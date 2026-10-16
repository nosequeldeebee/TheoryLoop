"""The loop, the conditions we compare, and scoring against planted truth."""
from __future__ import annotations

import numpy as np

from .agents import (HeuristicProposer, HeuristicFalsifier, Reviser,
                     LLMProposer, LLMFalsifier)
from .ledger import Ledger
from .theory import Proposition, test_proposition, ols
from .worlds import World, OUTCOME

CONDITIONS = ["single_pass", "loop", "loop_holdout"]


def split(df, seed, frac=0.5):
    idx = np.random.default_rng(seed + 10_000).permutation(len(df))
    cut = int(len(df) * frac)
    return df.iloc[idx[:cut]], df.iloc[idx[cut:]]


def make_agents(backend: str, **kw):
    if backend == "heuristic":
        return HeuristicProposer(), HeuristicFalsifier(), Reviser()
    if backend == "llm":
        return LLMProposer(**kw), LLMFalsifier(**{k: v for k, v in kw.items() if k != "k"}), Reviser()
    raise ValueError(backend)


def run(world: World, condition: str, seed: int, backend="heuristic",
        max_iter: int = 6, **agent_kw) -> dict:
    discovery, holdout = split(world.data, seed)
    proposer, falsifier, reviser = make_agents(backend, **agent_kw)
    ledger = Ledger()

    theory = proposer.propose(discovery, ledger)
    iters = 0
    if condition in ("loop", "loop_holdout"):
        for iters in range(1, max_iter + 1):
            objections = falsifier.critique(theory, discovery, ledger)
            if not objections:
                break
            theory = reviser.revise(theory, objections, ledger)

    if condition == "loop_holdout":
        theory = holdout_gate(theory, holdout, ledger)

    return {"world": world.name, "condition": condition, "seed": seed, "backend": backend,
            "theory": theory, "iterations": iters, "n_tests": ledger.n_tests,
            "ledger": ledger, **score(theory, world.truth)}


def holdout_gate(theory, holdout, ledger, alpha=0.05):
    """Run the theory's executable claims on cases no agent has seen.

    A plain proposition must hold. A scoped proposition makes two claims --
    the effect holds inside the scope AND differs outside it -- so both are
    tested. A scope that fails its own test is stripped; the unscoped claim
    then has to hold on its own.
    """
    kept = []
    for p in theory:
        r = test_proposition(p, holdout, OUTCOME)
        ledger.log("holdout", "test", spec=p.label(), coef=r["coef"], p=r["p"], set="holdout")
        if not r["holds"]:
            continue
        if p.condition:
            var, _ = p.condition
            s = holdout.copy()
            s["_x"] = s[p.predictor] / s[p.predictor].std()
            s["_ix"] = s["_x"] * s[var]
            _, pv = ols(s, OUTCOME, ["_x", var, "_ix"])
            ledger.log("holdout", "test", spec=f"scope of {p.label()}", p=pv["_ix"], set="holdout")
            if pv["_ix"] >= alpha:
                bare = Proposition(p.predictor, p.sign)
                rb = test_proposition(bare, holdout, OUTCOME)
                ledger.log("holdout", "test", spec=bare.label(), coef=rb["coef"], p=rb["p"], set="holdout")
                if not rb["holds"]:
                    continue
                p = bare
        if p not in kept:
            kept.append(p)
    return kept


def score(theory: list[Proposition], truth: list[frozenset]) -> dict:
    """Score a theory against planted truth slots.

    exact      -- a slot is recovered with the right variable, sign AND scope
    variable   -- a slot's variable and sign appear, scope possibly wrong
    false_props-- propositions about variables/signs that match no mechanism
    wrong_scope-- right variable and sign, wrong or missing scope condition
    perfect    -- every slot recovered exactly and nothing false or mis-scoped
    """
    accept = set().union(*truth) if truth else set()
    accept_vs = {(t.predictor, t.sign) for t in accept}
    exact = sum(1 for slot in truth if any(p in slot for p in theory))
    var_hits = sum(1 for slot in truth
                   if any((p.predictor, p.sign) in {(t.predictor, t.sign) for t in slot} for p in theory))
    false_props = sum(1 for p in theory if (p.predictor, p.sign) not in accept_vs)
    wrong_scope = sum(1 for p in theory if (p.predictor, p.sign) in accept_vs and p not in accept)
    n_t = len(truth)
    return {
        "n_props": len(theory),
        "recall_exact": exact / n_t if n_t else np.nan,
        "recall_variable": var_hits / n_t if n_t else np.nan,
        "false_props": false_props,
        "wrong_scope": wrong_scope,
        "perfect": int(exact == n_t and false_props == 0 and wrong_scope == 0),
        "any_false": int(false_props > 0),
    }
