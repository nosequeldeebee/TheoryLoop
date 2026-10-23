package theoryloop

import (
	"encoding/json"
	"math"
	"os"
)

// The ledger records every test any agent runs.
//
// Loops are forking-path (Gelman & Loken, 2013) machines. A loop that tries 200 specifications and
// reports the 3 that worked looks rigorous and is not. The ledger makes the
// garden of forking paths visible, so the number of tests behind each claim
// can be reported and corrected for.
//
// Gelman, A., & Loken, E. (2013). The garden of forking paths: Why multiple comparisons can be a problem, even when there is no “fishing expedition” or “p-hacking” and the research hypothesis was posited ahead of time. Department of Statistics, Columbia University, 348(1-17), 3.

// Entry is one ledger line.
type Entry struct {
	Step       int       `json:"step"`
	Agent      string    `json:"agent"`
	Action     string    `json:"action"`
	Spec       string    `json:"spec,omitempty"`
	Coef       *float64  `json:"coef,omitempty"`
	P          *float64  `json:"p,omitempty"`
	Set        string    `json:"set,omitempty"` // "holdout" for tests on held-out cases
	Theory     *[]string `json:"theory,omitempty"`
	Objections *[]string `json:"objections,omitempty"`
	Raw        string    `json:"raw,omitempty"` // a model's verbatim reply
}

// Ledger is an append-only log for one run.
type Ledger struct{ Entries []Entry }

// Log appends an entry, numbering it.
func (l *Ledger) Log(e Entry) {
	e.Step = len(l.Entries)
	l.Entries = append(l.Entries, e)
}

// Test records a statistical test with its coefficient and p-value.
func (l *Ledger) Test(agent, spec string, coef, p float64, set string) {
	l.Log(Entry{Agent: agent, Action: "test", Spec: spec, Coef: num(coef), P: num(p), Set: set})
}

// TestP records a test whose result is only a p-value.
func (l *Ledger) TestP(agent, spec string, p float64, set string) {
	l.Log(Entry{Agent: agent, Action: "test", Spec: spec, P: num(p), Set: set})
}

// NTests counts the statistical tests behind a run.
func (l *Ledger) NTests() int {
	n := 0
	for _, e := range l.Entries {
		if e.Action == "test" {
			n++
		}
	}
	return n
}

// Dump writes the ledger as JSON Lines.
func (l *Ledger) Dump(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, e := range l.Entries {
		if err := enc.Encode(e); err != nil {
			f.Close()
			return err
		}
	}
	return f.Close()
}

func num(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &v
}

func strs(s []string) *[]string {
	if s == nil {
		s = []string{}
	}
	return &s
}
