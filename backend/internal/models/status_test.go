package models

import "testing"

func TestIsValidStatus(t *testing.T) {
	for _, s := range AllStatuses {
		if !IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = false, but it is in AllStatuses", s)
		}
	}
	for _, s := range []string{"", "Aproved", "approved", "APPLIED", "Unknown"} {
		if IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = true, want false", s)
		}
	}
}

// AllStatuses and the transition graph are two hand-maintained lists; if they
// drift, the app's /statuses endpoint starts lying.
func TestAllStatusesMatchesTransitionGraph(t *testing.T) {
	if len(AllStatuses) != len(allowedTransitions) {
		t.Fatalf("AllStatuses has %d entries, transition graph has %d", len(AllStatuses), len(allowedTransitions))
	}
	for _, s := range AllStatuses {
		if _, ok := allowedTransitions[s]; !ok {
			t.Errorf("%q is in AllStatuses but has no entry in the transition graph", s)
		}
	}
	// Every target must itself be a known status, or a "legal" move could
	// produce a value the database CHECK constraint rejects.
	for from, targets := range allowedTransitions {
		for _, to := range targets {
			if !IsValidStatus(to) {
				t.Errorf("%q -> %q: target is not a valid status", from, to)
			}
		}
	}
}

func TestCanTransitionHappyPath(t *testing.T) {
	path := []string{
		StatusNew, StatusScored, StatusCVGenerated, StatusAwaitingApproval,
		StatusApproved, StatusApplied, StatusFollowUpSent, StatusClosed,
	}
	for i := 0; i < len(path)-1; i++ {
		if !CanTransition(path[i], path[i+1]) {
			t.Errorf("the documented happy path is broken: %s -> %s is rejected", path[i], path[i+1])
		}
	}
}

// The human approval gate is the point of the whole system. These are the
// moves that would bypass it.
func TestCanTransitionRefusesToSkipApproval(t *testing.T) {
	forbidden := [][2]string{
		{StatusNew, StatusApplied},
		{StatusScored, StatusApplied},
		{StatusScored, StatusApproved},
		{StatusCVGenerated, StatusApproved},
		{StatusAwaitingApproval, StatusApplied},
		{StatusNew, StatusAwaitingApproval},
	}
	for _, m := range forbidden {
		if CanTransition(m[0], m[1]) {
			t.Errorf("%s -> %s is allowed, but it bypasses the human approval gate", m[0], m[1])
		}
	}
}

func TestCanTransitionClosedIsTerminal(t *testing.T) {
	for _, s := range AllStatuses {
		if s == StatusClosed {
			continue
		}
		if CanTransition(StatusClosed, s) {
			t.Errorf("Closed -> %s is allowed, but Closed is terminal", s)
		}
	}
}

// A retried n8n workflow will re-issue the same PATCH. Setting a status to
// what it already is must succeed, or every retry becomes a 409.
func TestCanTransitionIsIdempotent(t *testing.T) {
	for _, s := range AllStatuses {
		if !CanTransition(s, s) {
			t.Errorf("%s -> %s (no-op) was rejected; retries would fail", s, s)
		}
	}
}

func TestCanTransitionRejectsUnknownStatuses(t *testing.T) {
	if CanTransition("Nonsense", StatusScored) {
		t.Error("transition out of an unknown status should be rejected")
	}
	if CanTransition(StatusNew, "Nonsense") {
		t.Error("transition into an unknown status should be rejected")
	}
}

func TestNextStatusesDoesNotAliasTheGraph(t *testing.T) {
	got := NextStatuses(StatusNew)
	if len(got) == 0 {
		t.Fatal("New should have outgoing transitions")
	}
	got[0] = "MUTATED"

	if allowedTransitions[StatusNew][0] == "MUTATED" {
		t.Error("NextStatuses returned a slice aliasing the package-level graph; a caller can corrupt the state machine")
	}
}
