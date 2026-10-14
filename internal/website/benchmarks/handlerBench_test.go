package benchmarks

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mcp-runtime/cully/internal/website"
)

// read-only routes.

func BenchmarkPublishedHandler(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	assertBenchmarkResponse(b, h, httptest.NewRequest(http.MethodGet, "/testimonials.json", nil), http.StatusOK, "[")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/testimonials.json", nil)
		for p.Next() {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
		}
	})
}

func BenchmarkPhotoHandler404(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	assertBenchmarkResponse(b, h, httptest.NewRequest(http.MethodGet, "/testimonial-photos/10", nil), http.StatusNotFound, "404 page not found")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		// assuming http.NotFound() responses:
		req := httptest.NewRequest(http.MethodGet, "/testimonial-photos/10", nil)
		for p.Next() {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
		}
	})
}

func BenchmarkPhotoHandler(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	id := "00000000000000000000000000000010" // 32 bit integer "10"
	dir := filepath.Join(s.DataDir, "approved", id)
	photoData := []byte("example photo")
	if err3 := os.MkdirAll(dir, os.ModePerm); err3 != nil {
		panic(err3)
	}
	if err2 := os.WriteFile(filepath.Join(dir, "photo"), photoData, os.ModePerm); err2 != nil {
		panic(err2)
	}
	assertBenchmarkResponse(b, h, httptest.NewRequest(http.MethodGet, "/testimonial-photos/"+id, nil), http.StatusOK, "example photo")
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			req := httptest.NewRequest(http.MethodGet, "/testimonial-photos/"+id, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				b.Fatalf("failed with status %d", rr.Code)
				return
			}
		}
	})
}

func BenchmarkInstillationHandler(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	assertBenchmarkResponse(b, h, httptest.NewRequest(http.MethodGet, "/install.sh", nil), http.StatusOK, "Cully installer")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
		for p.Next() {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
		}
	})
}

// write-included handlers.

func BenchmarkSubmitHandler(b *testing.B) {
	fixture := setupBenchmarkFixture(b)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	field := map[string]string{
		"name":    "eg.1",
		"quote":   "some testimonialssss",
		"consent": "on",
	}

	for key, value := range field {
		if err := mw.WriteField(key, value); err != nil {
			panic(err)
		}
	}

	if err := mw.Close(); err != nil {
		panic(err)
	}

	payload := append([]byte(nil), body.Bytes()...)
	contentType := mw.FormDataContentType()
	benchmarkIsolatedRequest(b, fixture, http.MethodPost, "/api/testimonials", payload,
		http.Header{"Content-Type": []string{contentType}, "X-Cully-Submission": []string{"1"}}, nil,
		http.StatusCreated, int64(len(payload)), "Thank you!")
}

func BenchmarkProfileImportHandler(b *testing.B) {
	fixture := setupBenchmarkFixture(b)
	profile := struct {
		URL string `json:"url"`
	}{URL: "https://www.linkedin.com/in/example-person"}
	js, err := json.Marshal(profile)
	if err != nil {
		b.Fatal(err)
	}
	benchmarkIsolatedRequest(b, fixture, http.MethodPost, "/api/linkedin-profile", js,
		http.Header{"X-Cully-Submission": []string{"1"}}, func(s *website.Server) {
			s.Client.Transport = &examReq{}
		}, http.StatusOK, int64(len(js)), `"name":"Example Person"`, `"context":"Engineer"`, `"workplace":"Example Co"`)
}

type examReq struct{}

func (s *examReq) RoundTrip(req *http.Request) (*http.Response, error) {
	const data = `<html><head><meta property="og:title" content="Example Person - Engineer | LinkedIn"><meta property="og:description" content="Experience: Example Co · Software Engineer"></head></html>`

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(data)),
		Request:    req,
	}, nil
}

func SetupBenchmarkHelper(b *testing.B) *website.Server {
	return setupBenchmarkFixture(b).newServer(b)
}

type benchmarkFixture struct {
	siteDir string
	dataDir string
}

func setupBenchmarkFixture(b *testing.B) benchmarkFixture {
	b.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("locate benchmark source")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	siteDir := b.TempDir()
	for _, file := range []struct{ source, target string }{
		{filepath.Join(repoRoot, "site", "index.html"), filepath.Join(siteDir, "index.html")},
		{filepath.Join(repoRoot, "site", "testimonials.json"), filepath.Join(siteDir, "testimonials.json")},
		{filepath.Join(repoRoot, "install.sh"), filepath.Join(siteDir, "install.sh")},
	} {
		data, err := os.ReadFile(file.source)
		if err != nil {
			b.Fatalf("read benchmark fixture %s: %v", file.source, err)
		}
		if err := os.WriteFile(file.target, data, 0600); err != nil {
			b.Fatalf("write benchmark fixture %s: %v", file.target, err)
		}
	}
	return benchmarkFixture{siteDir: siteDir, dataDir: b.TempDir()}
}

func (f benchmarkFixture) newServer(b *testing.B) *website.Server {
	b.Helper()
	dataDir, err := os.MkdirTemp(f.dataDir, "request-")
	if err != nil {
		b.Fatal(err)
	}
	s, err := website.New(f.siteDir, dataDir)
	if err != nil {
		b.Fatal(err)
	}
	return s
}

func benchmarkIsolatedRequest(b *testing.B, fixture benchmarkFixture, method, path string, body []byte, headers http.Header, configure func(*website.Server), wantStatus int, bytesPerOp int64, wantContents ...string) {
	b.Helper()
	check := func(response *httptest.ResponseRecorder) {
		responseBody := response.Body.String()
		if response.Code != wantStatus {
			b.Fatalf("%s %s: status=%d body=%q; want status=%d", method, path, response.Code, responseBody, wantStatus)
		}
		for _, wantContent := range wantContents {
			if !strings.Contains(responseBody, wantContent) {
				b.Fatalf("%s %s: status=%d body=%q; want body containing %q", method, path, response.Code, responseBody, wantContent)
			}
		}
	}

	probeServer := fixture.newServer(b)
	if configure != nil {
		configure(probeServer)
	}
	probe := httptest.NewRecorder()
	probeRequest := httptest.NewRequest(method, path, bytes.NewReader(body))
	probeRequest.Header = headers.Clone()
	probeServer.Handler().ServeHTTP(probe, probeRequest)
	check(probe)
	if err := os.RemoveAll(probeServer.DataDir); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(bytesPerOp)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// A fresh default-configured server per request keeps rate-limit,
		// upload, and pending-inbox state out of the measured workload.
		b.StopTimer()
		server := fixture.newServer(b)
		if configure != nil {
			configure(server)
		}
		handler := server.Handler()
		b.StartTimer()

		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, bytes.NewReader(body))
		request.Header = headers.Clone()
		handler.ServeHTTP(response, request)

		b.StopTimer()
		check(response)
		if err := os.RemoveAll(server.DataDir); err != nil {
			b.Fatal(err)
		}
	}
}

func assertBenchmarkResponse(b *testing.B, handler http.Handler, request *http.Request, wantStatus int, wantContent string) {
	b.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus || !strings.Contains(response.Body.String(), wantContent) {
		b.Fatalf("%s %s: status=%d body=%q; want status=%d and body containing %q", request.Method, request.URL.Path, response.Code, response.Body.String(), wantStatus, wantContent)
	}
}
