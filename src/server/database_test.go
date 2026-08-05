package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImportIsIdempotentAndPreservesProgress(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "jobdeck.sqlite")
	database, err := openDatabase(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	score := 88
	artifact := batch{
		Report: batchReport{
			ID:        "calibration-1",
			Kind:      "calibration",
			Title:     "Calibration",
			Summary:   "Test batch",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Jobs: []batchJob{{
			CanonicalKey:   "greenhouse:123",
			Company:        "Example",
			Title:          "Product Engineer",
			ApplicationURL: "https://example.com/jobs/123?utm_source=test",
			Source:         "Official ATS",
			LiveVerifiedAt: time.Now().UTC().Format(time.RFC3339),
			Rank:           1,
			FitClass:       "A",
			Score:          &score,
			State:          "ready",
			Recommendation: "Direct test fit.",
		}},
	}

	if err := importBatch(context.Background(), database, artifact); err != nil {
		t.Fatal(err)
	}
	detail, err := getReport(context.Background(), database, artifact.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Jobs) != 1 {
		t.Fatalf("expected one job, got %d", len(detail.Jobs))
	}
	jobID := detail.Jobs[0].ID
	if detail.Jobs[0].ApplicationURL != "https://example.com/jobs/123" {
		t.Fatalf("unexpected canonical URL: %s", detail.Jobs[0].ApplicationURL)
	}

	if err := updateJobState(context.Background(), database, jobID, "applied", "test"); err != nil {
		t.Fatal(err)
	}
	if err := importBatch(context.Background(), database, artifact); err != nil {
		t.Fatal(err)
	}
	detail, err = getReport(context.Background(), database, artifact.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Jobs[0].State != "applied" {
		t.Fatalf("expected applied state to survive import, got %s", detail.Jobs[0].State)
	}

	status, err := databaseStatus(context.Background(), database)
	if err != nil {
		t.Fatal(err)
	}
	if status.Reports != 1 || status.Jobs != 1 || status.Applied != 1 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestFeedbackCanBeReviewed(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "jobdeck.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	artifact := testBatch()
	if err := importBatch(context.Background(), database, artifact); err != nil {
		t.Fatal(err)
	}
	detail, err := getReport(context.Background(), database, artifact.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := addFeedback(context.Background(), database, detail.Jobs[0].ID,
		artifact.Report.ID, "strong_fit", "scope", "Keep this shape"); err != nil {
		t.Fatal(err)
	}
	pending, err := listPendingFeedback(context.Background(), database)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Company != "Example" {
		t.Fatalf("unexpected pending feedback: %+v", pending)
	}
	if err := markFeedbackReviewed(context.Background(), database, pending[0].ID,
		"Confirmed product scope preference"); err != nil {
		t.Fatal(err)
	}
	pending, err = listPendingFeedback(context.Background(), database)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending feedback, got %+v", pending)
	}
}

func TestActivityLoggerWritesMachineAndReadableLogs(t *testing.T) {
	directory := t.TempDir()
	logger := newActivityLogger(directory)
	if err := logger.append("agent", "search_completed", "Completed calibration search",
		map[string]string{"reviewed": "120", "reported": "15"}); err != nil {
		t.Fatal(err)
	}

	events, err := os.ReadFile(filepath.Join(directory, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(events), `"kind":"search_completed"`) {
		t.Fatalf("event log is missing the event: %s", events)
	}
	readable, err := os.ReadFile(filepath.Join(directory, "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readable), "🤖 **Agent**") ||
		!strings.Contains(string(readable), "Reported: 15") {
		t.Fatalf("readable log has the wrong format: %s", readable)
	}
}

func TestFeedbackRoundTrip(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "jobdeck.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	artifact := testBatch()
	if err := importBatch(context.Background(), database, artifact); err != nil {
		t.Fatal(err)
	}
	detail, err := getReport(context.Background(), database, artifact.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	jobID := detail.Jobs[0].ID
	if err := addFeedback(context.Background(), database, jobID, artifact.Report.ID,
		"wrong_fit", "level", "Too senior"); err != nil {
		t.Fatal(err)
	}
	detail, err = getReport(context.Background(), database, artifact.Report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Jobs[0].FeedbackVerdict == nil || *detail.Jobs[0].FeedbackVerdict != "wrong_fit" {
		t.Fatal("feedback verdict was not returned")
	}
}

func testBatch() batch {
	score := 70
	return batch{
		Report: batchReport{
			ID:        "test-report",
			Kind:      "calibration",
			Title:     "Test report",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Jobs: []batchJob{{
			Company:        "Example",
			Title:          "Engineer",
			ApplicationURL: "https://example.com/jobs/1",
			Source:         "Official page",
			LiveVerifiedAt: time.Now().UTC().Format(time.RFC3339),
			Rank:           1,
			FitClass:       "B",
			Score:          &score,
			Recommendation: "Useful boundary case.",
		}},
	}
}
