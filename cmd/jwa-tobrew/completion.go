package main

import (
	"errors"
	"fmt"
	"strings"
)

func runCompletion(args []string) error {
	fs := subFlagSet("completion", "generate shell completion for bash, zsh, or fish")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("usage: jwa-tobrew completion <bash|zsh|fish>")
	}

	switch rest[0] {
	case "bash":
		fmt.Print(bashCompletion())
	case "zsh":
		fmt.Print(zshCompletion())
	case "fish":
		fmt.Print(fishCompletion())
	default:
		return fmt.Errorf("unknown shell %q (want bash, zsh, or fish)", rest[0])
	}
	return nil
}

func completionCommandNames() []string {
	names := make([]string, 0, len(commandSpecs))
	for _, cmd := range commandSpecs {
		names = append(names, cmd.Name)
	}
	return names
}

func bashCompletion() string {
	commands := strings.Join(completionCommandNames(), " ")
	return fmt.Sprintf(`# bash completion for jwa-tobrew
_jwa_tobrew_completion()
{
  local cur subcommands
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  subcommands="%s"

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "${subcommands}" -- "${cur}") )
    return 0
  fi

  case "${COMP_WORDS[1]}" in
    completion)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
      ;;
  esac
}
complete -F _jwa_tobrew_completion jwa-tobrew
`, commands)
}

func zshCompletion() string {
	var b strings.Builder
	b.WriteString(`#compdef jwa-tobrew

_jwa_tobrew() {
  local -a commands
  commands=(
`)
	for _, cmd := range commandSpecs {
		fmt.Fprintf(&b, "    %s\n", shellSingleQuote(cmd.Name+":"+cmd.Summary))
	}
	b.WriteString(`  )

  _arguments -C \
    '1:command:->commands' \
    '*::argument:->arguments'

  case "$state" in
    commands)
      _describe 'command' commands
      ;;
    arguments)
      case "$words[2]" in
        completion)
          _values 'shell' bash zsh fish
          ;;
      esac
      ;;
  esac
}

_jwa_tobrew "$@"
`)
	return b.String()
}

func fishCompletion() string {
	var b strings.Builder
	b.WriteString("# fish completion for jwa-tobrew\n")
	b.WriteString("complete -c jwa-tobrew -f\n")
	for _, cmd := range commandSpecs {
		fmt.Fprintf(
			&b,
			"complete -c jwa-tobrew -n '__fish_use_subcommand' -a %s -d %s\n",
			shellSingleQuote(cmd.Name),
			shellSingleQuote(cmd.Summary),
		)
	}
	b.WriteString("complete -c jwa-tobrew -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'\n")
	return b.String()
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
