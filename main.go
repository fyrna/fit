// i mean yeah, fyrna/cli still WIP but who care?
// it's not like you gonna use this!
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fyrna/cli"
)

var (
	commitTypes = []string{"feat", "fix", "docs", "style", "refactor", "test", "chore", "ci", "build", "NONE"}
	typeEmojis  = map[string]string{
		"feat": "✨", "fix": "🐛", "docs": "📝", "style": "🎨",
		"refactor": "🔨", "test": "✅", "chore": "🔧",
		"ci": "🤖", "build": "🛠️", "NONE": "❌",
	}
)

const (
	red    = "\033[1;31m"
	pink   = "\033[1;35m"
	reset  = "\033[0m"
	cursor = "\033[1;35m█" // pink block
)

func write(out io.Writer, color, msg string) {
	fmt.Fprintf(out, color+"%s"+reset+"\n", msg)
}

func showPinkCursor() {
	fmt.Print("\033[?25l")
	fmt.Print("\033[2 q")
	fmt.Print("\033[?25h")
}

// ee... it works, so, i think i'll leave it
// func resetCursor() {
// 	fmt.Print("\033[?25h")
// 	fmt.Print("\033[0 q")
// 	fmt.Print("\033]12;\a")
// }

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
func interactiveCommit() error {
	if !isGitRepo() {
		return fmt.Errorf("💥 not a git repo")
	}
	if !hasStaged() {
		return fmt.Errorf("💥 no staged changes")
	}

	write(os.Stdout, pink, "🌸 fit interactive mode\n")

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
			return fmt.Errorf("❌ aborted")
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
	commitType := commitTypes[idx]

	fmt.Print(clearLine)

	for range len(commitTypes) {
		fmt.Print(clearLine)
	}

	var scope string

	fmt.Print("Scope (optional): ")

	showPinkCursor()

	scope, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	// resetCursor()
	scope = strings.TrimSpace(scope)

	// required
	var message string
	for {
		fmt.Print("Message (required): ")
		message, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		message = strings.TrimSpace(message)
		if message != "" {
			break
		}
		return fmt.Errorf("%s message cannot be empty %s", red, reset)
	}

	// body
	fmt.Println("Body (optional, empty to skip):")
	bodyLines := []string{}
	for {
		showPinkCursor()
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			break
		}
		bodyLines = append(bodyLines, line)
	}
	body := strings.Join(bodyLines, "\n")

	// build commit
	var header string
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

	fmt.Printf("\n📋 preview:\n\033[1;33m%s\033[0m\n\n", full)
	fmt.Print("\033[38;5;213mCommit? [y/N]:\033[0m ")
	ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(strings.ToLower(ans)) != "y" {
		return fmt.Errorf("❌ cancelled")
	}

	return runGit("commit", "-m", full)
}

func main() {
	a := cli.New("fit")

	a.Command("", func(c *cli.Context) error {
		runGit(append([]string{"commit"}, c.Args()...)...)
		return nil
	})

	a.Command("m", func(ctx *cli.Context) error {
		err := interactiveCommit()
		return err
	})

	a.Command("edit", func(ctx *cli.Context) error {
		if !isGitRepo() || !lastCommit() {
			write(ctx.App.Err, red, "nothing to amend")
			os.Exit(1)
		}
		runGit("commit", "--amend")
		return nil
	})

	a.Command("undo", func(ctx *cli.Context) error {
		if !isGitRepo() || !lastCommit() {
			write(ctx.App.Err, red, "nothing to undo")
			os.Exit(1)
		}
		runGit("reset", "--soft", "HEAD~1")
		return nil
	})

	a.Run()
}
