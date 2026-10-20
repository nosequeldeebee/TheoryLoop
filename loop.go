package theoryloop

import (
	"fmt"
	"math"
)

// Conditions are the procedures the benchmark compares.
//
//	single_pass   proposer only, no critique
//	loop          proposer, then falsifier and reviser until no objections remain
//	loop_holdout  the loop, then every surviving claim must hold on held-out cases
var Conditions = []string{"single_pass", "loop", "loop_holdout"}

// Config selects the agents a run uses.
type Config struct {
	Backend   string // "heuristic" or "llm"
	Model     string // llm only
	Anonymize bool   // llm only: hide variable names from the model
	MaxIter   int    // falsifier-reviser rounds; default 6
	Call      Caller // llm only; nil means ClaudeCaller
}

// Score compares a final theory with the planted truth.
type Score struct {
	NProps         int
	RecallExact    float64 // share of mechanisms recovered with right variable, sign and scope (NaN in the null world)
	RecallVariable float64 // share recovered with right variable and sign, scope possibly wrong
	FalseProps     int     // claims about a variable or sign that matches no mechanism
	WrongScope     int     // right variable and sign, wrong or missing scope
	Perfect        int     // every mechanism recovered exactly, nothing false or mis-scoped
	AnyFalse       int     // at least one false claim
}

// Result is the outcome of one run.
type Result struct {
	World, Condition, Backend string
	Seed                      uint64
	Theory                    []Proposition
	Iterations, NTests        int
	Ledger                    *Ledger
	Score
	Err string // set when the run failed (e.g. a model call)
}

func agentsFor(cfg Config) (Proposer, Falsifier, error) {
	switch cfg.Backend {
	case "", "heuristic":
		return HeuristicProposer{K: 3, Alpha: 0.05}, HeuristicFalsifier{}, nil
	case "llm":
		call := cfg.Call
		if call == nil {
			call = ClaudeCaller
		}
		model := cfg.Model
		if model == "" {
			model = DefaultModel
		}
		return LLMProposer{Model: model, K: 3, Anonymize: cfg.Anonymize, Call: call},
			LLMFalsifier{Model: model, Anonymize: cfg.Anonymize, Call: call}, nil
	}
	return nil, nil, fmt.Errorf("unknown backend %q", cfg.Backend)
}

// Run executes one condition on one world. Discovery and holdout halves are
// fixed by the seed, so every condition sees the same data.
func Run(w World, condition string, seed uint64, cfg Config) (Result, error) {
	res := Result{World: w.Name, Condition: condition, Seed: seed, Backend: cfg.Backend, Ledger: &Ledger{}}
	if res.Backend == "" {
		res.Backend = "heuristic"
	}
	switch condition {
	case "single_pass", "loop", "loop_holdout":
	default:
		return res, fmt.Errorf("unknown condition %q", condition)
	}
	proposer, falsifier, err := agentsFor(cfg)
	if err != nil {
		return res, err
	}
	discovery, holdout := Split(w.Data, seed)
	l := res.Ledger

	theory, err := proposer.Propose(discovery, l)
	if err != nil {
		return res, err
	}
	if condition != "single_pass" {
		maxIter := cfg.MaxIter
		if maxIter == 0 {
			maxIter = 6
		}
		for it := 1; it <= maxIter; it++ {
			res.Iterations = it
			obs, err := falsifier.Critique(theory, discovery, l)
			if err != nil {
				return res, err
			}
			if len(obs) == 0 {
				break
			}
			theory = Revise(theory, obs, l)
		}
	}
	if condition == "loop_holdout" {
		theory = HoldoutGate(theory, holdout, l, 0.05)
	}
	res.Theory, res.NTests = theory, l.NTests()
	res.Score = ScoreTheory(theory, w.Truth)
	return res, nil
}

// HoldoutGate runs a theory's executable claims on cases no agent has seen.
//
// A plain claim must hold. A scoped claim makes two claims, that the effect
// holds inside the scope AND differs outside it, so both are tested. A scope
// that fails its own test is stripped, and the unscoped claim must then hold
// on its own.
func HoldoutGate(theory []Proposition, holdout Data, l *Ledger, alpha float64) []Proposition {
	var kept []Proposition
	for _, p := range theory {
		r := Evaluate(p, holdout, nil)
		l.Test("holdout", p.Label(), r.Coef, r.P, "holdout")
		if !r.Holds {
			continue
		}
		if p.Scoped() {
			x := standardize(holdout.Col(p.Predictor))
			z := holdout.Col(p.Cond.Var)
			pInt := 1.0
			if _, pv, ok := ols(holdout.Col(Outcome), x, z, mul(x, z)); ok {
				pInt = pv[2]
			}
			l.TestP("holdout", "scope of "+p.Label(), pInt, "holdout")
			if pInt >= alpha {
				bare := P(p.Predictor, p.Sign)
				rb := Evaluate(bare, holdout, nil)
				l.Test("holdout", bare.Label(), rb.Coef, rb.P, "holdout")
				if !rb.Holds {
					continue
				}
				p = bare
			}
		}
		if indexOf(kept, p) < 0 {
			kept = append(kept, p)
		}
	}
	return kept
}

type varSign struct {
	v string
	s int
}

// ScoreTheory scores a theory against planted truth slots.
func ScoreTheory(theory []Proposition, truth []Slot) Score {
	accept := map[Proposition]bool{}
	acceptVS := map[varSign]bool{}
	for _, slot := range truth {
		for _, t := range slot {
			accept[t] = true
			acceptVS[varSign{t.Predictor, t.Sign}] = true
		}
	}
	exact, varHits := 0, 0
	for _, slot := range truth {
		slotVS := map[varSign]bool{}
		inSlot := map[Proposition]bool{}
		for _, t := range slot {
			slotVS[varSign{t.Predictor, t.Sign}] = true
			inSlot[t] = true
		}
		hitExact, hitVar := false, false
		for _, p := range theory {
			hitExact = hitExact || inSlot[p]
			hitVar = hitVar || slotVS[varSign{p.Predictor, p.Sign}]
		}
		if hitExact {
			exact++
		}
		if hitVar {
			varHits++
		}
	}
	s := Score{NProps: len(theory), RecallExact: math.NaN(), RecallVariable: math.NaN()}
	for _, p := range theory {
		vs := varSign{p.Predictor, p.Sign}
		if !acceptVS[vs] {
			s.FalseProps++
		} else if !accept[p] {
			s.WrongScope++
		}
	}
	if n := len(truth); n > 0 {
		s.RecallExact = float64(exact) / float64(n)
		s.RecallVariable = float64(varHits) / float64(n)
	}
	if exact == len(truth) && s.FalseProps == 0 && s.WrongScope == 0 {
		s.Perfect = 1
	}
	if s.FalseProps > 0 {
		s.AnyFalse = 1
	}
	return s
}
