package familylint_test

import (
	"testing"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

func TestRepoRule001CmdMainGo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(f *repoFixture)
		want  familylint.Status
	}{
		{
			name:  "missing main.go fails",
			setup: func(*repoFixture) {},
			want:  familylint.StatusFail,
		},
		{
			name: "canonical main.go passes",
			setup: func(f *repoFixture) {
				f.write("cmd/demo/main.go", "package main\n")
			},
			want: familylint.StatusPass,
		},
		{
			name: "tool surface main.go passes",
			setup: func(f *repoFixture) {
				f.write("tools/demo/main.go", "package main\n")
			},
			want: familylint.StatusPass,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			tt.setup(f)
			if got := f.runRule("F-repo-001").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoRule002VersionVarsAcceptToolsMain(t *testing.T) {
	t.Parallel()

	f := newRepoFixture(t, "demo")
	f.write("tools/demo/main.go", `package main

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
`)
	if got := f.runRule("F-repo-002").Status; got != familylint.StatusPass {
		t.Errorf("status = %v, want %v", got, familylint.StatusPass)
	}
}

func TestRepoRule003GoModule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want familylint.Status
	}{
		{name: "missing go.mod", body: "", want: familylint.StatusFail},
		{name: "correct module path", body: "module github.com/jwa91/demo\n\ngo 1.23\n", want: familylint.StatusPass},
		{name: "wrong owner", body: "module github.com/someone-else/demo\n\ngo 1.23\n", want: familylint.StatusFail},
		{name: "wrong name", body: "module github.com/jwa91/something-else\n\ngo 1.23\n", want: familylint.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			if tt.body != "" {
				f.write("go.mod", tt.body)
			}
			if got := f.runRule("F-repo-003").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoRule010GitignoreEnvBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		body        string
		envTemplate bool
		want        familylint.Status
	}{
		{name: "missing .gitignore", body: "", want: familylint.StatusFail},
		{name: "incomplete block", body: "bin/\n", want: familylint.StatusFail},
		{name: "complete block", body: "bin/\n.env\n.env.local\n.env.*.local\n", want: familylint.StatusPass},
		{name: "block with extra entries", body: ".DS_Store\n.env\n.env.local\n.env.*.local\nfoo\n", want: familylint.StatusPass},
		{name: "broad env glob with template exception", body: ".env\n.env.*\n!.env.template\n", envTemplate: true, want: familylint.StatusPass},
		{name: "broad env glob without template exception fails when template exists", body: ".env\n.env.*\n", envTemplate: true, want: familylint.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			if tt.body != "" {
				f.write(".gitignore", tt.body)
			}
			if tt.envTemplate {
				f.write(".env.template", "TOKEN=op://Example/Token/credential\n")
			}
			if got := f.runRule("F-repo-010").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoRule009EnvTemplateForReleaseEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		goreleaser string
		env        string
		want       familylint.Status
	}{
		{
			name: "direct op reference needs template",
			goreleaser: `
version: 2
env:
  - GITHUB_TOKEN=op://Personal/token/credential
`,
			want: familylint.StatusFail,
		},
		{
			name: "goreleaser env token with op template passes",
			goreleaser: `
version: 2
homebrew_casks:
  - repository:
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
`,
			env:  "HOMEBREW_TAP_GITHUB_TOKEN=op://Personal/tap/credential\n",
			want: familylint.StatusPass,
		},
		{
			name: "template without op references warns",
			goreleaser: `
version: 2
`,
			env:  "GITHUB_TOKEN=replace-me\n",
			want: familylint.StatusWarn,
		},
		{
			name: "no release env and no template passes",
			goreleaser: `
version: 2
`,
			want: familylint.StatusPass,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			f.write(".goreleaser.yaml", tt.goreleaser)
			if tt.env != "" {
				f.write(".env.template", tt.env)
			}
			if got := f.runRule("F-repo-009").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCfgRule007NoBrewsBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		goreleaser string
		want       familylint.Status
	}{
		{
			name:       "no .goreleaser.yaml skips",
			goreleaser: "",
			want:       familylint.StatusSkip,
		},
		{
			name: "brews block fails",
			goreleaser: `
version: 2
project_name: demo
brews:
  - name: demo
`,
			want: familylint.StatusFail,
		},
		{
			name: "only homebrew_casks passes",
			goreleaser: `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    directory: Casks
    binaries: [demo]
`,
			want: familylint.StatusPass,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			if tt.goreleaser != "" {
				f.write(".goreleaser.yaml", tt.goreleaser)
			}
			if got := f.runRule("F-cfg-007").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCfgRule010BinariesPlural(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		goreleaser string
		want       familylint.Status
	}{
		{
			name: "singular binary fails (deprecated)",
			goreleaser: `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    binary: demo
`,
			want: familylint.StatusFail,
		},
		{
			name: "plural binaries passes",
			goreleaser: `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    binaries: [demo]
`,
			want: familylint.StatusPass,
		},
		{
			name: "neither key fails",
			goreleaser: `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
`,
			want: familylint.StatusFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			f.write(".goreleaser.yaml", tt.goreleaser)
			if got := f.runRule("F-cfg-010").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoRule014NoBrewfile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  familylint.Status
	}{
		{name: "no Brewfile anywhere", files: nil, want: familylint.StatusPass},
		{name: "Brewfile at root", files: map[string]string{"Brewfile": ""}, want: familylint.StatusFail},
		{name: "Brewfile nested", files: map[string]string{"docs/Brewfile": ""}, want: familylint.StatusFail},
		{name: "Brewfile.local", files: map[string]string{"Brewfile.local": ""}, want: familylint.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRepoFixture(t, "demo")
			for path, body := range tt.files {
				f.write(path, body)
			}
			if got := f.runRule("F-repo-014").Status; got != tt.want {
				t.Errorf("status = %v, want %v", got, tt.want)
			}
		})
	}
}
