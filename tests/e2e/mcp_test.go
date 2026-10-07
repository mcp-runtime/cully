package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/mcp-runtime/cully/internal/memory"
	cullymcp "github.com/mcp-runtime/cully/internal/transport/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestContainerStack crosses the live MCP, data API, PostgreSQL and Mem0 services.
// The setup workflow supplies a disposable stack; local unit runs skip it.
func TestContainerStack(t *testing.T) {
	endpoint := os.Getenv("CULLY_E2E_MCP_URL")
	if endpoint == "" {
		t.Skip("CULLY_E2E_MCP_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := sdk.NewClient(&sdk.Implementation{Name: "cully-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: endpoint, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) == 0 {
		t.Fatalf("list tools: %v, %+v", err, tools)
	}
	call := func(name string, input any) memory.Result {
		t.Helper()
		response, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: input})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if response.IsError {
			data, _ := json.Marshal(response)
			t.Fatalf("%s returned a tool error: %s", name, data)
		}
		data, err := json.Marshal(response.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var result memory.Result
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	projectURL := "https://github.com/mcp-runtime/cully"
	logged := call("cully_log", memory.LogInput{Summary: "Disposable container end-to-end check", Assistant: "other", Section: "personal", ProjectURL: &projectURL}).Entry
	if logged == nil || logged.ID == "" {
		t.Fatal("MCP write did not reach PostgreSQL")
	}
	id := memory.IDInput{EntryID: logged.ID}
	got := call("cully_get", id).Entry
	if got == nil || got.Summary != logged.Summary {
		t.Fatalf("MCP read did not return the written record: %+v", got)
	}
	found := func(entries []memory.Entry) bool {
		for _, entry := range entries {
			if entry.ID == logged.ID {
				return true
			}
		}
		return false
	}
	if !found(call("cully_search", memory.SearchInput{Query: logged.Summary}).Entries) {
		t.Fatal("PostgreSQL text search did not find the written record")
	}
	if !found(call("cully_recent", memory.RecentInput{Limit: 20}).Entries) {
		t.Fatal("recent records did not include the written record")
	}
	projects := call("cully_projects", memory.ProjectsInput{Limit: 20}).Projects
	projectFound := false
	for _, project := range projects {
		projectFound = projectFound || project.ProjectURL == projectURL
	}
	if !projectFound {
		t.Fatal("project list did not include the written record")
	}
	preview, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "cully_context", Arguments: cullymcp.ContextInput{ProjectURL: &projectURL, Mode: "recent"}})
	if err != nil || preview.IsError {
		t.Fatalf("context preview failed: %v, %+v", err, preview)
	}
	previewJSON, err := json.Marshal(preview.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var contextResult cullymcp.ContextResult
	if err := json.Unmarshal(previewJSON, &contextResult); err != nil || len(contextResult.Notes) == 0 || contextResult.Notes[0].ID != logged.ID {
		t.Fatalf("context preview did not include the written record: %+v, %v", contextResult, err)
	}
	for {
		if found(call("cully_recall", memory.SearchInput{Query: logged.Summary}).Entries) {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("Mem0 did not index and recall the written record before timeout")
		}
		time.Sleep(2 * time.Second)
	}
	updatedSummary := logged.Summary + " updated"
	updated := call("cully_update", memory.UpdateInput{EntryID: logged.ID, Summary: &updatedSummary}).Entry
	if updated == nil || updated.Summary != updatedSummary {
		t.Fatalf("MCP update did not change the record: %+v", updated)
	}
	if !call("cully_delete", id).Deleted {
		t.Fatal("MCP delete did not delete the record")
	}
	if found(call("cully_search", memory.SearchInput{Query: updatedSummary}).Entries) {
		t.Fatal("deleted record remained visible to text search")
	}
	if found(call("cully_recall", memory.SearchInput{Query: logged.Summary}).Entries) {
		t.Fatal("deleted record remained visible to semantic recall")
	}
}
