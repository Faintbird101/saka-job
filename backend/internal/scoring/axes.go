package scoring

import (
	"encoding/json"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// Axes is the score broken into the five things it is actually made of.
//
// Every field is a pointer, and nil means "the posting did not say" rather
// than zero. That distinction is the whole point: most postings in the live
// feed state no salary, and scoring silence as zero would drag every total
// down and paint the pay bar red for reasons that have nothing to do with the
// job. An unknown axis is excluded from the weighted total instead.
type Axes struct {
	Skills    *int `json:"skills"`
	Seniority *int `json:"seniority"`
	Domain    *int `json:"domain"`
	Location  *int `json:"location"`
	Pay       *int `json:"pay"`
}

// Weakest returns the lowest known axis and its score — what the app calls out
// as the thing holding a match back.
func (a Axes) Weakest() (name string, score int, ok bool) {
	for _, c := range []struct {
		name string
		v    *int
	}{
		{"skills", a.Skills}, {"seniority", a.Seniority}, {"domain", a.Domain},
		{"location", a.Location}, {"pay", a.Pay},
	} {
		if c.v == nil {
			continue
		}
		if !ok || *c.v < score {
			name, score, ok = c.name, *c.v, true
		}
	}
	return
}

// JSON renders the axes for the score_axes column.
func (a Axes) JSON() json.RawMessage {
	b, err := json.Marshal(a)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}

// Weights are the per-axis weights from the profile. They need not sum to 100:
// Combine normalises by the weights that actually applied, so setting one to 0
// removes that axis cleanly and an unknown axis redistributes rather than
// penalises.
type Weights struct {
	Skills, Seniority, Domain, Location, Pay int
}

// WeightsFrom reads the profile, falling back to sensible defaults if a
// profile predates the weights columns.
func WeightsFrom(p models.Profile) Weights {
	w := Weights{
		Skills: p.WeightSkills, Seniority: p.WeightSeniority, Domain: p.WeightDomain,
		Location: p.WeightLocation, Pay: p.WeightPay,
	}
	if w.Skills+w.Seniority+w.Domain+w.Location+w.Pay == 0 {
		return Weights{Skills: 45, Seniority: 15, Domain: 20, Location: 12, Pay: 8}
	}
	return w
}

// Combine folds the axes into the single 0-100 score.
//
// Only known axes contribute, and the divisor is the sum of the weights that
// applied — so a job with no stated salary is scored on the other four rather
// than losing the pay weight outright.
func Combine(a Axes, w Weights) int {
	total, used := 0, 0
	for _, c := range []struct {
		v      *int
		weight int
	}{
		{a.Skills, w.Skills}, {a.Seniority, w.Seniority}, {a.Domain, w.Domain},
		{a.Location, w.Location}, {a.Pay, w.Pay},
	} {
		if c.v == nil || c.weight <= 0 {
			continue
		}
		total += *c.v * c.weight
		used += c.weight
	}
	if used == 0 {
		return 0
	}
	return clamp((total + used/2) / used)
}

// ---------- location ----------

// locationAxis scores how well a posting's location and work arrangement match
// what the candidate said they want.
//
// Returns nil when the profile states no preference — with nothing to compare
// against, any number would be invented.
func locationAxis(p models.Profile, j models.Job) *int {
	prefs := normalizeAll(jsonList(p.PreferredLocations))
	pref := strings.ToLower(strings.TrimSpace(p.RemotePreference))
	if len(prefs) == 0 && (pref == "" || pref == "any") {
		return nil
	}

	arrangement := normalizeText(j.WorkArrangement)
	isRemote := strings.Contains(arrangement, "remote")
	isHybrid := strings.Contains(arrangement, "hybrid")

	// Remote work makes the posting's location largely irrelevant, so it
	// satisfies a location preference regardless of where the office is.
	switch pref {
	case "remote_only":
		if isRemote {
			return axis(100)
		}
		if isHybrid {
			return axis(45)
		}
		return axis(10)
	case "hybrid_ok":
		if isRemote || isHybrid {
			return axis(100)
		}
	}

	if isRemote {
		return axis(100)
	}

	if len(prefs) == 0 {
		// A work-arrangement preference only, and this is on-site.
		if pref == "onsite_ok" {
			return axis(80)
		}
		return axis(60)
	}

	// Compare the posting's place against the wanted places.
	place := normalizeText(j.LocationRaw + " " + j.Country)
	if place == "" {
		return nil
	}
	for _, want := range prefs {
		if strings.Contains(place, want.normalized) || strings.Contains(want.normalized, place) {
			if isHybrid {
				return axis(90)
			}
			return axis(100)
		}
	}
	return axis(20)
}

// ---------- pay ----------

// payAxis compares the posting's salary against the candidate's floor.
//
// Returns nil — not zero — when either side is silent, which is the common
// case: most postings state no salary at all.
func payAxis(p models.Profile, j models.Job) *int {
	if p.SalaryFloor <= 0 {
		return nil
	}
	// Currency mismatches are not converted. Guessing an exchange rate would
	// produce a confident number from an assumption, so the axis stays unknown.
	if j.SalaryCurrency != "" && p.SalaryCurrency != "" &&
		!strings.EqualFold(j.SalaryCurrency, p.SalaryCurrency) {
		return nil
	}

	// Judge on the top of the range: that is what is actually negotiable.
	offered := 0.0
	switch {
	case j.SalaryMax != nil && *j.SalaryMax > 0:
		offered = *j.SalaryMax
	case j.SalaryMin != nil && *j.SalaryMin > 0:
		offered = *j.SalaryMin
	default:
		return nil
	}

	floor := float64(p.SalaryFloor)
	ratio := offered / floor
	switch {
	case ratio >= 1.25:
		return axis(100)
	case ratio >= 1.0:
		return axis(90)
	case ratio >= 0.9:
		return axis(70)
	case ratio >= 0.75:
		return axis(45)
	default:
		return axis(15)
	}
}

// axis wraps a value as a known axis score. The pointer is what makes
// "unknown" expressible at all.
func axis(n int) *int { return &n }

// jsonList is shared with prompt.go; declared there.
var _ = jsonList
