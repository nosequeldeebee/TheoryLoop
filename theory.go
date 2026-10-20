package theoryloop

import "fmt"

// Theory as code. Each proposition is
// an executable claim:
//
//	"Among cases satisfying Cond, higher Predictor goes with higher (Sign=+1)
//	 or lower (Sign=-1) legitimacy."
//
// Because a proposition is executable, it can be run against any dataset,
// including cases the theory-builder never saw. That is what lets held-out
// cases play the role of a test suite, the way unit tests do in a coding loop.

// Cond is a scope condition on a binary variable. The zero value means "everywhere".
type Cond struct {
	Var string
	Val int
}

// Proposition is one executable claim. It is comparable with ==.
type Proposition struct {
	Predictor string
	Sign      int // +1 or -1
	Cond      Cond
}

// P is an unscoped proposition.
func P(predictor string, sign int) Proposition {
	return Proposition{Predictor: predictor, Sign: sign}
}

// PWhen is a proposition that holds only where v == val.
func PWhen(predictor string, sign int, v string, val int) Proposition {
	return Proposition{Predictor: predictor, Sign: sign, Cond: Cond{v, val}}
}

// Scoped reports whether the proposition carries a scope condition.
func (p Proposition) Scoped() bool { return p.Cond.Var != "" }

// Label renders the proposition in words.
func (p Proposition) Label() string { return p.LabelWith(func(s string) string { return s }) }

// LabelWith renders the proposition with variable names passed through name
// (used to anonymize variables before they reach a model).
func (p Proposition) LabelWith(name func(string) string) string {
	verb := "raises"
	if p.Sign < 0 {
		verb = "lowers"
	}
	s := fmt.Sprintf("%s %s %s", name(p.Predictor), verb, Outcome)
	if p.Scoped() {
		s += fmt.Sprintf(" when %s=%d", name(p.Cond.Var), p.Cond.Val)
	}
	return s
}

// Evaluation is the result of running one proposition against data.
type Evaluation struct {
	Coef  float64 // effect per SD of the predictor
	P     float64
	N     int
	Holds bool // right sign and p < .05
}

// Evaluate runs a proposition against data, optionally with controls.
func Evaluate(p Proposition, d Data, controls []string) Evaluation {
	sub := d.Where(p.Cond)
	x := sub.Col(p.Predictor)
	s := sdOf(x)
	if sub.N < 30 || s == 0 {
		return Evaluation{P: 1, N: sub.N}
	}
	xs := [][]float64{scale(x, s)}
	for _, c := range controls {
		if c == p.Predictor || c == p.Cond.Var {
			continue
		}
		xs = append(xs, sub.Col(c))
	}
	coef, pv, ok := ols(sub.Col(Outcome), xs...)
	if !ok {
		return Evaluation{P: 1, N: sub.N}
	}
	return Evaluation{Coef: coef[0], P: pv[0], N: sub.N,
		Holds: sign(coef[0]) == p.Sign && pv[0] < 0.05}
}

// ---------------------------------------------------------------- helpers

func labels(ps []Proposition) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Label()
	}
	return out
}

func indexOf(ps []Proposition, p Proposition) int {
	for i, q := range ps {
		if q == p {
			return i
		}
	}
	return -1
}

func dedupe(ps []Proposition) []Proposition {
	var out []Proposition
	for _, p := range ps {
		if indexOf(out, p) < 0 {
			out = append(out, p)
		}
	}
	return out
}
