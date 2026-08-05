package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type activityLogger struct {
	directory string
	mutex     sync.Mutex
}

type activityEvent struct {
	OccurredAt string            `json:"occurredAt"`
	Actor      string            `json:"actor"`
	Kind       string            `json:"kind"`
	Summary    string            `json:"summary"`
	Details    map[string]string `json:"details,omitempty"`
}

func newActivityLogger(directory string) *activityLogger {
	return &activityLogger{directory: directory}
}

func (logger *activityLogger) append(
	actor string,
	kind string,
	summary string,
	details map[string]string,
) error {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	if err := os.MkdirAll(logger.directory, 0o700); err != nil {
		return fmt.Errorf("create activity log directory: %w", err)
	}
	event := activityEvent{
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Actor:      actor,
		Kind:       kind,
		Summary:    cleanLogText(summary),
		Details:    cleanLogDetails(details),
	}
	if err := logger.appendJSON(event); err != nil {
		return err
	}
	if err := logger.appendMarkdown(event); err != nil {
		return err
	}
	return nil
}

func (logger *activityLogger) appendJSON(event activityEvent) error {
	path := filepath.Join(logger.directory, "events.jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(event); err != nil {
		return fmt.Errorf("write event log: %w", err)
	}
	return nil
}

func (logger *activityLogger) appendMarkdown(event activityEvent) error {
	path := filepath.Join(logger.directory, "log.md")
	isEmpty := true
	if info, err := os.Stat(path); err == nil {
		isEmpty = info.Size() == 0
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open readable activity log: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if isEmpty {
		if _, err := writer.WriteString("# Job Deck activity\n\n"); err != nil {
			return err
		}
	}
	when, _ := time.Parse(time.RFC3339, event.OccurredAt)
	if _, err := fmt.Fprintf(writer, "## %s — %s\n\n%s **%s**\n\n",
		when.Local().Format("2006-01-02 15:04"), event.Summary,
		actorIcon(event.Actor), actorLabel(event.Actor)); err != nil {
		return err
	}
	keys := make([]string, 0, len(event.Details))
	for key := range event.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(writer, "- %s: %s\n", displayLogKey(key),
			event.Details[key]); err != nil {
			return err
		}
	}
	if _, err := writer.WriteString("\n---\n\n"); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush readable activity log: %w", err)
	}
	return nil
}

func cleanLogDetails(details map[string]string) map[string]string {
	if len(details) == 0 {
		return nil
	}
	cleaned := make(map[string]string, len(details))
	for key, value := range details {
		cleaned[cleanLogText(key)] = cleanLogText(value)
	}
	return cleaned
}

func cleanLogText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func actorIcon(actor string) string {
	switch actor {
	case "user":
		return "👤"
	case "agent":
		return "🤖"
	default:
		return "⚙️"
	}
}

func actorLabel(actor string) string {
	switch actor {
	case "user":
		return "User"
	case "agent":
		return "Agent"
	default:
		return "System"
	}
}

func displayLogKey(key string) string {
	if key == "" {
		return "Detail"
	}
	return strings.ToUpper(key[:1]) + strings.ReplaceAll(key[1:], "_", " ")
}
