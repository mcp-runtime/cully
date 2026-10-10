package website

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var metaTag = regexp.MustCompile(`(?is)<meta\s+[^>]*>`)
var metaAttr = regexp.MustCompile(`(?i)(property|name|content)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
var experience = regexp.MustCompile(`(?i)Experience:\s*([^·•\n]+)`)

func linkedInURL(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.User == nil && u.Port() == "" &&
		(strings.EqualFold(u.Hostname(), "www.linkedin.com") || strings.EqualFold(u.Hostname(), "linkedin.com")) &&
		strings.HasPrefix(u.Path, "/in/") && len(u.Path) > 4
}

func linkedInPhotoURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Hostname() != "media.licdn.com" {
		return nil, InvalidLinkedinPhotoErr
	}
	return u, nil
}

// This reads only metadata LinkedIn chooses to expose without authentication.
// It never uses member credentials or bypasses sign-in and access controls.
func (s *Server) importProfile(w http.ResponseWriter, r *http.Request) {
	if !s.allow(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxHTMLBodyReader)
	var input struct {
		URL string `json:"url"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		problem(w, 400, "Enter a LinkedIn profile URL.")
		return
	}

	

	u, err := url.Parse(input.URL)
	if err != nil || !linkedInURL(u) || len(input.URL) > 300 {
		problem(w, 400, "Enter a profile URL starting with https://www.linkedin.com/in/.")
		return
	}
	u.RawQuery, u.Fragment = "", ""
	req, err := http.NewRequestWithContext(r.Context(), "GET", u.String(), nil)
	if err != nil {
		problem(w, 400, "Enter a valid LinkedIn profile URL.")
		return
	}
	req.Header.Set("User-Agent", "Cully/1.0 (+https://cully.net; user-requested public profile preview)")
	response, err := s.Client.Do(req)
	if err != nil {
		problem(w, 502, "LinkedIn could not be reached. Please fill your details and upload a photo manually.")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		problem(w, 502, "LinkedIn did not make this profile available. Please fill your details and upload a photo manually.")
		return
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (512<<10)+1))
	if err != nil || len(body) > 512<<10 {
		problem(w, 502, "Could not read this profile. Please fill your details manually.")
		return
	}
	metadata := publicMetadata(string(body))
	title := metadata["og:title"]
	// Sign-in pages also have Open Graph metadata; they are not profiles.
	if title == "" || !strings.HasSuffix(title, " | LinkedIn") {
		problem(w, 502, "This profile has no public details to import. Please fill your details and upload a photo manually.")
		return
	}
	title = strings.TrimSuffix(title, " | LinkedIn")
	parts := strings.SplitN(title, " - ", 2)
	if strings.TrimSpace(parts[0]) == "" || len(parts) < 2 {
		problem(w, 502, "This profile has no public details to import. Please fill your details manually.")
		return
	}
	workplace := ""
	if matches := experience.FindStringSubmatch(metadata["og:description"]); len(matches) == 2 {
		workplace = strings.TrimSpace(matches[1])
	}
	photo := metadata["og:image"]
	if _, err := linkedInPhotoURL(photo); err != nil {
		photo = ""
	}
	respond(w, 200, map[string]string{"name": strings.TrimSpace(parts[0]), "context": strings.TrimSpace(parts[1]), "workplace": workplace, "photoUrl": photo})
}

func publicMetadata(page string) map[string]string {
	metadata := make(map[string]string)
	for _, tag := range metaTag.FindAllString(page, -1) {
		key, value := "", ""
		for _, attr := range metaAttr.FindAllStringSubmatch(tag, -1) {
			text := attr[2]
			if text == "" {
				text = attr[3]
			}
			if strings.EqualFold(attr[1], "content") {
				value = html.UnescapeString(text)
			} else {
				key = strings.ToLower(text)
			}
		}
		metadata[key] = value
	}
	return metadata
}

func (s *Server) fetchPhoto(ctx context.Context, raw string) ([]byte, error) {
	u, err := linkedInPhotoURL(raw)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, PhotoUnavailableErr
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxPhotoSize+1))
	if err != nil || len(data) > MaxPhotoSize {
		return nil, PhotoTooLargeErr
	}
	return data, validatePhoto(data)
}
