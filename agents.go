package theoryloop

import (
	"fmt"
	"math"
	"sort"
)

// Subagent roles. Each runs with a fresh context: it sees only its inputs,
// never another role's reasoning.
//
//	Proposer   looks at discovery data and proposes propositions.
//	Falsifier  tries to break the theory: does each claim fail, is it
//	           confounded by another variable, does it vanish in a subgroup?
//	           (Reviewer 2 as a process.)
//	Revise     turns objections into revisions: drop, replace with the
//	           confounder, or add a scope condition.
//
// Two interchangeable backends implement Proposer and Falsifier:
//
//	heuristic  deterministic reference agents. They are not the object of
//	           study; they validate the harness and set a floor.
//	llm        each role is a separate Claude call over a table of evidence
//	           computed by code. The model supplies judgment; every number
//	           comes from code, so it cannot report a result that was not run.

// Proposer drafts a theory from data.
type Proposer interface {
	Propose(d Data, l *Ledger) ([]Proposition, error)
}

// Falsifier raises objections to a theory.
type Falsifier interface {
	Critique(theory []Proposition, d Data, l *Ledger) ([]Objection, error)
}

// Objection is one falsifier finding.
type Objection struct {
	Prop  Proposition
	Kind  string // "fails", "confound" or "scope"
	Var   string // the confounder or moderator
	Sign  int    // sign of the confounder's effect
	Value int    // level of the moderator at which the effect holds
}

func (o Objection) String() string {
	s := o.Kind + ": " + o.Prop.Label()
	if o.Var != "" {
		s += " -> " + o.Var
		if o.Kind == "scope" {
			s += fmt.Sprintf("=%d", o.Value)
		}
	}
	return s
}

func objectionStrings(obs []Objection) []string {
	out := make([]string, len(obs))
	for i, o := range obs {
		out[i] = o.String()
	}
	return out
}

// ---------------------------------------------------------------- evidence

// UniRow is one line of the proposer's evidence table.
type UniRow struct {
	Predictor string
	Coef, P   float64
}

func univariateTable(d Data, l *Ledger, agent string) []UniRow {
	rows := make([]UniRow, 0, len(Predictors))
	for _, x := range Predictors {
		r := Evaluate(P(x, 1), d, nil)
		l.Test(agent, Outcome+" ~ "+x, r.Coef, r.P, "")
		rows = append(rows, UniRow{x, r.Coef, r.P})
	}
	return rows
}

// ControlDiag is what happens to a claim when one other variable is controlled.
type ControlDiag struct {
	Control         string
	CoefWith, PWith float64 // the claim's effect with the control in
	ControlP        float64 // the control's own p-value
	ControlSign     int
}

// ScopeDiag is how a claim behaves across the levels of a binary moderator.
type ScopeDiag struct {
	Moderator      string
	InteractionP   float64
	Holds0, Holds1 bool
	Coef0, Coef1   float64
}

// Diagnostics is the battery of attacks a falsifier can run on one claim.
type Diagnostics struct {
	Base     Evaluation
	Controls []ControlDiag
	Scopes   []ScopeDiag
}

func diagnose(p Proposition, d Data, l *Ledger, agent string) Diagnostics {
	var dg Diagnostics
	dg.Base = Evaluate(p, d, nil)
	l.Test(agent, p.Label(), dg.Base.Coef, dg.Base.P, "")
	sub := d.Where(p.Cond)
	y := sub.Col(Outcome)
	for _, z := range Predictors {
		if z == p.Predictor || z == p.Cond.Var {
			continue
		}
		r := Evaluate(p, d, []string{z})
		l.Test(agent, fmt.Sprintf("%s | control %s", p.Label(), z), r.Coef, r.P, "")
		cd := ControlDiag{Control: z, CoefWith: r.Coef, PWith: r.P, ControlP: 1}
		if sdOf(sub.Col(z)) > 0 {
			if c, pv, ok := ols(y, sub.Col(p.Predictor), sub.Col(z)); ok {
				cd.ControlP, cd.ControlSign = pv[1], sign(c[1])
			}
		}
		dg.Controls = append(dg.Controls, cd)

		if binary[z] && !p.Scoped() {
			x := standardize(sub.Col(p.Predictor))
			c, pv, _ := ols(y, x, sub.Col(z), mul(x, sub.Col(z)))
			l.Test(agent, fmt.Sprintf("%s x %s", p.Label(), z), c[2], pv[2], "")
			sd := ScopeDiag{Moderator: z, InteractionP: pv[2]}
			for _, v := range []int{0, 1} {
				q := PWhen(p.Predictor, p.Sign, z, v)
				rv := Evaluate(q, d, nil)
				l.Test(agent, q.Label(), rv.Coef, rv.P, "")
				if v == 0 {
					sd.Holds0, sd.Coef0 = rv.Holds, rv.Coef
				} else {
					sd.Holds1, sd.Coef1 = rv.Holds, rv.Coef
				}
			}
			dg.Scopes = append(dg.Scopes, sd)
		}
	}
	return dg
}

// -------------------------------------------------------- heuristic agents

// HeuristicProposer proposes the K most significant predictors at p < Alpha.
type HeuristicProposer struct {
	K     int
	Alpha float64
}

func (h HeuristicProposer) Propose(d Data, l *Ledger) ([]Proposition, error) {
	rows := univariateTable(d, l, "proposer")
	var sig []UniRow
	for _, r := range rows {
		if r.P < h.Alpha {
			sig = append(sig, r)
		}
	}
	sort.SliceStable(sig, func(i, j int) bool { return sig[i].P < sig[j].P })
	if len(sig) > h.K {
		sig = sig[:h.K]
	}
	var props []Proposition
	for _, r := range sig {
		props = append(props, P(r.Predictor, sign(r.Coef)))
	}
	l.Log(Entry{Agent: "proposer", Action: "propose", Theory: strs(labels(props))})
	return props, nil
}

// HeuristicFalsifier applies fixed rules to the diagnostic battery:
//
//	fails     the base test fails
//	confound  a control with p < .01 cuts the effect by more than half and to
//	          non-significance (the strongest such control is named)
//	scope     the interaction has p < .01 and the claim holds at exactly one level
type HeuristicFalsifier struct{}

func (HeuristicFalsifier) Critique(theory []Proposition, d Data, l *Ledger) ([]Objection, error) {
	var obs []Objection
	for _, p := range theory {
		dg := diagnose(p, d, l, "falsifier")
		if !dg.Base.Holds {
			obs = append(obs, Objection{Prop: p, Kind: "fails"})
			continue
		}
		var worst *ControlDiag
		for i := range dg.Controls {
			c := &dg.Controls[i]
			collapsed := c.PWith > 0.05 && math.Abs(c.CoefWith) < 0.5*math.Abs(dg.Base.Coef)
			if collapsed && c.ControlP < 0.01 &&
				(worst == nil || math.Abs(c.CoefWith) < math.Abs(worst.CoefWith)) {
				worst = c
			}
		}
		if worst != nil {
			obs = append(obs, Objection{Prop: p, Kind: "confound", Var: worst.Control, Sign: worst.ControlSign})
			continue
		}
		for _, s := range dg.Scopes {
			if s.InteractionP < 0.01 && s.Holds0 != s.Holds1 {
				val := 0
				if s.Holds1 {
					val = 1
				}
				obs = append(obs, Objection{Prop: p, Kind: "scope", Var: s.Moderator, Value: val})
				break
			}
		}
	}
	l.Log(Entry{Agent: "falsifier", Action: "critique", Objections: strs(objectionStrings(obs))})
	return obs, nil
}

// --------------------------------------------------------------- reviser

// Revise applies objections to a theory. It is shared by both backends:
// applying an objection is mechanical.
func Revise(theory []Proposition, obs []Objection, l *Ledger) []Proposition {
	cur := append([]Proposition(nil), theory...)
	for _, o := range obs {
		i := indexOf(cur, o.Prop)
		if i < 0 {
			continue
		}
		switch o.Kind {
		case "fails":
			cur = append(cur[:i:i], cur[i+1:]...)
		case "confound":
			s := o.Sign
			if s == 0 {
				s = 1
			}
			repl := P(o.Var, s)
			rest := append(cur[:i:i], cur[i+1:]...)
			if indexOf(rest, repl) < 0 {
				next := append([]Proposition(nil), rest[:i]...)
				next = append(next, repl)
				rest = append(next, rest[i:]...)
			}
			cur = rest
		case "scope":
			cur[i] = PWhen(o.Prop.Predictor, o.Prop.Sign, o.Var, o.Value)
		}
	}
	cur = dedupe(cur)
	l.Log(Entry{Agent: "reviser", Action: "revise", Theory: strs(labels(cur))})
	return cur
}
