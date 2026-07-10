package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ── ANSI colours ─────────────────────────────────────────────────────────────
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func bold(s string) string   { return colorBold + s + colorReset }
func dim(s string) string    { return colorDim + s + colorReset }

func info(msg string)    { fmt.Println(yellow("  ▶  ") + msg) }
func success(msg string) { fmt.Println(green("  ✔  ") + msg) }
func fail(msg string)    { fmt.Fprintln(os.Stderr, red("  ✘  ")+msg) }
func step(msg string)    { fmt.Println(dim("  →  ") + msg) }

// ── Logo ─────────────────────────────────────────────────────────────────────
const AsciiArt = colorCyan + `  _____ _ _ _____           _ 
 / ____(_)__   __|         | |
| |  __ _   | | ___   ___  | |
| | |_ | |  | |/ _ \ / _ \ | |
| |__| | |  | | (_) | (_) || |
 \_____|_|  |_|\___/ \___/ |_|` + colorReset

const configFile = ".gittool_config.json"

// ── Config ───────────────────────────────────────────────────────────────────
type Config struct {
	Repo string `json:"repo"`
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

// ── SSH helpers ───────────────────────────────────────────────────────────────

// convertToSSH converts an HTTPS GitHub/GitLab URL to its SSH equivalent.
// e.g. https://github.com/user/repo  →  git@github.com:user/repo.git
func convertToSSH(rawURL string) string {
	// Already SSH
	if strings.HasPrefix(rawURL, "git@") {
		return rawURL
	}

	// Match https://github.com/user/repo or https://gitlab.com/user/repo
	re := regexp.MustCompile(`https?://([^/]+)/([^/]+/[^/]+?)(?:\.git)?$`)
	m := re.FindStringSubmatch(rawURL)
	if m == nil {
		return rawURL // unknown format, leave unchanged
	}
	host := m[1]   // github.com | gitlab.com
	path := m[2]   // user/repo
	return fmt.Sprintf("git@%s:%s.git", host, path)
}

// sshKeyExists checks whether the user has at least one SSH key.
func sshKeyExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	keyFiles := []string{
		home + "/.ssh/id_ed25519",
		home + "/.ssh/id_rsa",
		home + "/.ssh/id_ecdsa",
	}
	for _, f := range keyFiles {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	return false
}

// testSSHAuth tests whether SSH auth to the given host works (github.com / gitlab.com).
func testSSHAuth(host string) bool {
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes", "-T", "git@"+host)
	// GitHub returns exit code 1 on success (unauthenticated banner),
	// GitLab returns 0 – we check stderr for the "Welcome" greeting.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Run() // ignore error; exit code 1 is expected for GitHub
	out := stderr.String()
	return strings.Contains(out, "successfully authenticated") ||
		strings.Contains(out, "Welcome to GitLab") ||
		strings.Contains(out, "Hi ")
}

// ── Git command runner ────────────────────────────────────────────────────────
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
		fail(errMsg)
		return false
	}
	return true
}

// ── Commands ──────────────────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(AsciiArt)
	fmt.Println(bold("gittool") + " — " + dim("Making Git simple."))
	fmt.Println()
	fmt.Println(bold("Commands:"))
	fmt.Println("  " + green("gittool init") + "              Initialize a brand new local repository")
	fmt.Println("  " + green("gittool repo add <url>") + "    Link a remote GitHub/GitLab repo (auto-uses SSH)")
	fmt.Println("  " + green("gittool status") + "            See what files you have changed")
	fmt.Println("  " + green("gittool save [\"message\"]") + "  Stage, commit & push (auto message if omitted)")
	fmt.Println("  " + green("gittool sync") + "              Pull latest remote changes then push yours")
	fmt.Println()
	fmt.Println(dim("  SSH keys are used by default. HTTPS URLs are converted automatically."))
	fmt.Println()
}

func initRepo() {
	info("Initializing local repository...")
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if !runCommand([]string{"git", "init"}) {
			return
		}
		if !runCommand([]string{"git", "branch", "-M", "main"}) {
			return
		}
	} else {
		step("Git repository already exists, skipping init.")
	}
	success("Local repository ready on branch 'main'!")
}

func setupRepo(repoURL string) {
	// Auto-convert to SSH
	sshURL := convertToSSH(repoURL)
	if sshURL != repoURL {
		info(fmt.Sprintf("Converting URL to SSH: %s", sshURL))
	}

	// Warn if SSH key is missing
	if !sshKeyExists() {
		fmt.Println(yellow("\n  ⚠   No SSH key detected!"))
		fmt.Println(dim("      Run the following to create one:"))
		fmt.Println(dim("        ssh-keygen -t ed25519 -C \"your@email.com\""))
		fmt.Println(dim("      Then add ~/.ssh/id_ed25519.pub to your GitHub/GitLab account."))
		fmt.Println()
	}

	// Test SSH connectivity
	for _, host := range []string{"github.com", "gitlab.com"} {
		if strings.Contains(sshURL, host) {
			step(fmt.Sprintf("Testing SSH connection to %s...", host))
			if testSSHAuth(host) {
				success(fmt.Sprintf("SSH authentication to %s is working!", host))
			} else {
				fmt.Println(yellow("  ⚠   Could not verify SSH auth to " + host + "."))
				fmt.Println(dim("      Make sure your public key is added to your account."))
			}
			break
		}
	}

	// Remove old origin silently
	exec.Command("git", "remote", "remove", "origin").Run()

	step(fmt.Sprintf("Linking remote: %s", sshURL))
	if runCommand([]string{"git", "remote", "add", "origin", sshURL}) {
		success("Repository successfully linked!")
	}

	// Persist the SSH URL
	if err := saveRepo(sshURL); err != nil {
		fail(fmt.Sprintf("Could not save config: %v", err))
	}
}

func getChangesSummary() string {
	cmd := exec.Command("git", "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	var added, modified, deleted []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		if len(line) < 4 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[3:])
		switch xy[0] {
		case 'A':
			added = append(added, file)
		case 'M':
			modified = append(modified, file)
		case 'D':
			deleted = append(deleted, file)
		case 'R':
			modified = append(modified, file)
		case '?':
			if xy[1] == '?' {
				added = append(added, file)
			}
		}
	}

	if len(added)+len(modified)+len(deleted) == 0 {
		return ""
	}

	var sb strings.Builder
	if len(added) > 0 {
		sb.WriteString("\nAdded:\n")
		for _, f := range added {
			sb.WriteString(fmt.Sprintf("  + %s\n", f))
		}
	}
	if len(modified) > 0 {
		sb.WriteString("\nModified:\n")
		for _, f := range modified {
			sb.WriteString(fmt.Sprintf("  ~ %s\n", f))
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
	step("Staging all changes...")
	if !runCommand([]string{"git", "add", "."}) {
		return
	}

	if message == "" {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		summary := getChangesSummary()
		if summary != "" {
			message = fmt.Sprintf("Automatic backup: %s%s", timestamp, summary)
		} else {
			message = fmt.Sprintf("Automatic backup: %s", timestamp)
		}
	}

	info("Saving with message:\n" + dim("      "+strings.ReplaceAll(message, "\n", "\n      ")))
	if !runCommand([]string{"git", "commit", "-m", message}) {
		fmt.Println(yellow("  ⚠   Nothing new to save."))
		return
	}

	step("Pushing to remote 'main'...")
	if runCommand([]string{"git", "push", "-u", "origin", "main"}) {
		success("Save and push complete!")
	}
}

func syncChanges() {
	step("Fetching latest from remote...")
	if !runCommand([]string{"git", "pull", "origin", "main", "--rebase"}) {
		return
	}
	step("Pushing your changes...")
	if runCommand([]string{"git", "push", "-u", "origin", "main"}) {
		success("Sync complete!")
	}
}

func showStatus() {
	step("Checking for changes...")
	runCommand([]string{"git", "status", "-s"})
}

// ── Entry point ───────────────────────────────────────────────────────────────
func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp()
		return
	}

	fmt.Println(AsciiArt)
	fmt.Println(strings.Repeat("─", 45))

	switch args[0] {
	case "init":
		initRepo()

	case "repo":
		if len(args) >= 3 && args[1] == "add" {
			setupRepo(args[2])
		} else {
			fail("Usage: gittool repo add <repository_url>")
		}

	case "status":
		showStatus()

	case "save":
		message := ""
		if len(args) >= 2 {
			message = strings.Join(args[1:], " ")
		}
		saveAndPush(message)

	case "sync":
		repoURL, err := loadRepo()
		if err != nil {
			fail(fmt.Sprintf("Error loading config: %v", err))
			return
		}
		if repoURL == "" {
			fail("No repository linked. Run 'gittool repo add <url>' first.")
			return
		}
		syncChanges()

	default:
		fail(fmt.Sprintf("Unknown command: '%s'", args[0]))
		printHelp()
	}
}
