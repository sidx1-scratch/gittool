package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// ASCII Art Logo
	AsciiArt = `[36m  _____ _ _ _____           _ 
 / ____(_)__   __|         | |
| |  __ _   | | ___   ___  | |
| | |_ | |  | |/ _ \ / _ \ | |
| |__| | |  | | (_) | (_) || |
 \_____|_|  |_|\___/ \___/ |_|[0m`

	configFile = ".gittool_config.json"
)

type Config struct {
	Repo string `json:"repo"`
}

func printHelp() {
	fmt.Println(AsciiArt)
	fmt.Println("gittool - Making Git simple.")
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  gittool init             Initialize a brand new local repository")
	fmt.Println("  gittool repo add <url>   Link a remote GitHub/GitLab repository")
	fmt.Println("  gittool status           See what files you have changed")
	fmt.Println("  gittool save [\"message\"] Track, commit, and PUSH all changes (message optional)")
	fmt.Println("  gittool sync             Pull latest remote changes and push yours")
	fmt.Println()
}

func saveRepo(repoURL string) error {
	cfg := Config{Repo: repoURL}
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func loadRepo() (string, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return "", nil
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Repo, nil
}

func runCommand(args []string) bool {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if stdout.Len() > 0 {
		fmt.Println(strings.TrimSpace(stdout.String()))
	}
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
		return false
	}
	return true
}

func setupRepo(repoURL string) {
	fmt.Println("Initializing local repository...")
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if !runCommand([]string{"git", "init"}) {
			return
		}
		if !runCommand([]string{"git", "branch", "-M", "main"}) {
			return
		}
	}

	fmt.Printf("Linking remote repository: %s\n", repoURL)
	// Remove origin if it exists (equivalent to subprocess.run(..., capture_output=True) in Python)
	removeCmd := exec.Command("git", "remote", "remove", "origin")
	_ = removeCmd.Run()

	if runCommand([]string{"git", "remote", "add", "origin", repoURL}) {
		fmt.Println("Repository successfully linked!")
	}
}

func getChangesSummary() string {
	cmd := exec.Command("git", "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	lines := strings.Split(stdout.String(), "\n")
	var added []string
	var modified []string
	var deleted []string

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[2:])

		switch status[0] {
		case 'A':
			added = append(added, file)
		case 'M':
			modified = append(modified, file)
		case 'D':
			deleted = append(deleted, file)
		case 'R':
			modified = append(modified, file)
		case '?':
			added = append(added, file)
		}
	}

	if len(added) == 0 && len(modified) == 0 && len(deleted) == 0 {
		return ""
	}

	var sb strings.Builder
	if len(added) > 0 {
		sb.WriteString("\nAdded:\n")
		for _, f := range added {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}
	if len(modified) > 0 {
		sb.WriteString("\nModified:\n")
		for _, f := range modified {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}
	if len(deleted) > 0 {
		sb.WriteString("\nDeleted:\n")
		for _, f := range deleted {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}

	return sb.String()
}

func saveAndPush(message string) {
	fmt.Println("Staging all changes...")
	if !runCommand([]string{"git", "add", "."}) {
		return
	}

	if message == "" {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		summary := getChangesSummary()
		if summary != "" {
			message = fmt.Sprintf("Automatic backup: %s\n%s", timestamp, summary)
		} else {
			message = fmt.Sprintf("Automatic backup: %s", timestamp)
		}
	}

	fmt.Printf("Saving changes with message:\n%s\n", message)
	if !runCommand([]string{"git", "commit", "-m", message}) {
		fmt.Println("Nothing to save or push.")
		return
	}

	fmt.Println("Pushing changes immediately to remote 'main'...")
	if runCommand([]string{"git", "push", "-u", "origin", "main"}) {
		fmt.Println("Save and push complete!")
	}
}

func syncChanges() {
	fmt.Println("Fetching latest updates from remote...")
	if !runCommand([]string{"git", "pull", "origin", "main", "--rebase"}) {
		return
	}

	fmt.Println("Pushing your changes to remote...")
	if runCommand([]string{"git", "push", "-u", "origin", "main"}) {
		return
	}
	fmt.Println("Sync complete!")
}

func showStatus() {
	runCommand([]string{"git", "status", "-s"})
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp()
		return
	}

	command := args[0]

	// Print ASCII art and a separator for every command to make it look premium
	fmt.Println(AsciiArt)
	fmt.Println(strings.Repeat("-", 45))

	switch command {
	case "init":
		fmt.Println("Initializing local repository...")
		if runCommand([]string{"git", "init"}) {
			if runCommand([]string{"git", "branch", "-M", "main"}) {
				fmt.Println("Successfully initialized local repository on branch 'main'!")
			}
		}

	case "repo":
		if len(args) >= 3 && args[1] == "add" {
			repoURL := args[2]
			if err := saveRepo(repoURL); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving repo configuration: %v\n", err)
				return
			}
			setupRepo(repoURL)
		} else {
			fmt.Println("Usage: gittool repo add <repository_url>")
		}

	case "status":
		showStatus()

	case "save":
		message := ""
		if len(args) >= 2 {
			message = args[1]
		}
		saveAndPush(message)

	case "sync":
		repoURL, err := loadRepo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading repo configuration: %v\n", err)
			return
		}
		if repoURL == "" {
			fmt.Println("Error: No repository linked yet. Run 'gittool repo add <url>' first.")
			return
		}
		syncChanges()

	default:
		fmt.Printf("Unknown command: '%s'\n", command)
		printHelp()
	}
}
