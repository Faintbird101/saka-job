package inbox

import (
	"regexp"
	"strings"
)

// Kind is what an email is saying about an application.
type Kind string

const (
	KindAcknowledgement Kind = "acknowledgement" // "we received your application"
	KindRejection       Kind = "rejection"
	KindInterview       Kind = "interview"
	KindOffer           Kind = "offer"
	KindOther           Kind = "other"
)

// Classification is the verdict on one message.
type Classification struct {
	Kind       Kind
	Confidence int    // 0-100
	Classifier string // "keyword" or "llm:<model>"
	Evidence   string // the phrase that decided it, for the audit trail
}

// Consequential reports whether acting on this classification is hard to
// undo. These are the ones that must be confirmed by a human rather than
// applied automatically: silently marking a live opportunity as rejected is a
// far worse outcome than asking.
func (k Kind) Consequential() bool {
	switch k {
	case KindRejection, KindOffer, KindInterview:
		return true
	default:
		return false
	}
}

// A phrase and how much it tells us.
type rule struct {
	re         *regexp.Regexp
	kind       Kind
	confidence int
}

// Order matters. The list is scanned in order and the highest-confidence hit
// wins, but the *interview* rules are deliberately checked against the whole
// text before rejection rules are trusted — see Classify.
var rules = []rule{
	// --- Interview / advancement. Checked first and weighted high, because
	// mistaking one of these for a rejection is the expensive error. ---
	{regexp.MustCompile(`(?i)\b(invite|inviting|invitation) (you )?(to|for) (an? )?(interview|call|chat|conversation|assessment)`), KindInterview, 95},
	{regexp.MustCompile(`(?i)\b(schedule|arrange|book) (a|an|your) (interview|call|meeting|chat)`), KindInterview, 92},
	{regexp.MustCompile(`(?i)\b(move|proceed|advance|progress) (you |your application )?(to|with|forward to) the (next|second|final) (stage|round|step)`), KindInterview, 92},
	{regexp.MustCompile(`(?i)\b(would like|we'?d like|pleased) to (invite|speak|meet|schedule|arrange)`), KindInterview, 90},
	{regexp.MustCompile(`(?i)\b(technical|coding|take[- ]home) (assessment|challenge|test|exercise)\b`), KindInterview, 80},
	{regexp.MustCompile(`(?i)\bavailability (for|to) (a|an|the) (call|interview|chat)`), KindInterview, 85},

	// --- Offer ---
	{regexp.MustCompile(`(?i)\b(pleased|delighted|happy) to (offer|extend)\b`), KindOffer, 95},
	{regexp.MustCompile(`(?i)\b(offer of employment|employment offer|job offer|offer letter)\b`), KindOffer, 92},

	// --- Rejection ---
	{regexp.MustCompile(`(?i)\bwe (regret|are sorry) to inform\b`), KindRejection, 95},
	{regexp.MustCompile(`(?i)\b(not|won'?t be) (moving|proceeding|progressing) (forward|ahead) with your\b`), KindRejection, 93},
	{regexp.MustCompile(`(?i)\b(decided|chosen) to (move forward|proceed|continue) with (other|another)\b`), KindRejection, 93},
	{regexp.MustCompile(`(?i)\bunfortunately,? (we|your|after|following)\b`), KindRejection, 80},
	{regexp.MustCompile(`(?i)\b(will not|not) be (progressing|considered|shortlisted)\b`), KindRejection, 85},
	{regexp.MustCompile(`(?i)\b(unsuccessful|not successful) (on this occasion|at this time|this time)\b`), KindRejection, 92},
	{regexp.MustCompile(`(?i)\b(other|another) candidate(s)? (whose|who)\b`), KindRejection, 85},
	{regexp.MustCompile(`(?i)\bkeep your (cv|resume|details) on file\b`), KindRejection, 70},

	// --- Acknowledgement. Low stakes, so a modest threshold is fine. ---
	{regexp.MustCompile(`(?i)\b(received|got) your (application|cv|resume|submission)\b`), KindAcknowledgement, 90},
	{regexp.MustCompile(`(?i)\bthank you for (applying|your application|your interest)\b`), KindAcknowledgement, 85},
	{regexp.MustCompile(`(?i)\bapplication (has been )?(received|submitted successfully)\b`), KindAcknowledgement, 90},
	{regexp.MustCompile(`(?i)\bwe are (currently )?reviewing your\b`), KindAcknowledgement, 80},
	{regexp.MustCompile(`(?i)\bunder (review|consideration)\b`), KindAcknowledgement, 70},
}

// LLMThreshold is the confidence below which a keyword verdict is not trusted
// on its own and the message should go to a model instead.
const LLMThreshold = 75

// ClassifyKeywords applies the rule list. Free, deterministic, and adequate for
// the formulaic majority of recruiting mail.
//
// The one piece of real logic: a message containing BOTH interview and
// rejection language is treated as an interview. "Unfortunately we cannot match
// your salary expectations, but we would like to invite you to interview" is a
// real shape of email, and reading it as a rejection is the error that actually
// costs you something.
func ClassifyKeywords(msg Message) Classification {
	text := msg.Subject + "\n" + msg.Body

	best := Classification{Kind: KindOther, Confidence: 0, Classifier: "keyword"}
	var sawInterview, sawOffer bool
	var interviewHit, offerHit Classification

	for _, r := range rules {
		m := r.re.FindString(text)
		if m == "" {
			continue
		}
		c := Classification{Kind: r.kind, Confidence: r.confidence, Classifier: "keyword", Evidence: trim(m, 120)}

		switch r.kind {
		case KindInterview:
			if !sawInterview || c.Confidence > interviewHit.Confidence {
				sawInterview, interviewHit = true, c
			}
		case KindOffer:
			if !sawOffer || c.Confidence > offerHit.Confidence {
				sawOffer, offerHit = true, c
			}
		}
		if c.Confidence > best.Confidence {
			best = c
		}
	}

	// Good news outranks bad news when both appear.
	if sawOffer {
		return offerHit
	}
	if sawInterview && best.Kind == KindRejection {
		interviewHit.Evidence += " (rejection wording also present; treated as an interview because that is the safer read)"
		return interviewHit
	}
	return best
}

// NeedsLLM reports whether a keyword verdict is too weak to rely on.
func NeedsLLM(c Classification) bool {
	return c.Kind == KindOther || c.Confidence < LLMThreshold
}

// SuggestedStatus maps a classification onto the status it implies, or "" when
// it implies no change.
func SuggestedStatus(k Kind) string {
	switch k {
	case KindAcknowledgement:
		return "Acknowledged"
	case KindInterview:
		return "Interviewing"
	case KindOffer:
		return "OfferReceived"
	case KindRejection:
		return "EmployerRejected"
	default:
		return ""
	}
}

func trim(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
