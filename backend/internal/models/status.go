package models

// The state machine. These constants must stay in sync with the CHECK
// constraint in 0001_init.sql — Postgres is the enforcement point, this is the
// Go-side mirror so a typo is a compile error rather than a 500.
const (
	StatusNew              = "New"
	StatusScored           = "Scored"
	StatusLowMatch         = "LowMatch"
	StatusScoreFailed      = "ScoreFailed"
	StatusCVGenerated      = "CVGenerated"
	StatusAwaitingApproval = "AwaitingApproval"
	StatusApproved         = "Approved"
	StatusRejected         = "Rejected"
	StatusApplied          = "Applied"
	StatusManualApply      = "ManualApply"
	StatusFollowUpSent     = "FollowUpSent"
	StatusClosed           = "Closed"

	// Driven by inbound email (0004). EmployerRejected is deliberately NOT
	// StatusRejected: that one means YOU declined the job.
	StatusAcknowledged     = "Acknowledged"
	StatusInterviewing     = "Interviewing"
	StatusOfferReceived    = "OfferReceived"
	StatusEmployerRejected = "EmployerRejected"
)

// AllStatuses is every legal value, in pipeline order.
var AllStatuses = []string{
	StatusNew, StatusScored, StatusCVGenerated, StatusAwaitingApproval,
	StatusApproved, StatusApplied, StatusFollowUpSent, StatusClosed,
	StatusLowMatch, StatusRejected, StatusScoreFailed, StatusManualApply,
	StatusAcknowledged, StatusInterviewing, StatusOfferReceived, StatusEmployerRejected,
}

// allowedTransitions is the edge list of the state machine.
//
// The DB CHECK constraint stops invalid *values*; this stops invalid *moves* —
// e.g. an app bug flipping a `New` job straight to `Applied`, skipping scoring,
// CV generation, and the human approval gate. That gate is the whole point of
// the system, so it gets enforced in code rather than trusted to clients.
var allowedTransitions = map[string][]string{
	StatusNew:              {StatusScored, StatusLowMatch, StatusScoreFailed},
	StatusScoreFailed:      {StatusNew, StatusScored, StatusLowMatch, StatusClosed},
	StatusScored:           {StatusCVGenerated, StatusLowMatch, StatusRejected, StatusClosed},
	StatusLowMatch:         {StatusScored, StatusClosed},
	StatusCVGenerated:      {StatusAwaitingApproval, StatusRejected, StatusClosed},
	StatusAwaitingApproval: {StatusApproved, StatusRejected, StatusCVGenerated},
	StatusApproved:         {StatusApplied, StatusManualApply, StatusRejected},
	StatusRejected:         {StatusClosed},
	// An application can be acknowledged, then advance, then conclude. Every
	// one of these is driven by a reply landing in the inbox.
	StatusApplied:      {StatusFollowUpSent, StatusClosed, StatusAcknowledged, StatusInterviewing, StatusOfferReceived, StatusEmployerRejected},
	StatusAcknowledged: {StatusInterviewing, StatusOfferReceived, StatusEmployerRejected, StatusFollowUpSent, StatusClosed},
	StatusInterviewing: {StatusOfferReceived, StatusEmployerRejected, StatusClosed, StatusInterviewing},
	// Terminal-ish: an offer is either accepted (Closed) or, rarely, withdrawn.
	StatusOfferReceived:    {StatusClosed, StatusEmployerRejected},
	StatusEmployerRejected: {StatusClosed},
	StatusManualApply:      {StatusApplied, StatusClosed, StatusAcknowledged, StatusEmployerRejected},
	// A follow-up can still be answered.
	StatusFollowUpSent: {StatusClosed, StatusFollowUpSent, StatusAcknowledged, StatusInterviewing, StatusOfferReceived, StatusEmployerRejected},
	StatusClosed:       {},
}

// IsValidStatus reports whether s is one of the twelve legal statuses.
func IsValidStatus(s string) bool {
	_, ok := allowedTransitions[s]
	return ok
}

// CanTransition reports whether from -> to is a legal move. A no-op move
// (from == to) is always allowed so a retried workflow is idempotent.
func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// NextStatuses lists the legal moves out of a status. Handy for the app's
// approval screen, which shouldn't hardcode the graph a second time.
func NextStatuses(from string) []string {
	out := append([]string(nil), allowedTransitions[from]...)
	return out
}
