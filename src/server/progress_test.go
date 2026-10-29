package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProgressApplicationDatesAndDeduplication(t *testing.T) {
	t.Setenv("JOBDECK_TIMEZONE", "UTC")
	db, err := openDatabase(filepath.Join(t.TempDir(), "progress.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO jobs(id,canonical_key,canonical_url,company,title,source,live_verified_at,first_seen_at,last_seen_at)
		VALUES(1,'one','https://example.com/one','Fictional','Engineer','fixture','2026-10-01','2026-10-01','2026-10-01'),
		(2,'two','https://example.com/two','Fictional','Engineer','fixture','2026-10-01','2026-10-01','2026-10-01');
		INSERT INTO job_events(job_id,event_type,occurred_at) VALUES
		(1,'applied','2026-10-28T02:00:00Z'), (1,'applied','2026-10-29T12:00:00Z'),
		(2,'opened','2026-10-28T12:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := readProgress(context.Background(), db, time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Days) != 1 || snapshot.Days[0].Date != "2026-10-28" || snapshot.Days[0].Applications != 1 {
		t.Fatalf("expected first Applied event in UTC, got %+v", snapshot.Days)
	}
	contents, _ := json.Marshal(snapshot)
	if strings.Contains(string(contents), "Fictional") || strings.Contains(string(contents), "example.com") {
		t.Fatal("job details leaked into snapshot")
	}
	t.Setenv("JOBDECK_TIMEZONE", "America/Chicago")
	snapshot, err = readProgress(context.Background(), db, time.Date(2026, 10, 29, 4, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Timezone != "America/Chicago" || snapshot.Today != "2026-10-28" || len(snapshot.Days) != 1 || snapshot.Days[0].Date != "2026-10-27" {
		t.Fatalf("local calendar day or first application grouping is wrong: %+v", snapshot)
	}
}

func TestProgressLocalMidnightAndDaylightSaving(t *testing.T) {
	t.Setenv("JOBDECK_TIMEZONE", "America/Chicago")
	for _, test := range []struct{ stamp, day string }{
		{"2026-10-29T04:59:59Z", "2026-10-28"},
		{"2026-10-29T05:00:00Z", "2026-10-29"},
		{"2026-11-02T05:59:59Z", "2026-11-01"},
		{"2026-11-02T06:00:00Z", "2026-11-02"},
		{"2026-03-08T07:59:59Z", "2026-03-08"},
		{"2026-03-08T08:00:00Z", "2026-03-08"},
	} {
		stamp, err := time.Parse(time.RFC3339, test.stamp)
		if err != nil {
			t.Fatal(err)
		}
		if got := progressDate(stamp); got != test.day {
			t.Errorf("%s: got %s, want %s", test.stamp, got, test.day)
		}
	}
	t.Setenv("JOBDECK_TIMEZONE", "Invalid/Timezone")
	if err := serveCommand(nil); err == nil || !strings.Contains(err.Error(), "JOBDECK_TIMEZONE") {
		t.Fatal("invalid time zone must fail before starting a server")
	}
}

func TestProgressRecurrencePeriods(t *testing.T) {
	for _, test := range []struct{ recurrence, date, want string }{
		{"once", "2026-10-28", "once"}, {"daily", "2026-10-28", "2026-10-28"},
		{"weekly", "2026-11-01", "2026-10-26"}, {"weekly", "2026-11-02", "2026-11-02"},
		{"monthly", "2026-11-01", "2026-11"}, {"quarterly", "2026-10-28", "2026-Q4"},
	} {
		t.Run(test.recurrence+test.date, func(t *testing.T) {
			if got := taskPeriod(progressTask{Recurrence: test.recurrence}, test.date); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestProgressMutationsPreserveHistoryAndRequireSession(t *testing.T) {
	t.Setenv("JOBDECK_ROOT", t.TempDir())
	db, err := openDatabase(filepath.Join(t.TempDir(), "progress.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &applicationServer{database: db, token: "test-token"}
	mutate := func(body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/progress", strings.NewReader(body))
		req.Header.Set("X-Jobdeck-Token", token)
		writer := httptest.NewRecorder()
		server.progressMutation(writer, req)
		return writer
	}
	if got := mutate(`{"action":"target","date":"2026-10-01","target":20}`, "").Code; got != 403 {
		t.Fatalf("unguarded mutation: %d", got)
	}
	for _, body := range []string{
		`{"action":"target","date":"2026-10-01","target":20}`,
		`{"action":"target","date":"2026-10-02","target":30}`,
		`{"action":"add","date":"2026-10-01","title":"Practice","recurrence":"daily"}`,
	} {
		if r := mutate(body, server.token); r.Code != 200 {
			t.Fatal(r.Body.String())
		}
	}
	var id string
	if err := db.QueryRow("SELECT id FROM progress_tasks").Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, day := range []string{"2026-10-01", "2026-10-02"} {
		r := mutate(`{"action":"complete","id":"`+id+`","date":"`+day+`","completed":true}`, server.token)
		if r.Code != 200 {
			t.Fatal(r.Body.String())
		}
	}
	r := mutate(`{"action":"complete","id":"`+id+`","date":"2026-10-02","completed":false}`, server.token)
	if r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	snapshot, err := readProgress(context.Background(), db, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Completions) != 1 || snapshot.Completions[0].CompletedOn != "2026-10-01" {
		t.Fatal("another day's completion was changed")
	}
	if len(snapshot.Targets) != 2 || snapshot.Targets[0].Target != 20 || snapshot.Targets[1].Target != 30 {
		t.Fatal("target history lost")
	}
	for _, body := range []string{
		`{"action":"target","date":"2099-01-01","target":20}`,
		`{"action":"target","date":"2026-02-30","target":20}`,
		`{"action":"target","date":"2026-10-01","target":-1}`,
		`{"action":"add","date":"2026-10-01","title":"Practice","recurrence":"invalid"}`,
	} {
		if r := mutate(body, server.token); r.Code != 400 {
			t.Fatalf("invalid input accepted: %s", body)
		}
	}
}

func TestSharingPublicationAndFailurePreserveLocalChanges(t *testing.T) {
	t.Setenv("JOBDECK_ROOT", t.TempDir())
	db, err := openDatabase(filepath.Join(t.TempDir(), "progress.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	status := http.StatusOK
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("x", 32) {
			t.Error("publisher authorization missing")
		}
		if r.URL.Path != "/api/progress" || r.Method != "PUT" {
			t.Error("wrong publisher endpoint")
		}
		var snapshot progressSnapshot
		if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			t.Error(err)
		}
		if snapshot.Version != 1 {
			t.Error("wrong snapshot version")
		}
		w.WriteHeader(status)
	}))
	defer remote.Close()
	config := sharingConfig{URL: remote.URL, Token: strings.Repeat("x", 32)}
	if err := saveSharingConfig(config); err != nil {
		t.Fatal(err)
	}
	server := &applicationServer{database: db}
	server.flushProgress(context.Background())
	config, err = readSharingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.LastPublishedAt == "" || config.Error != "" {
		t.Fatalf("publication not recorded: %+v", config)
	}
	status = http.StatusServiceUnavailable
	if _, err := db.Exec("INSERT INTO progress_targets(effective_on,target) VALUES(?,?)", "2026-01-01", 20); err != nil {
		t.Fatal(err)
	}
	server.flushProgress(context.Background())
	config, err = readSharingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Error == "" || config.LastPublishedAt == "" {
		t.Fatal("failure discarded last successful sync")
	}
	recorder := httptest.NewRecorder()
	server.sharing(recorder, httptest.NewRequest("GET", "/api/sharing", nil))
	if strings.Contains(recorder.Body.String(), config.Token) {
		t.Fatal("publishing credential exposed")
	}
}

func TestAutomaticPublicationRetriesAndStops(t *testing.T) {
	t.Setenv("JOBDECK_ROOT", t.TempDir())
	db, err := openDatabase(filepath.Join(t.TempDir(), "progress.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var status atomic.Int32
	status.Store(http.StatusServiceUnavailable)
	uploaded := make(chan progressSnapshot, 16)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var snapshot progressSnapshot
		if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			t.Error(err)
		}
		w.WriteHeader(int(status.Load()))
		uploaded <- snapshot
	}))
	defer remote.Close()
	if err := saveSharingConfig(sharingConfig{URL: remote.URL, Token: strings.Repeat("x", 32)}); err != nil {
		t.Fatal(err)
	}
	server := &applicationServer{database: db, sharingWake: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); server.watchProgress(ctx, 20*time.Millisecond) }()
	defer func() { cancel(); <-done }()
	awaitUpload := func() progressSnapshot {
		t.Helper()
		select {
		case snapshot := <-uploaded:
			return snapshot
		case <-time.After(2 * time.Second):
			t.Fatal("automatic update did not arrive")
			return progressSnapshot{}
		}
	}
	awaitUpload() // First upload fails.
	status.Store(http.StatusOK)
	awaitUpload()                              // A timer retries without a user action.
	server.flushProgress(context.Background()) // Wait for the successful receipt to be persisted.
	config, err := readSharingConfig()
	if err != nil || config.Error != "" || config.LastPublishedAt == "" {
		t.Fatalf("retry was not recorded: %+v, %v", config, err)
	}
	if _, err := db.Exec("INSERT INTO progress_targets(effective_on,target) VALUES(?,?)", "2026-01-01", 40); err != nil {
		t.Fatal(err)
	}
	for {
		snapshot := awaitUpload()
		if len(snapshot.Targets) == 1 && snapshot.Targets[0].Target == 40 {
			break
		}
	}
}
