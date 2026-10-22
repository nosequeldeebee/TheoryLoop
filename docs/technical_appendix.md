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
first benchmark run rather than from the initial design.

## A3. Theory as code

A proposition is a triple (predictor, sign, optional scope condition on a
binary variable). It is executable: `Evaluate` subsets to the scope,
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
  receives the same battery and decides which objections to raise. An LLM
  objection that names no real variable is discarded rather than applied.
- **Reviser.** Applies objections mechanically: drop failed claims, replace a
  confounded claim with the confounder, add a scope condition.

The loop runs proposer, then up to six falsifier–reviser rounds, stopping when
the falsifier raises nothing.

**Division of labor in the LLM backend.** Every number comes from code. The
model chooses which claims to make and which objections to press, but it cannot
report a statistic that was not computed. This removes one failure mode
(fabricated results) so the benchmark can isolate the one that matters for
theory building: judgment.

**Anonymization contrast.** With `-anonymize`, predictor names are replaced
with x1–x10 in everything the model sees and mapped back on return (a test
checks that no real name reaches the model). Comparing named and anonymized
runs estimates how much of an LLM loop's "theory" comes from the data and how
much from priors about what variables called *verified* or *community* ought to
do. The confound world is built for this: priors favour verification as a
legitimacy signal, and in that world it is a decoy.

## A5. Ledger

Every regression run by any agent is written to a JSON Lines ledger with the
agent, the specification, coefficient, p-value, whether it used held-out data,
and, for LLM agents, the model's verbatim reply. Ledgers for the first three
seeds of every world × condition cell are in `results/heuristic_ledgers/`.

## A6. Results with reference agents (5 worlds × 500 seeds × 3 conditions)

*Modal share* is the fraction of runs that end in the single most common final
theory: a measure of how stable a procedure's answer is across samples from the
same world.

| world | condition | exact recovery | any false mechanism | tests in ledger | modal share |
|---|---|---|---|---|---|
| main | single pass | .62 | .38 | 10 | .62 |
| | loop | .61 | .38 | 31 | .61 |
| | loop + holdout | **.97** | **.02** | 32 | .97 |
| scope | single pass | .00 | .34 | 10 | .48 |
| | loop | .54 | .34 | 56 | .40 |
| | loop + holdout | **.77** | **.01** | 60 | .58 |
| confound | single pass | .00 | 1.00 | 10 | .65 |
| | loop | .62 | .38 | 65 | .62 |
| | loop + holdout | **.93** | **.07** | 66 | .93 |
| null | single pass | .58 | .42 | 10 | .58 |
| | loop | .58 | .42 | 18 | .58 |
| | loop + holdout | **.98** | **.02** | 19 | .98 |
| multi | single pass | .65 | .35 | 10 | .65 |
| | loop | .63 | .35 | 48 | .63 |
| | loop + holdout | **.99** | **.01** | 51 | .99 |

**Harness check.** In the null world a single pass tests ten noise variables at
p < .05, so it should assert something in 1 − 0.95¹⁰ ≈ 40% of runs. It does so
in 42%.

Three patterns from these benchmark runs:

1. **Critique fixes structure, not luck.** The adversarial loop repairs
   confounds (false-mechanism rate in the confound world 1.00 → .38) and
   recovers scope (exact recovery .00 → .54). It does nothing for chance
   findings: in the null world the false-discovery rate is .42 before and after
   critique, and the same holds in the main and multi worlds. A falsifier
   examining the same data that produced a lucky correlation sees the same lucky
   correlation. Only new cases expose it.
2. **Loops multiply forking paths.** The loop runs 1.8 to 6.5 times as many
   tests as a single pass. Without a gate this is a garden of forking paths with
   a rigorous-looking process on top.
3. **Instability is diagnostic.** Ungated procedures reach their most common
   theory in 40–65% of runs; the gated loop in 93–99%. The exception is the 
   scope world (58%). Most runs state both equivalent framings together; 
   in the rest, the gate strips one framing or the scope itself. 

## A7. Known limitations of the current prototype

- Mechanisms are linear-additive or single interactions; real organizational
  mechanisms involve feedback, timing, and multi-level structure.
- The proposition language cannot express mediation, temporal ordering, or
  continuous moderators.
- The scope condition is the only theoretical move beyond main effects; theory
  building also involves construct redefinition, which this benchmark does not
  test.
- Reference agents are rules. The LLM results are a pilot: 20 datasets per world (100 runs per condition), named and anonymized. Replies came from Claude subagents through reply files (`-replay`), not from direct API calls. Each prompt got one reply, so Claude's own sampling variance is not measured. Each Claude agent is a single call over a table computed by code, with no tools. See `results/llm_pilot_20seeds/` and `results/llm_pilot_20seeds_anonymized/`.
- The holdout gate is a single split; the gated rate of false mechanisms is
  bounded by p < .05 per surviving claim, not by a family-wise correction. 

## A8. Reproducibility

The benchmark is written in Go with no third-party dependencies; the
statistics (OLS, t-distribution p-values) and random draws are implemented in
the package and covered by tests against known values. All randomness comes from
a PCG generator with transforms written in the package, so a seed yields the
same data on any Go version. On a given CPU architecture, runs reproduce
exactly; across architectures the compiler
may combine floating-point operations differently, which in rare cases can change the results of 
a test sitting at p = .05. The full reference benchmark (7,500 runs) takes about
11 seconds on a two-core machine.

`theoryloop export` writes the generated datasets to CSV, with each row marked
discovery or holdout, so any result can be checked in without
running Go. The LLM backend takes the model name as a flag, and the ledger
stores every raw model reply, so runs can be audited and repeated on other
models.
