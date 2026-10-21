package theoryloop

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustWorld(t *testing.T, name string, seed uint64, n int) World {
	t.Helper()
	w, err := MakeWorld(name, seed, n)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// ------------------------------------------------------------- statistics

func TestTDistributionPValues(t *testing.T) {
	cases := []struct{ t, dof, want float64 }{
		{1.959964, 1e7, 0.05}, // normal limit
		{2.0, 10, 0.073388},   // standard t table
		{2.228139, 10, 0.05},  // 10 df critical value
		{0, 5, 1},
	}
	for _, c := range cases {
		if got := tTwoSided(c.t, c.dof); math.Abs(got-c.want) > 1e-4 {
			t.Errorf("tTwoSided(%v, %v) = %v, want %v", c.t, c.dof, got, c.want)
		}
	}
}

func TestOLSRecoversCoefficients(t *testing.T) {
	r := newRNG(1, 1)
	n := 500
	x1, x2, y := make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range y {
		x1[i], x2[i] = r.normal(), r.normal()
		y[i] = 1 + 2*x1[i] - 0.5*x2[i] + 0.1*r.normal()
	}
	coef, p, ok := ols(y, x1, x2)
	if !ok || math.Abs(coef[0]-2) > 0.05 || math.Abs(coef[1]+0.5) > 0.05 || p[0] > 1e-10 || p[1] > 1e-10 {
		t.Fatalf("ols = %v %v %v", coef, p, ok)
	}
	if _, p, ok := ols(y, x1, x1); ok || p[0] != 1 {
		t.Fatalf("collinear design should be rejected, got ok=%v p=%v", ok, p)
	}
}

func TestDistributions(t *testing.T) {
	r := newRNG(7, 1)
	var b, z, zz []float64
	for i := 0; i < 50000; i++ {
		b = append(b, r.beta(2, 8))
		v := r.normal()
		z, zz = append(z, v), append(zz, v*v)
	}
	if m := meanOf(b); math.Abs(m-0.2) > 0.005 {
		t.Errorf("Beta(2,8) mean = %v, want 0.2", m)
	}
	if m, v := meanOf(z), meanOf(zz); math.Abs(m) > 0.02 || math.Abs(v-1) > 0.03 {
		t.Errorf("normal mean %v, variance %v", m, v)
	}
}

// ------------------------------------------------------------------ worlds

func TestWorldsReproducible(t *testing.T) {
	a, b := mustWorld(t, "scope", 7, 2000), mustWorld(t, "scope", 7, 2000)
	for name, col := range a.Data.Cols {
		for i := range col {
			if col[i] != b.Data.Cols[name][i] {
				t.Fatalf("column %s differs at row %d", name, i)
			}
		}
	}
}

func TestPlantedEffectsArePresent(t *testing.T) {
	for _, name := range WorldNames {
		w := mustWorld(t, name, 0, 20000)
		for _, slot := range w.Truth {
			for _, p := range slot {
				if !Evaluate(p, w.Data, nil).Holds {
					t.Errorf("%s: planted %q does not hold", name, p.Label())
				}
			}
		}
	}
}

func TestDecoyIsADecoy(t *testing.T) {
	w := mustWorld(t, "confound", 0, 20000)
	raw := Evaluate(P("verified", 1), w.Data, nil)
	ctl := Evaluate(P("verified", 1), w.Data, []string{"community"})
	if !raw.Holds || ctl.Holds {
		t.Fatalf("raw holds=%v, controlled holds=%v", raw.Holds, ctl.Holds)
	}
}

// ----------------------------------------------------------------- scoring

func TestEquivalentFramingsBothCount(t *testing.T) {
	truth := mustWorld(t, "scope", 0, 100).Truth
	for _, p := range []Proposition{PWhen("verified", 1, "bear_launch", 1), PWhen("bear_launch", 1, "verified", 1)} {
		if s := ScoreTheory([]Proposition{p}, truth); s.Perfect != 1 {
			t.Errorf("%q should count as recovering the mechanism", p.Label())
		}
	}
}

func TestMissingScopeIsWrongScopeNotFalse(t *testing.T) {
	s := ScoreTheory([]Proposition{P("verified", 1)}, mustWorld(t, "scope", 0, 100).Truth)
	if s.WrongScope != 1 || s.FalseProps != 0 || s.Perfect != 0 {
		t.Fatalf("got %+v", s)
	}
}

func TestNullWorldScoring(t *testing.T) {
	truth := mustWorld(t, "null", 0, 100).Truth
	if s := ScoreTheory(nil, truth); s.Perfect != 1 || !math.IsNaN(s.RecallExact) {
		t.Errorf("empty theory in null world: %+v", s)
	}
	if s := ScoreTheory([]Proposition{P("royalty", 1)}, truth); s.AnyFalse != 1 {
		t.Errorf("any claim in the null world is false: %+v", s)
	}
}

// -------------------------------------------------------------------- loop

func TestRunsAreDeterministic(t *testing.T) {
	w := mustWorld(t, "multi", 3, 2000)
	a, _ := Run(w, "loop_holdout", 3, Config{})
	b, _ := Run(w, "loop_holdout", 3, Config{})
	if strings.Join(labels(a.Theory), ";") != strings.Join(labels(b.Theory), ";") || a.NTests != b.NTests {
		t.Fatal("same world and seed gave different runs")
	}
}

func TestLoopRepairsConfound(t *testing.T) {
	single, loop := 0, 0
	for s := uint64(0); s < 20; s++ {
		w := mustWorld(t, "confound", s, 2000)
		a, _ := Run(w, "single_pass", s, Config{})
		b, _ := Run(w, "loop", s, Config{})
		single += a.AnyFalse
		loop += b.AnyFalse
	}
	if loop >= single {
		t.Fatalf("loop should assert the decoy less often: single %d/20, loop %d/20", single, loop)
	}
}

func TestLedgerRecordsEveryTest(t *testing.T) {
	r, err := Run(mustWorld(t, "main", 0, 2000), "loop_holdout", 0, Config{})
	if err != nil {
		t.Fatal(err)
	}
	holdout := false
	for _, e := range r.Ledger.Entries {
		holdout = holdout || e.Set == "holdout"
	}
	if r.NTests <= len(Predictors) || !holdout {
		t.Fatalf("ledger has %d tests, holdout entries: %v", r.NTests, holdout)
	}
}

// ------------------------------------------------------------- LLM backend

type fakeClaude struct {
	replies []string
	calls   []string
}

func (f *fakeClaude) call(system, user, model string, maxTokens int) (string, error) {
	f.calls = append(f.calls, user)
	r := f.replies[0]
	f.replies = f.replies[1:]
	return r, nil
}

func TestParseJSONWithProse(t *testing.T) {
	v, err := parseJSON("Here you go:\n[{\"a\": 1}]\nHope that helps.")
	if err != nil || len(asObjects(v)) != 1 {
		t.Fatalf("got %v, %v", v, err)
	}
}

func TestAnonymizedRoundTrip(t *testing.T) {
	w := mustWorld(t, "confound", 0, 2000)
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	fake := &fakeClaude{replies: []string{
		j([]map[string]any{{"predictor": "x1", "sign": 1}}),                           // x1 = verified
		j([]map[string]any{{"index": 0, "kind": "confound", "var": "x3", "sign": 1}}), // x3 = community
	}}
	l := &Ledger{}
	theory, err := LLMProposer{Anonymize: true, Call: fake.call}.Propose(w.Data, l)
	if err != nil || len(theory) != 1 || theory[0] != P("verified", 1) {
		t.Fatalf("proposer: %v %v", theory, err)
	}
	if strings.Contains(fake.calls[0], "verified") {
		t.Error("variable names leaked to the proposer")
	}
	obs, err := LLMFalsifier{Anonymize: true, Call: fake.call}.Critique(theory, w.Data, l)
	if err != nil || len(obs) != 1 || obs[0].Var != "community" {
		t.Fatalf("falsifier: %v %v", obs, err)
	}
	if strings.Contains(fake.calls[1], "community") || strings.Contains(fake.calls[1], "verified") {
		t.Error("variable names leaked to the falsifier")
	}
	if got := Revise(theory, obs, l); len(got) != 1 || got[0] != P("community", 1) {
		t.Errorf("revise: %v", got)
	}
}

func TestLLMObjectionNamingNoVariableIsDropped(t *testing.T) {
	w := mustWorld(t, "main", 0, 2000)
	fake := &fakeClaude{replies: []string{`[{"index": 0, "kind": "confound", "var": "market_mood"}]`}}
	obs, err := LLMFalsifier{Call: fake.call}.Critique([]Proposition{P("verified", 1)}, w.Data, &Ledger{})
	if err != nil || len(obs) != 0 {
		t.Fatalf("got %v, %v", obs, err)
	}
}

func TestFileCallerWritesPromptThenReadsReply(t *testing.T) {
	dir := t.TempDir()
	fc := &FileCaller{Dir: dir}
	if _, err := fc.Call("sys", "usr", "m", 100); !errors.Is(err, ErrPending) {
		t.Fatalf("want ErrPending, got %v", err)
	}
	id := PromptID("sys", "usr", "m")
	if b, err := os.ReadFile(filepath.Join(dir, "pending", id+".txt")); err != nil || !strings.Contains(string(b), "usr") {
		t.Fatalf("pending prompt not written: %v", err)
	}
	os.MkdirAll(filepath.Join(dir, "replies"), 0o755)
	os.WriteFile(filepath.Join(dir, "replies", id+".txt"), []byte(`[{"predictor":"verified","sign":1}]`), 0o644)
	got, err := fc.Call("sys", "usr", "m", 100)
	if err != nil || !strings.Contains(got, "verified") {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestEmptyProposalIsAValidTheory(t *testing.T) {
	w := mustWorld(t, "null", 0, 2000)
	fake := &fakeClaude{replies: []string{"[]"}}
	theory, err := LLMProposer{Call: fake.call}.Propose(w.Data, &Ledger{})
	if err != nil || len(theory) != 0 {
		t.Fatalf("got %v, %v", theory, err)
	}
	// the falsifier must not be called (or fail) on an empty theory
	obs, err := LLMFalsifier{Call: fake.call}.Critique(nil, w.Data, &Ledger{})
	if err != nil || obs != nil || len(fake.calls) != 1 {
		t.Fatalf("falsifier on empty theory: %v %v, calls=%d", obs, err, len(fake.calls))
	}
}

func TestPromptVersionsDiffer(t *testing.T) {
	w := mustWorld(t, "null", 0, 2000)
	fake := &fakeClaude{replies: []string{"[]", "[]"}}
	var systems []string
	rec := func(system, user, model string, n int) (string, error) {
		systems = append(systems, system)
		return fake.call(system, user, model, n)
	}
	LLMProposer{Call: rec}.Propose(w.Data, &Ledger{})
	LLMProposer{Call: rec, PromptV1: true}.Propose(w.Data, &Ledger{})
	if systems[0] == systems[1] || !strings.Contains(systems[0], "empty array") || strings.Contains(systems[1], "empty array") {
		t.Fatal("v1 and v2 prompts should differ in the empty-answer sentence only")
	}
}
