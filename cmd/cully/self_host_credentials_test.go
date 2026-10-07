package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSelfHostCredentialsArePrivateAndStable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".cully", "config.json")
	env := filepath.Join(root, ".env")
	if err := os.WriteFile(env, []byte("CULLY_DB_PASSWORD=existing-db\nCULLY_MCP_OWNER=existing-owner\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := ensureSelfHostCredentials(path, env)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ensureSelfHostCredentials(path, env)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("rerun changed service credentials")
	}
	if first["db_password"] != "existing-db" || first["mcp_owner"] != "existing-owner" {
		t.Fatal("existing credentials were not imported")
	}
	if first["data_api_token"] == first["mem0_api_key"] || first["db_password"] == first["mem0_db_password"] {
		t.Fatal("service credentials were reused across services")
	}
	for _, item := range []struct {
		path string
		mode os.FileMode
	}{{filepath.Dir(path), 0o700}, {path, 0o600}} {
		info, err := os.Stat(item.path)
		if err != nil || info.Mode().Perm() != item.mode {
			t.Fatalf("wrong private mode for %s: %v, %v", item.path, info, err)
		}
	}
	stored, err := loadSelfHostCredentials(path)
	if err != nil || !reflect.DeepEqual(first, stored) {
		t.Fatalf("credentials did not reload: %v", err)
	}
}

func TestSelfHostCredentialsPreserveConfigAndRejectUnsafePaths(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, ".cully")
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "config.json")
	if err := os.WriteFile(path, []byte(`{"agent":{"enabled":true},"self_hosted":{"db_password":"kept","custom":"value"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureSelfHostCredentials(path, filepath.Join(root, "missing.env")); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatal(err)
	}
	var agent map[string]bool
	var section map[string]string
	if err := json.Unmarshal(document["agent"], &agent); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(document["self_hosted"], &section); err != nil {
		t.Fatal(err)
	}
	if !agent["enabled"] || section["custom"] != "value" || section["db_password"] != "kept" {
		t.Fatal("existing config fields were lost")
	}
	link := filepath.Join(private, "linked.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureSelfHostCredentials(link, filepath.Join(root, "missing.env")); err == nil {
		t.Fatal("accepted a symlinked credential file")
	}
	shared := filepath.Join(root, "shared")
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureSelfHostCredentials(filepath.Join(shared, "config.json"), filepath.Join(root, "missing.env")); err == nil {
		t.Fatal("accepted a shared credential directory")
	}
}

func TestSelfHostOAuthDerivesURLsAndChecksConnector(t *testing.T) {
	for _, key := range []string{"CULLY_MCP_HOST", "CULLY_MCP_AUTH_CONNECTOR", "CULLY_JWKS_URL", "MCP_AUTH_UPSTREAM_CLIENT_SECRET", "CULLY_AUTH_HOST", "CULLY_AUTH_ISSUER", "CULLY_AUTH_RESOURCE"} {
		t.Setenv(key, "")
	}
	root := t.TempDir()
	env := "CULLY_MCP_HOST=mcp.acme.test\nCULLY_AUTH_HOST=auth.acme.test\nMCP_AUTH_UPSTREAM_CLIENT_SECRET=private-value\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	connectors := `{"org":{"client_secret_env":"MCP_AUTH_UPSTREAM_CLIENT_SECRET"}}`
	if err := os.WriteFile(filepath.Join(root, "connectors.json"), []byte(connectors), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := selfHostOAuthValues(root)
	if err != nil {
		t.Fatal(err)
	}
	if values["CULLY_AUTH_RESOURCE"] != "https://mcp.acme.test/mcp" || values["CULLY_AUTH_ISSUER"] != "https://auth.acme.test/mcp-auth" || values["CULLY_JWKS_URL"] != "https://auth.acme.test/mcp-auth/.well-known/jwks.json" || values["CULLY_MCP_AUTH_CONNECTOR"] != "org" {
		t.Fatalf("unexpected OAuth settings: %v", values)
	}
	if err := os.WriteFile(filepath.Join(root, "connectors.json"), []byte(`{"other":{"client_secret_env":"MCP_AUTH_UPSTREAM_CLIENT_SECRET"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CULLY_MCP_AUTH_CONNECTOR", "org")
	if _, err := selfHostOAuthValues(root); err == nil {
		t.Fatal("accepted a missing selected connector")
	}
}

func TestSelfHostCredentialsImportDatabaseURLAndQuoteShellValues(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".cully", "config.json")
	env := filepath.Join(root, ".env")
	if err := os.WriteFile(env, []byte("CULLY_DATABASE_URL=postgres://cully:old%27password@db:5432/cully\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := ensureSelfHostCredentials(path, env)
	if err != nil {
		t.Fatal(err)
	}
	if values["db_password"] != "old'password" {
		t.Fatal("existing database URL password was not imported")
	}
	value := "a'b;$(echo unsafe)"
	command := exec.Command("/bin/sh", "-c", "value="+shellQuote(value)+"; printf %s \"$value\"")
	output, err := command.Output()
	if err != nil || string(output) != value {
		t.Fatalf("shell export quoting failed: %q, %v", output, err)
	}
}
