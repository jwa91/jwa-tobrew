package familylint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

// repoFixture is a freshly-staged jwa-* repo at t.TempDir()/<name>.
// Use newRepoFixture to construct; layer files on top with writeFile,
// then call ctx() to get a (cache-invalidated) Context.
type repoFixture struct {
	t    *testing.T
	root string
	name string
}

func newRepoFixture(t *testing.T, name string) *repoFixture {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", root, err)
	}
	return &repoFixture{t: t, root: root, name: name}
}

// write writes (or replaces) rel relative to the fixture root. Creates
// any intermediate directories.
func (f *repoFixture) write(rel, body string) {
	f.t.Helper()
	abs := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", abs, err)
	}
}

func (f *repoFixture) writeExecutable(rel, body string) string {
	f.t.Helper()
	abs := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o755); err != nil {
		f.t.Fatalf("write %s: %v", abs, err)
	}
	return abs
}

// ctx builds a fresh Context against the current state of the fixture.
// Caller passes any options (e.g. WithBinary). Always returns a fresh
// Context so the lazy file cache is empty.
func (f *repoFixture) ctx(opts ...familylint.Option) *familylint.Context {
	f.t.Helper()
	c, err := familylint.NewContext(f.root, opts...)
	if err != nil {
		f.t.Fatalf("NewContext: %v", err)
	}
	return c
}

// runRule resolves the rule from DefaultRegistry and runs it against the
// fixture's current state in a fresh Context. Returns the Result so the
// caller can assert on Status / Message / Hint.
func (f *repoFixture) runRule(id string, opts ...familylint.Option) familylint.Result {
	f.t.Helper()
	rule := familylint.RuleByID(id)
	if rule == nil {
		f.t.Fatalf("rule %s not registered", id)
	}
	return rule.Check(f.ctx(opts...))
}
