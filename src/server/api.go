package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type applicationServer struct {
	database *sql.DB
	logger   *activityLogger
	port     int
	token    string
	ui       http.Handler
}

func newApplicationServer(
	database *sql.DB,
	port int,
	logger *activityLogger,
) (*applicationServer, error) {
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("create session token: %w", err)
	}
	uiHandler, err := embeddedUIHandler()
	if err != nil {
		return nil, err
	}
	return &applicationServer{
		database: database,
		logger:   logger,
		port:     port,
		token:    hex.EncodeToString(tokenBytes),
		ui:       uiHandler,
	}, nil
}

func (server *applicationServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", server.health)
	mux.HandleFunc("GET /api/session", server.session)
	mux.HandleFunc("GET /api/summary", server.summary)
	mux.HandleFunc("GET /api/reports", server.reports)
	mux.HandleFunc("GET /api/reports/{reportID}", server.report)
	mux.HandleFunc("PUT /api/jobs/{jobID}/state", server.jobState)
	mux.HandleFunc("POST /api/jobs/{jobID}/feedback", server.feedback)
	mux.Handle("/", server.ui)
	return server.securityHeaders(server.validateRequest(mux))
}

func (server *applicationServer) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"app":     "jobdeck",
		"status":  "ok",
		"version": version,
	})
}

func (server *applicationServer) session(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"token": server.token})
}

func (server *applicationServer) summary(writer http.ResponseWriter, request *http.Request) {
	summary, err := databaseStatus(request.Context(), server.database)
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, summary)
}

func (server *applicationServer) reports(writer http.ResponseWriter, request *http.Request) {
	reports, err := listReports(request.Context(), server.database)
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, reports)
}

func (server *applicationServer) report(writer http.ResponseWriter, request *http.Request) {
	reportID := request.PathValue("reportID")
	detail, err := getReport(request.Context(), server.database, reportID)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(writer, http.StatusNotFound, errors.New("report not found"))
		return
	}
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, detail)
}

func (server *applicationServer) jobState(writer http.ResponseWriter, request *http.Request) {
	if !server.authorizedMutation(request) {
		writeAPIError(writer, http.StatusForbidden, errors.New("invalid session token"))
		return
	}
	jobID, err := strconv.ParseInt(request.PathValue("jobID"), 10, 64)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, errors.New("invalid job id"))
		return
	}
	var input struct {
		State string `json:"state"`
		Note  string `json:"note"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if err := updateJobState(request.Context(), server.database, jobID,
		input.State, input.Note); errors.Is(err, sql.ErrNoRows) {
		writeAPIError(writer, http.StatusNotFound, errors.New("job not found"))
		return
	} else if err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	company, title, labelErr := jobIdentity(request.Context(), server.database, jobID)
	if labelErr == nil {
		if logErr := server.logger.append("user", "job_state_changed",
			"Updated application progress", map[string]string{
				"job":   company + " — " + title,
				"state": input.State,
			}); logErr != nil {
			fmt.Println("jobdeck: activity log:", logErr)
		}
	}
	writeJSON(writer, http.StatusOK, map[string]string{"state": input.State})
}

func (server *applicationServer) feedback(writer http.ResponseWriter, request *http.Request) {
	if !server.authorizedMutation(request) {
		writeAPIError(writer, http.StatusForbidden, errors.New("invalid session token"))
		return
	}
	jobID, err := strconv.ParseInt(request.PathValue("jobID"), 10, 64)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, errors.New("invalid job id"))
		return
	}
	var input struct {
		ReportID string `json:"reportId"`
		Verdict  string `json:"verdict"`
		Reason   string `json:"reason"`
		Note     string `json:"note"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if err := addFeedback(request.Context(), server.database, jobID, input.ReportID,
		input.Verdict, input.Reason, input.Note); errors.Is(err, sql.ErrNoRows) {
		writeAPIError(writer, http.StatusNotFound, errors.New("job not found"))
		return
	} else if err != nil {
		writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	company, title, labelErr := jobIdentity(request.Context(), server.database, jobID)
	if labelErr == nil {
		if logErr := server.logger.append("user", "job_feedback_saved",
			"Saved job-fit feedback", map[string]string{
				"job":     company + " — " + title,
				"verdict": input.Verdict,
				"reason":  input.Reason,
			}); logErr != nil {
			fmt.Println("jobdeck: activity log:", logErr)
		}
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (server *applicationServer) authorizedMutation(request *http.Request) bool {
	return request.Header.Get("X-Jobdeck-Token") == server.token
}

func (server *applicationServer) validateRequest(next http.Handler) http.Handler {
	allowedHosts := map[string]bool{
		net.JoinHostPort("127.0.0.1", strconv.Itoa(server.port)): true,
		net.JoinHostPort("localhost", strconv.Itoa(server.port)): true,
	}
	allowedOrigins := map[string]bool{
		fmt.Sprintf("http://127.0.0.1:%d", server.port): true,
		fmt.Sprintf("http://localhost:%d", server.port): true,
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !allowedHosts[request.Host] {
			writeAPIError(writer, http.StatusForbidden, errors.New("unexpected host"))
			return
		}
		origin := request.Header.Get("Origin")
		if origin != "" && !allowedOrigins[origin] {
			writeAPIError(writer, http.StatusForbidden, errors.New("unexpected origin"))
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (server *applicationServer) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy",
			"default-src 'self'; connect-src 'self'; img-src 'self' data:; "+
				"style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'none'; "+
				"form-action 'none'; frame-ancestors 'none'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(request.URL.Path, "/api/") {
			writer.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(writer, request)
	})
}

func decodeJSON(request *http.Request, destination any) error {
	defer request.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		return
	}
}

func writeAPIError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{
		"error":     http.StatusText(status),
		"message":   err.Error(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
