// Package website serves the public site and its moderated testimonial inbox.
// These submissions are separate from Cully's owner-scoped agent memory.
package website

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Testimonial struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Context   string    `json:"context,omitempty"`
	Workplace string    `json:"workplace,omitempty"`
	LinkedIn  string    `json:"linkedin,omitempty"`
	Quote     string    `json:"quote"`
	PhotoURL  string    `json:"photoUrl,omitempty"`
	Consent   bool      `json:"consent"`
	CreatedAt time.Time `json:"createdAt"`
}

type Server struct {
	SiteDir string
	DataDir string
	Client  *http.Client
	mu      sync.Mutex
	window  time.Time
	count   int
	uploads chan struct{}
}

func New(site, data string) (*Server, error) {
	for _, state := range []string{"pending", "approved"} {
		if err := os.MkdirAll(filepath.Join(data, state), 0700); err != nil {
			return nil, err
		}
	}
	return &Server{SiteDir: site, DataDir: data, uploads: make(chan struct{}, 4), Client: &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 || !linkedInURL(req.URL) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/testimonials", s.submit)
	mux.HandleFunc("POST /api/linkedin-profile", s.importProfile)
	mux.HandleFunc("GET /testimonials.json", s.published)
	mux.HandleFunc("GET /testimonial-photos/{id}", s.photo)
	mux.HandleFunc("GET /install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, filepath.Join(s.SiteDir, "install.sh"))
	})
	mux.Handle("GET /", http.FileServer(http.Dir(s.SiteDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		mux.ServeHTTP(w, r)
	})
}

// A custom header prevents cross-origin form posts. No cross-origin preflight
// permissions are exposed. A bounded global budget also covers proxied traffic.
func (s *Server) allow(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("X-Cully-Submission") != "1" {
		problem(w, 403, "Submit using the form on cully.net.")
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || (u.Scheme != "https" && u.Scheme != "http") {
			problem(w, 403, "Submit using the form on cully.net.")
			return false
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.window) >= time.Minute {
		s.window, s.count = time.Now(), 0
	}
	if s.count >= MaxRequestsPerIPLimit {
		w.Header().Set("Retry-After", "60")
		problem(w, 429, "Too many requests. Please try again in a minute.")
		return false
	}
	s.count++
	return true
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	if !s.allow(w, r) {
		return
	}
	select {
	case s.uploads <- struct{}{}:
		defer func() { <-s.uploads }()
	default:
		problem(w, 429, "Uploads are busy. Please try again shortly.")
		return
	}
	pending, err := os.ReadDir(filepath.Join(s.DataDir, "pending"))
	if err != nil {
		problem(w, 500, "Could not save your submission. Please retry.")
		return
	}
	if len(pending) >= 100 {
		problem(w, 503, "The review inbox is full. Please try again later.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxPhotoSize+(64<<10))
	if err := r.ParseMultipartForm(MaxPhotoSize + (64 << 10)); err != nil {
		problem(w, 400, "Choose a photo up to 5 MB and retry your submission.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	if r.FormValue("website") != "" { // Bot honeypot.
		problem(w, 400, "Could not accept this submission.")
		return
	}
	t := Testimonial{Name: strings.TrimSpace(r.FormValue("name")),
		Context: strings.TrimSpace(r.FormValue("context")), Workplace: strings.TrimSpace(r.FormValue("workplace")),
		LinkedIn: strings.TrimSpace(r.FormValue("linkedin")), Quote: strings.TrimSpace(r.FormValue("quote")),
		Consent: r.FormValue("consent") == "on", CreatedAt: time.Now().UTC()}
	if err := validate(t); err != nil {
		problem(w, 400, err.Error())
		return
	}
	var photo []byte // defaulted nil slice. 
	f, _, err := r.FormFile("photo")
	if err == nil {
		defer f.Close()
		photo, err = io.ReadAll(io.LimitReader(f, MaxPhotoSize+1))
		if err != nil || len(photo) > MaxPhotoSize {
			problem(w, 400, "Choose a photo up to 5 MB.")
			return
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		problem(w, 400, "Could not read your photo.")
		return
	}
	if len(photo) == 0 && r.FormValue("importedPhoto") != "" {
		photo, err = s.fetchPhoto(r.Context(), r.FormValue("importedPhoto"))
		if err != nil {
			problem(w, 400, "The LinkedIn photo could not be saved. Please upload a photo instead.")
			return
		}
	}
	if len(photo) > 0 {
		if err := validatePhoto(photo); err != nil {
			problem(w, 400, err.Error())
			return
		}
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		problem(w, 500, "Could not save your submission. Please retry.")
		return
	}
	t.ID = hex.EncodeToString(id)
	if len(photo) > 0 {
		t.PhotoURL = "/testimonial-photos/" + t.ID
	}
	// Only complete submissions become visible to the moderation CLI.
	dir, err := os.MkdirTemp(s.DataDir, ".submission-")
	if err != nil {
		problem(w, 500, "Could not save your submission. Please retry.")
		return
	}
	defer os.RemoveAll(dir)
	if len(photo) > 0 {
		err = os.WriteFile(filepath.Join(dir, "photo"), photo, 0600)
	}
	if err == nil {
		var data []byte
		data, err = json.MarshalIndent(t, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, "testimonial.json"), data, 0600)
		}
	}
	if err == nil {
		err = os.Rename(dir, filepath.Join(s.DataDir, "pending", t.ID))
	}
	if err != nil {
		problem(w, 500, "Could not save your submission. Please retry.")
		return
	}
	respond(w, 201, map[string]string{"id": t.ID, "message": "Thank you! Your testimonial and photo were saved for review. They will appear here after approval."})
}

func validate(t Testimonial) error {
	if !t.Consent {
		return InvalidTestimonialConsentErr
	}
	if n := utf8.RuneCountInString(t.Name); n < 1 || n > 80 {
		return InvalidNameLengthErr
	}
	if n := utf8.RuneCountInString(t.Quote); n < 20 || n > 1000 {
		return InvalidTestimonialLengthErr
	}
	if utf8.RuneCountInString(t.Context) > 100 || utf8.RuneCountInString(t.Workplace) > 100 {
		return InvalidRolesLengthErr
	}
	if t.LinkedIn != "" {
		u, err := url.Parse(t.LinkedIn)
		if err != nil || len(t.LinkedIn) > 300 || !linkedInURL(u) {
			return InvalidLinkedinURLErr
		}
	}
	return nil
}

func validatePhoto(photo []byte) error {
	content := http.DetectContentType(photo)
	if content != "image/jpeg" && content != "image/png" && content != "image/webp" {
		return NonexistentImageErr
	}
	if content != "image/webp" {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(photo))
		if err != nil || cfg.Width > 4096 || cfg.Height > 4096 {
			return InvalidImageSizeErr
		}
	}
	return nil
}

func (s *Server) published(w http.ResponseWriter, r *http.Request) {
	entries, err := List(s.DataDir, "approved")
	if err != nil {
		problem(w, 500, "Could not load testimonials.")
		return
	}
	// Keep support for previously curated website testimonials.
	if data, err := os.ReadFile(filepath.Join(s.SiteDir, "testimonials.json")); err == nil {
		var curated []Testimonial
		if json.Unmarshal(data, &curated) == nil {
			entries = append(entries, curated...)
		}
	}
	w.Header().Set("Cache-Control", "no-cache")
	respond(w, 200, entries)
}

func (s *Server) photo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(filepath.Join(s.DataDir, "approved", id, "photo"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

func List(data, state string) ([]Testimonial, error) {
	if state != "pending" && state != "approved" {
		return nil, InvalidReviewStateErr
	}
	dirs, err := os.ReadDir(filepath.Join(data, state))
	if err != nil {
		return nil, err
	}
	entries := make([]Testimonial, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() || !validID.MatchString(dir.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(data, state, dir.Name(), "testimonial.json"))
		if err != nil {
			return nil, err
		}
		var t Testimonial
		if err := json.Unmarshal(b, &t); err != nil {
			return nil, err
		}
		entries = append(entries, t)
	}
	return entries, nil
}

// Review is available only through the operator's CLI, never a public route.
func Review(data, action, id string) error {
	if !validID.MatchString(id) {
		return InvalidSubmissionIDErr
	}
	source := filepath.Join(data, "pending", id)
	b, err := os.ReadFile(filepath.Join(source, "testimonial.json"))
	if err != nil {
		return err
	}
	var t Testimonial
	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}
	switch action {
	case "approve":
		if err := validate(t); err != nil {
			return err
		}
		return os.Rename(source, filepath.Join(data, "approved", id))
	case "reject":
		return os.RemoveAll(source)
	default:
		return fmt.Errorf("unknown review action %q", action)
	}
}

func respond(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(value)
}

func problem(w http.ResponseWriter, code int, message string) {
	respond(w, code, map[string]string{"error": message})
}
