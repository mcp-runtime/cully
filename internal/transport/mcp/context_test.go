package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mcp-runtime/cully/internal/memory"
)

type contextStore struct {
	owner   string
	request memory.Request
	entries []memory.Entry
}

func (s *contextStore) Execute(_ context.Context, owner string, request memory.Request) (memory.Result, error) {
	s.owner, s.request = owner, request
	return memory.Result{Entries: s.entries}, nil
}

func TestCompactContextBoundsAndOwner(t *testing.T) {
	long := strings.Repeat("useful detail ", 90)
	ref := "codex-0123456789abcdef"
	agentID := "01a13b57-f08a-77a1-a18f-5daaf9d50ec4"
	store := &contextStore{}
	for i := 0; i < 4; i++ {
		store.entries = append(store.entries, memory.Entry{
			ID: "id", OccurredAt: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC),
			Summary: long, Approach: &long, Outcome: &long, Issue: &long, Learning: &long, NextSteps: &long, SessionRef: &ref, AgentSessionID: &agentID,
		})
	}
	project := "https://github.com/Owner/Repo"
	out, err := compactContext(context.Background(), memory.Service{Store: store}, "owner-a", ContextInput{Query: "memory", ProjectURL: &project, SessionRef: &ref})
	if err != nil {
		t.Fatal(err)
	}
	if store.owner != "owner-a" || store.request.Operation != "search" || store.request.Search.Limit != 4 {
		t.Fatalf("owner or bounded request lost: %+v", store)
	}
	if store.request.Search.ProjectURL == nil || *store.request.Search.ProjectURL != "https://github.com/owner/repo" {
		t.Fatalf("project validation skipped: %+v", store.request.Search)
	}
	if store.request.Search.SessionRef == nil || *store.request.Search.SessionRef != ref || out.Notes[0].SessionRef == nil || *out.Notes[0].SessionRef != ref {
		t.Fatal("session reference lost")
	}
	if out.Notes[0].AgentSessionID == nil || *out.Notes[0].AgentSessionID != agentID {
		t.Fatal("resumable agent session ID lost")
	}
	if len(out.Notes) != 3 || !out.HasMore || out.Source != "text" {
		t.Fatalf("bad context page: %+v", out)
	}
	if len([]rune(out.Notes[0].Summary)) > 180 || len([]rune(out.Notes[0].Approach)) > 120 || strings.Contains(out.Notes[0].Summary, long) {
		t.Fatalf("unbounded context: %+v", out.Notes[0])
	}
}

func TestCompactContextUsesSharedValidation(t *testing.T) {
	store := &contextStore{}
	service := memory.Service{Store: store}
	badProject := "https://github.com.evil.test/org/repo"
	badRef := "native-session-id"
	for _, input := range []ContextInput{
		{Mode: "semantic"},
		{Mode: "text", Query: "  "},
		{Mode: "recent", Query: "oauth"},
		{Mode: "recent", ProjectURL: &badProject},
		{Mode: "recent", SessionRef: &badRef},
		{Mode: "bogus"},
	} {
		if _, err := compactContext(context.Background(), service, "owner-a", input); !errors.Is(err, memory.ErrInvalid) {
			t.Fatalf("accepted invalid context %+v: %v", input, err)
		}
	}
	if store.owner != "" {
		t.Fatal("invalid request reached store")
	}
}
