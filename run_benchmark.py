"""Run the planted-mechanism benchmark.

    python run_benchmark.py                          # heuristic reference agents, 50 seeds
    python run_benchmark.py --backend llm --seeds 10 # Claude subagents (needs ANTHROPIC_API_KEY)
    python run_benchmark.py --backend llm --anonymize --seeds 10   # variable names hidden

Writes results/<tag>_runs.csv, results/<tag>_summary.md, results/<tag>_ledgers/,
and results/<tag>_chart.png.
"""
from __future__ import annotations

import argparse
import os
import time

import pandas as pd

from theoryloop import make_world, run, WORLD_NAMES, CONDITIONS


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--backend", default="heuristic", choices=["heuristic", "llm"])
    ap.add_argument("--seeds", type=int, default=50)
    ap.add_argument("--worlds", nargs="*", default=WORLD_NAMES)
    ap.add_argument("--model", default="claude-sonnet-5-5")
    ap.add_argument("--anonymize", action="store_true",
                    help="hide variable names from LLM agents (tests reliance on priors)")
    ap.add_argument("--out", default="results")
    a = ap.parse_args()

    tag = a.backend + ("_anon" if a.anonymize else "")
    os.makedirs(f"{a.out}/{tag}_ledgers", exist_ok=True)
    kw = {} if a.backend == "heuristic" else {"model": a.model, "anonymize": a.anonymize}

    rows, t0 = [], time.time()
    for w in a.worlds:
        for s in range(a.seeds):
            world = make_world(w, seed=s)
            for c in CONDITIONS:
                r = run(world, c, seed=s, backend=a.backend, **kw)
                if s < 3:
                    r["ledger"].dump(f"{a.out}/{tag}_ledgers/{w}_{c}_seed{s}.jsonl")
                r["theory"] = " ; ".join(p.label() for p in r["theory"])
                r.pop("ledger")
                rows.append(r)
    df = pd.DataFrame(rows)
    df.to_csv(f"{a.out}/{tag}_runs.csv", index=False)

    summ = (df.groupby(["world", "condition"], sort=False)
              .agg(perfect=("perfect", "mean"), any_false=("any_false", "mean"),
                   recall_exact=("recall_exact", "mean"), wrong_scope=("wrong_scope", "mean"),
                   tests=("n_tests", "mean"), distinct_theories=("theory", "nunique"))
              .round(2))
    overall = df.groupby("condition", sort=False)[["perfect", "any_false"]].mean().round(2)

    with open(f"{a.out}/{tag}_summary.md", "w") as f:
        f.write(f"# Benchmark summary ({tag}, {a.seeds} seeds per world)\n\n")
        f.write("perfect = recovered every planted mechanism with correct scope and nothing else\n")
        f.write("any_false = share of runs asserting at least one mechanism that does not exist\n")
        f.write("tests = mean number of statistical tests in the ledger behind the final theory\n")
        f.write("distinct_theories = how many different final theories across seeds\n\n")
        f.write(summ.to_markdown() + "\n\n## Overall\n\n" + overall.to_markdown() + "\n")
    print(summ.to_string()); print(); print(overall.to_string())
    print(f"\n{len(df)} runs in {time.time()-t0:.1f}s")
    chart(df, f"{a.out}/{tag}_chart.png", tag)


def chart(df, path, tag):
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    colors = {"single_pass": "#9aa5b1", "loop": "#3b6ea8", "loop_holdout": "#1f3a5f"}
    labels = {"single_pass": "Single pass", "loop": "Adversarial loop", "loop_holdout": "Loop + holdout gate"}
    worlds = list(dict.fromkeys(df.world))
    fig, axes = plt.subplots(1, 2, figsize=(11, 4), sharey=True)
    for ax, metric, title in [(axes[0], "perfect", "Recovered the planted theory exactly"),
                              (axes[1], "any_false", "Asserted a mechanism that does not exist")]:
        m = df.groupby(["world", "condition"])[metric].mean()
        w = 0.26
        for i, c in enumerate(CONDITIONS):
            ax.bar([j + (i - 1) * w for j in range(len(worlds))], [m[(x, c)] for x in worlds],
                   width=w, color=colors[c], label=labels[c])
        ax.set_xticks(range(len(worlds)), worlds)
        ax.set_title(title, fontsize=11, loc="left")
        ax.set_ylim(0, 1); ax.spines[["top", "right"]].set_visible(False)
    axes[0].set_ylabel("share of runs")
    h, l = axes[0].get_legend_handles_labels()
    fig.legend(h, l, frameon=False, fontsize=9, ncol=3, loc="lower center")
    fig.suptitle(f"Planted-mechanism benchmark ({tag} agents)", x=0.01, ha="left", fontsize=12)
    fig.tight_layout(rect=(0, 0.07, 1, 1)); fig.savefig(path, dpi=160)


if __name__ == "__main__":
    main()
