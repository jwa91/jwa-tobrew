package familylint

import (
	"slices"
	"sort"
	"strings"
)

// RepoKind identifies which convention pack applies to a repository.
// The zero value means "auto/unset"; callers should use Context.RepoKind
// after NewContext has detected the repository shape.
type RepoKind string

const (
	RepoKindGoCLI     RepoKind = "go-cli"
	RepoKindSwiftCask RepoKind = "swift-cask"
	RepoKindCask      RepoKind = "cask"
	RepoKindFormula   RepoKind = "formula"
	RepoKindGeneric   RepoKind = "generic"
	RepoKindVPS       RepoKind = "vps"
)

var (
	allRepoKinds = []RepoKind{
		RepoKindGoCLI,
		RepoKindSwiftCask,
		RepoKindCask,
		RepoKindFormula,
		RepoKindGeneric,
		RepoKindVPS,
	}
	goCLIPack = []RepoKind{RepoKindGoCLI}
)

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
	// Kinds narrows the rule to specific repo kinds. Empty means every kind.
	Kinds []RepoKind
	// Check runs the validator against the given Context.
	Check func(c *Context) Result
}

// AppliesToKind reports whether this rule participates in kind's pack.
func (r Rule) AppliesToKind(kind RepoKind) bool {
	if kind == "" {
		kind = RepoKindGeneric
	}
	if len(r.Kinds) > 0 {
		return slices.Contains(r.Kinds, kind)
	}
	for _, k := range defaultKindsForRuleID(r.ID) {
		if k == kind {
			return true
		}
	}
	return false
}

func defaultKindsForRuleID(id string) []RepoKind {
	switch {
	case strings.HasPrefix(id, "F-cmd-"), strings.HasPrefix(id, "F-io-"):
		return allRepoKinds
	case id == "F-repo-001", id == "F-repo-002", id == "F-repo-003", id == "F-repo-007", id == "F-repo-011", id == "F-repo-012":
		return goCLIPack
	case id == "F-cfg-001", id == "F-cfg-002", id == "F-cfg-003", id == "F-cfg-004", id == "F-cfg-005", id == "F-cfg-006", id == "F-cfg-007", id == "F-cfg-008", id == "F-cfg-009", id == "F-cfg-010", id == "F-cfg-011", id == "F-cfg-012":
		return goCLIPack
	case id == "F-cfg-030", id == "F-cfg-031", id == "F-cfg-032", id == "F-cfg-033":
		return goCLIPack
	case id == "F-ver-002", id == "F-ver-004", id == "F-ver-005":
		return goCLIPack
	default:
		return allRepoKinds
	}
}

// Registry holds a set of Rules. Rules are unique by ID. The zero value
// is NOT usable; construct with NewRegistry().
type Registry struct {
	rules []Rule
	byID  map[string]struct{}
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

// RulesForKind returns every rule that applies to kind, sorted by ID.
func RulesForKind(kind RepoKind) []Rule {
	all := Rules()
	out := make([]Rule, 0, len(all))
	for _, rule := range all {
		if rule.AppliesToKind(kind) {
			out = append(out, rule)
		}
	}
	return out
}

// RulesByLayer returns rules in DefaultRegistry filtered by Layer.
func RulesByLayer(layer Layer) []Rule { return DefaultRegistry.ByLayer(layer) }

// RuleByID returns a rule from DefaultRegistry, or nil.
func RuleByID(id string) *Rule { return DefaultRegistry.ByID(id) }
