// Package remote keeps database credentials out of the public MCP service.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mcp-runtime/cully/internal/memory"
	"github.com/mcp-runtime/cully/internal/transport/datahttp"
)

type Store struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func New(base, token string) (*Store, error) {
	u, e := url.Parse(base)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || token == "" {
		return nil, fmt.Errorf("CULLY_DATA_API_URL and CULLY_DATA_API_TOKEN are required")
	}
	return &Store{BaseURL: strings.TrimRight(base, "/"), Token: token, Client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (s *Store) Execute(ctx context.Context, owner string, input memory.Request) (memory.Result, error) {
	out := memory.Result{}
	data, err := json.Marshal(datahttp.Envelope{Owner: owner, Request: input})
	if err != nil {
		return out, memory.ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.BaseURL+"/v1/memory", bytes.NewReader(data))
	if err != nil {
		return out, memory.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return out, memory.ErrUnavailable
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
	case 400:
		return out, memory.ErrInvalid
	case 403:
		return out, memory.ErrForbidden
	case 409:
		return out, memory.ErrConflict
	default:
		return out, memory.ErrUnavailable
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&out); err != nil {
		return out, memory.ErrUnavailable
	}
	return out, nil
}
