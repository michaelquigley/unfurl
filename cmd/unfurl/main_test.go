package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandStdinToStdout(t *testing.T) {
	stdout, stderr, err := executeRoot(t, []string{}, "alpha\nbeta\n")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if stdout != "alpha\nbeta\n" {
		t.Fatalf("stdout mismatch:\nwant %q\n got %q", "alpha\nbeta\n", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr mismatch: %q", stderr)
	}
}

func TestRootCommandFileToStdout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.md")
	if err := os.WriteFile(path, []byte("file\ncontent\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stdout, _, err := executeRoot(t, []string{path}, "")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if stdout != "file\ncontent\n" {
		t.Fatalf("stdout mismatch:\nwant %q\n got %q", "file\ncontent\n", stdout)
	}
}

func TestRootCommandInPlaceRewritesFileAndPreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.md")
	if err := os.WriteFile(path, []byte("file\ncontent\n"), 0o640); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stdout, _, err := executeRoot(t, []string{"--in-place", path}, "")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if string(got) != "file\ncontent\n" {
		t.Fatalf("file mismatch:\nwant %q\n got %q", "file\ncontent\n", string(got))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat rewritten file: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o640 {
		t.Fatalf("mode mismatch: want %v got %v", os.FileMode(0o640), gotMode)
	}
}

func TestRootCommandInPlaceRequiresFileArgument(t *testing.T) {
	_, _, err := executeRoot(t, []string{"--in-place"}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--in-place requires a file argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommandMissingFileReturnsWrappedError(t *testing.T) {
	_, _, err := executeRoot(t, []string{filepath.Join(t.TempDir(), "missing.md")}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "open ") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeRoot(t *testing.T, args []string, stdin string) (string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	opts := &rootOptions{
		stdin:  strings.NewReader(stdin),
		stdout: &stdout,
		stderr: &stderr,
	}
	cmd := newRootCmd(opts)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
