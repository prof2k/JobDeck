package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type buildReceipt struct {
	Version          string `json:"version"`
	BuiltAt          string `json:"builtAt"`
	Platform         string `json:"platform"`
	GoVersion        string `json:"goVersion"`
	ServerSourceHash string `json:"serverSourceHash"`
	WebDistHash      string `json:"webDistHash"`
	BinaryHash       string `json:"binaryHash"`
}

func recordBuild(goVersion string, platform string) (buildReceipt, error) {
	root, err := requireProjectRoot()
	if err != nil {
		return buildReceipt{}, err
	}
	receipt, err := currentBuildReceipt(root, goVersion, platform)
	if err != nil {
		return buildReceipt{}, err
	}
	path := filepath.Join(root, "private", "runtime", "build.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return buildReceipt{}, err
	}
	contents, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return buildReceipt{}, err
	}
	contents = append(contents, '\n')
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return buildReceipt{}, fmt.Errorf("write build receipt: %w", err)
	}
	return receipt, nil
}

func verifyBuild() (buildReceipt, error) {
	root, err := requireProjectRoot()
	if err != nil {
		return buildReceipt{}, err
	}
	receiptPath := filepath.Join(root, "private", "runtime", "build.json")
	contents, err := os.ReadFile(receiptPath)
	if err != nil {
		return buildReceipt{}, fmt.Errorf("read build receipt: %w", err)
	}
	var expected buildReceipt
	if err := json.Unmarshal(contents, &expected); err != nil {
		return buildReceipt{}, fmt.Errorf("decode build receipt: %w", err)
	}
	actual, err := currentBuildReceipt(root, expected.GoVersion, expected.Platform)
	if err != nil {
		return buildReceipt{}, err
	}
	mismatches := make([]string, 0, 3)
	if expected.ServerSourceHash != actual.ServerSourceHash {
		mismatches = append(mismatches, "server source")
	}
	if expected.WebDistHash != actual.WebDistHash {
		mismatches = append(mismatches, "web output")
	}
	if expected.BinaryHash != actual.BinaryHash {
		mismatches = append(mismatches, "root executable")
	}
	if len(mismatches) > 0 {
		return expected, fmt.Errorf("build is stale or changed: %s",
			strings.Join(mismatches, ", "))
	}
	return expected, nil
}

func currentBuildReceipt(root string, goVersion string, platform string) (buildReceipt, error) {
	serverHash, err := hashTree(filepath.Join(root, "src", "server"), func(path string) bool {
		return !strings.Contains(path, string(filepath.Separator)+"webdist"+string(filepath.Separator))
	})
	if err != nil {
		return buildReceipt{}, fmt.Errorf("hash server source: %w", err)
	}
	webHash, err := hashTree(filepath.Join(root, "src", "web", "dist"), nil)
	if err != nil {
		return buildReceipt{}, fmt.Errorf("hash web output: %w", err)
	}
	binaryName := "jobdeck"
	if filepath.Separator == '\\' {
		binaryName += ".exe"
	}
	binaryHash, err := hashFile(filepath.Join(root, binaryName))
	if err != nil {
		return buildReceipt{}, fmt.Errorf("hash root executable: %w", err)
	}
	return buildReceipt{
		Version:          version,
		BuiltAt:          time.Now().UTC().Format(time.RFC3339),
		Platform:         platform,
		GoVersion:        goVersion,
		ServerSourceHash: serverHash,
		WebDistHash:      webHash,
		BinaryHash:       binaryHash,
	}, nil
}

func hashTree(root string, include func(string) bool) (string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if include == nil || include(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(hash, filepath.ToSlash(relative)); err != nil {
			return "", err
		}
		if _, err := io.WriteString(hash, "\x00"); err != nil {
			return "", err
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("root executable has not been built")
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
