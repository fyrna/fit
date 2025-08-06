// i mean yeah, fyrna/cli still WIP but who care?
// it's not like you gonna use this!
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fyrna/cli"
)

var (
	commitTypes = []string{"feat", "fix", "docs", "style", "refactor", "test", "chore", "ci", "build", "NONE"}

	typeEmojis = map[string]string{
		"feat":     "✨",
		"fix":      "🐛",
		"docs":     "📝",
		"style":    "🎨",
		"refactor": "🔨",
		"test":     "✅",
		"chore":    "🔧",
		"ci":       "🤖",
		"build":    "🛠️",
		"NONE":     "❌",
	}
)

const (
	red    = "\033[1;31m"
	pink   = "\033[1;35m"
	brPink = "\033[38;5;213m"
	yellow = "\033[1;33m"
	reset  = "\033[0m"
	cursor = "\033[1;35m█" // pink block
)

func write(color, msg string) {
	fmt.Fprintf(os.Stdout, color+"%s"+reset, msg)
}

func writeln(color, msg string) {
	write(color, msg+"\n")
}

func writerr(msg string) error {
	return fmt.Errorf(red+"%s"+reset, msg)
}

func showPinkCursor() {
	fmt.Print("\033[?25l")
	fmt.Print("\033[2 q")
	fmt.Print("\033[?25h")
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func isGitRepo() bool {
	return runGit("rev-parse", "--git-dir") == nil
}

func hasStaged() bool {
	return runGit("diff", "--cached", "--quiet") != nil
}

func lastCommit() bool {
	return runGit("rev-parse", "--verify", "HEAD") == nil
}

// tiny TUI helpers
const (
	hideCursor = "\033[?25l"
	showCursor = "\033[?25h"
	clearLine  = "\033[2K\r"
	up         = "\033[A"
	down       = "\033[B"
)

func getch() (byte, error) {
	// raw tanpa buffering
	_ = exec.Command("stty", "-F", "/dev/tty", "raw", "-echo").Run()
	defer exec.Command("stty", "-F", "/dev/tty", "-raw", "echo").Run()

	b := make([]byte, 1)
	_, err := os.Stdin.Read(b)
	return b[0], err
}

// interactive commit
func interactiveCommit(ctx *cli.Context) error {
	if !isGitRepo() {
		return writerr("not a git repo!")
	}
	if !hasStaged() {
		return writerr("no staged changes")
	}

	writeln(pink, "🌸 fit interactive mode\n")

	// choose type
	idx := 0

	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)

	for {
		fmt.Print(clearLine)

		for i, t := range commitTypes {
			if i == idx {
				fmt.Printf("\033[1;35m▶ %s %s\033[0m\n", typeEmojis[t], t)
			} else {
				fmt.Printf("  %s %s\n", typeEmojis[t], t)
			}
		}

		c, _ := getch()

		switch c {
		case 'q', 3, 17: // q, ^C, ^Q
			return writerr("❌ aborted")
		case 13: // Enter
			goto chosen
		case 27: // arrow escape
			next, _ := getch()
			if next == 91 {
				dir, _ := getch()
				switch dir {
				case 65: // up
					if idx > 0 {
						idx--
					}
				case 66: // down
					if idx < len(commitTypes)-1 {
						idx++
					}
				}
			}
		}

		for range len(commitTypes) {
			fmt.Print(up)
		}
	}
chosen:
	var (
		scope, message, header string
	)

	reader := bufio.NewReader(os.Stdin)
	commitType := commitTypes[idx]

	writeln(yellow, clearLine)

	for range len(commitTypes) {
		fmt.Print(clearLine)
	}

	fmt.Print("Scope (optional): ")

	showPinkCursor()

	scope, _ = reader.ReadString('\n')
	// resetCursor()
	scope = strings.TrimSpace(scope)

	// required
	for {
		fmt.Print("Message (required): ")

		message, _ = reader.ReadString('\n')
		message = strings.TrimSpace(message)

		if message != "" {
			break
		}

		return writerr("message cannot be empty!")
	}

	// body
	fmt.Println("Body (optional, empty to skip):")
	bodyLines := []string{}

	for {
		showPinkCursor()

		line, _ := reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")

		if line == "" {
			break
		}

		bodyLines = append(bodyLines, line)
	}

	body := strings.Join(bodyLines, "\n")

	// build commit
	if commitType == "NONE" {
		if scope != "" {
			header = fmt.Sprintf("%s(%s): %s", commitType, scope, message)
		} else {
			header = message
		}
	} else {
		emoji := typeEmojis[commitType]
		if scope != "" {
			header = fmt.Sprintf("%s %s(%s): %s", emoji, commitType, scope, message)
		} else {
			header = fmt.Sprintf("%s %s: %s", emoji, commitType, message)
		}
	}

	full := header

	if body != "" {
		full = header + "\n\n" + body
	}

	fmt.Printf("\n📋 preview:\n%s%s%s\n\n", yellow, full, reset)
	write(brPink, "Commit? [y/N] ")

	ans, _ := reader.ReadString('\n')

	if strings.TrimSpace(strings.ToLower(ans)) != "y" {
		return writerr("❌ cancelled")
	}

	return runGit("commit", "-m", full)
}

func main() {
	a := cli.New("fit")

	a.Command("", func(ctx *cli.Context) error {
		if !isGitRepo() {
			return writerr("not a git repo\nyou may want to \"git init\" manualy")
		}

		args := []string{}

		for _, arg := range ctx.Args() {
			if strings.TrimSpace(arg) != "" {
				args = append(args, arg)
			}
		}

		return runGit(append([]string{"commit"}, args...)...)
	})

	a.Command("m", interactiveCommit)

	a.Command("edit", func(ctx *cli.Context) error {
		if !isGitRepo() || !lastCommit() {
			return writerr("nothing to amend")
		}
		return runGit("commit", "--amend")
	})

	a.Command("undo", func(ctx *cli.Context) error {
		if !isGitRepo() || !lastCommit() {
			return writerr("nothing to undo")
		}
		return runGit("reset", "--soft", "HEAD~1")
	})

	a.Run()
}
