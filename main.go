package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	server := flag.String("server", "https://ntfy.sh", "ntfy server URL")
	topic := flag.String("topic", "", "topic to publish to (required)")
	title := flag.String("title", "", "notification title")
	priority := flag.String("priority", "", "priority: min, low, default, high, max")
	tags := flag.String("tags", "", "comma-separated tags (e.g. warning,skull)")
	click := flag.String("click", "", "URL to open when notification is tapped")
	actions := flag.String("actions", "", "action buttons (e.g. \"view, Open PR, https://...\")")
	batch := flag.Bool("batch", false, "combine all stdin into one notification")
	repo := flag.String("repo", ".", "path to git repo (used to resolve commit URLs)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `ntfy-test — push notifications via ntfy

Reads messages from stdin. Sends one notification per line, or use -batch to
combine all input into a single notification.

When piping git log --oneline, the tool detects commit hashes and adds a
click URL pointing to the commit on GitHub or Gitea (auto-detected from
the git remote).

Usage:
  echo "hello" | ntfy-test -topic <topic>
  git log --oneline -3 | ntfy-test -topic <topic> -batch -title "Commits"
  git -C /path/to/repo log --oneline -1 | ntfy-test -topic <topic> -repo /path/to/repo

Options:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *topic == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Resolve the web base URL from the git remote
	webBase := repoWebURL(*repo)

	if *batch {
		all, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			os.Exit(1)
		}
		msg := strings.TrimSpace(string(all))
		if msg == "" {
			os.Exit(0)
		}

		// In batch mode, use the first line's hash for the click URL
		clickURL := *click
		if clickURL == "" && webBase != "" {
			if hash := extractHash(strings.SplitN(msg, "\n", 2)[0]); hash != "" {
				clickURL = webBase + "/commit/" + hash
			}
		}

		if err := send(*server, *topic, msg, *title, *priority, *tags, clickURL, *actions); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(msg)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := strings.TrimSpace(scanner.Text())
			if msg == "" {
				continue
			}

			clickURL := *click
			if clickURL == "" && webBase != "" {
				if hash := extractHash(msg); hash != "" {
					clickURL = webBase + "/commit/" + hash
				}
			}

			if err := send(*server, *topic, msg, *title, *priority, *tags, clickURL, *actions); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(msg)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}
}

// repoWebURL reads the git remote origin URL and converts it to a web URL.
// Supports GitHub (github.com) and Gitea (gitea.t7a.org).
func repoWebURL(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	remote := strings.TrimSpace(string(out))

	// SSH: git@github.com:org/repo.git
	// HTTPS: https://github.com/org/repo.git
	remote = strings.TrimSuffix(remote, ".git")

	if strings.HasPrefix(remote, "git@") {
		// git@host:org/repo -> https://host/org/repo
		remote = strings.TrimPrefix(remote, "git@")
		remote = strings.Replace(remote, ":", "/", 1)
		remote = "https://" + remote
	}

	if !strings.HasPrefix(remote, "https://") && !strings.HasPrefix(remote, "http://") {
		return ""
	}

	return remote
}

// extractHash checks if a line starts with a git short hash (from git log --oneline).
func extractHash(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	candidate := fields[0]
	if len(candidate) >= 7 && len(candidate) <= 40 && isHex(candidate) {
		return candidate
	}
	return ""
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func send(server, topic, message, title, priority, tags, click, actions string) error {
	url := strings.TrimRight(server, "/") + "/" + topic

	req, err := http.NewRequest("POST", url, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if title != "" {
		req.Header.Set("Title", title)
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}
	if tags != "" {
		req.Header.Set("Tags", tags)
	}
	if click != "" {
		req.Header.Set("Click", click)
	}
	if actions != "" {
		req.Header.Set("Actions", actions)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("server returned %s: %s", resp.Status, string(body))
}
