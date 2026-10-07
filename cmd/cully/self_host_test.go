package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func stackArchive(t *testing.T, name string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipper := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(zipper)
	data := []byte("#!/bin/sh\n")
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestStackRef(t *testing.T) {
	for _, test := range []struct{ version, expected string }{
		{"dev", "main"},
		{"0.3.0", "v0.3.0"},
		{"v0.3.0", "v0.3.0"},
	} {
		got, err := stackRef(test.version)
		if err != nil || got != test.expected {
			t.Fatalf("stackRef(%q) = %q, %v", test.version, got, err)
		}
	}
	if _, err := stackRef("../main"); err == nil {
		t.Fatal("accepted unsafe stack ref")
	}
}

func TestExtractStackKeepsFilesInsideManagedDirectory(t *testing.T) {
	directory := t.TempDir()
	valid := stackArchive(t, "cully-v0.3.0/deploy/self-hosted/setup.sh")
	if err := extractStack(bytes.NewReader(valid), directory); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "deploy", "self-hosted", "setup.sh")); err != nil {
		t.Fatal(err)
	}
	unsafe := stackArchive(t, "cully-v0.3.0/../../outside")
	if err := extractStack(bytes.NewReader(unsafe), directory); err == nil {
		t.Fatal("accepted archive path traversal")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(directory), "outside")); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside managed directory: %v", err)
	}
}

func TestAgentSetupCommandRemoved(t *testing.T) {
	if err := run([]string{"agent", "setup", "codex"}); err == nil {
		t.Fatal("removed agent setup command still works")
	}
}

func TestLinkStackConfigKeepsPrivateSettingsAcrossRuns(t *testing.T) {
	root := t.TempDir()
	setupDir := filepath.Join(root, "release", "deploy", "self-hosted")
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(setupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		".env.example":                     "CULLY_MCP_HOST=mcp.example.com\n",
		"connectors.keycloak.example.json": "{}\n",
	} {
		if err := os.WriteFile(filepath.Join(setupDir, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := linkStackConfig(setupDir, configDir); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(configDir, ".env")
	if err := os.WriteFile(envFile, []byte("CULLY_MCP_HOST=mine.example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linkStackConfig(setupDir, configDir); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(setupDir, ".env"))
	if err != nil || string(contents) != "CULLY_MCP_HOST=mine.example\n" {
		t.Fatalf("configuration was replaced: %q, %v", contents, err)
	}
	info, err := os.Stat(envFile)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("configuration is not private: %v, %v", info, err)
	}
}

func TestExtractStackSkipsGlobalPAXHeader(t *testing.T) {
	var buffer bytes.Buffer
	zipper := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(zipper)
	if err := writer.WriteHeader(&tar.Header{
		Name:       "pax_global_header",
		Typeflag:   tar.TypeXGlobalHeader,
		PAXRecords: map[string]string{"comment": "GitHub source archive"},
	}); err != nil {
		t.Fatal(err)
	}
	name := "cully-0.4.1/deploy/self-hosted/setup.sh"
	content := []byte("#!/bin/sh\n")
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := extractStack(bytes.NewReader(buffer.Bytes()), directory); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "deploy", "self-hosted", "setup.sh")); err != nil {
		t.Fatal(err)
	}
}
