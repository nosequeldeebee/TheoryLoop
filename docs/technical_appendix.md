# Technical appendix

## A1. Design logic

The benchmark answers one question: **when an AI loop builds theory from data,
how often does the theory it converges on match the mechanism that actually
generated the data?** That question cannot be answered on real data, where the
true mechanism is unknown. It can be answered on synthetic data where we planted
the mechanism. The benchmark therefore separates two things that are usually
fused in AI-assisted research: the *procedure* that produces a theory, and the
*validity* of what it produces.

## A2. Worlds

Each world is a panel of n = 2,000 NFT projects with ten predictors named after
platform-legitimacy constructs (marketplace verification, creator reputation,
community size at launch, royalty rate, wash-trading share, bear-market launch)
and four distractors with no effect in any world (roadmap length, visual
complexity, mint hour, log supply). The outcome is a standardized 12-month
legitimacy score with unit-variance noise.

| world | generating equation | truth slots |
|---|---|---|
| main | y = 0.30·verified + e | {verified +} |
| scope | y = 0.45·verified·bear + e | {verified + \| bear=1, bear + \| verified=1} |
| confound | P(verified) = logit⁻¹(1.6·community − 0.4); y = 0.40·community + e | {community +} |
| null | y = e | none |
| multi | y = −0.55·1[wash > 0.30] + 0.20·creator_rep + e | {wash −}, {creator_rep +} |

**Truth slots and observational equivalence.** A pure interaction can be stated
with either variable as focal: "verification matters in bear markets" and
"bear-market launches matter for verified projects" fit the data identically.
Which one is the theory is a framing decision the data cannot make. The scorer
treats such statements as one slot and accepts either. This emerged from the
first benchmark run rather than from the initial design (A6).

## A3. Theory as code

A proposition is a triple (predictor, sign, optional scope condition on a
binary variable). It is executable: `test_proposition` subsets to the scope,
standardizes the predictor, fits OLS, and returns whether the coefficient has
the claimed sign at p < .05. A scoped proposition makes two claims (the effect
holds inside the scope, and differs outside it), and the holdout gate tests both
(interaction term p < .05). A scope that fails its own test is stripped and the
bare claim must then hold on its own.

## A4. Agents

All roles receive only their inputs; no role sees another role's reasoning.

- **Proposer.** Sees univariate regressions of the outcome on each predictor in
  the discovery half. Reference version: proposes the three most significant at
  p < .05. LLM version: receives the same table as text and returns up to three
  propositions as JSON.
- **Falsifier.** For each proposition, code computes a diagnostic battery: the
  base test; the effect after adding each other predictor as a control (with the
  control's own p-value and sign); and, for unscoped claims, interaction tests
  with each binary moderator and the effect at each level. Reference version
  rules: *fails* if the base test fails; *confound* if a control with p < .01
  cuts the effect by more than half and to non-significance; *scope* if the
  interaction p < .01 and the claim holds at exactly one level. LLM version:
  receives the same battery and decides which objections to raise.
- **Reviser.** Applies objections mechanically: drop failed claims, replace a
  confounded claim with the confounder, add a scope condition.

The loop runs proposer, then up to six falsifier–reviser rounds, stopping when
the falsifier raises nothing.

**Division of labor in the LLM backend.** Every number comes from code. The
model chooses which claims to make and which objections to press, but it cannot
report a statistic that was not computed. This removes one failure mode
(fabricated results) so the benchmark can isolate the one that matters for
theory building: judgment.

**Anonymization contrast.** With `--anonymize`, predictor names are replaced
with x1–x10 in everything the model sees and mapped back on return. Comparing
named and anonymized runs estimates how much of an LLM loop's "theory" comes
from the data and how much from priors about what variables called *verified*
or *community* ought to do. The confound world is built for this: priors
favour verification as a legitimacy signal, and in that world it is a decoy.

## A5. Ledger

Every regression run by any agent is written to a JSONL ledger with the agent,
the specification, coefficient, p-value, and whether it used held-out data.
Ledgers for the first three seeds of every world × condition cell are in
`results/heuristic_ledgers/`.

## A6. Results with reference agents (5 worlds × 50 seeds × 3 conditions)

| world | condition | exact recovery | any false mechanism | tests in ledger | distinct final theories |
|---|---|---|---|---|---|
| main | single pass | .64 | .36 | 10 | 15 |
| | loop | .64 | .36 | 30 | 15 |
| | loop + holdout | **1.00** | **.00** | 31 | 1 |
| scope | single pass | .00 | .34 | 10 | 20 |
| | loop | .54 | .34 | 55 | 24 |
| | loop + holdout | **.74** | **.00** | 59 | 9 |
| confound | single pass | .00 | 1.00 | 10 | 10 |
| | loop | .68 | .32 | 62 | 12 |
| | loop + holdout | **.90** | **.10** | 63 | 3 |
| null | single pass | .54 | .46 | 10 | 18 |
| | loop | .54 | .46 | 19 | 18 |
| | loop + holdout | **1.00** | **.00** | 19 | 1 |
| multi | single pass | .64 | .36 | 10 | 15 |
| | loop | .62 | .36 | 48 | 16 |
| | loop + holdout | **.98** | **.00** | 50 | 3 |

Four patterns, each a hypothesis for the LLM runs:

1. **Critique fixes structure, not luck.** The adversarial loop repairs
   confounds (single-pass false-mechanism rate 1.00 → .32 in the confound world)
   and recovers scope (exact recovery .00 → .54). It does nothing for chance
   findings: the null world's false-discovery rate is .46 before and after
   critique. A falsifier examining the same data that produced a lucky
   correlation sees the same lucky correlation. Only new cases expose it.
2. **Loops multiply forking paths.** The loop runs two to six times as many
   tests as a single pass. Without a gate this is a garden of forking paths
   with a rigorous-looking process on top.
3. **Variance is a validity signal.** Across seeds, ungated procedures produce
   10–24 different final theories per world; the gated loop converges to 1–3
   (9 in the scope world, mostly equivalent framings). Instability across runs
   is diagnostic of a procedure fitting noise.
4. **Scope has a power cost.** The gate lowers scope recall (.82 → .76) because
   it demands that boundary conditions replicate too. Some true boundary
   conditions are lost in exchange for eliminating false ones.

## A7. Known limitations of the current prototype

- Mechanisms are linear-additive or single interactions; real organizational
  mechanisms involve feedback, timing, and multi-level structure.
- The proposition language cannot express mediation, temporal ordering, or
  continuous moderators.
- The scope condition is the only theoretical move beyond main effects; theory
  building also involves construct redefinition, which this benchmark does not
  test.
- Reference agents are rules. LLM results are not yet in.
- The holdout gate is a single split; the gated rate of false mechanisms is
  bounded by alpha per surviving claim, not by a family-wise correction across
  the ledger.

## A8. Reproducibility

All randomness is seeded (`make_world(name, seed)`; discovery/holdout split seed
+ 10,000). The reference benchmark reproduces bit-for-bit in about 45 seconds on
a laptop. The LLM backend uses a publicly available model through a standard
API; model name is a command-line argument so runs can be repeated on open-weight
models served behind a compatible endpoint.
