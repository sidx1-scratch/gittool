package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── True-colour helpers ────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	italic = "\033[3m"
)

func rgb(r, g, b int) string     { return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b) }
func bgRGB(r, g, b int) string   { return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b) }

var (
	cGreen  = rgb(80, 220, 120)
	cRed    = rgb(255, 90, 90)
	cYellow = rgb(255, 200, 60)
	cBlue   = rgb(80, 160, 255)
	cCyan   = rgb(0, 230, 255)
	cPurple = rgb(180, 100, 255)
	cGray   = rgb(130, 130, 150)
)

func green(s string) string  { return cGreen + s + reset }
func red(s string) string    { return cRed + s + reset }
func yellow(s string) string { return cYellow + s + reset }
func blue(s string) string   { return cBlue + s + reset }
func gray(s string) string   { return cGray + s + reset }
func Purple(s string) string { return cPurple + s + reset }
func b(s string) string      { return bold + s + reset }
func d(s string) string      { return dim + s + reset }

// ─── Gradient ASCII logo ─────────────────────────────────────────────────────

type logoLine struct {
	text    string
	r, g, b int
}

var logo = []logoLine{
	{"  _____ _ _ _____           _ ", 0, 230, 255},
	{" / ____(_)__   __|         | |", 0, 195, 255},
	{"| |  __ _   | | ___   ___  | |", 0, 160, 255},
	{"| | |_ | |  | |/ _ \\ / _ \\ | |", 0, 120, 255},
	{"| |__| | |  | | (_) | (_) || |", 0, 85, 255},
	{" \\_____|_|  |_|\\___/ \\___/ |_|", 0, 50, 255},
}

func printLogo() {
	for _, l := range logo {
		fmt.Printf("%s%s%s\n", rgb(l.r, l.g, l.b), l.text, reset)
	}
}

func printSeparator() {
	fmt.Println(d(strings.Repeat("─", 47)))
}

// ─── Animated spinner ────────────────────────────────────────────────────────

type Spinner struct {
	label string
	done  chan struct{}
	wg    sync.WaitGroup
}

func spin(label string) *Spinner {
	s := &Spinner{label: label, done: make(chan struct{})}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		frames := []string{"   ", ".  ", ".. ", "..."}
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-time.After(220 * time.Millisecond):
				fmt.Printf("\r  %s%s%s%s", cGray, label, reset, frames[i%len(frames)])
				i++
			}
		}
	}()
	// Print initial state
	fmt.Printf("  %s%s%s   ", cGray, label, reset)
	return s
}

func (s *Spinner) ok(msg string) {
	close(s.done)
	s.wg.Wait()
	fmt.Printf("\r%-60s\r", "") // clear line
	fmt.Printf("  %s✔%s  %s\n", cGreen, reset, msg)
}

func (s *Spinner) fail(msg string) {
	close(s.done)
	s.wg.Wait()
	fmt.Printf("\r%-60s\r", "")
	fmt.Fprintf(os.Stderr, "  %s✘%s  %s\n", cRed, reset, msg)
}

// ─── Git helpers ─────────────────────────────────────────────────────────────

func git(args ...string) (string, string, bool) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err == nil
}

func gitSilent(args ...string) bool {
	_, _, ok := git(args...)
	return ok
}

func runVisible(args []string) bool {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stdout.Len() > 0 {
		fmt.Println(strings.TrimSpace(stdout.String()))
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		fmt.Fprintf(os.Stderr, "  %s✘%s  %s\n", cRed, reset, msg)
		return false
	}
	return true
}

// ─── Config ───────────────────────────────────────────────────────────────────

const configFile = ".gittool_config.json"

type Config struct {
	Repo string `json:"repo"`
}

func saveConfig(repoURL string) error {
	data, err := json.MarshalIndent(Config{Repo: repoURL}, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func loadConfig() (string, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return "", nil
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}
	var cfg Config
	return cfg.Repo, json.Unmarshal(data, &cfg)
}

// ─── SSH helpers ─────────────────────────────────────────────────────────────

func toSSH(rawURL string) string {
	if strings.HasPrefix(rawURL, "git@") {
		return rawURL
	}
	re := regexp.MustCompile(`https?://([^/]+)/([^/]+/[^/]+?)(?:\.git)?$`)
	m := re.FindStringSubmatch(rawURL)
	if m == nil {
		return rawURL
	}
	return fmt.Sprintf("git@%s:%s.git", m[1], m[2])
}

func hasSSHKey() bool {
	home, _ := os.UserHomeDir()
	for _, f := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		if _, err := os.Stat(home + "/.ssh/" + f); err == nil {
			return true
		}
	}
	return false
}

func testSSH(host string) bool {
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-T", "git@"+host)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Run()
	out := stderr.String()
	return strings.Contains(out, "successfully authenticated") ||
		strings.Contains(out, "Welcome to GitLab") ||
		strings.Contains(out, "Hi ")
}

// ─── Change summary ───────────────────────────────────────────────────────────

type changeSummary struct {
	added, modified, deleted []string
}

func getChanges() changeSummary {
	out, _, _ := git("status", "--porcelain")
	var cs changeSummary
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[2:])
		switch xy[0] {
		case 'A':
			cs.added = append(cs.added, file)
		case 'M':
			cs.modified = append(cs.modified, file)
		case 'D':
			cs.deleted = append(cs.deleted, file)
		case 'R':
			// Renamed: "old -> new", take new name
			if parts := strings.Split(file, " -> "); len(parts) == 2 {
				file = parts[1]
			}
			cs.modified = append(cs.modified, file)
		case '?':
			if xy[1] == '?' {
				cs.added = append(cs.added, file)
			}
		}
	}
	return cs
}

func (cs changeSummary) message(timestamp string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Automatic backup: %s", timestamp))
	if len(cs.added) > 0 {
		sb.WriteString("\n\nAdded:\n")
		for _, f := range cs.added {
			sb.WriteString(fmt.Sprintf("  + %s\n", f))
		}
	}
	if len(cs.modified) > 0 {
		sb.WriteString("\nModified:\n")
		for _, f := range cs.modified {
			sb.WriteString(fmt.Sprintf("  ~ %s\n", f))
		}
	}
	if len(cs.deleted) > 0 {
		sb.WriteString("\nDeleted:\n")
		for _, f := range cs.deleted {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}
	return sb.String()
}

func (cs changeSummary) empty() bool {
	return len(cs.added)+len(cs.modified)+len(cs.deleted) == 0
}

// ─── Open $EDITOR ────────────────────────────────────────────────────────────

func openEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"nvim", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return initial, nil // fallback: just use the auto-message
	}

	tmpFile, err := os.CreateTemp("", "gittool-commit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	// Pre-fill with the auto-generated message as a reference
	header := fmt.Sprintf("# Write your commit message above.\n# Lines starting with '#' will be removed.\n#\n# Auto-generated message:\n")
	for _, line := range strings.Split(initial, "\n") {
		header += "# " + line + "\n"
	}
	tmpFile.WriteString(header)
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	// Strip comment lines and trim
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "#") {
			lines = append(lines, l)
		}
	}
	msg := strings.TrimSpace(strings.Join(lines, "\n"))
	return msg, nil
}

// ─── Box renderer ─────────────────────────────────────────────────────────────

func box(lines []string, width int) string {
	var sb strings.Builder
	topColor := rgb(0, 160, 255)
	textColor := rgb(200, 210, 230)

	pad := func(s string, w int) string {
		// Strip ANSI for length calculation
		ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		plain := ansi.ReplaceAllString(s, "")
		spaces := w - len(plain)
		if spaces < 0 {
			spaces = 0
		}
		return s + strings.Repeat(" ", spaces)
	}

	border := func(r string) string { return topColor + r + reset }

	sb.WriteString(border("╭" + strings.Repeat("─", width+2) + "╮") + "\n")
	for _, l := range lines {
		sb.WriteString(border("│") + " " + textColor + pad(l, width) + reset + " " + border("│") + "\n")
	}
	sb.WriteString(border("╰" + strings.Repeat("─", width+2) + "╯") + "\n")
	return sb.String()
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func cmdHelp() {
	printLogo()
	fmt.Printf("\n%s%s%s %s— %sMaking Git simple.%s\n\n", bold, cGreen+"git", cBlue+"tool", reset, dim, reset)
	fmt.Println(b("Commands:"))
	cmds := [][2]string{
		{"gittool init", "Initialize a brand new local repository"},
		{"gittool repo add <url>", "Link a remote GitHub/GitLab repo (SSH auto-configured)"},
		{"gittool status", "Beautiful overview of changes, branch & remote info"},
		{"gittool save [-m]", "Stage, commit & push (opens $EDITOR with -m flag)"},
		{"gittool sync", "Pull remote changes then push yours"},
		{"gittool log", "Pretty commit log with relative timestamps"},
		{"gittool undo", "Undo the last commit (keeps changes staged)"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%-30s%s %s\n", cGreen, c[0], reset, gray(c[1]))
	}
	fmt.Printf("\n%s\n", gray("  SSH keys are used by default. HTTPS URLs auto-convert to SSH."))
	fmt.Println()
}

func cmdInit() {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		s := spin("Initializing repository")
		if !gitSilent("init") || !gitSilent("branch", "-M", "main") {
			s.fail("Failed to initialize repository")
			return
		}
		s.ok(b("Repository initialized") + gray(" on branch ") + blue("main"))
	} else {
		fmt.Printf("  %s✔%s  %s\n", cBlue, reset, "Repository already exists. Nothing to do.")
	}
}

func cmdRepoAdd(rawURL string) {
	sshURL := toSSH(rawURL)
	if sshURL != rawURL {
		fmt.Printf("  %s⟳%s  %s → %s\n", cCyan, reset, gray(rawURL), blue(sshURL))
	}

	if !hasSSHKey() {
		fmt.Printf("\n  %s⚠%s  %s\n", cYellow, reset, yellow("No SSH key found!"))
		fmt.Printf("     %s\n", gray("Create one: ssh-keygen -t ed25519 -C \"your@email.com\""))
		fmt.Printf("     %s\n\n", gray("Then add ~/.ssh/id_ed25519.pub to your GitHub/GitLab account."))
	}

	// Test SSH auth
	for _, host := range []string{"github.com", "gitlab.com"} {
		if strings.Contains(sshURL, host) {
			s := spin(fmt.Sprintf("Testing SSH auth to %s", host))
			if testSSH(host) {
				s.ok(green("SSH authentication working") + gray(" → "+host))
			} else {
				s.fail(fmt.Sprintf("SSH auth failed to %s — add your public key to your account", host))
			}
			break
		}
	}

	exec.Command("git", "remote", "remove", "origin").Run()
	s := spin("Linking remote")
	if !gitSilent("remote", "add", "origin", sshURL) {
		s.fail("Failed to link remote")
		return
	}
	s.ok(b("Remote linked") + gray(" → ") + blue(sshURL))

	if err := saveConfig(sshURL); err != nil {
		fmt.Printf("  %s⚠%s  %s\n", cYellow, reset, yellow("Could not save config: "+err.Error()))
	}
}

func cmdStatus() {
	// Gather info
	branch, _, _ := git("branch", "--show-current")
	remote, _, _ := git("remote", "get-url", "origin")
	aheadStr, _, _ := git("rev-list", "--count", "@{u}..HEAD")
	behindStr, _, _ := git("rev-list", "--count", "HEAD..@{u}")
	ahead, _ := strconv.Atoi(aheadStr)
	behind, _ := strconv.Atoi(behindStr)

	cs := getChanges()

	// Build box content
	var lines []string

	// Branch
	branchColor := green(branch)
	if branch == "" {
		branch = "unknown"
		branchColor = gray(branch)
	}
	lines = append(lines, fmt.Sprintf("%s Branch%s  %s", b(cBlue), reset, branchColor))

	// Remote
	if remote != "" {
		lines = append(lines, fmt.Sprintf("%s Remote%s  %s", b(cBlue), reset, gray(remote)))
	}

	// Ahead/behind
	sync := ""
	if ahead > 0 {
		sync += green(fmt.Sprintf("↑ %d ahead", ahead)) + "  "
	}
	if behind > 0 {
		sync += red(fmt.Sprintf("↓ %d behind", behind))
	}
	if ahead == 0 && behind == 0 && aheadStr != "" {
		sync = green("✔ up to date")
	}
	if sync != "" {
		lines = append(lines, fmt.Sprintf("%s  Sync%s   %s", b(cBlue), reset, sync))
	}

	// File changes
	if !cs.empty() {
		lines = append(lines, d("───────────────────────────────────────"))
		for _, f := range cs.added {
			lines = append(lines, green(" + ")+b(f))
		}
		for _, f := range cs.modified {
			lines = append(lines, yellow(" ~ ")+f)
		}
		for _, f := range cs.deleted {
			lines = append(lines, red(" - ")+gray(f))
		}

		// Summary count
		parts := []string{}
		if len(cs.added) > 0 {
			parts = append(parts, green(fmt.Sprintf("%d added", len(cs.added))))
		}
		if len(cs.modified) > 0 {
			parts = append(parts, yellow(fmt.Sprintf("%d modified", len(cs.modified))))
		}
		if len(cs.deleted) > 0 {
			parts = append(parts, red(fmt.Sprintf("%d deleted", len(cs.deleted))))
		}
		lines = append(lines, d("───────────────────────────────────────"))
		lines = append(lines, strings.Join(parts, gray("  ·  ")))
	} else {
		lines = append(lines, d("───────────────────────────────────────"))
		lines = append(lines, green("✔ Working tree clean"))
	}

	fmt.Print(box(lines, 45))
}

func cmdSave(args []string) {
	// Determine if editor mode (-m flag)
	editorMode := false
	manualMsg := ""
	for i, a := range args {
		if a == "-m" {
			editorMode = true
			_ = i
		} else if !strings.HasPrefix(a, "-") {
			manualMsg = strings.Join(args[i:], " ")
			break
		}
	}

	// Stage first
	s := spin("Staging all changes")
	if !gitSilent("add", ".") {
		s.fail("Failed to stage changes")
		return
	}
	cs := getChanges()
	// Note: after git add, status --porcelain shows staged differently
	// We re-run with cached flag
	s.ok("All changes staged")

	// Build message
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	autoMsg := cs.message(timestamp)

	var message string
	if editorMode {
		// Open editor
		s2 := spin("Opening $EDITOR")
		msg, err := openEditor(autoMsg)
		if err != nil || strings.TrimSpace(msg) == "" {
			s2.fail("Editor returned empty message, using auto-message")
			message = autoMsg
		} else {
			s2.ok("Commit message written")
			message = msg
		}
	} else if manualMsg != "" {
		message = manualMsg
	} else {
		message = autoMsg
	}

	// Commit
	sc := spin("Committing")
	if !gitSilent("commit", "-m", message) {
		sc.fail("Nothing to commit")
		return
	}
	sc.ok(b("Committed") + gray(" — ") + gray(strings.Split(message, "\n")[0]))

	// Push
	sp := spin("Pushing to origin/main")
	if !gitSilent("push", "-u", "origin", "main") {
		sp.fail("Push failed — check your remote and SSH key")
		return
	}
	sp.ok(green("Pushed successfully!"))
}

func cmdSync() {
	s := spin("Pulling from origin/main")
	if !gitSilent("pull", "origin", "main", "--rebase") {
		s.fail("Pull/rebase failed")
		return
	}
	s.ok("Pulled latest changes")

	s2 := spin("Pushing to origin/main")
	if !gitSilent("push", "-u", "origin", "main") {
		s2.fail("Push failed")
		return
	}
	s2.ok(green("Sync complete!"))
}

func cmdLog() {
	out, _, ok := git("log", "--pretty=format:%h|%s|%cr|%an", "-20")
	if !ok || out == "" {
		fmt.Printf("  %s\n", gray("No commits yet."))
		return
	}

	fmt.Println()
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		hash, subject, timeAgo, author := parts[0], parts[1], parts[2], parts[3]
		fmt.Printf("  %s%s%s  %s%-52s%s  %s%s%s  %s%s%s\n",
			cPurple, hash, reset,
			bold, subject, reset,
			cGray, timeAgo, reset,
			dim, author, reset,
		)
	}
	fmt.Println()
}

func cmdUndo() {
	// Show the commit being undone
	lastCommit, _, ok := git("log", "-1", "--pretty=format:%h %s")
	if !ok || lastCommit == "" {
		fmt.Printf("  %s✘%s  Nothing to undo.\n", cRed, reset)
		return
	}

	fmt.Printf("  %s⚠%s  Undoing: %s%s%s\n", cYellow, reset, gray("[ "), lastCommit, gray(" ]"))
	s := spin("Resetting last commit (changes kept staged)")
	if !gitSilent("reset", "--soft", "HEAD~1") {
		s.fail("Failed to undo last commit")
		return
	}
	s.ok(green("Undo complete!") + gray(" — changes are still staged, ready to re-commit"))
}

// ─── Entry point ─────────────────────────────────────────────────────────────

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		cmdHelp()
		return
	}

	printLogo()
	printSeparator()

	switch args[0] {
	case "init":
		cmdInit()

	case "repo":
		if len(args) >= 3 && args[1] == "add" {
			cmdRepoAdd(args[2])
		} else {
			fmt.Printf("  %s✘%s  Usage: gittool repo add <url>\n", cRed, reset)
		}

	case "status":
		cmdStatus()

	case "save":
		cmdSave(args[1:])

	case "sync":
		repo, err := loadConfig()
		if err != nil {
			fmt.Printf("  %s✘%s  Error loading config: %v\n", cRed, reset, err)
			return
		}
		if repo == "" {
			fmt.Printf("  %s✘%s  No repository linked. Run: %s\n", cRed, reset, blue("gittool repo add <url>"))
			return
		}
		cmdSync()

	case "log":
		cmdLog()

	case "undo":
		cmdUndo()

	default:
		fmt.Printf("  %s✘%s  Unknown command: %s%s%s\n", cRed, reset, yellow("'"), args[0], yellow("'"))
		fmt.Println()
		cmdHelp()
	}
}
