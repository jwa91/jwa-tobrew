package familylint

import (
	"strings"
	"testing"
)

// TestRulesPresent locks in the expected rule IDs at the package level.
// If you add a new rule, append it here; if you intentionally remove one,
// drop it here too. The list serves as the canonical roll-call of what
// the family contract enforces.
func TestRulesPresent(t *testing.T) {
	want := []string{
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
	got := map[string]struct{}{}
	for _, r := range Rules() {
		got[r.ID] = struct{}{}
	}
	for _, id := range want {
		if _, ok := got[id]; !ok {
			t.Errorf("rule %s not registered", id)
		}
	}
	for id := range got {
		found := false
		for _, want := range want {
			if want == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rule %s registered but not in expected list — update want[]", id)
		}
	}
}

func TestRulesSortedByID(t *testing.T) {
	rules := Rules()
	for i := 1; i < len(rules); i++ {
		if rules[i-1].ID >= rules[i].ID {
			t.Errorf("rules not sorted: %s before %s", rules[i-1].ID, rules[i].ID)
		}
	}
}

func TestRuleByID(t *testing.T) {
	r := RuleByID("F-cmd-001")
	if r == nil {
		t.Fatal("F-cmd-001 not found")
	}
	if r.Layer != LayerCmd {
		t.Errorf("F-cmd-001 layer = %q, want cmd", r.Layer)
	}
	if RuleByID("F-nope-999") != nil {
		t.Error("RuleByID returned non-nil for unknown ID")
	}
}

func TestRulesByLayer(t *testing.T) {
	cmd := RulesByLayer(LayerCmd)
	if len(cmd) == 0 {
		t.Fatal("no rules in LayerCmd")
	}
	for _, r := range cmd {
		if r.Layer != LayerCmd {
			t.Errorf("rule %s in cmd layer slice but Layer=%q", r.ID, r.Layer)
		}
		if !strings.HasPrefix(r.ID, "F-cmd-") {
			t.Errorf("rule %s in cmd layer slice but ID doesn't match F-cmd-*", r.ID)
		}
	}
}

func TestRegisterPanicsOnDuplicateID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate ID")
		} else if s, ok := r.(string); !ok || !strings.Contains(s, "duplicate") {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	Register(Rule{
		ID:          "F-cmd-001", // duplicate
		Layer:       LayerCmd,
		Description: "duplicate test",
		Check:       func(*Context) Result { return Pass() },
	})
}

func TestRegisterPanicsOnEmptyID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty ID")
		}
	}()
	Register(Rule{Check: func(*Context) Result { return Pass() }})
}

func TestRegisterPanicsOnNilCheck(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil Check")
		}
	}()
	Register(Rule{ID: "F-test-nil-check"})
}
