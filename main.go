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

var commitTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"test", "chore", "ci", "build",
}

var typeEmojis = map[string]string{
	"feat":     "✨",
	"fix":      "🐛",
	"docs":     "📝",
	"style":    "🎨",
	"refactor": "🔨",
	"test":     "✅",
	"chore":    "🔧",
	"ci":       "🤖",
	"build":    "🛠️",
}

func isGitStagedEmpty() bool {
	err := runGit("diff", "--cached", "--quiet")
	// Exit status 0 = no diff, exit status 1 = ada diff
	return err == nil
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGitAdd(files []string) error {
	if len(files) == 0 {
		files = []string{"."}
	}
	return runGit(append([]string{"add"}, files...)...)
}

func runSemanticCommit(c *cli.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// cek sebelum commit
	if isGitStagedEmpty() {
		fmt.Println("\n\033[1;31m⚠️  Tidak ada file yang di-add (git add).\033[0m")
		fmt.Print("\033[1;34mTambah semua file dengan 'git add .' sekarang? (y/n): \033[0m")

		confirm, _ := reader.ReadString('\n')

		if strings.TrimSpace(strings.ToLower(confirm)) == "y" {
			err := runGit("add", ".")
			if err != nil {
				return fmt.Errorf("\033[1;31mGagal menjalankan git add .\033[0m")
			}
		} else {
			fmt.Println("\033[1;90mCommit dibatalkan (っ﹏-) .｡o\033[0m")
			return nil
		}
	}

	fmt.Println("\033[1;35mPilih tipe commit:\033[0m")
	for i, t := range commitTypes {
		fmt.Printf("  \033[1;36m%2d)\033[0m %s\n", i+1, t)
	}
	fmt.Print("\033[1;34m> Nomor pilihan:\033[0m ")

	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)

	var idx int
	fmt.Sscanf(choiceStr, "%d", &idx)
	if idx < 1 || idx > len(commitTypes) {
		return fmt.Errorf("\033[1;31mPilihan nggak valid >///<\033[0m")
	}

	commitType := commitTypes[idx-1]
	emoji := typeEmojis[commitType]

	fmt.Print("\033[1;34m📦 Scope (optional):\033[0m ")
	scope, _ := reader.ReadString('\n')
	scope = strings.TrimSpace(scope)

	fmt.Print("\033[1;34m💬 Deskripsi singkat:\033[0m ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	fmt.Println("\033[1;34m📝 Tulis body (opsional, kosongkan untuk skip):\033[0m")
	fmt.Println("\033[0;90m  (Enter baris kosong dua kali (setelah input) atau satu kali (jika input kosong)  untuk mengakhiri)\033[0m")
	bodyLines := []string{}

	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		bodyLines = append(bodyLines, line)
	}

	// Format: type(scope): desc\n\nbody
	var header string
	if scope != "" {
		header = fmt.Sprintf("%s%s(%s): %s", emoji, commitType, scope, desc)
	} else {
		header = fmt.Sprintf("%s %s: %s", emoji, commitType, desc)
	}

	// Join body (if any)
	msg := header
	if len(bodyLines) > 0 {
		msg += "\n\n" + strings.Join(bodyLines, "\n")
	}

	// preview warna
	fmt.Println("\n\033[1;32m📤 Commit Preview:\033[0m")
	fmt.Printf("\033[1;33m%s\033[0m\n", header)
	if len(bodyLines) > 0 {
		fmt.Println(strings.Join(bodyLines, "\n"))
	}

	// Konfirmasi
	fmt.Print("\n\033[1;34mLanjut commit? (y/n):\033[0m ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println("\033[1;90mCommit dibatalkan~ (๑´•﹏•`๑)\033[0m")
		return nil
	}

	return runGit("commit", "-m", msg)
}

func main() {
	app := cli.New("fit",
		cli.SetVersion("0.1.0"),
		cli.SetDesc("A Cute Fyrna's gIt commiT cli wrapper"))

	app.Command("", func(c *cli.Context) error {
		return runGit("commit")
	})

	app.Command("m",
		cli.Short("Add commit message with semantic type"),
		cli.Action(runSemanticCommit),
	)

	app.Command("a", func(c *cli.Context) error {
		return runGitAdd(c.Args())
	})

	app.Command("aa", func(c *cli.Context) error {
		return runGitAdd(nil)
	})

	app.Command("help", func(c *cli.Context) error {
		return app.PrintRootHelp()
	})

	app.Run()
}
