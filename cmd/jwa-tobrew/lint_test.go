package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunLintJSONReportsPolicyFailures(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(t, func() error {
		return runLint([]string{"--repo", dir, "--kind", "generic", "--binary", "", "--format", "json"})
	})

	var report lintReport
	if err := json.Unmarshal([]byte(out.stdout), &report); err != nil {
		t.Fatalf("lint json did not decode: %v\n%s", err, out.stdout)
	}
	if report.Repo != filepath.Base(dir) {
		t.Fatalf("repo = %q, want %q", report.Repo, filepath.Base(dir))
	}
	if report.Kind != "generic" {
		t.Fatalf("kind = %q, want generic", report.Kind)
	}
	if report.Failed == 0 {
		t.Fatalf("failed = 0, want policy failures in empty repo")
	}
	var exit exitCodeError
	if !errors.As(out.err, &exit) {
		t.Fatalf("error = %v, want exitCodeError", out.err)
	}
	if exit.code != 4 {
		t.Fatalf("exit code = %d, want 4", exit.code)
	}
	if !lintReportHasStatus(report, "F-repo-004", "fail") {
		t.Fatalf("report missing failed F-repo-004 result: %#v", report.Results)
	}
}

type capturedOutput struct {
	stdout string
	err    error
}

func captureStdout(t *testing.T, fn func() error) capturedOutput {
	t.Helper()
	oldStdout := os.Stdout
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writeEnd
	runErr := fn()
	if err := writeEnd.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	os.Stdout = oldStdout
	body, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return capturedOutput{stdout: string(body), err: runErr}
}

func lintReportHasStatus(report lintReport, id, status string) bool {
	for _, result := range report.Results {
		if result.ID == id && result.Status == status {
			return true
		}
	}
	return false
}
