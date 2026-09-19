package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStorage_EnsureDailyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := &Storage{baseDir: filepath.Join(tempDir, "Daily")}
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)

	filePath, err := storage.EnsureDailyFile(now)
	if err != nil {
		t.Fatalf("EnsureDailyFile failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read daily file: %v", err)
	}

	expectedHeader := "---\ndate: 2026-09-19\ntype: daily-log\ntags: [timeline]\n---\n# 2026-09-19\n"
	if !strings.HasPrefix(string(content), expectedHeader) {
		t.Errorf("expected header %q, got %q", expectedHeader, string(content))
	}
}

func TestStorage_AppendAndLoadPosts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := &Storage{baseDir: tempDir}
	baseTime := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	// Post 1
	p1Time := baseTime
	p1, err := storage.AppendPost(p1Time, "First thought of the day")
	if err != nil {
		t.Fatalf("AppendPost 1 failed: %v", err)
	}
	if p1.TimeStr != "12:00" || p1.FullTime != "12:00:00" {
		t.Errorf("unexpected time in p1: %s, %s", p1.TimeStr, p1.FullTime)
	}

	// Post 2 (Multi-line)
	p2Time := baseTime.Add(15 * time.Minute)
	_, err = storage.AppendPost(p2Time, "Second thought line 1\nSecond thought line 2")
	if err != nil {
		t.Fatalf("AppendPost 2 failed: %v", err)
	}

	// Post 3
	p3Time := baseTime.Add(30 * time.Minute)
	_, err = storage.AppendPost(p3Time, "日本語のテスト投稿 🚀")
	if err != nil {
		t.Fatalf("AppendPost 3 failed: %v", err)
	}

	// Load posts: should be newest first (p3, p2, p1)
	posts, err := storage.LoadTodayPosts(baseTime)
	if err != nil {
		t.Fatalf("LoadTodayPosts failed: %v", err)
	}

	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	// Check order (newest first)
	if posts[0].Content != "日本語のテスト投稿 🚀" {
		t.Errorf("expected posts[0] to be post 3, got: %s", posts[0].Content)
	}
	if posts[0].TimeStr != "12:30" {
		t.Errorf("expected posts[0] time to be 12:30, got: %s", posts[0].TimeStr)
	}

	if !strings.Contains(posts[1].Content, "Second thought line 1\nSecond thought line 2") {
		t.Errorf("expected posts[1] multiline content, got: %s", posts[1].Content)
	}
	if posts[1].TimeStr != "12:15" {
		t.Errorf("expected posts[1] time to be 12:15, got: %s", posts[1].TimeStr)
	}

	if posts[2].Content != "First thought of the day" {
		t.Errorf("expected posts[2] to be post 1, got: %s", posts[2].Content)
	}
	if posts[2].TimeStr != "12:00" {
		t.Errorf("expected posts[2] time to be 12:00, got: %s", posts[2].TimeStr)
	}
}
