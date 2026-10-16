"""Subagent roles.

The loop has three roles, each run with a fresh context (no memory of the
others' reasoning, only their outputs):

  Proposer  -- looks at discovery data and proposes propositions.
  Falsifier -- receives a theory and tries to break it: does each claim fail,
               is it confounded by another variable, does it vanish in a
               subgroup? (Reviewer 2 as a process.)
  Reviser   -- turns objections into revisions: drop, replace with the
               confounder, or add a scope condition.

Two interchangeable backends:

  heuristic -- deterministic reference agents. They are not the object of
               study; they validate the benchmark harness and give a floor.
  llm       -- each role is a separate Claude call that sees a table of
               computed evidence and returns JSON. The LLM supplies judgment;
               all numbers come from code, so the model cannot invent results.
"""
from __future__ import annotations

import json
import os
import urllib.request

import numpy as np
import pandas as pd

from .ledger import Ledger
from .theory import Proposition, test_proposition, subset, ols
from .worlds import PREDICTORS, BINARY, OUTCOME


# ---------------------------------------------------------------- evidence
def univariate_table(df: pd.DataFrame, ledger: Ledger, agent: str) -> list[dict]:
    rows = []
    for x in PREDICTORS:
        r = test_proposition(Proposition(x, +1), df, OUTCOME)
        ledger.log(agent, "test", spec=f"{OUTCOME} ~ {x}", coef=r["coef"], p=r["p"])
        rows.append({"predictor": x, "coef_per_sd": round(r["coef"], 3), "p": r["p"]})
    return rows


def diagnostics(prop: Proposition, df: pd.DataFrame, ledger: Ledger, agent: str) -> dict:
    """The battery of attacks a falsifier can run on one proposition."""
    base = test_proposition(prop, df, OUTCOME)
    ledger.log(agent, "test", spec=prop.label(), coef=base["coef"], p=base["p"])
    controls, scopes = [], []
    cond_var = prop.condition[0] if prop.condition else None
    for z in PREDICTORS:
        if z in (prop.predictor, cond_var):
            continue
        r = test_proposition(prop, df, OUTCOME, controls=[z])
        sub = subset(df, prop.condition)
        coef_z, pz = ols(sub, OUTCOME, [prop.predictor, z]) if sub[z].std() > 0 else ({z: 0.0}, {z: 1.0})
        ledger.log(agent, "test", spec=f"{prop.label()} | control {z}", coef=r["coef"], p=r["p"])
        controls.append({"control": z, "coef_with_control": r["coef"], "p_with_control": r["p"],
                         "control_p": pz.get(z, 1.0), "control_sign": int(np.sign(coef_z.get(z, 0.0)))})
        if z in BINARY and cond_var is None:
            s = subset(df, prop.condition).copy()
            s["_ix"] = s[prop.predictor] / s[prop.predictor].std() * s[z]
            s["_x"] = s[prop.predictor] / s[prop.predictor].std()
            c, p = ols(s, OUTCOME, ["_x", z, "_ix"])
            ledger.log(agent, "test", spec=f"{prop.label()} x {z}", coef=c["_ix"], p=p["_ix"])
            by_level = {}
            for v in (0, 1):
                rv = test_proposition(Proposition(prop.predictor, prop.sign, (z, v)), df, OUTCOME)
                ledger.log(agent, "test", spec=f"{prop.predictor} when {z}={v}", coef=rv["coef"], p=rv["p"])
                by_level[v] = rv
            scopes.append({"moderator": z, "interaction_p": float(p["_ix"]),
                           "holds_when_0": by_level[0]["holds"], "holds_when_1": by_level[1]["holds"],
                           "coef_when_0": by_level[0]["coef"], "coef_when_1": by_level[1]["coef"]})
    return {"base": base, "controls": controls, "scopes": scopes}


# ---------------------------------------------------------- heuristic agents
class HeuristicProposer:
    def __init__(self, k: int = 3, alpha: float = 0.05):
        self.k, self.alpha = k, alpha

    def propose(self, df, ledger):
        rows = univariate_table(df, ledger, "proposer")
        rows = sorted([r for r in rows if r["p"] < self.alpha], key=lambda r: r["p"])[: self.k]
        props = [Proposition(r["predictor"], int(np.sign(r["coef_per_sd"]))) for r in rows]
        ledger.log("proposer", "propose", theory=[p.label() for p in props])
        return props


class HeuristicFalsifier:
    def critique(self, theory, df, ledger):
        objections = []
        for prop in theory:
            d = diagnostics(prop, df, ledger, "falsifier")
            if not d["base"]["holds"]:
                objections.append({"prop": prop, "kind": "fails"})
                continue
            # confound: effect collapses when a significant control enters
            worst = None
            for c in d["controls"]:
                collapsed = c["p_with_control"] > 0.05 and abs(c["coef_with_control"]) < 0.5 * abs(d["base"]["coef"])
                if collapsed and c["control_p"] < 0.01:
                    if worst is None or abs(c["coef_with_control"]) < abs(worst["coef_with_control"]):
                        worst = c
            if worst:
                objections.append({"prop": prop, "kind": "confound", "var": worst["control"],
                                   "sign": worst["control_sign"]})
                continue
            # scope: effect lives in only one level of a binary moderator
            for s in d["scopes"]:
                if s["interaction_p"] < 0.01 and s["holds_when_0"] != s["holds_when_1"]:
                    objections.append({"prop": prop, "kind": "scope", "var": s["moderator"],
                                       "value": 1 if s["holds_when_1"] else 0})
                    break
        ledger.log("falsifier", "critique", objections=[_obj_str(o) for o in objections])
        return objections


class Reviser:
    """Shared by both backends: applying an objection is mechanical."""
    def revise(self, theory, objections, ledger):
        new = list(theory)
        for o in objections:
            p = o["prop"]
            if p not in new:
                continue
            i = new.index(p)
            if o["kind"] == "fails":
                new.pop(i)
            elif o["kind"] == "confound":
                repl = Proposition(o["var"], o.get("sign", 1) or 1)
                new.pop(i)
                if repl not in new:
                    new.insert(i, repl)
            elif o["kind"] == "scope":
                new[i] = Proposition(p.predictor, p.sign, (o["var"], o["value"]))
        new = list(dict.fromkeys(new))
        ledger.log("reviser", "revise", theory=[p.label() for p in new])
        return new


def _obj_str(o):
    s = f"{o['kind']}: {o['prop'].label()}"
    if "var" in o:
        s += f" -> {o['var']}" + (f"={o['value']}" if "value" in o else "")
    return s


# --------------------------------------------------------------- LLM agents
API_URL = "https://api.anthropic.com/v1/messages"


def _anon_map(anonymize: bool):
    return {x: (f"x{i+1}" if anonymize else x) for i, x in enumerate(PREDICTORS)}


def call_claude(system: str, user: str, model: str, max_tokens: int = 1500) -> str:
    key = os.environ.get("ANTHROPIC_API_KEY")
    if not key:
        raise RuntimeError("Set ANTHROPIC_API_KEY to use the llm backend.")
    body = json.dumps({"model": model, "max_tokens": max_tokens, "system": system,
                       "messages": [{"role": "user", "content": user}]}).encode()
    req = urllib.request.Request(API_URL, data=body, headers={
        "x-api-key": key, "anthropic-version": "2023-06-01", "content-type": "application/json"})
    with urllib.request.urlopen(req, timeout=120) as r:
        out = json.load(r)
    return "".join(b.get("text", "") for b in out["content"])


def parse_json(text: str):
    """Pull the last JSON object/array out of a model reply."""
    for open_, close in (("[", "]"), ("{", "}")):
        start, end = text.find(open_), text.rfind(close)
        if start != -1 and end > start:
            try:
                return json.loads(text[start:end + 1])
            except json.JSONDecodeError:
                continue
    raise ValueError(f"no JSON found in: {text[:200]}")


class LLMProposer:
    SYSTEM = ("You are an organizational theorist building theory from data on NFT projects. "
              "The outcome is a 12-month legitimacy score. You see univariate regression "
              "results computed by code. Propose up to {k} propositions that you believe reflect "
              "real mechanisms. Reply with a JSON array of objects "
              '{{"predictor": name, "sign": 1 or -1}} and nothing after it.')

    def __init__(self, model="claude-sonnet-5-5", k=3, anonymize=False, caller=call_claude):
        self.model, self.k, self.anonymize, self.caller = model, k, anonymize, caller

    def propose(self, df, ledger):
        m = _anon_map(self.anonymize); inv = {v: k for k, v in m.items()}
        rows = univariate_table(df, ledger, "proposer")
        table = "\n".join(f"{m[r['predictor']]}: coef/SD={r['coef_per_sd']}, p={r['p']:.4g}" for r in rows)
        reply = self.caller(self.SYSTEM.format(k=self.k), table, self.model)
        props = [Proposition(inv.get(o["predictor"], o["predictor"]), int(o["sign"]))
                 for o in parse_json(reply) if inv.get(o["predictor"], o["predictor"]) in PREDICTORS]
        ledger.log("proposer", "propose", theory=[p.label() for p in props], raw=reply)
        return props[: self.k]


class LLMFalsifier:
    SYSTEM = ("You are an adversarial reviewer. For each proposition you get diagnostics "
              "computed by code: the base test, the effect after adding each control, and "
              "interaction tests with binary moderators. Raise an objection only when the "
              "evidence supports it. Kinds: 'fails' (claim does not hold), 'confound' "
              "(name the variable that explains it away and its sign), 'scope' (name the "
              "moderator and the level at which the effect holds). Reply with a JSON array of "
              '{"index": i, "kind": ..., "var": ..., "sign": ..., "value": ...}; '
              "an empty array if the theory survives.")

    def __init__(self, model="claude-sonnet-5-5", anonymize=False, caller=call_claude):
        self.model, self.anonymize, self.caller = model, anonymize, caller

    def critique(self, theory, df, ledger):
        m = _anon_map(self.anonymize); inv = {v: k for k, v in m.items()}
        blocks = []
        for i, prop in enumerate(theory):
            d = diagnostics(prop, df, ledger, "falsifier")
            lab = prop.label()
            for k, v in m.items():
                lab = lab.replace(k, v)
            ctrl = "; ".join(f"+{m[c['control']]}: coef={c['coef_with_control']:.3f} p={c['p_with_control']:.3g} "
                             f"(control p={c['control_p']:.3g}, sign={c['control_sign']})" for c in d["controls"])
            scp = "; ".join(f"{m[s['moderator']]}: int p={s['interaction_p']:.3g}, coef@0={s['coef_when_0']:.3f}, "
                            f"coef@1={s['coef_when_1']:.3f}" for s in d["scopes"])
            blocks.append(f"[{i}] {lab}\n base: coef={d['base']['coef']:.3f} p={d['base']['p']:.3g}\n controls: {ctrl}\n moderators: {scp}")
        reply = self.caller(self.SYSTEM, "\n\n".join(blocks), self.model, max_tokens=2000)
        objections = []
        for o in parse_json(reply):
            i = int(o["index"])
            if 0 <= i < len(theory) and o.get("kind") in {"fails", "confound", "scope"}:
                ob = {"prop": theory[i], "kind": o["kind"]}
                if o.get("var"):
                    ob["var"] = inv.get(o["var"], o["var"])
                    if ob["var"] not in PREDICTORS:
                        continue
                if o["kind"] == "confound":
                    ob["sign"] = int(o.get("sign") or 1)
                if o["kind"] == "scope":
                    ob["value"] = int(o.get("value", 1))
                objections.append(ob)
        ledger.log("falsifier", "critique", objections=[_obj_str(o) for o in objections], raw=reply)
        return objections
