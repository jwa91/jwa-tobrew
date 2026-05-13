package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

type lintResult struct {
	ID          string `json:"id"`
	Layer       string `json:"layer"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Message     string `json:"message,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

type lintReport struct {
	Repo    string       `json:"repo"`
	Kind    string       `json:"kind"`
	Passed  int          `json:"passed"`
	Warned  int          `json:"warned"`
	Failed  int          `json:"failed"`
	Skipped int          `json:"skipped"`
	Results []lintResult `json:"results"`
}

func runLint(args []string) error {
	fs := subFlagSet("lint", "run the jwa-* family policy lint rules")
	repo := fs.String("repo", ".", "repository root to lint")
	kindFlag := fs.String("kind", "auto", "repo kind: auto | go-cli | swift-cask | cask | formula | generic | vps")
	binary := fs.String("binary", os.Args[0], "binary to use for command-surface checks (empty to skip)")
	format := fs.String("format", "human", "output format: human | json")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	kind, err := parseRepoKind(*kindFlag)
	if err != nil {
		return err
	}
	var opts []familylint.Option
	if *binary != "" {
		opts = append(opts, familylint.WithBinary(*binary))
	}
	if kind != "" {
		opts = append(opts, familylint.WithRepoKind(kind))
	}
	ctx, err := familylint.NewContext(*repo, opts...)
	if err != nil {
		return err
	}

	report := buildLintReport(ctx)

	switch *format {
	case "human":
		printLintHuman(report)
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown --format %q", *format)
	}
	if report.Failed > 0 {
		return exitCodeError{code: 4}
	}
	return nil
}

func buildLintReport(ctx *familylint.Context) lintReport {
	report := lintReport{
		Repo: filepathBase(ctx.RepoRoot),
		Kind: string(ctx.RepoKind),
	}
	for _, rule := range familylint.RulesForKind(ctx.RepoKind) {
		result := rule.Check(ctx)
		row := lintResult{
			ID:          rule.ID,
			Layer:       string(rule.Layer),
			Severity:    severityString(rule.Severity),
			Status:      statusString(result.Status),
			Description: rule.Description,
			Message:     result.Message,
			Hint:        result.Hint,
		}
		report.Results = append(report.Results, row)
		switch result.Status {
		case familylint.StatusPass:
			report.Passed++
		case familylint.StatusWarn:
			report.Warned++
		case familylint.StatusFail:
			report.Failed++
		case familylint.StatusSkip:
			report.Skipped++
		default:
			report.Failed++
		}
	}
	return report
}

func parseRepoKind(raw string) (familylint.RepoKind, error) {
	switch familylint.RepoKind(raw) {
	case "auto", "":
		return "", nil
	case familylint.RepoKindGoCLI, familylint.RepoKindSwiftCask, familylint.RepoKindCask, familylint.RepoKindFormula, familylint.RepoKindGeneric, familylint.RepoKindVPS:
		return familylint.RepoKind(raw), nil
	default:
		return "", fmt.Errorf("unknown --kind %q", raw)
	}
}

func printLintHuman(report lintReport) {
	banner("family lint — %s (%s)", report.Repo, report.Kind)
	for _, row := range report.Results {
		switch row.Status {
		case "pass":
			ok("%s %s", row.ID, row.Description)
		case "warn":
			warn("%s %s", row.ID, coalesce(row.Message, row.Description))
			if row.Hint != "" {
				hint(row.Hint)
			}
		case "fail":
			fail("%s %s", row.ID, coalesce(row.Message, row.Description))
			if row.Hint != "" {
				hint(row.Hint)
			}
		case "skip":
			info("%s skipped: %s", row.ID, row.Message)
		}
	}
	fmt.Fprintf(os.Stdout, "\npassed=%d warned=%d failed=%d skipped=%d\n", report.Passed, report.Warned, report.Failed, report.Skipped)
}

func severityString(severity familylint.Severity) string {
	switch severity {
	case familylint.SeverityWarn:
		return "warn"
	case familylint.SeverityFail:
		return "fail"
	default:
		return "unknown"
	}
}

func statusString(status familylint.Status) string {
	switch status {
	case familylint.StatusPass:
		return "pass"
	case familylint.StatusWarn:
		return "warn"
	case familylint.StatusFail:
		return "fail"
	case familylint.StatusSkip:
		return "skip"
	default:
		return "unknown"
	}
}

func filepathBase(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i < 0 {
		return path
	}
	return path[i+1:]
}
