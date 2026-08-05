package main

import "encoding/json"

type batch struct {
	Report batchReport `json:"report"`
	Jobs   []batchJob  `json:"jobs"`
}

type batchReport struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	CreatedAt      string `json:"createdAt"`
	RequestedCount int    `json:"requestedCount"`
}

type batchJob struct {
	CanonicalKey   string         `json:"canonicalKey"`
	Company        string         `json:"company"`
	Title          string         `json:"title"`
	Location       string         `json:"location"`
	Arrangement    string         `json:"arrangement"`
	Salary         string         `json:"salary"`
	ApplicationURL string         `json:"applicationUrl"`
	Source         string         `json:"source"`
	LiveVerifiedAt string         `json:"liveVerifiedAt"`
	Rank           int            `json:"rank"`
	FitClass       string         `json:"fitClass"`
	Score          *int           `json:"score"`
	State          string         `json:"state"`
	Recommendation string         `json:"recommendation"`
	Requirements   []evidenceItem `json:"requirements"`
	Gaps           []string       `json:"gaps"`
	Risks          []string       `json:"risks"`
	CompanyPolicy  *companyPolicy `json:"companyPolicy,omitempty"`
}

type evidenceItem struct {
	Requirement string `json:"requirement"`
	Match       string `json:"match"`
	Evidence    string `json:"evidence"`
	Source      string `json:"source"`
}

type companyPolicy struct {
	Summary             string `json:"summary"`
	MaximumApplications *int   `json:"maximumApplications,omitempty"`
	WindowDays          *int   `json:"windowDays,omitempty"`
	SourceURL           string `json:"sourceUrl"`
	VerifiedAt          string `json:"verifiedAt"`
}

type evaluationPayload struct {
	Requirements  []evidenceItem `json:"requirements"`
	Gaps          []string       `json:"gaps"`
	Risks         []string       `json:"risks"`
	CompanyPolicy *companyPolicy `json:"companyPolicy,omitempty"`
}

type reportListItem struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Summary      string  `json:"summary"`
	CreatedAt    string  `json:"createdAt"`
	LastOpenedAt *string `json:"lastOpenedAt"`
	JobCount     int     `json:"jobCount"`
	ReadyCount   int     `json:"readyCount"`
	AppliedCount int     `json:"appliedCount"`
}

type reportDetail struct {
	Report reportListItem `json:"report"`
	Jobs   []reportJob    `json:"jobs"`
}

type reportJob struct {
	ID              int64           `json:"id"`
	CanonicalKey    string          `json:"canonicalKey"`
	Company         string          `json:"company"`
	Title           string          `json:"title"`
	Location        string          `json:"location"`
	Arrangement     string          `json:"arrangement"`
	Salary          string          `json:"salary"`
	ApplicationURL  string          `json:"applicationUrl"`
	Source          string          `json:"source"`
	LiveVerifiedAt  string          `json:"liveVerifiedAt"`
	State           string          `json:"state"`
	Rank            int             `json:"rank"`
	FitClass        string          `json:"fitClass"`
	Score           *int            `json:"score"`
	Recommendation  string          `json:"recommendation"`
	Evaluation      json.RawMessage `json:"evaluation"`
	FeedbackVerdict *string         `json:"feedbackVerdict"`
	FeedbackReason  *string         `json:"feedbackReason"`
	FeedbackNote    *string         `json:"feedbackNote"`
}

type statusSummary struct {
	Reports int `json:"reports"`
	Jobs    int `json:"jobs"`
	Ready   int `json:"ready"`
	Opened  int `json:"opened"`
	Applied int `json:"applied"`
	Skipped int `json:"skipped"`
}

type pendingFeedbackRecord struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"createdAt"`
	ReportID  string `json:"reportId"`
	JobID     int64  `json:"jobId"`
	Company   string `json:"company"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Verdict   string `json:"verdict"`
	Reason    string `json:"reason"`
	Note      string `json:"note"`
}
