package familylint

import (
	"sort"
	"sync"
)

// Layer is the taxonomy bucket a rule belongs to.
type Layer string

const (
	LayerCmd  Layer = "cmd"
	LayerIO   Layer = "io"
	LayerRepo Layer = "repo"
	LayerCfg  Layer = "cfg"
	LayerVer  Layer = "ver"
)

// Severity controls how a non-passing Check result is treated.
// SeverityFail is the default; Warn is for "noted, but won't block".
type Severity int

const (
	SeverityFail Severity = iota
	SeverityWarn
)

// Status is the outcome of a Check.
type Status int

const (
	// StatusPass means the rule's invariant holds.
	StatusPass Status = iota
	// StatusWarn means the invariant doesn't hold but the rule is advisory.
	StatusWarn
	// StatusFail means the invariant doesn't hold and the rule is mandatory.
	StatusFail
	// StatusSkip means the rule doesn't apply in this Context — typically
	// because the artefact it inspects isn't present (e.g., no .goreleaser.yaml
	// in a repo that doesn't release). Skipped rules don't count as failures.
	StatusSkip
)

// Result is what Rule.Check returns.
type Result struct {
	Status  Status
	Message string // for Warn/Fail/Skip: terse description
	Hint    string // for Fail: actionable fix suggestion
}

// Pass is the canonical passing result.
func Pass() Result { return Result{Status: StatusPass} }

// Warn returns a warning result with a message.
func Warn(msg string) Result {
	return Result{Status: StatusWarn, Message: msg}
}

// Fail returns a failing result with a message and fix hint.
func Fail(msg, hint string) Result {
	return Result{Status: StatusFail, Message: msg, Hint: hint}
}

// Skip indicates the rule doesn't apply (e.g., the file it inspects is absent).
func Skip(reason string) Result {
	return Result{Status: StatusSkip, Message: reason}
}

// Rule is one validator.
type Rule struct {
	// ID is the canonical identifier, e.g. "F-cmd-001". Must be unique.
	ID string
	// Layer is the taxonomy bucket.
	Layer Layer
	// Severity controls non-passing treatment. Defaults to SeverityFail.
	Severity Severity
	// Description is the one-line "what this rule checks" statement.
	Description string
	// Check runs the validator against the given Context.
	Check func(c *Context) Result
}

var (
	registryMu sync.Mutex
	registry   []Rule
	registered = map[string]struct{}{}
)

// Register adds a Rule to the global registry. It panics if the rule's
// ID is empty, the Check is nil, or the ID is already registered — these
// indicate a programmer error that should surface at init / first test run.
func Register(r Rule) {
	if r.ID == "" {
		panic("familylint.Register: Rule.ID is required")
	}
	if r.Check == nil {
		panic("familylint.Register: Rule.Check is required for " + r.ID)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registered[r.ID]; dup {
		panic("familylint.Register: duplicate rule ID " + r.ID)
	}
	registered[r.ID] = struct{}{}
	registry = append(registry, r)
}

// Rules returns a copy of every registered rule, sorted by ID. The sort
// makes output stable regardless of init() ordering.
func Rules() []Rule {
	registryMu.Lock()
	defer registryMu.Unlock()
	out := make([]Rule, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// RulesByLayer returns only rules in the given layer, sorted by ID.
func RulesByLayer(layer Layer) []Rule {
	all := Rules()
	out := make([]Rule, 0, len(all))
	for _, r := range all {
		if r.Layer == layer {
			out = append(out, r)
		}
	}
	return out
}

// RuleByID returns the rule with the given ID, or nil.
func RuleByID(id string) *Rule {
	all := Rules()
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// resetRegistry is a test-only helper. It clears the registry so tests
// for Register() can run hermetically.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = nil
	registered = map[string]struct{}{}
}
