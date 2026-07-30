package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// ApplyPack is everything needed to apply to one job by hand, in one place.
type ApplyPack struct {
	JobID          string `json:"job_id"`
	Title          string `json:"title"`
	Organization   string `json:"organization"`
	Location       string `json:"location"`
	Score          *int   `json:"score"`
	ApplyURL       string `json:"apply_url"`
	DirectApply    bool   `json:"direct_apply"`
	CVURL          string `json:"cv_url"`
	CoverLetterURL string `json:"cover_letter_url"`
}

// ApplyRun is the outcome of one WF-D pass.
type ApplyRun struct {
	Packs []ApplyPack `json:"packs"`
	Count int         `json:"count"`
	// DigestText is a ready-to-send plain-text body. The backend renders it
	// rather than n8n so the wording lives in version control, and so any
	// delivery channel — email, Slack, a webhook — gets identical content.
	DigestText string `json:"digest_text"`
	NotifyTo   string `json:"notify_to"`
	// Expired lists jobs closed for sitting in ManualApply past the grace
	// period. Reported rather than silently tidied.
	Expired []string `json:"expired,omitempty"`
}

// PrepareApplyPacks is WF-D.
//
// It does NOT email employers. Across every job ingested, zero carried a
// hiring-manager address — every posting is an apply URL — so an automated
// sender would have nothing to send to. Instead each Approved job moves to
// ManualApply and you get one digest with the link and the documents, which is
// what the data actually supports.
//
// The state stays honest: ManualApply means "ready for you to submit", and only
// you can move it to Applied. Nothing here claims an application was made.
func (s *Service) PrepareApplyPacks(ctx context.Context, limit int) (ApplyRun, error) {
	profile, err := s.GetProfile(ctx)
	if err != nil {
		return ApplyRun{}, err
	}

	if limit <= 0 || limit > MaxScoreBatch {
		limit = MaxScoreBatch
	}

	jobs, err := s.jobsWithStatus(ctx, models.StatusApproved, limit)
	if err != nil {
		return ApplyRun{}, err
	}

	run := ApplyRun{Packs: make([]ApplyPack, 0, len(jobs)), NotifyTo: profile.NotifyEmail}

	for _, job := range jobs {
		status := models.StatusManualApply
		if _, err := s.UpdateJob(ctx, job.ID, models.JobUpdate{Status: &status}); err != nil {
			s.log.Error("could not move job to ManualApply", "job_id", job.ID, "error", err)
			s.logErrorDetached(ctx, "WF-D", &job.ID, fmt.Sprintf("apply pack failed: %v", err))
			continue
		}
		// Stamp the grace-period clock separately from updated_at, which any
		// later edit would touch and silently restart.
		if _, err := s.db.Pool.Exec(ctx, queries.MarkManualApply, job.ID); err != nil {
			s.log.Warn("could not stamp manual_apply_at", "job_id", job.ID, "error", err)
		}

		run.Packs = append(run.Packs, ApplyPack{
			JobID:          job.ID,
			Title:          job.Title,
			Organization:   job.Organization,
			Location:       strings.TrimSpace(job.LocationRaw + " " + job.Country),
			Score:          job.Score,
			ApplyURL:       job.URL,
			DirectApply:    job.DirectApply,
			CVURL:          s.publicURL("/jobs/" + job.ID + "/cv"),
			CoverLetterURL: s.publicURL("/jobs/" + job.ID + "/cover-letter"),
		})
	}
	run.Count = len(run.Packs)

	expired, err := s.expireManualApply(ctx, profile.ManualApplyGraceDays)
	if err != nil {
		s.log.Error("manual-apply expiry sweep failed", "error", err)
	}
	run.Expired = expired

	run.DigestText = renderDigest(run)

	s.log.Info("apply packs prepared",
		"count", run.Count, "expired", len(run.Expired), "notify_to", profile.NotifyEmail)
	return run, nil
}

// expireManualApply closes jobs left in ManualApply past the grace period.
//
// Without it the list grows forever: jobs you decided against but never
// explicitly rejected sit as "pending" indefinitely, and a dashboard that is
// always amber is one you stop reading.
func (s *Service) expireManualApply(ctx context.Context, graceDays int) ([]string, error) {
	if graceDays <= 0 {
		return nil, nil // expiry disabled
	}

	rows, err := s.db.Pool.Query(ctx, queries.ExpireManualApply, graceDays)
	if err != nil {
		return nil, fmt.Errorf("expire manual-apply jobs: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id, title, org string
		if err := rows.Scan(&id, &title, &org); err != nil {
			return nil, fmt.Errorf("scan expired job: %w", err)
		}
		out = append(out, fmt.Sprintf("%s at %s", title, org))
		s.log.Info("closed stale ManualApply job", "job_id", id, "title", title, "grace_days", graceDays)
	}
	return out, rows.Err()
}

// renderDigest builds the notification body.
//
// Plain text on purpose: it renders identically in every mail client, in Slack,
// and in a terminal, and there is no styling here worth the fragility of HTML
// email.
func renderDigest(run ApplyRun) string {
	var b strings.Builder

	if run.Count == 0 {
		b.WriteString("No applications are ready to send.\n")
	} else {
		fmt.Fprintf(&b, "%d application(s) ready for you to submit.\n\n", run.Count)
		for i, p := range run.Packs {
			score := "unscored"
			if p.Score != nil {
				score = fmt.Sprintf("score %d", *p.Score)
			}
			fmt.Fprintf(&b, "%d. %s — %s\n", i+1, p.Title, p.Organization)
			if p.Location != "" {
				fmt.Fprintf(&b, "   %s | %s\n", p.Location, score)
			} else {
				fmt.Fprintf(&b, "   %s\n", score)
			}
			fmt.Fprintf(&b, "   Apply:  %s\n", p.ApplyURL)
			fmt.Fprintf(&b, "   CV:     %s\n", p.CVURL)
			fmt.Fprintf(&b, "   Letter: %s\n\n", p.CoverLetterURL)
		}
		b.WriteString("Mark each one Applied once you have submitted it, so the\n")
		b.WriteString("follow-up clock starts from a real date.\n")
	}

	if len(run.Expired) > 0 {
		fmt.Fprintf(&b, "\nClosed %d job(s) that sat unapplied past the grace period:\n", len(run.Expired))
		for _, e := range run.Expired {
			fmt.Fprintf(&b, "  - %s\n", e)
		}
	}

	return b.String()
}

// publicURL turns an API path into something clickable from a mail client.
//
// The backend sits behind Caddy and cannot infer its own external address, so
// it comes from config; without it the digest would carry paths nobody can
// follow.
func (s *Service) publicURL(path string) string {
	if s.publicBase == "" {
		return path
	}
	return strings.TrimRight(s.publicBase, "/") + path
}
