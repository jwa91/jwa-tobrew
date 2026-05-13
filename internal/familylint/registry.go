package familylint

import "sort"

// Layer is the taxonomy bucket a rule belongs to. Empty-string is invalid;
// every Rule must declare a non-empty Layer.
type Layer string

const (
	LayerCmd  Layer = "cmd"
	LayerIO   Layer = "io"
	LayerRepo Layer = "repo"
	LayerCfg  Layer = "cfg"
	LayerVer  Layer = "ver"
)

// Severity controls how a non-passing Check result is treated.
//
// Zero value is SeverityUnknown — every Rule must set a Severity at
// registration; Register panics when it's omitted, matching the
// "make illegal states unrepresentable" guidance.
type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityWarn
	SeverityFail
)

// Status is the outcome of a single Check invocation.
//
// Zero value is StatusUnknown so a Check that forgets to set a status
// cannot accidentally report Pass. Use Pass()/Warn()/Fail()/Skip() to
// build Results.
type Status int

const (
	StatusUnknown Status = iota
	StatusPass
	StatusWarn
	StatusFail
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

// Skip indicates the rule doesn't apply in this context.
func Skip(reason string) Result {
	return Result{Status: StatusSkip, Message: reason}
}

// Rule is a single validator. ID, Layer, Severity, and Check are all required.
type Rule struct {
	// ID is the canonical identifier, e.g. "F-cmd-001". Must be unique.
	ID string
	// Layer is the taxonomy bucket.
	Layer Layer
	// Severity controls non-passing treatment.
	Severity Severity
	// Description is the one-line "what this rule checks" statement.
	Description string
	// Check runs the validator against the given Context.
	Check func(c *Context) Result
}

// Registry holds a set of Rules. Rules are unique by ID. The zero value
// is NOT usable; construct with NewRegistry().
type Registry struct {
	rules    []Rule
	byID     map[string]struct{}
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byID: map[string]struct{}{}}
}

// Register adds a Rule to the registry. Panics on:
//   - empty ID, Layer, or Description
//   - nil Check
//   - SeverityUnknown
//   - duplicate ID
//
// These are all programmer errors that should surface at init / first
// test run, not silently shadow another rule or report ambiguously.
func (r *Registry) Register(rule Rule) {
	switch {
	case rule.ID == "":
		panic("familylint: Rule.ID is required")
	case rule.Layer == "":
		panic("familylint: Rule.Layer is required for " + rule.ID)
	case rule.Description == "":
		panic("familylint: Rule.Description is required for " + rule.ID)
	case rule.Check == nil:
		panic("familylint: Rule.Check is required for " + rule.ID)
	case rule.Severity == SeverityUnknown:
		panic("familylint: Rule.Severity is required for " + rule.ID)
	}
	if _, dup := r.byID[rule.ID]; dup {
		panic("familylint: duplicate rule ID " + rule.ID)
	}
	r.byID[rule.ID] = struct{}{}
	r.rules = append(r.rules, rule)
}

// Rules returns every registered rule, sorted by ID for stable output.
func (r *Registry) Rules() []Rule {
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ByLayer returns rules in the given layer, sorted by ID.
func (r *Registry) ByLayer(layer Layer) []Rule {
	all := r.Rules()
	out := make([]Rule, 0, len(all))
	for _, ru := range all {
		if ru.Layer == layer {
			out = append(out, ru)
		}
	}
	return out
}

// ByID returns the rule with the given ID, or nil.
func (r *Registry) ByID(id string) *Rule {
	for i := range r.rules {
		if r.rules[i].ID == id {
			// Return a copy so callers can't mutate registry state.
			rule := r.rules[i]
			return &rule
		}
	}
	return nil
}

// Len reports how many rules are registered.
func (r *Registry) Len() int { return len(r.rules) }

// DefaultRegistry is the package-level registry populated by every
// rule file's init() function. Production callers iterate this.
// Tests that need isolation construct their own via NewRegistry().
var DefaultRegistry = NewRegistry()

// Register adds a Rule to DefaultRegistry. Provided as a thin proxy so
// init() blocks in rule files don't need to reach into DefaultRegistry.
func Register(rule Rule) { DefaultRegistry.Register(rule) }

// Rules returns every rule in DefaultRegistry, sorted by ID.
func Rules() []Rule { return DefaultRegistry.Rules() }

// RulesByLayer returns rules in DefaultRegistry filtered by Layer.
func RulesByLayer(layer Layer) []Rule { return DefaultRegistry.ByLayer(layer) }

// RuleByID returns a rule from DefaultRegistry, or nil.
func RuleByID(id string) *Rule { return DefaultRegistry.ByID(id) }
