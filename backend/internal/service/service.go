// Package service holds the business logic that sits between the HTTP
// handlers and the database.
//
// Handlers here are deliberately thin: they parse, they call one service
// method, they render. Everything that constitutes a *rule* — which status
// transitions are legal, what counts as a duplicate, what a fetch_log row
// should say — lives in this package, so n8n and the app cannot diverge on it.
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourname/jobhunter/backend/internal/db"
	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// Sentinel errors the handler layer maps onto HTTP status codes.
var (
	// ErrNotFound → 404.
	ErrNotFound = errors.New("not found")
	// ErrInvalidInput → 400. Wrap it with detail: fmt.Errorf("%w: ...", ErrInvalidInput).
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict → 409. Used for illegal state-machine moves.
	ErrConflict = errors.New("conflict")
)

// Service is the single dependency handlers are constructed with.
type Service struct {
	db  *db.DB
	log *slog.Logger

	// scorer may be nil — only when the configured mode could not be built
	// (llm mode with no key). The API still boots and serves everything else;
	// only /internal/scoring/run reports itself unavailable.
	scorer scoring.Scorer

	// llm is the model client, built independently of the scoring mode.
	//
	// Generation always needs a model — there is no free substitute for writing
	// prose — while scoring usually should not use one. Tying generation to
	// SCORING_MODE would force you to pay for scoring just to get cover
	// letters, so the two are deliberately separate: keyword scoring plus LLM
	// generation is a supported, and probably the preferred, combination.
	llm      scoring.Client
	llmModel string

	// publicBase is the externally reachable origin (behind Caddy). The
	// process cannot infer it, and without it the apply-pack digest would
	// carry bare paths that nobody can click from a mail client.
	publicBase string
}

// New builds a Service. scorer and llm may each be nil.
func New(database *db.DB, log *slog.Logger, scorer scoring.Scorer, llm scoring.Client, llmModel, publicBase string) *Service {
	return &Service{db: database, log: log, scorer: scorer, llm: llm, llmModel: llmModel, publicBase: publicBase}
}

// row is the subset of pgx.Row / pgx.Rows that scanJob needs, so the same
// scanner serves both QueryRow and Query.
type row interface {
	Scan(dest ...any) error
}

// scanJob reads one row in the exact order of queries.JobColumns.
//
// The column list and this function are a matched pair: change one without the
// other and every job read breaks at runtime rather than compile time. That's
// the cost of hand-rolled SQL; the benefit is that the query plans are
// obvious. If this grows a third variant, it's time for sqlc.
func scanJob(r row) (models.Job, error) {
	var j models.Job
	var keySkills, keywords, matched, missing []byte

	err := r.Scan(
		&j.ID,
		&j.SourceJobID,
		&j.LinkedInID,
		&j.NormalizedURL,
		&j.Title,
		&j.Organization,
		&j.OrganizationURL,
		&j.URL,
		&j.Source,
		&j.SourceDomain,
		&j.DescriptionText,
		&j.DatePosted,
		&j.DateValidThru,
		&j.Country,
		&j.LocationRaw,
		&j.WorkArrangement,
		&j.EmploymentType,
		&j.Seniority,
		&j.ExperienceLevel,
		&j.DirectApply,
		&keySkills,
		&keywords,
		&j.AIRequirementsSummary,
		&j.AICoreResponsibilities,
		&j.SalaryCurrency,
		&j.SalaryMin,
		&j.SalaryMax,
		&j.SalaryUnit,
		&j.Status,
		&j.Score,
		&matched,
		&missing,
		&j.AISummary,
		&j.CVURL,
		&j.CoverLetterURL,
		&j.PromptUsed,
		&j.DateApplied,
		&j.EmailUsed,
		&j.GeneratedAt,
		&j.GeneratedBy,
		&j.CreatedAt,
		&j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			return models.Job{}, ErrNotFound
		}
		return models.Job{}, fmt.Errorf("scan job: %w", err)
	}

	j.AIKeySkills = jsonOr(keySkills, "[]")
	j.AIKeywords = jsonOr(keywords, "[]")
	j.MatchedSkills = jsonOr(matched, "[]")
	j.MissingSkills = jsonOr(missing, "[]")
	return j, nil
}

// isBadUUID reports whether an error is Postgres rejecting a malformed UUID.
//
// A job id that is not a UUID is a request for something that cannot exist, so
// it is a 404 — not the 500 that an unmapped driver error would produce. The
// check is on the SQLSTATE code (22P02, invalid_text_representation) rather
// than the message text, which is localised and version-dependent.
func isBadUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// jsonOr guards against a NULL jsonb sneaking past a missing COALESCE and
// producing literal `null` in the API response, which every client would then
// need a null check for.
func jsonOr(b []byte, def string) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage(def)
	}
	return json.RawMessage(b)
}

// nullIfEmpty converts "" to a SQL NULL, for columns where empty string and
// absent mean the same thing and NULL is the honest representation.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullIfZero does the same for the two integer dedup keys. linkedin_id is
// UNIQUE and nullable precisely so that non-LinkedIn sources — which have no
// such id — don't all collide on 0.
func nullIfZero(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

// jsonbOrNull passes a JSONB value through, or NULL when it's empty, so
// COALESCE-style partial updates treat "not supplied" correctly.
func jsonbOrNull(m json.RawMessage) any {
	if len(m) == 0 {
		return nil
	}
	return []byte(m)
}
