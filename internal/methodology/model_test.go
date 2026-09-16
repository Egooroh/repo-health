package methodology

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
func score(t *testing.T, v Value, want float64) {
	t.Helper()
	if err := v.Validate(); err != nil {
		t.Fatal(err)
	}
	if v.Score == nil {
		t.Fatal("missing score")
	}
	near(t, *v.Score, want)
}
func fixture() Snapshot {
	at := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	s := Snapshot{Profile: "public", At: at, AsOf: at, PublicRepository: true, Categories: map[string]Value{}}
	for _, id := range []string{"documentation", "ci", "code", "activity", "issues"} {
		s.Categories[id] = measured(90)
	}
	return s
}
func evaluate(t *testing.T, s Snapshot) Report {
	t.Helper()
	r, e := Evaluate(s)
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestCounterexamples(t *testing.T) {
	t.Run("issues_percent_units", func(t *testing.T) {
		old := .5*60 + (100.-0)*40
		near(t, old, 4030)
		// Direct correction of PDF A, not the new response-based Issues metric.
		corrected := 60*.5 + 40*(1.-0)
		near(t, corrected, 70)
		r, _ := Response(5, 10)
		b, _ := Backlog(0, 10)
		v, e := Issues(r, b)
		if e != nil {
			t.Fatal(e)
		}
		score(t, v, 70)
	})
	t.Run("adding_todos_does_not_improve_code", func(t *testing.T) {
		before := 100. - (1*2 + 1*50)
		after := 100. - (10*2 + .1*50)
		near(t, before, 48)
		near(t, after, 75)
		// Comments are excluded from authored executable-line counts.
		a, _ := Code(100, 1000, 40, 1000)
		b, _ := Code(100, 1000, 40, 1000)
		score(t, a, 86.5)
		if !reflect.DeepEqual(a, b) {
			t.Fatal("comment-only edit changed structural score")
		}
	})
	t.Run("category_cap_is_not_overall_cap", func(t *testing.T) {
		near(t, .2*30+.8*100, 86)
		s := fixture()
		s.Profile = "personal"
		s.SecurityState = Known
		s.Findings = []Finding{{"a", "critical", "open"}}
		r := evaluate(t, s)
		near(t, *r.Base, 80)
		near(t, *r.Score, 59)
	})
}

func TestDocumentation(t *testing.T) {
	yes, no := true, false
	var checks [10]*bool
	for i := range checks {
		checks[i] = &no
	}
	v, _ := Documentation(checks)
	score(t, v, 0)
	for i := range checks {
		checks[i] = &yes
	}
	v, _ = Documentation(checks)
	score(t, v, 100)
	checks[2] = nil
	v, _ = Documentation(checks)
	score(t, v, 100)
	near(t, v.Coverage, .85)
	checks[2] = &no
	v, _ = Documentation(checks)
	score(t, v, 85)
	near(t, v.Coverage, 1)
}
func TestCIAndIssues(t *testing.T) {
	for _, tc := range []struct {
		name        string
		good, total int
		want        float64
		unknown     bool
	}{
		{"one_success", 1, 1, 0, true}, {"five_successes", 5, 5, 100, false}, {"all_failed", 0, 5, 0, false}, {"mixed", 4, 5, 80, false}, {"empty", 0, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, e := CI(tc.good, tc.total)
			if e != nil {
				t.Fatal(e)
			}
			if tc.unknown {
				if v.State != Insufficient || v.Score != nil {
					t.Fatal(v)
				}
			} else {
				score(t, v, tc.want)
			}
		})
	}
	if _, e := CI(6, 5); e == nil {
		t.Fatal("accepted impossible counts")
	}
	b, _ := Backlog(0, 0)
	if b.Score != nil {
		t.Fatal("empty backlog fabricated support quality")
	}
	b, _ = Backlog(2, 10)
	score(t, b, 80)
	r, _ := Response(5, 10)
	v, _ := Issues(r, Unknown(Missing))
	score(t, v, 50)
	near(t, v.Coverage, .6)
}
func TestRhythmAndCode(t *testing.T) {
	for _, tc := range []struct{ gap, want float64 }{{0, 100}, {30, 100}, {60, 50}, {90, 0}, {1000, 0}} {
		v, e := Rhythm([]float64{10, 20, 30, 40}, tc.gap)
		if e != nil {
			t.Fatal(e)
		}
		score(t, v, tc.want)
	}
	x := []float64{40, 10, 30, 20}
	saved := append([]float64(nil), x...)
	Rhythm(x, 60)
	if !reflect.DeepEqual(x, saved) {
		t.Fatal("mutated history")
	}
	v, _ := Rhythm([]float64{1, 2, 3}, 1)
	if v.Score != nil {
		t.Fatal("short history scored")
	}
	v, _ = Code(0, 99, 0, 99)
	if v.Score != nil {
		t.Fatal("tiny code sample scored")
	}
	v, _ = Code(0, 0, 0, 1000)
	score(t, v, 100)
	near(t, v.Coverage, .35)
	v, _ = Code(1000, 1000, 1000, 1000)
	score(t, v, 0)
	if _, e := Code(1001, 1000, 0, 1000); e == nil {
		t.Fatal("invalid code counters")
	}
	for d := 0; d <= 1000; d++ {
		v, e := Code(100, 1000, d, 1000)
		if e != nil {
			t.Fatal(e)
		}
		if *v.Score < 0 || *v.Score > 100 {
			t.Fatal(v)
		}
	}
}
func TestSecurity(t *testing.T) {
	tests := []struct {
		name string
		fs   []Finding
		want float64
		gate bool
	}{
		{"clean", nil, 100, false},
		{"high_and_medium", []Finding{{"a", "high", "open"}, {"b", "medium", "open"}, {"c", "medium", "open"}}, 71, false},
		{"critical", []Finding{{"a", "critical", "open"}}, 40, true},
		{"accepted_is_not_fixed", []Finding{{"a", "critical", "accepted"}}, 40, true},
		{"resolved", []Finding{{"a", "critical", "resolved"}}, 100, false},
		{"false_positive", []Finding{{"a", "critical", "false_positive"}}, 100, false},
		{"deduplicate", []Finding{{"a", "high", "open"}, {"a", "high", "open"}}, 75, false},
		{"low_is_not_ignored", []Finding{{"a", "low", "open"}}, 98, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, e := Security(Known, tc.fs)
			if e != nil {
				t.Fatal(e)
			}
			score(t, r.Value, tc.want)
			if r.Critical != tc.gate {
				t.Fatal("wrong gate")
			}
		})
	}
	for _, state := range []State{Missing, Denied, Failed, Incomplete, Stale, Insufficient, NotApplicable} {
		r, e := Security(state, nil)
		if e != nil || r.Value.Score != nil || r.Critical {
			t.Fatal(state, r, e)
		}
	}
	for _, fs := range [][]Finding{
		{{"a", "unknown", "open"}}, {{"a", "high", "unknown"}}, {{"", "high", "open"}}, {{"a", "high", "open"}, {"a", "critical", "open"}},
	} {
		if _, e := Security(Known, fs); e == nil {
			t.Fatal("invalid findings accepted")
		}
	}
}
func TestAggregation(t *testing.T) {
	t.Run("complete_public", func(t *testing.T) {
		r := evaluate(t, fixture())
		near(t, *r.Score, 90)
		if !r.Rankable || r.Status != "complete" {
			t.Fatal(r)
		}
	})
	t.Run("public_never_reads_security", func(t *testing.T) {
		s := fixture()
		before := evaluate(t, s)
		s.SecurityState = "invalid"
		s.Findings = []Finding{{"secret", "invalid", "invalid"}}
		s.PreviousSecurityBlock = true
		after := evaluate(t, s)
		a, _ := json.Marshal(before)
		b, _ := json.Marshal(after)
		if string(a) != string(b) {
			t.Fatal("private input affected public output")
		}
	})
	t.Run("hidden_bad_ci_loses_rank", func(t *testing.T) {
		s := fixture()
		s.Categories["ci"] = measured(20)
		before := evaluate(t, s)
		near(t, *before.Score, 72.5)
		s.Categories["ci"] = Unknown(Denied)
		after := evaluate(t, s)
		near(t, *after.Score, 90)
		near(t, after.Coverage, .75)
		if after.Rankable || after.Status != "partial" {
			t.Fatal(after)
		}
	})
	t.Run("only_docs_has_no_overall_score", func(t *testing.T) {
		s := fixture()
		s.Categories = map[string]Value{"documentation": measured(100)}
		r := evaluate(t, s)
		if r.Score != nil || r.Status != "insufficient" {
			t.Fatal(r)
		}
	})
	t.Run("partial_category_counts_fractional_coverage", func(t *testing.T) {
		s := fixture()
		v := measured(90)
		v.Coverage = .5
		s.Categories["code"] = v
		r := evaluate(t, s)
		near(t, r.Coverage, .875)
		if r.Rankable {
			t.Fatal("partial ranked")
		}
	})
	t.Run("personal_missing_security", func(t *testing.T) {
		s := fixture()
		s.Profile = "personal"
		s.SecurityState = Denied
		r := evaluate(t, s)
		near(t, *r.Score, 90)
		near(t, r.Coverage, .8)
		if r.Rankable || r.Status != "partial" {
			t.Fatal(r)
		}
	})
	t.Run("lost_security_does_not_remove_block", func(t *testing.T) {
		s := fixture()
		s.Profile = "personal"
		s.SecurityState = Denied
		s.PreviousSecurityBlock = true
		r := evaluate(t, s)
		if r.Score != nil || r.Status != "blocked_unverified" {
			t.Fatal(r)
		}
	})
	t.Run("fresh_clean_scan_removes_block", func(t *testing.T) {
		s := fixture()
		s.Profile = "personal"
		s.SecurityState = Known
		s.PreviousSecurityBlock = true
		r := evaluate(t, s)
		near(t, *r.Score, 92)
		if r.Gate != "" || r.Status != "complete" {
			t.Fatal(r)
		}
	})
	t.Run("freshness_boundary", func(t *testing.T) {
		s := fixture()
		s.AsOf = s.At.Add(24 * time.Hour)
		if !evaluate(t, s).Rankable {
			t.Fatal("boundary rejected")
		}
		s.AsOf = s.AsOf.Add(time.Nanosecond)
		r := evaluate(t, s)
		if r.Score != nil || r.Rankable || r.Status != "stale" {
			t.Fatal(r)
		}
	})
	t.Run("private_repo_not_public", func(t *testing.T) {
		s := fixture()
		s.PublicRepository = false
		if _, e := Evaluate(s); e == nil {
			t.Fatal("private repo leaked")
		}
	})
	t.Run("future_snapshot_invalid", func(t *testing.T) {
		s := fixture()
		s.At = s.At.Add(time.Second)
		if _, e := Evaluate(s); e == nil {
			t.Fatal("future snapshot accepted")
		}
	})
	t.Run("repeatable", func(t *testing.T) {
		s := fixture()
		a := evaluate(t, s)
		for i := 0; i < 100; i++ {
			if !reflect.DeepEqual(a, evaluate(t, s)) {
				t.Fatal("non deterministic")
			}
		}
	})
}

func TestInputValidation(t *testing.T) {
	nan, inf, negative, over := math.NaN(), math.Inf(1), -1.0, 101.0
	for _, v := range []Value{{Known, nil, 1}, {Known, &nan, 1}, {Known, &inf, 1}, {Known, &negative, 1}, {Known, &over, 1}, {Known, measured(50).Score, 0}, {Denied, measured(50).Score, 0}, {Missing, nil, 1}, {"wat", nil, 0}} {
		if v.Validate() == nil {
			t.Fatalf("accepted %+v", v)
		}
	}
	if _, e := Combine([]Value{measured(1)}, []float64{0}); e == nil {
		t.Fatal("zero weight")
	}
	if _, e := Combine(nil, nil); e == nil {
		t.Fatal("empty weights")
	}
}

func TestCoverageThresholds(t *testing.T) {
	s := fixture()
	s.Categories = map[string]Value{"documentation": measured(90), "code": measured(90), "activity": measured(90)}
	r := evaluate(t, s)
	near(t, r.Coverage, .6)
	if r.Score == nil || r.Status != "partial" || r.Rankable {
		t.Fatal(r)
	}
	s.Categories["activity"] = Unknown(Missing)
	if evaluate(t, s).Score != nil {
		t.Fatal("below coverage threshold scored")
	}
	s = fixture()
	for _, id := range []string{"documentation", "activity", "issues"} {
		v := measured(90)
		v.Coverage = .59
		s.Categories[id] = v
	}
	r = evaluate(t, s)
	if r.Coverage < .6 || r.Score != nil {
		t.Fatal("broad but shallow evidence accepted", r)
	}
}

func TestSecurityImprovementAndOrder(t *testing.T) {
	fs := []Finding{{"a", "critical", "open"}, {"b", "high", "open"}, {"c", "low", "open"}}
	a, _ := Security(Known, fs)
	b, _ := Security(Known, []Finding{fs[2], fs[0], fs[1]})
	if !reflect.DeepEqual(a, b) {
		t.Fatal("input order changed score")
	}
	for i := range fs {
		changed := append([]Finding(nil), fs...)
		changed[i].Status = "resolved"
		next, e := Security(Known, changed)
		if e != nil || *next.Value.Score < *a.Value.Score {
			t.Fatal("fix lowered score", next, e)
		}
	}
}
func TestMonotonicity(t *testing.T) {
	for _, id := range []string{"documentation", "ci", "code", "activity", "issues"} {
		previous := -1.0
		for n := 0; n <= 100; n++ {
			s := fixture()
			s.Categories[id] = measured(float64(n))
			r := evaluate(t, s)
			if *r.Score < previous {
				t.Fatal("improvement lowered score")
			}
			previous = *r.Score
		}
	}
	fs := []Finding{}
	previous := 100.0
	for i := 0; i < 50; i++ {
		fs = append(fs, Finding{ID: time.Unix(int64(i), 0).String(), Severity: "high", Status: "open"})
		r, e := Security(Known, fs)
		if e != nil {
			t.Fatal(e)
		}
		if *r.Value.Score > previous {
			t.Fatal("new finding improved score")
		}
		previous = *r.Value.Score
	}
}
