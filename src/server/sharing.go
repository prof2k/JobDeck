package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type sharingConfig struct {
	SnapshotHash    string `json:"snapshotHash,omitempty"`
	URL             string `json:"url"`
	Token           string `json:"token,omitempty"`
	LastPublishedAt string `json:"lastPublishedAt"`
	Error           string `json:"error"`
}

func readSharingConfig() (sharingConfig, error) {
	var config sharingConfig
	contents, err := os.ReadFile(privatePath("sharing.json"))
	if errors.Is(err, os.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(contents, &config)
	return config, err
}

func saveSharingConfig(config sharingConfig) error {
	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}
	path := privatePath("sharing.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".sharing-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func validateSharingURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("enter the shared site's base URL")
	}
	local := parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost"
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && local) {
		return "", errors.New("the shared site must use HTTPS")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (server *applicationServer) sharing(writer http.ResponseWriter, _ *http.Request) {
	server.sharingMu.Lock()
	defer server.sharingMu.Unlock()
	config, err := readSharingConfig()
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	configured := config.URL != "" && config.Token != ""
	config.Token = ""
	writeJSON(writer, http.StatusOK, struct {
		sharingConfig
		Configured bool `json:"configured"`
	}{config, configured})
}

func (server *applicationServer) configureSharing(writer http.ResponseWriter, request *http.Request) {
	if !server.authorizedMutation(request) {
		writeAPIError(writer, http.StatusForbidden, errors.New("invalid session token"))
		return
	}
	var input struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	config := sharingConfig{}
	if input.URL != "" {
		value, err := validateSharingURL(strings.TrimSpace(input.URL))
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, err)
			return
		}
		if len(input.Token) < 32 || len(input.Token) > 512 {
			writeAPIError(writer, http.StatusBadRequest, errors.New("publishing token must contain 32–512 characters"))
			return
		}
		config.URL, config.Token = value, input.Token
	}
	server.sharingMu.Lock()
	err := saveSharingConfig(config)
	server.sharingMu.Unlock()
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	server.publishAfterChange(request.Context())
	server.sharing(writer, request)
}

// A buffered notification coalesces concurrent saves without delaying local writes.
func (server *applicationServer) publishAfterChange(_ context.Context) {
	select {
	case server.sharingWake <- struct{}{}:
	default:
	}
}

func (server *applicationServer) watchProgress(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	server.flushProgress(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-server.sharingWake:
			server.flushProgress(ctx)
		case <-ticker.C:
			// This also catches imports, external ledger updates, retries, and UTC rollover.
			server.flushProgress(ctx)
		}
	}
}

func (server *applicationServer) flushProgress(ctx context.Context) {
	server.publicationMu.Lock()
	defer server.publicationMu.Unlock()
	server.sharingMu.Lock()
	config, err := readSharingConfig()
	server.sharingMu.Unlock()
	if err != nil {
		fmt.Println("jobdeck: read sharing configuration:", err)
		return
	}
	if config.URL == "" || config.Token == "" {
		return
	}
	snapshot, err := readProgress(ctx, server.database, time.Now())
	if err != nil {
		fmt.Println("jobdeck: read progress:", err)
		return
	}
	// Ignore the transport timestamp so unchanged snapshots don't trigger cloud writes.
	timestamp := snapshot.UpdatedAt
	snapshot.UpdatedAt = ""
	contents, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	digest := sha256.Sum256(contents)
	hash := hex.EncodeToString(digest[:])
	if hash == config.SnapshotHash && config.Error == "" {
		return
	}
	snapshot.UpdatedAt = timestamp
	err = server.uploadProgress(ctx, config, snapshot)
	server.sharingMu.Lock()
	defer server.sharingMu.Unlock()
	current, readErr := readSharingConfig()
	if readErr != nil {
		fmt.Println("jobdeck: read sharing status:", readErr)
		return
	}
	// A settings change while an upload was in flight must keep the newer credentials.
	if current.URL != config.URL || current.Token != config.Token {
		return
	}
	if err != nil {
		current.Error = err.Error()
	} else {
		current.Error = ""
		current.LastPublishedAt = timestamp
		current.SnapshotHash = hash
	}
	if err := saveSharingConfig(current); err != nil {
		fmt.Println("jobdeck: save sharing status:", err)
	}
}

func (server *applicationServer) uploadProgress(ctx context.Context, config sharingConfig, snapshot progressSnapshot) error {
	contents, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if len(contents) > 1<<20 {
		return errors.New("progress snapshot exceeds 1 MB")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, config.URL+"/api/progress", bytes.NewReader(contents))
	if err != nil {
		return errors.New("invalid shared site URL")
	}
	request.Header.Set("Authorization", "Bearer "+config.Token)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return errors.New("could not reach the shared page; your changes are saved locally")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("shared page returned HTTP %d; your changes are saved locally", response.StatusCode)
	}
	return nil
}
