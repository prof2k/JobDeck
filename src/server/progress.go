package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const progressTimezone = "UTC"

type progressTask struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Recurrence string  `json:"recurrence"`
	StartsOn   string  `json:"startsOn"`
	ArchivedOn *string `json:"archivedOn"`
}

type progressCompletion struct {
	TaskID      string `json:"taskId"`
	Period      string `json:"period"`
	CompletedOn string `json:"completedOn"`
}

type progressTarget struct {
	EffectiveOn string `json:"effectiveOn"`
	Target      int    `json:"target"`
}

type progressDay struct {
	Date         string `json:"date"`
	Applications int    `json:"applications"`
}

// Shared progress deliberately excludes job details, notes, resumes, and credentials.
type progressSnapshot struct {
	Version     int                  `json:"version"`
	UpdatedAt   string               `json:"updatedAt"`
	Timezone    string               `json:"timezone"`
	Today       string               `json:"today"`
	Tasks       []progressTask       `json:"tasks"`
	Completions []progressCompletion `json:"completions"`
	Targets     []progressTarget     `json:"targets"`
	Days        []progressDay        `json:"days"`
}

func progressDate(now time.Time) string {
	return now.UTC().Format(time.DateOnly)
}

func parseProgressDate(value string) (time.Time, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Format(time.DateOnly) != value {
		return time.Time{}, errors.New("use a valid date in YYYY-MM-DD format")
	}
	return parsed, nil
}

func taskPeriod(task progressTask, day string) string {
	date, _ := time.Parse(time.DateOnly, day)
	switch task.Recurrence {
	case "once":
		return "once"
	case "weekly":
		return date.AddDate(0, 0, -(int(date.Weekday())+6)%7).Format(time.DateOnly)
	case "monthly":
		return day[:7]
	case "quarterly":
		return fmt.Sprintf("%d-Q%d", date.Year(), (int(date.Month())-1)/3+1)
	default:
		return day
	}
}

func readProgress(ctx context.Context, database *sql.DB, now time.Time) (progressSnapshot, error) {
	result := progressSnapshot{Version: 1, UpdatedAt: now.UTC().Format(time.RFC3339Nano),
		Timezone: progressTimezone, Today: progressDate(now), Tasks: []progressTask{},
		Completions: []progressCompletion{}, Targets: []progressTarget{}, Days: []progressDay{}}
	// A transaction keeps a snapshot internally consistent while other tabs save changes.
	tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id, title, recurrence, starts_on, archived_on FROM progress_tasks ORDER BY starts_on, rowid")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var task progressTask
		if err := rows.Scan(&task.ID, &task.Title, &task.Recurrence, &task.StartsOn, &task.ArchivedOn); err != nil {
			rows.Close()
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	rows, err = tx.QueryContext(ctx, "SELECT task_id, period, completed_on FROM progress_completions ORDER BY completed_on, task_id")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var completion progressCompletion
		if err := rows.Scan(&completion.TaskID, &completion.Period, &completion.CompletedOn); err != nil {
			rows.Close()
			return result, err
		}
		result.Completions = append(result.Completions, completion)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	rows, err = tx.QueryContext(ctx, "SELECT effective_on, target FROM progress_targets ORDER BY effective_on")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var target progressTarget
		if err := rows.Scan(&target.EffectiveOn, &target.Target); err != nil {
			rows.Close()
			return result, err
		}
		result.Targets = append(result.Targets, target)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	// Count each job once, on its first explicit Applied event, in UTC.
	rows, err = tx.QueryContext(ctx, "SELECT MIN(occurred_at) FROM job_events WHERE event_type = 'applied' GROUP BY job_id")
	if err != nil {
		return result, err
	}
	counts := map[string]int{}
	for rows.Next() {
		var stamp string
		if err := rows.Scan(&stamp); err != nil {
			rows.Close()
			return result, err
		}
		when, err := time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			rows.Close()
			return result, fmt.Errorf("invalid application timestamp: %w", err)
		}
		counts[progressDate(when)]++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	for day, count := range counts {
		result.Days = append(result.Days, progressDay{Date: day, Applications: count})
	}
	sort.Slice(result.Days, func(i, j int) bool { return result.Days[i].Date < result.Days[j].Date })
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func (server *applicationServer) progress(writer http.ResponseWriter, request *http.Request) {
	result, err := readProgress(request.Context(), server.database, time.Now())
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (server *applicationServer) progressMutation(writer http.ResponseWriter, request *http.Request) {
	if !server.authorizedMutation(request) {
		writeAPIError(writer, http.StatusForbidden, errors.New("invalid session token"))
		return
	}
	var input struct {
		Action     string `json:"action"`
		ID         string `json:"id"`
		Title      string `json:"title"`
		Recurrence string `json:"recurrence"`
		Date       string `json:"date"`
		Completed  bool   `json:"completed"`
		Target     int    `json:"target"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	today := progressDate(time.Now())
	if _, err := parseProgressDate(input.Date); err != nil || input.Date > today {
		writeAPIError(writer, http.StatusBadRequest, errors.New("choose a valid date on or before today"))
		return
	}
	var err error
	switch input.Action {
	case "add":
		title := strings.TrimSpace(input.Title)
		if title == "" || len(title) > 200 {
			writeAPIError(writer, http.StatusBadRequest, errors.New("task title must contain 1–200 bytes"))
			return
		}
		if !strings.Contains("|once|daily|weekly|monthly|quarterly|", "|"+input.Recurrence+"|") || input.Recurrence == "" {
			writeAPIError(writer, http.StatusBadRequest, errors.New("choose a supported recurrence"))
			return
		}
		id := make([]byte, 16)
		if _, err = rand.Read(id); err == nil {
			_, err = server.database.ExecContext(request.Context(), "INSERT INTO progress_tasks(id,title,recurrence,starts_on) VALUES(?,?,?,?)", hex.EncodeToString(id), title, input.Recurrence, input.Date)
		}
	case "target":
		if input.Target < 0 || input.Target > 1000 {
			writeAPIError(writer, http.StatusBadRequest, errors.New("target must be between 0 and 1000"))
			return
		}
		_, err = server.database.ExecContext(request.Context(), "INSERT INTO progress_targets(effective_on,target) VALUES(?,?) ON CONFLICT(effective_on) DO UPDATE SET target=excluded.target", input.Date, input.Target)
	case "complete", "archive":
		var task progressTask
		err = server.database.QueryRowContext(request.Context(), "SELECT id,title,recurrence,starts_on,archived_on FROM progress_tasks WHERE id=?", input.ID).Scan(&task.ID, &task.Title, &task.Recurrence, &task.StartsOn, &task.ArchivedOn)
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(writer, http.StatusNotFound, errors.New("task not found"))
			return
		}
		if err != nil {
			break
		}
		if input.Date < task.StartsOn || (task.ArchivedOn != nil && input.Date >= *task.ArchivedOn) {
			writeAPIError(writer, http.StatusBadRequest, errors.New("task is not active on this date"))
			return
		}
		if input.Action == "archive" {
			_, err = server.database.ExecContext(request.Context(), "UPDATE progress_tasks SET archived_on=? WHERE id=?", input.Date, input.ID)
		} else if input.Completed {
			_, err = server.database.ExecContext(request.Context(), "INSERT INTO progress_completions(task_id,period,completed_on) VALUES(?,?,?) ON CONFLICT(task_id,period) DO NOTHING", input.ID, taskPeriod(task, input.Date), input.Date)
		} else {
			_, err = server.database.ExecContext(request.Context(), "DELETE FROM progress_completions WHERE task_id=? AND period=?", input.ID, taskPeriod(task, input.Date))
		}
	default:
		writeAPIError(writer, http.StatusBadRequest, errors.New("unknown progress action"))
		return
	}
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	server.publishAfterChange(request.Context())
	server.progress(writer, request)
}
