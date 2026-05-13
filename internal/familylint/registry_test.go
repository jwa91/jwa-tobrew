package familylint_test

import (
	"strings"
	"testing"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

// validRule returns a baseline Rule that passes every Register precondition.
// Tests mutate one field to exercise each panic path.
func validRule() familylint.Rule {
	return familylint.Rule{
		ID:          "F-test-001",
		Layer:       familylint.LayerCmd,
		Severity:    familylint.SeverityFail,
		Description: "test fixture",
		Check:       func(*familylint.Context) familylint.Result { return familylint.Pass() },
	}
}

// expectedRuleIDs is the canonical roll-call of rules registered by init()
// functions across the package. Update this list when adding or removing
// a rule; TestExpectedRulesRegistered enforces it on every test run.
var expectedRuleIDs = []string{
	"F-cmd-001", "F-cmd-002", "F-cmd-003", "F-cmd-004", "F-cmd-005", "F-cmd-006",
	"F-io-001", "F-io-002", "F-io-003", "F-io-004", "F-io-005", "F-io-006", "F-io-007", "F-io-008",
	"F-repo-001", "F-repo-002", "F-repo-003", "F-repo-004", "F-repo-005", "F-repo-006",
	"F-repo-007", "F-repo-008", "F-repo-009", "F-repo-010", "F-repo-011", "F-repo-012",
	"F-repo-013", "F-repo-014",
	"F-cfg-001", "F-cfg-002", "F-cfg-003", "F-cfg-004", "F-cfg-005", "F-cfg-006",
	"F-cfg-007", "F-cfg-008", "F-cfg-009", "F-cfg-010", "F-cfg-011", "F-cfg-012",
	"F-cfg-020", "F-cfg-021", "F-cfg-022", "F-cfg-023",
	"F-cfg-030", "F-cfg-031", "F-cfg-032", "F-cfg-033",
	"F-cfg-040", "F-cfg-041", "F-cfg-042", "F-cfg-043",
	"F-ver-001", "F-ver-002", "F-ver-003", "F-ver-004", "F-ver-005",
}

func TestExpectedRulesRegistered(t *testing.T) {
	t.Parallel()
	got := map[string]struct{}{}
	for _, r := range familylint.Rules() {
		got[r.ID] = struct{}{}
	}
	for _, id := range expectedRuleIDs {
		if _, ok := got[id]; !ok {
			t.Errorf("rule %s missing from DefaultRegistry", id)
		}
	}
	for id := range got {
		if !containsString(expectedRuleIDs, id) {
			t.Errorf("rule %s registered but not in expectedRuleIDs — update the list", id)
		}
	}
}

func TestRulesSortedByID(t *testing.T) {
	t.Parallel()
	rules := familylint.Rules()
	for i := 1; i < len(rules); i++ {
		if rules[i-1].ID >= rules[i].ID {
			t.Errorf("rules not sorted: %s before %s", rules[i-1].ID, rules[i].ID)
		}
	}
}

func TestRuleByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      string
		wantNil bool
	}{
		{name: "registered rule", id: "F-cmd-001"},
		{name: "unknown rule", id: "F-nope-999", wantNil: true},
		{name: "empty id", id: "", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := familylint.RuleByID(tt.id)
			if (got == nil) != tt.wantNil {
				t.Fatalf("RuleByID(%q) = %v, wantNil=%v", tt.id, got, tt.wantNil)
			}
		})
	}
}

func TestRulesByLayer(t *testing.T) {
	t.Parallel()
	for _, layer := range []familylint.Layer{
		familylint.LayerCmd, familylint.LayerIO, familylint.LayerRepo,
		familylint.LayerCfg, familylint.LayerVer,
	} {
		layer := layer // capture
		t.Run(string(layer), func(t *testing.T) {
			t.Parallel()
			rules := familylint.RulesByLayer(layer)
			if len(rules) == 0 {
				t.Fatalf("no rules in layer %q", layer)
			}
			prefix := "F-" + string(layer) + "-"
			for _, r := range rules {
				if r.Layer != layer {
					t.Errorf("rule %s in layer-%q slice but Layer=%q", r.ID, layer, r.Layer)
				}
				if !strings.HasPrefix(r.ID, prefix) {
					t.Errorf("rule %s in layer-%q slice but ID doesn't start with %s", r.ID, layer, prefix)
				}
			}
		})
	}
}

func TestRegistryRegisterPanics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mutate      func(*familylint.Rule)
		wantContain string
	}{
		{
			name:        "empty ID",
			mutate:      func(r *familylint.Rule) { r.ID = "" },
			wantContain: "Rule.ID is required",
		},
		{
			name:        "empty Layer",
			mutate:      func(r *familylint.Rule) { r.Layer = "" },
			wantContain: "Rule.Layer is required",
		},
		{
			name:        "empty Description",
			mutate:      func(r *familylint.Rule) { r.Description = "" },
			wantContain: "Rule.Description is required",
		},
		{
			name:        "nil Check",
			mutate:      func(r *familylint.Rule) { r.Check = nil },
			wantContain: "Rule.Check is required",
		},
		{
			name:        "SeverityUnknown",
			mutate:      func(r *familylint.Rule) { r.Severity = familylint.SeverityUnknown },
			wantContain: "Rule.Severity is required",
		},
	}
	for _, tt := range tests {
		tt := tt // capture for parallel
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := familylint.NewRegistry()
			rule := validRule()
			tt.mutate(&rule)

			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected panic containing %q", tt.wantContain)
				}
				msg, ok := r.(string)
				if !ok {
					t.Fatalf("panic value is %T (%v), want string", r, r)
				}
				if !strings.Contains(msg, tt.wantContain) {
					t.Errorf("panic %q doesn't contain %q", msg, tt.wantContain)
				}
			}()
			reg.Register(rule)
		})
	}
}

func TestRegistryRegisterPanicsOnDuplicateID(t *testing.T) {
	t.Parallel()
	reg := familylint.NewRegistry()
	reg.Register(validRule())

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate ID")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "duplicate rule ID") {
			t.Errorf("panic %q doesn't mention duplicate", msg)
		}
	}()
	reg.Register(validRule()) // same ID — duplicate
}

func TestRegistryLen(t *testing.T) {
	t.Parallel()
	reg := familylint.NewRegistry()
	if reg.Len() != 0 {
		t.Errorf("new registry: Len=%d, want 0", reg.Len())
	}
	reg.Register(validRule())
	if reg.Len() != 1 {
		t.Errorf("after one Register: Len=%d, want 1", reg.Len())
	}
}

func TestRegistryByIDReturnsCopy(t *testing.T) {
	// ByID must return a copy so callers can't mutate registry state.
	t.Parallel()
	reg := familylint.NewRegistry()
	reg.Register(validRule())

	got := reg.ByID("F-test-001")
	if got == nil {
		t.Fatal("ByID returned nil for registered rule")
	}
	got.Description = "mutated"

	again := reg.ByID("F-test-001")
	if again.Description == "mutated" {
		t.Error("mutation through ByID return leaked into registry")
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
