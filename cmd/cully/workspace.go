package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mcp-runtime/cully/internal/workspace"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type workspaceTransport struct{ token string }

func (t workspaceTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(r)
}

func runWorkspace(args []string) error {
	fs := flag.NewFlagSet("workspace", flag.ContinueOnError)
	endpoint := fs.String("mcp-url", os.Getenv("CULLY_MCP_URL"), "OAuth-protected public MCP endpoint")
	file := fs.String("input", "-", "workspace action JSON file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: cully workspace --mcp-url URL --input FILE")
	}
	u, err := url.Parse(*endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		return fmt.Errorf("workspace requires an HTTPS MCP URL (HTTP is allowed on loopback)")
	}
	token := os.Getenv("CULLY_WORKSPACE_TOKEN")
	if token == "" {
		return fmt.Errorf("set CULLY_WORKSPACE_TOKEN to a short-lived user OAuth access token; or use your agent's configured Cully MCP tools")
	}
	var input io.Reader = os.Stdin
	if *file != "-" {
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	}
	d := json.NewDecoder(io.LimitReader(input, (128<<10)+1))
	d.DisallowUnknownFields()
	var v workspace.Input
	if err = d.Decode(&v); err != nil {
		return fmt.Errorf("invalid workspace JSON")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("provide exactly one workspace action")
	}
	if err = v.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := sdk.NewClient(&sdk.Implementation{Name: "cully-workspace", Version: version}, nil)
	httpClient := &http.Client{Transport: workspaceTransport{token: token}, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: *endpoint, HTTPClient: httpClient, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return fmt.Errorf("workspace MCP connection failed; check endpoint and user sign-in")
	}
	defer session.Close()
	name := "cully_workspace_read"
	if v.Write() {
		name = "cully_workspace_write"
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: v})
	if err != nil {
		return fmt.Errorf("workspace MCP request failed")
	}
	if result.IsError {
		for _, c := range result.Content {
			if text, ok := c.(*sdk.TextContent); ok {
				return fmt.Errorf("workspace: %s", text.Text)
			}
		}
		return fmt.Errorf("workspace action rejected")
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result.StructuredContent)
}
