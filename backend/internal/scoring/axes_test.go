package scoring

import (
	"encoding/json"
	"testing"

	"github.com/yourname/jobhunter/backend/internal/models"
)

func TestCombineIgnoresUnknownAxes(t *testing.T) {
	w := Weights{Skills: 45, Seniority: 15, Domain: 20, Location: 12, Pay: 8}

	// All five known, all 80 -> 80. Nothing clever, just the baseline.
	all := Axes{axis(80), axis(80), axis(80), axis(80), axis(80)}
	if got := Combine(all, w); got != 80 {
		t.Errorf("all-80 combined to %d, want 80", got)
	}

	// The same four, with pay unknown, must still be 80 — NOT dragged down.
	// This is the whole reason the axes are pointers: most live postings state
	// no salary, and treating that silence as zero would sink every score.
	noPay := Axes{axis(80), axis(80), axis(80), axis(80), nil}
	if got := Combine(noPay, w); got != 80 {
		t.Errorf("unknown pay changed the score to %d; silence must not be penalised", got)
	}

	// Whereas a genuinely bad pay score SHOULD pull it down.
	badPay := Axes{axis(80), axis(80), axis(80), axis(80), axis(0)}
	if got := Combine(badPay, w); got >= 80 {
		t.Errorf("a zero pay axis left the score at %d; a known-bad axis must count", got)
	}
}

func TestCombineHandlesEverythingUnknown(t *testing.T) {
	if got := Combine(Axes{}, Weights{Skills: 45}); got != 0 {
		t.Errorf("no known axes gave %d, want 0", got)
	}
}

// A zero weight removes an axis, so the dial in the app can switch one off.
func TestZeroWeightRemovesAnAxis(t *testing.T) {
	a := Axes{axis(100), axis(0), axis(100), axis(100), axis(100)}
	withSeniority := Combine(a, Weights{Skills: 25, Seniority: 25, Domain: 25, Location: 25})
	without := Combine(a, Weights{Skills: 25, Seniority: 0, Domain: 25, Location: 25})

	if without <= withSeniority {
		t.Errorf("zeroing the weight of a failing axis did not raise the score (%d vs %d)", without, withSeniority)
	}
	if without != 100 {
		t.Errorf("with seniority removed and everything else perfect, want 100, got %d", without)
	}
}

func TestWeakestNamesTheLowestKnownAxis(t *testing.T) {
	name, score, ok := Axes{axis(96), axis(88), axis(94), axis(90), axis(72)}.Weakest()
	if !ok || name != "pay" || score != 72 {
		t.Errorf("Weakest() = %q/%d/%v, want pay/72/true", name, score, ok)
	}

	// An unknown axis is never "the weakest" — it is simply not known.
	name, _, ok = Axes{axis(96), nil, nil, nil, nil}.Weakest()
	if !ok || name != "skills" {
		t.Errorf("Weakest() = %q, want skills — unknown axes must not win", name)
	}

	if _, _, ok = (Axes{}).Weakest(); ok {
		t.Error("Weakest() claimed a result with no known axes")
	}
}

func TestPayAxisIsUnknownWhenEitherSideIsSilent(t *testing.T) {
	withFloor := models.Profile{SalaryFloor: 400000, SalaryCurrency: "KES"}
	max := 500000.0

	if got := payAxis(models.Profile{}, models.Job{SalaryMax: &max}); got != nil {
		t.Errorf("no floor set should be unknown, got %d", *got)
	}
	if got := payAxis(withFloor, models.Job{}); got != nil {
		t.Errorf("posting states no salary — should be unknown, got %d", *got)
	}
	// A currency mismatch is unknown rather than converted: guessing a rate
	// would turn an assumption into a confident number.
	if got := payAxis(withFloor, models.Job{SalaryMax: &max, SalaryCurrency: "USD"}); got != nil {
		t.Errorf("currency mismatch should be unknown, got %d", *got)
	}
}

func TestPayAxisScoresAgainstTheFloor(t *testing.T) {
	p := models.Profile{SalaryFloor: 400000, SalaryCurrency: "KES"}
	for _, tc := range []struct{ offer, wantAtLeast, wantAtMost int }{
		{600000, 95, 100}, // well above
		{420000, 85, 95},  // just above
		{380000, 60, 80},  // just below
		{200000, 0, 30},   // far below
	} {
		v := float64(tc.offer)
		got := payAxis(p, models.Job{SalaryMax: &v, SalaryCurrency: "KES"})
		if got == nil {
			t.Fatalf("offer %d gave unknown", tc.offer)
		}
		if *got < tc.wantAtLeast || *got > tc.wantAtMost {
			t.Errorf("offer %d scored %d, want %d-%d", tc.offer, *got, tc.wantAtLeast, tc.wantAtMost)
		}
	}
}

func TestLocationAxisIsUnknownWithoutAPreference(t *testing.T) {
	if got := locationAxis(models.Profile{}, models.Job{Country: "Kenya"}); got != nil {
		t.Errorf("no stated preference should be unknown, got %d", *got)
	}
}

func TestLocationAxisRemoteSatisfiesAnyPlace(t *testing.T) {
	p := models.Profile{
		PreferredLocations: json.RawMessage(`["Nairobi"]`),
		RemotePreference:   "any",
	}
	// Remote work makes the office location irrelevant.
	got := locationAxis(p, models.Job{Country: "Philippines", WorkArrangement: "Remote Solely"})
	if got == nil || *got < 90 {
		t.Errorf("remote role scored %v against a Nairobi preference; location should not matter", got)
	}
}

func TestLocationAxisRemoteOnlyRejectsOnsite(t *testing.T) {
	p := models.Profile{RemotePreference: "remote_only"}
	onsite := locationAxis(p, models.Job{Country: "Kenya", WorkArrangement: "On-site"})
	remote := locationAxis(p, models.Job{Country: "Kenya", WorkArrangement: "Remote OK"})

	if onsite == nil || remote == nil {
		t.Fatal("a stated remote_only preference should always produce a score")
	}
	if *onsite >= *remote {
		t.Errorf("on-site (%d) did not score below remote (%d) for a remote-only candidate", *onsite, *remote)
	}
}

func TestLocationAxisMatchesThePreferredPlace(t *testing.T) {
	p := models.Profile{PreferredLocations: json.RawMessage(`["Nairobi","Kenya"]`)}
	near := locationAxis(p, models.Job{LocationRaw: "Nairobi", Country: "Kenya", WorkArrangement: "On-site"})
	far := locationAxis(p, models.Job{LocationRaw: "Manila", Country: "Philippines", WorkArrangement: "On-site"})

	if near == nil || far == nil {
		t.Fatal("both should score with a stated preference")
	}
	if *near <= *far {
		t.Errorf("Nairobi (%d) should beat Manila (%d) for a Nairobi-based candidate", *near, *far)
	}
}

func TestAxesJSONRoundTrips(t *testing.T) {
	a := Axes{axis(96), axis(88), axis(94), axis(90), nil}
	var back Axes
	if err := json.Unmarshal(a.JSON(), &back); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if back.Skills == nil || *back.Skills != 96 {
		t.Errorf("skills lost in the round trip: %v", back.Skills)
	}
	// nil must survive as null, not become 0 — the app renders it as "—".
	if back.Pay != nil {
		t.Errorf("unknown pay came back as %d instead of null", *back.Pay)
	}
}

// The keyword scorer must now produce a breakdown, not just a number.
func TestKeywordScorerEmitsAxes(t *testing.T) {
	r := score(t, kwProfile(), kwJob("Flutter Engineer", []string{"Flutter", "Dart"}, nil))

	if r.Axes.Skills == nil {
		t.Error("skills axis missing")
	}
	if r.Axes.Domain == nil {
		t.Error("domain axis missing")
	}
	if r.Axes.Seniority == nil {
		t.Error("seniority axis missing")
	}
	// The test profile states neither location nor pay, so both must be
	// unknown rather than invented.
	if r.Axes.Location != nil {
		t.Errorf("location scored %d with no stated preference", *r.Axes.Location)
	}
	if r.Axes.Pay != nil {
		t.Errorf("pay scored %d with no stated floor", *r.Axes.Pay)
	}
}
