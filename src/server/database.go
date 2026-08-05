package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

func openDatabase(databasePath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	database.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	}
	for _, statement := range pragmas {
		if _, err := database.Exec(statement); err != nil {
			database.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	if _, err := database.Exec(schemaSQL); err != nil {
		database.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := ensureColumn(database, "job_feedback", "reviewed_at", "TEXT"); err != nil {
		database.Close()
		return nil, err
	}
	if err := ensureColumn(database, "job_feedback", "review_note",
		"TEXT NOT NULL DEFAULT ''"); err != nil {
		database.Close()
		return nil, err
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		database.Close()
		return nil, fmt.Errorf("protect database file: %w", err)
	}
	return database, nil
}

func ensureColumn(database *sql.DB, table string, column string, definition string) error {
	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var id int
		var name string
		var dataType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&id, &name, &dataType, &notNull, &defaultValue,
			&primaryKey); err != nil {
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	if _, err := database.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return nil
		}
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}

func importBatch(ctx context.Context, database *sql.DB, artifact batch) error {
	if err := validateBatch(artifact); err != nil {
		return err
	}

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer transaction.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO reports(id, kind, title, summary, created_at, imported_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind,
			title = excluded.title,
			summary = excluded.summary,
			created_at = excluded.created_at,
			imported_at = excluded.imported_at
	`, artifact.Report.ID, artifact.Report.Kind, artifact.Report.Title,
		artifact.Report.Summary, artifact.Report.CreatedAt, now)
	if err != nil {
		return fmt.Errorf("store report: %w", err)
	}
	if _, err := transaction.ExecContext(ctx,
		"DELETE FROM report_jobs WHERE report_id = ?", artifact.Report.ID); err != nil {
		return fmt.Errorf("replace report jobs: %w", err)
	}

	for index := range artifact.Jobs {
		job := &artifact.Jobs[index]
		canonicalURL, err := canonicalizeURL(job.ApplicationURL)
		if err != nil {
			return fmt.Errorf("job %d: %w", index+1, err)
		}
		if strings.TrimSpace(job.CanonicalKey) == "" {
			job.CanonicalKey = derivedCanonicalKey(canonicalURL)
		}
		if job.State == "" {
			job.State = "ready"
		}

		var jobID int64
		err = transaction.QueryRowContext(ctx, `
			INSERT INTO jobs(
				canonical_key, canonical_url, company, title, location, arrangement,
				salary, source, live_verified_at, current_state, first_seen_at, last_seen_at
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(canonical_key) DO UPDATE SET
				canonical_url = excluded.canonical_url,
				company = excluded.company,
				title = excluded.title,
				location = excluded.location,
				arrangement = excluded.arrangement,
				salary = excluded.salary,
				source = excluded.source,
				live_verified_at = excluded.live_verified_at,
				current_state = CASE
					WHEN jobs.current_state IN ('opened', 'applied', 'skipped')
					THEN jobs.current_state
					ELSE excluded.current_state
				END,
				last_seen_at = excluded.last_seen_at
			RETURNING id
		`, job.CanonicalKey, canonicalURL, job.Company, job.Title,
			valueOrDefault(job.Location, "Not listed"),
			valueOrDefault(job.Arrangement, "Not listed"),
			valueOrDefault(job.Salary, "Not listed"), job.Source,
			job.LiveVerifiedAt, job.State, now, now).Scan(&jobID)
		if err != nil {
			return fmt.Errorf("store job %d: %w", index+1, err)
		}

		evaluation, err := json.Marshal(evaluationPayload{
			Requirements:  job.Requirements,
			Gaps:          emptyIfNil(job.Gaps),
			Risks:         emptyIfNil(job.Risks),
			CompanyPolicy: job.CompanyPolicy,
		})
		if err != nil {
			return fmt.Errorf("encode evaluation %d: %w", index+1, err)
		}
		_, err = transaction.ExecContext(ctx, `
			INSERT INTO report_jobs(
				report_id, job_id, rank, fit_class, score, recommendation, evaluation_json
			) VALUES(?, ?, ?, ?, ?, ?, ?)
		`, artifact.Report.ID, jobID, job.Rank, job.FitClass, job.Score,
			job.Recommendation, string(evaluation))
		if err != nil {
			return fmt.Errorf("attach job %d to report: %w", index+1, err)
		}

		if job.CompanyPolicy != nil {
			policy := job.CompanyPolicy
			_, err = transaction.ExecContext(ctx, `
				INSERT INTO company_policies(
					company_key, summary, maximum_applications, window_days,
					source_url, verified_at
				) VALUES(?, ?, ?, ?, ?, ?)
				ON CONFLICT(company_key, source_url) DO UPDATE SET
					summary = excluded.summary,
					maximum_applications = excluded.maximum_applications,
					window_days = excluded.window_days,
					verified_at = excluded.verified_at
			`, normalizedCompanyKey(job.Company), policy.Summary,
				policy.MaximumApplications, policy.WindowDays, policy.SourceURL,
				policy.VerifiedAt)
			if err != nil {
				return fmt.Errorf("store company policy %d: %w", index+1, err)
			}
		}
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}

func validateBatch(artifact batch) error {
	if strings.TrimSpace(artifact.Report.ID) == "" {
		return errors.New("report id is required")
	}
	if artifact.Report.Kind != "calibration" && artifact.Report.Kind != "search" {
		return errors.New("report kind must be calibration or search")
	}
	if strings.TrimSpace(artifact.Report.Title) == "" {
		return errors.New("report title is required")
	}
	if _, err := time.Parse(time.RFC3339, artifact.Report.CreatedAt); err != nil {
		return errors.New("report createdAt must be RFC3339")
	}
	if artifact.Report.RequestedCount < 1 {
		return errors.New("report requestedCount must be a positive integer")
	}
	if artifact.Report.Kind == "calibration" && artifact.Report.RequestedCount != 15 {
		return errors.New("calibration reports require exactly 15 jobs")
	}
	if len(artifact.Jobs) == 0 {
		return errors.New("at least one job is required")
	}
	if len(artifact.Jobs) != artifact.Report.RequestedCount {
		return fmt.Errorf(
			"report requires exactly %d jobs, got %d; partial batches cannot be imported",
			artifact.Report.RequestedCount,
			len(artifact.Jobs),
		)
	}

	ranks := make(map[int]struct{}, len(artifact.Jobs))
	for index, job := range artifact.Jobs {
		if strings.TrimSpace(job.Company) == "" || strings.TrimSpace(job.Title) == "" {
			return fmt.Errorf("job %d requires company and title", index+1)
		}
		if strings.TrimSpace(job.Source) == "" {
			return fmt.Errorf("job %d requires an official source", index+1)
		}
		if _, err := time.Parse(time.RFC3339, job.LiveVerifiedAt); err != nil {
			return fmt.Errorf("job %d liveVerifiedAt must be RFC3339", index+1)
		}
		if job.Rank < 1 {
			return fmt.Errorf("job %d rank must be positive", index+1)
		}
		if _, exists := ranks[job.Rank]; exists {
			return fmt.Errorf("job %d duplicates rank %d", index+1, job.Rank)
		}
		ranks[job.Rank] = struct{}{}
		if !strings.Contains("ABCD", job.FitClass) || len(job.FitClass) != 1 {
			return fmt.Errorf("job %d fitClass must be A, B, C, or D", index+1)
		}
		if job.Score != nil && (*job.Score < 0 || *job.Score > 100) {
			return fmt.Errorf("job %d score must be between 0 and 100", index+1)
		}
		state := valueOrDefault(job.State, "ready")
		if !validState(state) {
			return fmt.Errorf("job %d has invalid state %q", index+1, state)
		}
		if strings.TrimSpace(job.Recommendation) == "" {
			return fmt.Errorf("job %d requires a recommendation", index+1)
		}
	}
	return nil
}

func listReports(ctx context.Context, database *sql.DB) ([]reportListItem, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT
			r.id, r.kind, r.title, r.summary, r.created_at, r.last_opened_at,
			COUNT(rj.job_id),
			SUM(CASE WHEN j.current_state IN ('ready', 'opened') THEN 1 ELSE 0 END),
			SUM(CASE WHEN j.current_state = 'applied' THEN 1 ELSE 0 END)
		FROM reports r
		LEFT JOIN report_jobs rj ON rj.report_id = r.id
		LEFT JOIN jobs j ON j.id = rj.job_id
		GROUP BY r.id
		ORDER BY datetime(r.created_at) DESC, r.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	reports := make([]reportListItem, 0)
	for rows.Next() {
		var item reportListItem
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Summary,
			&item.CreatedAt, &item.LastOpenedAt, &item.JobCount, &item.ReadyCount,
			&item.AppliedCount); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
}

func getReport(ctx context.Context, database *sql.DB, reportID string) (reportDetail, error) {
	reports, err := listReports(ctx, database)
	if err != nil {
		return reportDetail{}, err
	}

	var detail reportDetail
	found := false
	for _, report := range reports {
		if report.ID == reportID {
			detail.Report = report
			found = true
			break
		}
	}
	if !found {
		return reportDetail{}, sql.ErrNoRows
	}

	rows, err := database.QueryContext(ctx, `
		SELECT
			j.id, j.canonical_key, j.company, j.title, j.location, j.arrangement,
			j.salary, j.canonical_url, j.source, j.live_verified_at,
			j.current_state, rj.rank, rj.fit_class, rj.score,
			rj.recommendation, rj.evaluation_json,
			f.verdict, f.reason, f.note
		FROM report_jobs rj
		JOIN jobs j ON j.id = rj.job_id
		LEFT JOIN job_feedback f ON f.id = (
			SELECT id FROM job_feedback
			WHERE job_id = j.id AND (report_id = ? OR report_id IS NULL)
			ORDER BY datetime(created_at) DESC, id DESC LIMIT 1
		)
		WHERE rj.report_id = ?
		ORDER BY rj.rank
	`, reportID, reportID)
	if err != nil {
		return reportDetail{}, fmt.Errorf("get report jobs: %w", err)
	}
	defer rows.Close()

	detail.Jobs = make([]reportJob, 0, detail.Report.JobCount)
	for rows.Next() {
		var job reportJob
		var evaluation string
		if err := rows.Scan(&job.ID, &job.CanonicalKey, &job.Company, &job.Title,
			&job.Location, &job.Arrangement, &job.Salary, &job.ApplicationURL,
			&job.Source, &job.LiveVerifiedAt, &job.State, &job.Rank,
			&job.FitClass, &job.Score, &job.Recommendation, &evaluation,
			&job.FeedbackVerdict, &job.FeedbackReason, &job.FeedbackNote); err != nil {
			return reportDetail{}, fmt.Errorf("scan report job: %w", err)
		}
		job.Evaluation = json.RawMessage(evaluation)
		detail.Jobs = append(detail.Jobs, job)
	}
	if err := rows.Err(); err != nil {
		return reportDetail{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.ExecContext(ctx,
		"UPDATE reports SET last_opened_at = ? WHERE id = ?", now, reportID); err != nil {
		return reportDetail{}, fmt.Errorf("touch report: %w", err)
	}
	detail.Report.LastOpenedAt = &now
	return detail, nil
}

func updateJobState(
	ctx context.Context,
	database *sql.DB,
	jobID int64,
	state string,
	note string,
) error {
	if !validState(state) {
		return fmt.Errorf("invalid state %q", state)
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(ctx,
		"UPDATE jobs SET current_state = ? WHERE id = ?", state, jobID)
	if err != nil {
		return fmt.Errorf("update job state: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO job_events(job_id, event_type, occurred_at, note)
		VALUES(?, ?, ?, ?)
	`, jobID, state, time.Now().UTC().Format(time.RFC3339), note)
	if err != nil {
		return fmt.Errorf("record job event: %w", err)
	}
	return transaction.Commit()
}

func addFeedback(
	ctx context.Context,
	database *sql.DB,
	jobID int64,
	reportID string,
	verdict string,
	reason string,
	note string,
) error {
	validVerdicts := map[string]bool{
		"strong_fit": true,
		"maybe":      true,
		"wrong_fit":  true,
		"never_show": true,
	}
	if !validVerdicts[verdict] {
		return fmt.Errorf("invalid feedback verdict %q", verdict)
	}
	result, err := database.ExecContext(ctx, `
		INSERT INTO job_feedback(job_id, report_id, verdict, reason, note, created_at)
		SELECT id, ?, ?, ?, ?, ? FROM jobs WHERE id = ?
	`, valueOrNil(reportID), verdict, reason, note,
		time.Now().UTC().Format(time.RFC3339), jobID)
	if err != nil {
		return fmt.Errorf("store feedback: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func listPendingFeedback(ctx context.Context, database *sql.DB) ([]pendingFeedbackRecord, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT
			f.id, f.created_at, COALESCE(f.report_id, ''), f.job_id,
			j.company, j.title, j.canonical_url,
			f.verdict, f.reason, f.note
		FROM job_feedback f
		JOIN jobs j ON j.id = f.job_id
		WHERE f.reviewed_at IS NULL
		ORDER BY datetime(f.created_at), f.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list pending feedback: %w", err)
	}
	defer rows.Close()
	items := make([]pendingFeedbackRecord, 0)
	for rows.Next() {
		var item pendingFeedbackRecord
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.ReportID, &item.JobID,
			&item.Company, &item.Title, &item.URL, &item.Verdict, &item.Reason,
			&item.Note); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func markFeedbackReviewed(
	ctx context.Context,
	database *sql.DB,
	feedbackID int64,
	note string,
) error {
	result, err := database.ExecContext(ctx, `
		UPDATE job_feedback
		SET reviewed_at = ?, review_note = ?
		WHERE id = ? AND reviewed_at IS NULL
	`, time.Now().UTC().Format(time.RFC3339), note, feedbackID)
	if err != nil {
		return fmt.Errorf("resolve feedback: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func databaseStatus(ctx context.Context, database *sql.DB) (statusSummary, error) {
	var summary statusSummary
	err := database.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM reports),
			COUNT(*),
			COALESCE(SUM(CASE WHEN current_state = 'ready' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN current_state = 'opened' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN current_state = 'applied' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN current_state = 'skipped' THEN 1 ELSE 0 END), 0)
		FROM jobs
	`).Scan(&summary.Reports, &summary.Jobs, &summary.Ready, &summary.Opened,
		&summary.Applied, &summary.Skipped)
	if err != nil {
		return statusSummary{}, err
	}
	return summary, nil
}

func jobIdentity(ctx context.Context, database *sql.DB, jobID int64) (string, string, error) {
	var company string
	var title string
	if err := database.QueryRowContext(ctx,
		"SELECT company, title FROM jobs WHERE id = ?", jobID).Scan(&company,
		&title); err != nil {
		return "", "", err
	}
	return company, title, nil
}

func validState(state string) bool {
	switch state {
	case "ready", "opened", "applied", "skipped", "expired", "rejected":
		return true
	default:
		return false
	}
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func valueOrNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
