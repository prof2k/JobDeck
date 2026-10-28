package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharedSnapshotIncludesOnlyNewestBatchWithoutChangingActivity(t *testing.T) {
	db, err := openDatabase(filepath.Join(t.TempDir(), "latest.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fixture := func(id, created string) batch {
		return batch{Report: batchReport{ID: id, Kind: "search", Title: id, CreatedAt: created, RequestedCount: 1}, Jobs: []batchJob{{
			CanonicalKey: id + ":job", Company: id, Title: "Frontend Engineer", ApplicationURL: "https://example.com/" + id,
			Source: "Official ATS", LiveVerifiedAt: created, Rank: 1, FitClass: "A", State: "ready", Recommendation: "Fixture fit.",
		}}}
	}
	for _, artifact := range []batch{fixture("older", "2026-10-01T12:00:00Z"), fixture("newest", "2026-10-02T12:00:00Z")} {
		if err := importBatch(context.Background(), db, artifact); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE reports SET last_opened_at='2026-10-28T12:00:00Z' WHERE id='older';
		UPDATE report_jobs SET evaluation_json='{"requirements":[],"gaps":[],"risks":[],"roleType":"Frontend","screeningVerdict":"pass","privateNotes":"DO_NOT_SHARE"}' WHERE report_id='newest';
		INSERT INTO job_feedback(job_id,report_id,verdict,reason,note,created_at) SELECT job_id,'newest','strong_fit','DO_NOT_SHARE','DO_NOT_SHARE','2026-10-28' FROM report_jobs WHERE report_id='newest';`); err != nil {
		t.Fatal(err)
	}
	read := func() progressSnapshot {
		t.Helper()
		snapshot, err := readProgress(context.Background(), db, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	snapshot := read()
	if snapshot.LatestReport == nil || snapshot.LatestReport.Report.ID != "newest" || len(snapshot.LatestReport.Jobs) != 1 {
		t.Fatal("snapshot did not select the newest complete batch")
	}
	contents, err := json.Marshal(snapshot.LatestReport)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "DO_NOT_SHARE") || strings.Contains(string(contents), "older") {
		t.Fatal("private feedback or older batches leaked")
	}
	if !strings.Contains(string(contents), `"roleType":"Frontend"`) {
		t.Fatal("role grouping was lost")
	}
	var lastOpened *string
	if err := db.QueryRow("SELECT last_opened_at FROM reports WHERE id='newest'").Scan(&lastOpened); err != nil {
		t.Fatal(err)
	}
	if lastOpened != nil {
		t.Fatal("background sharing marked a report opened")
	}
	if _, err := db.Exec("ALTER TABLE reports ADD COLUMN group_by TEXT NOT NULL DEFAULT 'roleType'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE jobs SET current_state='applied' WHERE canonical_key='newest:job'"); err != nil {
		t.Fatal(err)
	}
	updated := read()
	if updated.LatestReport.Report.GroupBy != "roleType" || updated.LatestReport.Report.AppliedCount != 1 || updated.LatestReport.Jobs[0].State != "applied" {
		t.Fatal("grouping or application progress did not update")
	}
}
