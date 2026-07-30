// Package models holds the wire and storage shapes, plus the mapping between
// them. It has no dependencies on the database or HTTP layers on purpose:
// Normalize is pure, so it can be unit tested against real API payloads
// without a container running.
package models

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

// ---------- Incoming: the shape of ONE item from the RapidAPI response ----------
// Only the fields we actually use are mapped; the rest ride along in RawPayload.
type APIJob struct {
	ID              int64    `json:"id"`
	LinkedInID      int64    `json:"linkedin_id"`
	Title           string   `json:"title"`
	Organization    string   `json:"organization"`
	OrganizationURL string   `json:"organization_url"`
	URL             string   `json:"url"`
	Source          string   `json:"source"`
	SourceDomain    string   `json:"source_domain"`
	DescriptionText string   `json:"description_text"`
	DatePosted      string   `json:"date_posted"`
	DateValidThru   string   `json:"date_valid_through"`
	EmploymentType  []string `json:"employment_type"`
	Seniority       string   `json:"seniority"`
	DirectApply     bool     `json:"direct_apply"`

	CountriesDerived []string `json:"countries_derived"`
	LocationsDerived []string `json:"locations_derived"`

	AIKeySkills            []string `json:"ai_key_skills"`
	AIKeywords             []string `json:"ai_keywords"`
	AIExperienceLevel      string   `json:"ai_experience_level"`
	AIWorkArrangement      string   `json:"ai_work_arrangement"`
	AIRequirementsSummary  string   `json:"ai_requirements_summary"`
	AICoreResponsibilities string   `json:"ai_core_responsibilities"`

	AISalaryCurrency string   `json:"ai_salary_currency"`
	AISalaryMin      *float64 `json:"ai_salary_min_value"`
	AISalaryMax      *float64 `json:"ai_salary_max_value"`
	AISalaryUnit     string   `json:"ai_salary_unit_text"`

	// The employer's real website. organization_url is a LinkedIn company page
	// and is useless for matching inbound mail; this is the actual domain.
	OrgWebsite string `json:"org_linkedin_website"`

	Locations []struct {
		Address struct {
			AddressCountry  string `json:"addressCountry"`
			AddressLocality string `json:"addressLocality"`
		} `json:"address"`
	} `json:"locations"`
}

// ---------- Stored: our jobs row (what the Go API serves to Flutter) ----------
type Job struct {
	ID          string `json:"id"`
	SourceJobID int64  `json:"source_job_id"`
	// LinkedInID and NormalizedURL are dedup keys, not display data. Zero /
	// empty means "this source didn't give us one" — see internal/ingest.
	LinkedInID    int64  `json:"linkedin_id,omitempty"`
	NormalizedURL string `json:"normalized_url,omitempty"`

	Title           string     `json:"title"`
	Organization    string     `json:"organization"`
	OrganizationURL string     `json:"organization_url,omitempty"`
	URL             string     `json:"url"`
	Source          string     `json:"source,omitempty"`
	SourceDomain    string     `json:"source_domain,omitempty"`
	DescriptionText string     `json:"description_text,omitempty"`
	DatePosted      *time.Time `json:"date_posted"`
	DateValidThru   *time.Time `json:"date_valid_through,omitempty"`

	// OrgDomain is the employer's REAL domain, lifted from
	// raw_payload.org_linkedin_website at ingest. organization_url cannot serve
	// here: it is a linkedin.com/company/... page, so every job would appear to
	// share one domain and inbox matching would attribute mail at random.
	OrgDomain string `json:"org_domain,omitempty"`

	Country         string `json:"country"`
	LocationRaw     string `json:"location_raw"`
	WorkArrangement string `json:"work_arrangement"`
	EmploymentType  string `json:"employment_type"`
	Seniority       string `json:"seniority,omitempty"`
	ExperienceLevel string `json:"experience_level"`
	DirectApply     bool   `json:"direct_apply"`

	AIKeySkills            json.RawMessage `json:"ai_key_skills"`
	AIKeywords             json.RawMessage `json:"ai_keywords,omitempty"`
	AIRequirementsSummary  string          `json:"ai_requirements_summary,omitempty"`
	AICoreResponsibilities string          `json:"ai_core_responsibilities,omitempty"`

	SalaryCurrency string   `json:"salary_currency,omitempty"`
	SalaryMin      *float64 `json:"salary_min,omitempty"`
	SalaryMax      *float64 `json:"salary_max,omitempty"`
	SalaryUnit     string   `json:"salary_unit,omitempty"`

	// ---- our pipeline fields ----
	Status         string          `json:"status"`
	Score          *int            `json:"score"`
	MatchedSkills  json.RawMessage `json:"matched_skills,omitempty"`
	MissingSkills  json.RawMessage `json:"missing_skills,omitempty"`
	AISummary      string          `json:"ai_summary,omitempty"`
	CVURL          string          `json:"cv_url,omitempty"`
	CoverLetterURL string          `json:"cover_letter_url,omitempty"`
	PromptUsed     string          `json:"prompt_used,omitempty"`
	DateApplied    *time.Time      `json:"date_applied"`
	EmailUsed      string          `json:"email_used,omitempty"`

	// Generated documents (WF-C). Stored as text on the row rather than files
	// on a volume: no orphans when a job is rejected, nothing extra to back up,
	// and the text stays editable right up to approval.
	CVText          string     `json:"cv_text,omitempty"`
	CoverLetterText string     `json:"cover_letter_text,omitempty"`
	GeneratedAt     *time.Time `json:"generated_at,omitempty"`
	GeneratedBy     string     `json:"generated_by,omitempty"`

	// RawPayload is the untouched API item, kept so a scoring or generation
	// bug can be replayed without burning another API call from the quota.
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Profile is the single settings row (id = 1) the app writes and n8n reads.
type Profile struct {
	MasterCV          string          `json:"master_cv"`
	SearchTitles      json.RawMessage `json:"search_titles"`
	PreferredSkills   json.RawMessage `json:"preferred_skills"`
	MinScoreThreshold int             `json:"min_score_threshold"`
	MaxJobsPerRun     int             `json:"max_jobs_per_run"`

	// Per-stage model overrides. Empty means "use the LLM_MODEL environment
	// value". Kept here rather than in the environment so they are editable
	// from the app without a restart — and because provider free tiers meter
	// quota per model, pointing the two stages at different models doubles the
	// daily allowance rather than sharing one.
	ScoringModel     string `json:"scoring_model"`
	GenerationModel  string `json:"generation_model"`
	CoverLetterNotes string `json:"cover_letter_notes"`

	// ManualApplyGraceDays is how long a job may sit in ManualApply before it
	// is closed automatically. 0 disables expiry.
	ManualApplyGraceDays int `json:"manual_apply_grace_days"`
	// NotifyEmail is where the apply-pack digest goes. Deliberately the
	// candidate's own address — nothing in this pipeline emails an employer.
	NotifyEmail string `json:"notify_email"`
	// InboxAutoConfidence is the floor below which a classification is only
	// ever suggested, never applied.
	InboxAutoConfidence int `json:"inbox_auto_confidence"`

	// FollowUpAfterDays is how long to wait after applying before chasing.
	// FollowUpCloseDays is how long after chasing to give up; 0 disables it.
	FollowUpAfterDays int `json:"followup_after_days"`
	FollowUpCloseDays int `json:"followup_close_days"`

	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileUpdate is the PATCH body for the profile. Every field is a pointer so
// "absent" and "explicitly set to empty/zero" stay distinguishable.
type ProfileUpdate struct {
	MasterCV             *string          `json:"master_cv"`
	SearchTitles         *json.RawMessage `json:"search_titles"`
	PreferredSkills      *json.RawMessage `json:"preferred_skills"`
	MinScoreThreshold    *int             `json:"min_score_threshold"`
	MaxJobsPerRun        *int             `json:"max_jobs_per_run"`
	ScoringModel         *string          `json:"scoring_model"`
	GenerationModel      *string          `json:"generation_model"`
	CoverLetterNotes     *string          `json:"cover_letter_notes"`
	ManualApplyGraceDays *int             `json:"manual_apply_grace_days"`
	InboxAutoConfidence  *int             `json:"inbox_auto_confidence"`
	FollowUpAfterDays    *int             `json:"followup_after_days"`
	FollowUpCloseDays    *int             `json:"followup_close_days"`
	NotifyEmail          *string          `json:"notify_email"`
}

// JobUpdate is the PATCH body for a job. Same pointer rule as ProfileUpdate.
type JobUpdate struct {
	Status         *string          `json:"status"`
	Score          *int             `json:"score"`
	MatchedSkills  *json.RawMessage `json:"matched_skills"`
	MissingSkills  *json.RawMessage `json:"missing_skills"`
	AISummary      *string          `json:"ai_summary"`
	CVURL          *string          `json:"cv_url"`
	CoverLetterURL *string          `json:"cover_letter_url"`
	PromptUsed     *string          `json:"prompt_used"`
	EmailUsed      *string          `json:"email_used"`
	DateApplied    *time.Time       `json:"date_applied"`
}

// JobEvent is one inbound email matched (or not) to an application.
//
// Kept as its own table rather than columns on the job because a single
// application can accumulate several replies — acknowledgement, then interview,
// then offer — and flattening them onto the row would lose all but the last.
type JobEvent struct {
	ID     string  `json:"id"`
	JobID  *string `json:"job_id"`
	Source string  `json:"source"`

	Kind       string `json:"kind"`
	Confidence int    `json:"confidence"`
	Classifier string `json:"classifier"`

	Sender       string `json:"sender"`
	SenderDomain string `json:"sender_domain"`
	Subject      string `json:"subject"`
	// Excerpt is a fragment, not the whole body — see service.excerptOf.
	Excerpt    string     `json:"excerpt"`
	ReceivedAt *time.Time `json:"received_at"`

	MatchScore  int    `json:"match_score"`
	MatchReason string `json:"match_reason"`

	// SuggestedStatus is empty when the email implies no change.
	SuggestedStatus string `json:"suggested_status,omitempty"`
	// Confirmed is nil while a suggestion awaits your decision.
	Confirmed *bool      `json:"confirmed"`
	AppliedAt *time.Time `json:"applied_at"`

	MessageID string    `json:"message_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrorRecord is one row of the application error log.
type ErrorRecord struct {
	ID        string          `json:"id"`
	Workflow  string          `json:"workflow"`
	JobID     *string         `json:"job_id"`
	Message   string          `json:"message"`
	Context   json.RawMessage `json:"context,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// FetchLog is one ingest run — what makes API quota consumption visible
// instead of guesswork.
type FetchLog struct {
	ID            string    `json:"id"`
	FetchedAt     time.Time `json:"fetched_at"`
	QueryTitle    string    `json:"query_title"`
	ReturnedCount int       `json:"returned_count"`
	InsertedCount int       `json:"inserted_count"`
	SkippedCount  int       `json:"skipped_count"`
	Notes         string    `json:"notes,omitempty"`
}

// IngestResult is what POST /internal/jobs/ingest returns: the same three
// numbers that get written to fetch_log, so n8n can log or alert on them
// without a second round trip.
type IngestResult struct {
	Returned   int      `json:"returned"`
	Inserted   int      `json:"inserted"`
	Skipped    int      `json:"skipped"`
	JobIDs     []string `json:"job_ids"`
	FetchLogID string   `json:"fetch_log_id"`
}

// StatusCount is one bar of the pipeline dashboard.
type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// Stats is the dashboard payload.
type Stats struct {
	ByStatus     []StatusCount `json:"by_status"`
	Total        int           `json:"total"`
	LastFetch    *FetchLog     `json:"last_fetch"`
	RecentErrors int           `json:"recent_errors_24h"`
}

// Lenient time parsing: the API mixes "2026-07-26T16:26:11.63" (frac seconds,
// no zone) with "2026-07-27T04:19:19" (no frac, no zone). Try known layouts.
var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func firstOr(s []string, def string) string {
	if len(s) > 0 {
		return s[0]
	}
	return def
}

// jsonArray marshals a string slice for a JSONB column, normalising nil to an
// empty array so the app never has to handle both null and [].
func jsonArray(v []string) json.RawMessage {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("[]")
	}
	return b
}

// Normalize maps one raw API item to the values we insert.
// Location: trust the raw address country over the flaky *_derived geo fields
// (the sample data put a Monterrey job in Chiapas).
func (a APIJob) Normalize(raw json.RawMessage) Job {
	country := ""
	locality := ""
	if len(a.Locations) > 0 {
		country = a.Locations[0].Address.AddressCountry
		locality = a.Locations[0].Address.AddressLocality
	}
	if country == "" {
		country = firstOr(a.CountriesDerived, "")
	}
	locRaw := locality
	if locRaw == "" {
		locRaw = firstOr(a.LocationsDerived, "")
	}

	return Job{
		SourceJobID:     a.ID,
		LinkedInID:      a.LinkedInID,
		Title:           strings.TrimSpace(a.Title),
		Organization:    strings.TrimSpace(a.Organization),
		OrganizationURL: a.OrganizationURL,
		URL:             strings.TrimSpace(a.URL),
		Source:          a.Source,
		SourceDomain:    a.SourceDomain,
		DescriptionText: a.DescriptionText,
		DatePosted:      parseTime(a.DatePosted),
		DateValidThru:   parseTime(a.DateValidThru),

		Country:         country,
		LocationRaw:     locRaw,
		WorkArrangement: a.AIWorkArrangement,
		EmploymentType:  firstOr(a.EmploymentType, ""),
		Seniority:       a.Seniority,
		ExperienceLevel: a.AIExperienceLevel,
		DirectApply:     a.DirectApply,

		AIKeySkills:            jsonArray(a.AIKeySkills),
		AIKeywords:             jsonArray(a.AIKeywords),
		AIRequirementsSummary:  a.AIRequirementsSummary,
		AICoreResponsibilities: a.AICoreResponsibilities,

		SalaryCurrency: a.AISalaryCurrency,
		SalaryMin:      a.AISalaryMin,
		SalaryMax:      a.AISalaryMax,
		SalaryUnit:     a.AISalaryUnit,

		OrgDomain: hostFromURL(a.OrgWebsite),

		RawPayload: raw,
		Status:     StatusNew,
	}
}

// hostFromURL reduces a website URL to a bare, comparable domain:
// "https://www.Div-Systems.com/careers" -> "div-systems.com".
//
// Stored at ingest rather than parsed out of raw_payload on demand, so the
// inbox matcher can index it — it runs against every candidate job for every
// incoming email.
func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "//") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return strings.TrimPrefix(host, "www.")
}
