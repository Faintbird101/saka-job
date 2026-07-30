package service

import (
	"strings"
	"testing"
)

func intp(n int) *int { return &n }

func TestRenderDigestListsEveryPack(t *testing.T) {
	run := ApplyRun{
		Count: 2,
		Packs: []ApplyPack{
			{Title: "Flutter Product Engineer", Organization: "WorkBuddy", Location: "Nairobi Kenya",
				Score: intp(97), ApplyURL: "https://linkedin.com/jobs/view/a",
				CVURL:          "https://api.sakajob.home:7443/jobs/1/cv",
				CoverLetterURL: "https://api.sakajob.home:7443/jobs/1/cover-letter"},
			{Title: "Senior Flutter Developer", Organization: "Lighthouse",
				Score: intp(82), ApplyURL: "https://linkedin.com/jobs/view/b",
				CVURL:          "https://api.sakajob.home:7443/jobs/2/cv",
				CoverLetterURL: "https://api.sakajob.home:7443/jobs/2/cover-letter"},
		},
	}

	got := renderDigest(run)
	for _, want := range []string{
		"2 application(s) ready",
		"Flutter Product Engineer", "WorkBuddy", "score 97", "Nairobi Kenya",
		"Senior Flutter Developer", "Lighthouse", "score 82",
		"https://linkedin.com/jobs/view/a",
		"https://api.sakajob.home:7443/jobs/2/cover-letter",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest is missing %q\n---\n%s", want, got)
		}
	}
}

// The digest must never read as though something was sent on the candidate's
// behalf — the whole point of ManualApply is that submitting is still a human
// act.
func TestRenderDigestDoesNotClaimAnythingWasSent(t *testing.T) {
	got := renderDigest(ApplyRun{Count: 1, Packs: []ApplyPack{
		{Title: "X", Organization: "Y", ApplyURL: "u", CVURL: "c", CoverLetterURL: "l"},
	}})

	for _, forbidden := range []string{"we applied", "application sent", "we have sent", "submitted on your behalf"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Errorf("digest implies an application was sent (%q):\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "ready for you to submit") {
		t.Errorf("digest should make clear the submitting is yours to do:\n%s", got)
	}
}

func TestRenderDigestHandlesAnEmptyRun(t *testing.T) {
	got := renderDigest(ApplyRun{Count: 0})
	if !strings.Contains(got, "No applications are ready") {
		t.Errorf("empty run should say so plainly, got %q", got)
	}
}

func TestRenderDigestHandlesAnUnscoredJob(t *testing.T) {
	got := renderDigest(ApplyRun{Count: 1, Packs: []ApplyPack{
		{Title: "X", Organization: "Y", ApplyURL: "u", CVURL: "c", CoverLetterURL: "l"},
	}})
	if !strings.Contains(got, "unscored") {
		t.Errorf("a nil score should render as 'unscored', not 0 or a panic:\n%s", got)
	}
}

// Expiry is reported, not silent: a job disappearing from your list without
// explanation is worse than one that lingers.
func TestRenderDigestReportsExpiredJobs(t *testing.T) {
	got := renderDigest(ApplyRun{
		Count:   0,
		Expired: []string{"Old Role at OldCo", "Stale Role at StaleCo"},
	})
	if !strings.Contains(got, "Closed 2 job(s)") {
		t.Errorf("expired jobs are not reported:\n%s", got)
	}
	for _, want := range []string{"Old Role at OldCo", "Stale Role at StaleCo"} {
		if !strings.Contains(got, want) {
			t.Errorf("digest does not name the expired job %q", want)
		}
	}
}

func TestPublicURLBuildsClickableLinks(t *testing.T) {
	s := &Service{publicBase: "https://api.sakajob.home:7443"}
	if got := s.publicURL("/jobs/abc/cv"); got != "https://api.sakajob.home:7443/jobs/abc/cv" {
		t.Errorf("publicURL = %q", got)
	}

	// A trailing slash on the base must not produce a doubled separator.
	s2 := &Service{publicBase: "https://api.sakajob.home:7443/"}
	if got := s2.publicURL("/jobs/abc/cv"); strings.Contains(got, "7443//") {
		t.Errorf("doubled slash in %q", got)
	}

	// Unconfigured falls back to a bare path rather than something malformed.
	s3 := &Service{}
	if got := s3.publicURL("/jobs/abc/cv"); got != "/jobs/abc/cv" {
		t.Errorf("unconfigured base should yield the plain path, got %q", got)
	}
}
