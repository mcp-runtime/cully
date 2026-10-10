package website

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	site, data := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte("Cully website"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(site, data)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func photoFixture(t *testing.T) []byte {
	t.Helper()
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return photo.Bytes()
}

func submission(t *testing.T, fields map[string]string, photo []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := m.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if photo != nil {
		f, err := m.CreateFormFile("photo", "profile.png")
		if err != nil {
			t.Fatal(err)
		}
		f.Write(photo)
	}
	m.Close()
	r := httptest.NewRequest("POST", "https://cully.net/api/testimonials", &body)
	r.Header.Set("Content-Type", m.FormDataContentType())
	r.Header.Set("X-Cully-Submission", "1")
	r.Header.Set("Origin", "https://cully.net")
	return r
}

func fields() map[string]string {
	return map[string]string{"name": "Test Person", "workplace": "Test Team", "context": "Developer",
		"quote":    "Cully helped me find a useful project decision in the next session.",
		"linkedin": "https://www.linkedin.com/in/test-person", "consent": "on"}
}

func TestSubmissionReviewAndPersistence(t *testing.T) {
	s := testServer(t)
	photo := photoFixture(t)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, submission(t, fields(), photo))
	if w.Code != 201 {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	id := result["id"]
	if !validID.MatchString(id) {
		t.Fatalf("invalid ID %q", id)
	}
	for _, path := range []string{"/testimonials.json", "/testimonial-photos/" + id} {
		w = httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if path == "/testimonials.json" && strings.TrimSpace(w.Body.String()) != "[]" {
			t.Fatalf("pending submission exposed: %s", w.Body.String())
		}
		if strings.Contains(path, "/testimonial-photos/") && w.Code != 404 {
			t.Fatal("pending photo exposed")
		}
	}
	pending, err := List(s.DataDir, "pending")
	if err != nil || len(pending) != 1 || pending[0].Workplace != "Test Team" || !pending[0].Consent {
		t.Fatalf("stored data: %+v %v", pending, err)
	}
	if err := Review(s.DataDir, "approve", id); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(s.SiteDir, s.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	restarted.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/testimonials.json", nil))
	var approved []Testimonial
	json.Unmarshal(w.Body.Bytes(), &approved)
	if len(approved) != 1 || approved[0].ID != id {
		t.Fatalf("approved entries: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	restarted.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/testimonial-photos/"+id, nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), photo) || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("approved photo: %d", w.Code)
	}
	if err := Review(s.DataDir, "approve", "../../outside"); err == nil {
		t.Fatal("path traversal accepted")
	}
}

func TestRejectRemovesPendingPhotoAndRecord(t *testing.T) {
	s := testServer(t)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, submission(t, fields(), photoFixture(t)))
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	id := result["id"]
	if err := Review(s.DataDir, "reject", id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.DataDir, "pending", id)); !os.IsNotExist(err) {
		t.Fatal("rejected submission retained")
	}
}

func TestInvalidSubmissionsDoNotWriteRecords(t *testing.T) {
	for _, scenario := range []string{"consent", "short", "url", "origin", "header", "photo", "large-photo", "honeypot"} {
		t.Run(scenario, func(t *testing.T) {
			s := testServer(t)
			f := fields()
			var photo []byte
			switch scenario {
			case "consent":
				delete(f, "consent")
			case "short":
				f["quote"] = "too short"
			case "url":
				f["linkedin"] = "https://linkedin.com.evil.test/in/person"
			case "photo":
				photo = []byte("<svg onload='alert(1)'></svg>")
			case "large-photo":
				photo = make([]byte, MaxPhotoSize+1)
			case "honeypot":
				f["website"] = "bot"
			}
			r := submission(t, f, photo)
			if scenario == "origin" {
				r.Header.Set("Origin", "https://other.test")
			}
			if scenario == "header" {
				r.Header.Del("X-Cully-Submission")
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			if w.Code < 400 {
				t.Fatalf("accepted invalid submission: %d", w.Code)
			}
			pending, err := List(s.DataDir, "pending")
			if err != nil || len(pending) != 0 {
				t.Fatalf("invalid submission stored: %v", err)
			}
		})
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicLinkedInImportAndRestrictedFallback(t *testing.T) {
	s := testServer(t)
	calls := 0
	s.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Hostname() != "www.linkedin.com" {
			t.Fatalf("unexpected upstream: %s", r.URL)
		}
		body := `<meta content="Test Person - Developer | LinkedIn" property="og:title"><meta property='og:image' content='https://media.licdn.com/profile.png'><meta property="og:description" content="Experience: Test Team · Location: Somewhere">`
		code := 200
		if calls == 2 {
			code, body = 999, "Restricted"
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	for i, want := range []int{200, 502, 400} {
		link := "https://www.linkedin.com/in/test-person"
		if i == 2 {
			link = "https://127.0.0.1/in/private"
		}
		r := httptest.NewRequest("POST", "https://cully.net/api/linkedin-profile", strings.NewReader(`{"url":"`+link+`"}`))
		r.Header.Set("X-Cully-Submission", "1")
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("import: %d %s", w.Code, w.Body.String())
		}
		if i == 0 && !strings.Contains(w.Body.String(), `"workplace":"Test Team"`) {
			t.Fatalf("profile details: %s", w.Body.String())
		}
	}
	if calls != 2 {
		t.Fatal("invalid host was fetched")
	}
	for _, raw := range []string{"https://www.linkedin.com/in/name", "https://media.licdn.com/photo", "https://127.0.0.1/in/name"} {
		u, _ := url.Parse(raw)
		err := s.Client.CheckRedirect(&http.Request{URL: u}, []*http.Request{{}})
		if (err == nil) != linkedInURL(u) {
			t.Fatalf("unsafe redirect policy: %s", raw)
		}
	}
	if _, err := s.fetchPhoto(context.Background(), "https://127.0.0.1/photo"); err == nil {
		t.Fatal("unsafe photo URL accepted")
	}
}

func TestRateLimit(t *testing.T) {
	s := testServer(t)

	// temporary: ensuring fixed-window rate limiter state before loop
	s.mu.Lock()
	s.window = time.Now()
	s.count = 0
	s.mu.Unlock()

	for i := 0; i < 21; i++ {
		req := submission(t, fields(), nil)
		req.Host = "cully.net"

		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		if i < 20 && w.Code != 201 || i == 20 && w.Code != 429 {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
}
