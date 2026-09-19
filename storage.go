package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Post represents a single micro-post.
type Post struct {
	Timestamp time.Time
	TimeStr   string // e.g. "15:04" for display
	FullTime  string // e.g. "15:04:05" for file storage
	Content   string
}

// Storage handles reading and writing posts to Obsidian Markdown files.
type Storage struct {
	baseDir string
}

// expandHome expands a leading "~" or "~/" or "~\ " into the user's home directory.
func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// resolveVaultDir determines the directory path to use for daily notes:
// 1. If TSUB_VAULT_DIR environment variable is set, use it (with ~ expanded).
// 2. Otherwise default to ~/vault/Daily (with home dir expanded).
// Fallback: ./vault/Daily if home directory cannot be determined.
func resolveVaultDir() string {
	envDir := os.Getenv("TSUB_VAULT_DIR")
	if envDir != "" {
		return expandHome(envDir)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "vault", "Daily")
	}
	return filepath.Join(home, "vault", "Daily")
}

// NewStorage creates a new Storage instance based on TSUB_VAULT_DIR or default (~/vault/Daily).
func NewStorage() *Storage {
	return &Storage{baseDir: resolveVaultDir()}
}

// GetDailyFilePath returns the path for the given date's markdown file.
func (s *Storage) GetDailyFilePath(t time.Time) string {
	fileName := t.Format("2006-01-02") + ".md"
	return filepath.Join(s.baseDir, fileName)
}

// EnsureDailyFile ensures the directory and daily markdown file exist with header.
func (s *Storage) EnsureDailyFile(t time.Time) (string, error) {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", s.baseDir, err)
	}

	filePath := s.GetDailyFilePath(t)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dateStr := t.Format("2006-01-02")
		initialContent := fmt.Sprintf("---\ndate: %s\ntype: daily-log\ntags: [timeline]\n---\n# %s\n\n", dateStr, dateStr)
		if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
			return "", fmt.Errorf("failed to create daily file %s: %w", filePath, err)
		}
	}
	return filePath, nil
}

var postRegex = regexp.MustCompile(`^-\s+(\d{2}:\d{2}:\d{2})\s*(.*)$`)

// LoadTodayPosts loads existing posts for the date in newest-first order.
func (s *Storage) LoadTodayPosts(t time.Time) ([]Post, error) {
	filePath := s.GetDailyFilePath(t)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open daily file: %w", err)
	}
	defer file.Close()

	var chronologicalPosts []Post
	var currentPost *Post

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := postRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			if currentPost != nil {
				chronologicalPosts = append(chronologicalPosts, *currentPost)
			}
			fullTime := matches[1]
			displayTime := fullTime[:5] // "HH:MM"
			parsedTime, _ := time.Parse("15:04:05", fullTime)
			actualTime := time.Date(t.Year(), t.Month(), t.Day(), parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, t.Location())

			currentPost = &Post{
				Timestamp: actualTime,
				TimeStr:   displayTime,
				FullTime:  fullTime,
				Content:   matches[2],
			}
		} else if currentPost != nil {
			// Check if continuation of multi-line post
			trimmed := strings.TrimPrefix(line, "  ")
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				currentPost.Content += "\n" + trimmed
			}
		}
	}
	if currentPost != nil {
		chronologicalPosts = append(chronologicalPosts, *currentPost)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading daily file: %w", err)
	}

	// Reverse to newest-first
	newestFirst := make([]Post, len(chronologicalPosts))
	for i, p := range chronologicalPosts {
		newestFirst[len(chronologicalPosts)-1-i] = p
	}

	return newestFirst, nil
}

// AppendPost writes a post to the daily file.
func (s *Storage) AppendPost(t time.Time, content string) (Post, error) {
	filePath, err := s.EnsureDailyFile(t)
	if err != nil {
		return Post{}, err
	}

	fullTime := t.Format("15:04:05")
	displayTime := t.Format("15:04")

	lines := strings.Split(content, "\n")
	var formatted strings.Builder
	formatted.WriteString(fmt.Sprintf("- %s %s\n", fullTime, lines[0]))
	for i := 1; i < len(lines); i++ {
		formatted.WriteString(fmt.Sprintf("  %s\n", lines[i]))
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return Post{}, fmt.Errorf("failed to open daily file for append: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(formatted.String()); err != nil {
		return Post{}, fmt.Errorf("failed to write post to file: %w", err)
	}

	return Post{
		Timestamp: t,
		TimeStr:   displayTime,
		FullTime:  fullTime,
		Content:   content,
	}, nil
}
