package familylint_test

import (
	"fmt"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

// Example shows the canonical caller shape: build a Context against a
// repo (optionally pointing at a built binary), iterate Rules(), invoke
// each rule's Check, and render per-status.
func Example() {
	c, err := familylint.NewContext("/path/to/jwa-repo",
		familylint.WithBinary("/path/to/bin/jwa-repo"),
	)
	if err != nil {
		fmt.Println("ctx:", err)
		return
	}

	var pass, fail, warn, skip int
	for _, r := range familylint.Rules() {
		switch r.Check(c).Status {
		case familylint.StatusPass:
			pass++
		case familylint.StatusFail:
			fail++
		case familylint.StatusWarn:
			warn++
		case familylint.StatusSkip:
			skip++
		}
	}
	fmt.Printf("%d pass / %d fail / %d warn / %d skip\n", pass, fail, warn, skip)
}

// ExampleNewRegistry shows how a caller (typically a test) builds an
// isolated Registry instead of mutating the package-level
// DefaultRegistry. Useful when a future caller wants to compose a
// subset of rules into its own linter binary.
func ExampleNewRegistry() {
	reg := familylint.NewRegistry()
	reg.Register(familylint.Rule{
		ID:          "custom-001",
		Layer:       familylint.LayerRepo,
		Severity:    familylint.SeverityWarn,
		Description: "every repo must have a NOTES.md",
		Check: func(c *familylint.Context) familylint.Result {
			if c.FileExists("NOTES.md") {
				return familylint.Pass()
			}
			return familylint.Warn("NOTES.md missing")
		},
	})
	fmt.Println(reg.Len(), "rule(s) registered")
	// Output: 1 rule(s) registered
}
