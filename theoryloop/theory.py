"""Theory as code.

A theory is a list of propositions. Each proposition is an executable claim:

    "Among cases satisfying CONDITION, higher PREDICTOR goes with
     higher (sign=+1) or lower (sign=-1) OUTCOME."

Because a proposition is executable, it can be run against any dataset --
including cases the theory-builder never saw. That is what lets held-out
cases play the role of a test suite, the way unit tests do in a coding loop.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

import numpy as np
import pandas as pd
from scipy import stats


@dataclass(frozen=True)
class Proposition:
    predictor: str
    sign: int                                   # +1 or -1
    condition: Optional[tuple[str, int]] = None  # (binary var, value) or None

    def label(self) -> str:
        arrow = "raises" if self.sign > 0 else "lowers"
        s = f"{self.predictor} {arrow} legitimacy"
        if self.condition:
            s += f" when {self.condition[0]}={self.condition[1]}"
        return s

    def to_dict(self) -> dict:
        return {"predictor": self.predictor, "sign": self.sign,
                "condition": list(self.condition) if self.condition else None}

    @staticmethod
    def from_dict(d: dict) -> "Proposition":
        c = d.get("condition")
        return Proposition(d["predictor"], int(d["sign"]), tuple(c) if c else None)


def subset(df: pd.DataFrame, cond) -> pd.DataFrame:
    if cond is None:
        return df
    var, val = cond
    return df[df[var] == val]


def ols(df: pd.DataFrame, y: str, xs: list[str]):
    """OLS with intercept. Returns (coefs, pvalues) dicts keyed by x."""
    X = np.column_stack([np.ones(len(df))] + [df[x].to_numpy(float) for x in xs])
    Y = df[y].to_numpy(float)
    beta, *_ = np.linalg.lstsq(X, Y, rcond=None)
    resid = Y - X @ beta
    dof = max(len(Y) - X.shape[1], 1)
    sigma2 = resid @ resid / dof
    try:
        cov = sigma2 * np.linalg.inv(X.T @ X)
    except np.linalg.LinAlgError:
        cov = sigma2 * np.linalg.pinv(X.T @ X)
    se = np.sqrt(np.clip(np.diag(cov), 1e-300, None))
    t = beta / se
    p = 2 * stats.t.sf(np.abs(t), dof)
    return dict(zip(xs, beta[1:])), dict(zip(xs, p[1:]))


def test_proposition(prop: Proposition, df: pd.DataFrame, y: str,
                     controls: list[str] | None = None) -> dict:
    """Run one proposition against data. Effect is per-SD of the predictor."""
    sub = subset(df, prop.condition)
    xs = [prop.predictor] + [c for c in (controls or []) if c != prop.predictor
                             and (not prop.condition or c != prop.condition[0])]
    sub = sub.copy()
    sd = sub[prop.predictor].std()
    if not sd or len(sub) < 30:
        return {"coef": 0.0, "p": 1.0, "n": len(sub), "holds": False}
    sub[prop.predictor] = sub[prop.predictor] / sd
    coef, p = ols(sub, y, xs)
    c, pv = coef[prop.predictor], p[prop.predictor]
    return {"coef": float(c), "p": float(pv), "n": int(len(sub)),
            "holds": bool(np.sign(c) == prop.sign and pv < 0.05)}
