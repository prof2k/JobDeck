package main

import (
	"errors"
	"os"
	"path/filepath"
)

func projectRoot() string {
	if configured := os.Getenv("JOBDECK_ROOT"); configured != "" {
		if absolute, err := filepath.Abs(configured); err == nil {
			return absolute
		}
	}

	if executable, err := os.Executable(); err == nil {
		if root, found := findProjectRoot(filepath.Dir(executable)); found {
			return root
		}
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		if root, found := findProjectRoot(workingDirectory); found {
			return root
		}
		return workingDirectory
	}
	return "."
}

func findProjectRoot(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if isProjectRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func isProjectRoot(path string) bool {
	if _, err := os.Stat(filepath.Join(path, "AGENTS.md")); err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(path, "src", "server"))
	return err == nil && info.IsDir()
}

func privatePath(parts ...string) string {
	pathParts := append([]string{projectRoot(), "private"}, parts...)
	return filepath.Join(pathParts...)
}

func defaultDatabasePath() string {
	return privatePath("jobdeck.sqlite")
}

func ensurePrivateWorkspace() error {
	directories := []string{
		privatePath(),
		privatePath("logs"),
		privatePath("reports"),
		privatePath("resume", "original"),
		privatePath("runtime"),
		privatePath("agent-workspace"),
	}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func requireProjectRoot() (string, error) {
	root := projectRoot()
	if !isProjectRoot(root) {
		return "", errors.New("could not locate the Job Deck project root")
	}
	return root, nil
}
