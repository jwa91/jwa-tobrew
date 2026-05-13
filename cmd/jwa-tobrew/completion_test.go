package main

import (
	"strings"
	"testing"
)

func TestRunCompletionIncludesEveryCommand(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			out := captureStdout(t, func() error {
				return runCompletion([]string{shell})
			})
			if out.err != nil {
				t.Fatalf("runCompletion(%s): %v", shell, out.err)
			}
			for _, cmd := range commandSpecs {
				if !strings.Contains(out.stdout, cmd.Name) {
					t.Fatalf("%s completion missing command %q:\n%s", shell, cmd.Name, out.stdout)
				}
			}
		})
	}
}

func TestRunCompletionRejectsUnknownShell(t *testing.T) {
	out := captureStdout(t, func() error {
		return runCompletion([]string{"powershell"})
	})
	if out.err == nil {
		t.Fatal("runCompletion accepted an unknown shell")
	}
	if !strings.Contains(out.err.Error(), "unknown shell") {
		t.Fatalf("error = %v, want unknown shell", out.err)
	}
}
