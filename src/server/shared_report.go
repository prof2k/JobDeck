package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type sharedEvaluation struct {
	evaluationPayload
	RoleType         string `json:"roleType,omitempty"`
	ScreeningVerdict string `json:"screeningVerdict,omitempty"`
}

// Read within the snapshot transaction. getReport also marks reports opened,
// which would incorrectly record viewer activity as an action by the owner.
func readLatestSharedReport(ctx context.Context, tx *sql.Tx) (*reportDetail, error) {
	groupColumn := "'company'"
	columns, err := tx.QueryContext(ctx, "PRAGMA table_info(reports)")
	if err != nil {
		return nil, fmt.Errorf("inspect report fields: %w", err)
	}
	for columns.Next() {
		var id, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := columns.Scan(&id, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			columns.Close()
			return nil, err
		}
		if name == "group_by" {
			groupColumn = "group_by"
		}
	}
	if err := columns.Err(); err != nil {
		columns.Close()
		return nil, err
	}
	columns.Close()
	var detail reportDetail
	// Both possible expressions are fixed constants, never request input.
	err = tx.QueryRowContext(ctx, `SELECT id, kind, title, summary, created_at, `+groupColumn+`
		FROM reports ORDER BY datetime(created_at) DESC, id DESC LIMIT 1`).Scan(
		&detail.Report.ID, &detail.Report.Kind, &detail.Report.Title, &detail.Report.Summary,
		&detail.Report.CreatedAt, &detail.Report.GroupBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read latest report: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT j.id, j.canonical_key, j.company, j.title,
		j.location, j.arrangement, j.salary, j.canonical_url, j.source, j.live_verified_at,
		j.current_state, rj.rank, rj.fit_class, rj.score, rj.recommendation, rj.evaluation_json
		FROM report_jobs rj JOIN jobs j ON j.id = rj.job_id
		WHERE rj.report_id = ? ORDER BY rj.rank`, detail.Report.ID)
	if err != nil {
		return nil, fmt.Errorf("read latest report jobs: %w", err)
	}
	defer rows.Close()
	detail.Jobs = []reportJob{}
	for rows.Next() {
		var job reportJob
		var rawEvaluation string
		if err := rows.Scan(&job.ID, &job.CanonicalKey, &job.Company, &job.Title, &job.Location,
			&job.Arrangement, &job.Salary, &job.ApplicationURL, &job.Source, &job.LiveVerifiedAt,
			&job.State, &job.Rank, &job.FitClass, &job.Score, &job.Recommendation, &rawEvaluation); err != nil {
			return nil, err
		}
		var evaluation sharedEvaluation
		if err := json.Unmarshal([]byte(rawEvaluation), &evaluation); err != nil {
			return nil, fmt.Errorf("decode shared evaluation: %w", err)
		}
		if evaluation.Requirements == nil {
			evaluation.Requirements = []evidenceItem{}
		}
		if evaluation.Gaps == nil {
			evaluation.Gaps = []string{}
		}
		if evaluation.Risks == nil {
			evaluation.Risks = []string{}
		}
		job.Evaluation, err = json.Marshal(evaluation)
		if err != nil {
			return nil, err
		}
		// Feedback fields remain nil. Never load private feedback into this snapshot.
		detail.Jobs = append(detail.Jobs, job)
		switch job.State {
		case "ready", "opened":
			detail.Report.ReadyCount++
		case "applied":
			detail.Report.AppliedCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	detail.Report.JobCount = len(detail.Jobs)
	return &detail, nil
}
