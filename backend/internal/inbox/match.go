// Package inbox matches inbound employer email to job applications and works
// out what the email is saying.
//
// Everything here is pure: matching and classification take a message and a
// list of candidate jobs and return a verdict, with no database and no network.
// That matters more than usual because the cost of getting this wrong is
// asymmetric — a misread rejection could make you abandon a live opportunity —
// so the rules need to be exhaustively testable.
package inbox

import (
	"regexp"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// Message is one inbound email, already fetched by n8n's IMAP node.
type Message struct {
	MessageID  string
	From       string // may be "Jane Doe <jane@acme.com>"
	Subject    string
	Body       string
	ReceivedAt string
}

// Match is the matcher's verdict for one message.
type Match struct {
	JobID  string
	Score  int    // 0-100 confidence that this email belongs to that job
	Reason string // human-readable, so an attribution can be second-guessed
	// Ambiguous means several jobs matched about equally well. The caller must
	// NOT attribute the email in that case — two jobs at the same company are
	// common (three at "div Systems" in the current data) and picking one
	// arbitrarily would silently attach a rejection to the wrong application.
	Ambiguous bool
}

// Match-score weights. Domain is the strongest single signal because it is
// structural rather than textual; title is what disambiguates between several
// jobs at the same employer.
const (
	scoreDomain        = 55
	scoreTitleInText   = 30
	scoreOrgNameInText = 15
	// ambiguityMargin is how far ahead the best match must be to be trusted.
	// Below this, two candidates are treated as indistinguishable.
	ambiguityMargin = 15
	// MinMatchScore is the floor for attributing an email to a job at all.
	MinMatchScore = 40
)

var addrRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@([A-Za-z0-9.\-]+\.[A-Za-z]{2,})`)

// SenderDomain pulls the domain out of a From header, tolerating a display
// name around the address.
func SenderDomain(from string) string {
	m := addrRe.FindStringSubmatch(from)
	if len(m) < 2 {
		return ""
	}
	return normalizeDomain(m[1])
}

// normalizeDomain lowercases and strips a leading www.
func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(d, "www.")
	return strings.Trim(d, ".")
}

// sameOrgDomain reports whether two domains plausibly belong to one company.
//
// This is the heart of the idea that a reply from hr@acme.com belongs to the
// job at acme.com even though the application went to recruiting@acme.com. It
// compares the registrable part, so mail.acme.com, careers.acme.com and
// acme.com all match — while deliberately NOT matching acme.com against
// acme-recruiting.com, which is a different organisation.
func sameOrgDomain(a, b string) bool {
	a, b = normalizeDomain(a), normalizeDomain(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	// Subdomain either way: careers.acme.com vs acme.com.
	return strings.HasSuffix(a, "."+b) || strings.HasSuffix(b, "."+a)
}

// FindMatch picks the job an email most likely belongs to.
//
// candidates should already be narrowed to jobs that could plausibly have a
// reply — applied to, and applied to before this email arrived.
func FindMatch(msg Message, candidates []models.Job) Match {
	domain := SenderDomain(msg.From)
	haystack := strings.ToLower(msg.Subject + " \n " + msg.Body)

	type scored struct {
		job     models.Job
		score   int
		reasons []string
	}
	var results []scored

	for _, job := range candidates {
		s := scored{job: job}

		if domain != "" && job.OrgDomain != "" && sameOrgDomain(domain, job.OrgDomain) {
			s.score += scoreDomain
			s.reasons = append(s.reasons, "sender domain matches "+job.OrgDomain)
		}
		if t := strings.ToLower(strings.TrimSpace(job.Title)); t != "" && strings.Contains(haystack, t) {
			s.score += scoreTitleInText
			s.reasons = append(s.reasons, "job title appears in the email")
		}
		if o := strings.ToLower(strings.TrimSpace(job.Organization)); len(o) > 2 && strings.Contains(haystack, o) {
			s.score += scoreOrgNameInText
			s.reasons = append(s.reasons, "organisation name appears in the email")
		}

		if s.score > 0 {
			results = append(results, s)
		}
	}

	if len(results) == 0 {
		return Match{Reason: "no candidate job matched the sender domain, title, or organisation"}
	}

	// Highest score wins; ties are ambiguous rather than arbitrary.
	best := results[0]
	runnerUp := 0
	for _, r := range results[1:] {
		switch {
		case r.score > best.score:
			runnerUp = best.score
			best = r
		case r.score > runnerUp:
			runnerUp = r.score
		}
	}

	m := Match{
		JobID:  best.job.ID,
		Score:  min(best.score, 100),
		Reason: strings.Join(best.reasons, "; "),
	}
	if best.score < MinMatchScore {
		m.Reason += " (below the attribution threshold)"
		m.JobID = ""
		return m
	}
	if runnerUp > 0 && best.score-runnerUp < ambiguityMargin {
		// Same company, several open roles, nothing to tell them apart.
		m.Ambiguous = true
		m.Reason += "; another job scored within " + itoa(ambiguityMargin) + " points, so the attribution is not safe"
	}
	return m
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
