package inbox

import (
	"testing"

	"github.com/yourname/jobhunter/backend/internal/models"
)

func job(id, title, org, domain string) models.Job {
	return models.Job{ID: id, Title: title, Organization: org, OrgDomain: domain}
}

func TestSenderDomain(t *testing.T) {
	cases := map[string]string{
		"jane@acme.com":                      "acme.com",
		"Jane Doe <jane.doe@acme.com>":       "acme.com",
		"\"Recruiting, Acme\" <hr@ACME.com>": "acme.com",
		"noreply@www.acme.com":               "acme.com",
		"careers@mail.acme.co.uk":            "mail.acme.co.uk",
		"not an address":                     "",
		"":                                   "",
	}
	for in, want := range cases {
		if got := SenderDomain(in); got != want {
			t.Errorf("SenderDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

// The core premise: you applied via recruiting@ and they reply from hr@, or
// from a subdomain. Same company, must still match.
func TestSameOrgDomainAcceptsTheSameCompany(t *testing.T) {
	pairs := [][2]string{
		{"acme.com", "acme.com"},
		{"hr.acme.com", "acme.com"},
		{"acme.com", "careers.acme.com"},
		{"mail.recruiting.acme.com", "acme.com"},
		{"ACME.com", "www.acme.com"},
	}
	for _, p := range pairs {
		if !sameOrgDomain(p[0], p[1]) {
			t.Errorf("sameOrgDomain(%q, %q) = false, want true", p[0], p[1])
		}
	}
}

// And must NOT match a different company that merely looks similar — a
// lookalike domain is how a rejection gets attached to the wrong employer.
func TestSameOrgDomainRejectsDifferentCompanies(t *testing.T) {
	pairs := [][2]string{
		{"acme.com", "acme-recruiting.com"},
		{"acme.com", "notacme.com"},
		{"acme.com", "acme.co"},
		{"acme.com", "gmail.com"},
		{"acme.com", ""},
		{"", "acme.com"},
	}
	for _, p := range pairs {
		if sameOrgDomain(p[0], p[1]) {
			t.Errorf("sameOrgDomain(%q, %q) = true, want false", p[0], p[1])
		}
	}
}

func TestFindMatchByDomain(t *testing.T) {
	jobs := []models.Job{
		job("j1", "Flutter Engineer", "Yondu, Inc.", "yondu.com"),
		job("j2", "Backend Engineer", "Kellton", "kellton.com"),
	}
	m := FindMatch(Message{From: "hr@yondu.com", Subject: "Your application", Body: "Thanks for applying."}, jobs)

	if m.JobID != "j1" {
		t.Fatalf("JobID = %q, want j1 (reason: %s)", m.JobID, m.Reason)
	}
	if m.Ambiguous {
		t.Error("should not be ambiguous — only one company matched")
	}
	if m.Score < MinMatchScore {
		t.Errorf("score %d below threshold", m.Score)
	}
}

// The situation already present in the live data: three open roles at one
// employer. Domain alone cannot say which, and guessing would attach the reply
// to the wrong application.
func TestFindMatchIsAmbiguousForTwoRolesAtOneCompany(t *testing.T) {
	jobs := []models.Job{
		job("j1", "Flutter Developer", "div Systems", "div-systems.com"),
		job("j2", "React Developer", "div Systems", "div-systems.com"),
	}
	m := FindMatch(Message{From: "hr@div-systems.com", Subject: "Update", Body: "About your application."}, jobs)

	if !m.Ambiguous {
		t.Errorf("expected ambiguous, got JobID=%q score=%d (%s)", m.JobID, m.Score, m.Reason)
	}
}

// The title in the subject is what breaks the tie.
func TestTitleDisambiguatesSameCompany(t *testing.T) {
	jobs := []models.Job{
		job("j1", "Flutter Developer", "div Systems", "div-systems.com"),
		job("j2", "React Developer", "div Systems", "div-systems.com"),
	}
	m := FindMatch(Message{
		From:    "hr@div-systems.com",
		Subject: "Your application for Flutter Developer",
		Body:    "We received it.",
	}, jobs)

	if m.Ambiguous {
		t.Errorf("title should have disambiguated: %s", m.Reason)
	}
	if m.JobID != "j1" {
		t.Errorf("JobID = %q, want j1", m.JobID)
	}
}

// Unmatched mail must come back with no job rather than being forced onto the
// closest one.
func TestFindMatchRefusesWhenNothingMatches(t *testing.T) {
	jobs := []models.Job{job("j1", "Flutter Engineer", "Yondu, Inc.", "yondu.com")}
	for _, msg := range []Message{
		{From: "newsletter@random.com", Subject: "Weekly digest", Body: "Unrelated."},
		{From: "", Subject: "", Body: ""},
	} {
		m := FindMatch(msg, jobs)
		if m.JobID != "" {
			t.Errorf("attributed %q to %s on no evidence", msg.Subject, m.JobID)
		}
		if m.Reason == "" {
			t.Error("no reason given for the non-match")
		}
	}
}

// A generic sender (recruiting platform, personal Gmail) with the org name in
// the body is weak evidence — it must stay below the attribution threshold
// rather than being trusted on the strength of a name appearing in text.
func TestOrgNameAloneIsNotEnough(t *testing.T) {
	jobs := []models.Job{job("j1", "Flutter Engineer", "Yondu, Inc.", "yondu.com")}
	m := FindMatch(Message{
		From:    "someone@gmail.com",
		Subject: "Question",
		Body:    "I saw you mention yondu, inc. the other day.",
	}, jobs)

	if m.JobID != "" {
		t.Errorf("org name alone (score %d) should not attribute: %s", m.Score, m.Reason)
	}
}

func TestFindMatchScoreIsCapped(t *testing.T) {
	jobs := []models.Job{job("j1", "Flutter Engineer", "Yondu, Inc.", "yondu.com")}
	m := FindMatch(Message{
		From:    "hr@yondu.com",
		Subject: "Flutter Engineer at Yondu, Inc.",
		Body:    "Flutter Engineer, Yondu, Inc.",
	}, jobs)

	if m.Score > 100 {
		t.Errorf("score %d exceeds 100", m.Score)
	}
	if m.JobID != "j1" {
		t.Errorf("strongest possible signal did not match")
	}
}

func TestFindMatchHandlesJobsWithNoDomain(t *testing.T) {
	// 4 of 24 live jobs have no org_linkedin_website. They must not crash the
	// matcher, and must not match on an empty domain.
	jobs := []models.Job{job("j1", "Flutter Engineer", "NoWebsite Ltd", "")}
	m := FindMatch(Message{From: "hr@somewhere.com", Subject: "Hi", Body: "Hello"}, jobs)
	if m.JobID != "" {
		t.Error("matched a job with no domain against an unrelated sender")
	}
}
