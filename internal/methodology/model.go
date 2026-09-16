// Package methodology implements the candidate methodology over normalized facts.
// It does not collect data or authorize access to repository reports.
package methodology

import (
	"fmt"
	"math"
	"time"
)

const Version = "rhs-2.0-rc1"

type State string

const (
	Known         State = "known"
	Missing       State = "missing"
	Denied        State = "denied"
	Failed        State = "error"
	Incomplete    State = "incomplete"
	Stale         State = "stale"
	Insufficient  State = "insufficient"
	NotApplicable State = "not_applicable"
)

// Value carries coverage independently of the observed score. Unknown has no score.
type Value struct {
	State    State    `json:"state"`
	Score    *float64 `json:"score"`
	Coverage float64  `json:"coverage"`
}

func measured(n float64) Value { return Value{Known, &n, 1} }
func Unknown(s State) Value    { return Value{State: s} }
func finite(n float64) bool    { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func (v Value) Validate() error {
	switch v.State {
	case Known:
		if v.Score == nil || !finite(*v.Score) || *v.Score < 0 || *v.Score > 100 || !finite(v.Coverage) || v.Coverage <= 0 || v.Coverage > 1 {
			return fmt.Errorf("invalid measured value")
		}
	case Missing, Denied, Failed, Incomplete, Stale, Insufficient, NotApplicable:
		if v.Score != nil || v.Coverage != 0 {
			return fmt.Errorf("unknown value carries a score or coverage")
		}
	default:
		return fmt.Errorf("invalid state %q", v.State)
	}
	return nil
}

// Combine keeps missing weight in the coverage denominator, not in the score.
func Combine(values []Value, weights []float64) (Value, error) {
	if len(values) == 0 || len(values) != len(weights) {
		return Value{}, fmt.Errorf("invalid dimensions")
	}
	total, known, sum := 0.0, 0.0, 0.0
	for i, v := range values {
		if err := v.Validate(); err != nil {
			return Value{}, err
		}
		w := weights[i]
		if !finite(w) || w <= 0 {
			return Value{}, fmt.Errorf("invalid weight")
		}
		total += w
		if v.State == Known {
			k := w * v.Coverage
			known += k
			sum += k * (*v.Score)
		}
	}
	if known == 0 {
		return Unknown(Missing), nil
	}
	v := measured(sum / known)
	v.Coverage = known / total
	return v, nil
}

// Documentation: five dimensions, each containing exactly two evidence checks.
// nil means unavailable, false means confirmed absent, true means found.
func Documentation(checks [10]*bool) (Value, error) {
	weights := []float64{10, 10, 15, 15, 10, 10, 7.5, 7.5, 7.5, 7.5}
	values := make([]Value, 10)
	for i, p := range checks {
		values[i] = Unknown(Missing)
		if p != nil {
			n := 0.0
			if *p {
				n = 100
			}
			values[i] = measured(n)
		}
	}
	return Combine(values, weights)
}

// Ratio accepts complete, deduplicated eligible cohorts only.
func Ratio(good, total, minSample int) (Value, error) {
	if minSample < 1 || good < 0 || total < 0 || good > total {
		return Value{}, fmt.Errorf("invalid cohort")
	}
	if total < minSample {
		return Unknown(Insufficient), nil
	}
	return measured(100 * float64(good) / float64(total)), nil
}
func CI(success, total int) (Value, error)      { return Ratio(success, total, 5) }
func Response(onTime, total int) (Value, error) { return Ratio(onTime, total, 5) }
func Backlog(stuck, open int) (Value, error) {
	if stuck < 0 || open < 0 || stuck > open {
		return Value{}, fmt.Errorf("invalid backlog")
	}
	if open == 0 {
		return Unknown(Insufficient), nil
	}
	return Ratio(open-stuck, open, 1)
}
func Issues(response, backlog Value) (Value, error) {
	return Combine([]Value{response, backlog}, []float64{60, 40})
}
func Activity(rhythm, response Value) (Value, error) {
	return Combine([]Value{rhythm, response}, []float64{60, 40})
}

// Code measures structural properties only. TODO count is intentionally not an input.
func Code(complexLines, functionLines, duplicateLines, authoredLines int) (Value, error) {
	if complexLines < 0 || functionLines < 0 || complexLines > functionLines || duplicateLines < 0 || authoredLines < 0 || duplicateLines > authoredLines || functionLines > authoredLines {
		return Value{}, fmt.Errorf("invalid code counters")
	}
	a, b := Unknown(Insufficient), Unknown(Insufficient)
	if functionLines >= 100 {
		a = measured(100 * (1 - float64(complexLines)/float64(functionLines)))
	}
	if authoredLines >= 100 {
		b = measured(100 * math.Max(0, 1-float64(duplicateLines)/float64(authoredLines)/0.2))
	}
	return Combine([]Value{a, b}, []float64{65, 35})
}

// Rhythm takes sorted-independent historical intervals in days and the current gap.
// The current unfinished interval must NOT be included in history.
func Rhythm(intervals []float64, gap float64) (Value, error) {
	if !finite(gap) || gap < 0 {
		return Value{}, fmt.Errorf("invalid gap")
	}
	x := append([]float64(nil), intervals...)
	for _, n := range x {
		if !finite(n) || n <= 0 {
			return Value{}, fmt.Errorf("invalid interval")
		}
	}
	if len(x) < 4 {
		return Unknown(Insufficient), nil
	}
	// Insertion sort avoids modifying the caller's input.
	for i := 1; i < len(x); i++ {
		for j := i; j > 0 && x[j] < x[j-1]; j-- {
			x[j], x[j-1] = x[j-1], x[j]
		}
	}
	b := math.Max(30, x[int(math.Ceil(.75*float64(len(x))))-1])
	return measured(100 * math.Max(0, math.Min(1, (3*b-gap)/(2*b)))), nil
}

type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}
type SecurityResult struct {
	Value    Value `json:"value"`
	Critical bool  `json:"critical"`
}

// Security requires a complete current AppSec snapshot, supplied by its adapter.
func Security(state State, findings []Finding) (SecurityResult, error) {
	if state != Known {
		v := Unknown(state)
		if err := v.Validate(); err != nil {
			return SecurityResult{}, err
		}
		return SecurityResult{Value: v}, nil
	}
	seen := map[string]Finding{}
	sum, maxPenalty := 0.0, 0.0
	critical := false
	for _, f := range findings {
		if f.ID == "" {
			return SecurityResult{}, fmt.Errorf("empty finding identity")
		}
		if previous, ok := seen[f.ID]; ok {
			if previous != f {
				return SecurityResult{}, fmt.Errorf("conflicting duplicate")
			}
			continue
		}
		seen[f.ID] = f
		p, ok := map[string]float64{"critical": 60, "high": 25, "medium": 8, "low": 2}[f.Severity]
		if !ok {
			return SecurityResult{}, fmt.Errorf("unknown severity")
		}
		switch f.Status {
		case "resolved", "false_positive":
			continue
		case "open", "accepted":
		default:
			return SecurityResult{}, fmt.Errorf("unknown finding status")
		}
		critical = critical || f.Severity == "critical"
		sum += p
		maxPenalty = math.Max(maxPenalty, p)
	}
	p := math.Min(100, maxPenalty+math.Min(30, .25*(sum-maxPenalty)))
	return SecurityResult{measured(100 - p), critical}, nil
}

type Snapshot struct {
	Profile               string           `json:"profile"`
	At                    time.Time        `json:"at"`
	AsOf                  time.Time        `json:"as_of"`
	PublicRepository      bool             `json:"public_repository"`
	Categories            map[string]Value `json:"categories"`
	SecurityState         State            `json:"security_state"`
	Findings              []Finding        `json:"findings"`
	PreviousSecurityBlock bool             `json:"previous_security_block"`
}
type Report struct {
	Version    string           `json:"version"`
	Profile    string           `json:"profile"`
	Categories map[string]Value `json:"categories"`
	Base       *float64         `json:"base"`
	Score      *float64         `json:"score"`
	Coverage   float64          `json:"coverage"`
	Status     string           `json:"status"`
	Gate       string           `json:"gate"`
	Rankable   bool             `json:"rankable"`
}

// Evaluate receives an already authorized snapshot. Public projections never read
// security inputs, including invalid or historical private security facts.
func Evaluate(s Snapshot) (Report, error) {
	r := Report{Version: Version, Profile: s.Profile, Categories: map[string]Value{}}
	if s.Profile != "public" && s.Profile != "personal" {
		return r, fmt.Errorf("unknown profile")
	}
	if s.At.IsZero() || s.AsOf.IsZero() || s.At.After(s.AsOf) {
		return r, fmt.Errorf("invalid snapshot time")
	}
	if s.Profile == "public" && !s.PublicRepository {
		return r, fmt.Errorf("repository is not public")
	}
	ids := []string{"documentation", "ci", "code", "activity", "issues"}
	weights := []float64{20, 25, 25, 15, 15}
	for id := range s.Categories {
		found := false
		for _, known := range ids {
			found = found || known == id
		}
		if !found {
			return r, fmt.Errorf("unknown category %s", id)
		}
	}
	values := make([]Value, 0, 6)
	usable := 0
	for _, id := range ids {
		v, ok := s.Categories[id]
		if !ok {
			v = Unknown(Missing)
		}
		if err := v.Validate(); err != nil {
			return r, fmt.Errorf("%s: %w", id, err)
		}
		r.Categories[id] = v
		values = append(values, v)
		if v.State == Known && v.Coverage >= .6 {
			usable++
		}
	}
	blockedPending := false
	if s.Profile == "personal" {
		state := s.SecurityState
		if state == "" {
			state = Missing
		}
		sec, err := Security(state, s.Findings)
		if err != nil {
			return r, err
		}
		for i := range weights {
			weights[i] *= .8
		}
		weights = append(weights, 20)
		values = append(values, sec.Value)
		r.Categories["security"] = sec.Value
		if sec.Value.State == Known {
			usable++
			if sec.Critical {
				r.Gate = "critical_appsec"
			}
		} else if s.PreviousSecurityBlock {
			blockedPending = true
			r.Gate = "security_status_unverified"
		}
	}
	combined, err := Combine(values, weights)
	if err != nil {
		return r, err
	}
	r.Coverage = combined.Coverage
	if s.AsOf.Sub(s.At) > 24*time.Hour {
		r.Status = "stale"
		return r, nil
	}
	if combined.Score == nil || r.Coverage < .6 || usable < 3 {
		r.Status = "insufficient"
		return r, nil
	}
	r.Base = combined.Score
	if blockedPending {
		r.Status = "blocked_unverified"
		return r, nil
	}
	n := *r.Base
	if r.Gate == "critical_appsec" {
		n = math.Min(n, 59)
	}
	r.Score = &n
	r.Status = "partial"
	if math.Abs(r.Coverage-1) < 1e-12 {
		r.Status = "complete"
	}
	r.Rankable = s.Profile == "public" && r.Status == "complete"
	return r, nil
}
