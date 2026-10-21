# LLM pilot: 20 datasets per world (100 runs per condition)

Claude subagents as proposer and falsifier, new proposer prompt (an empty answer is allowed), named variables.
Seeds 0-19 of all five worlds, answered through `-replay`. 222 model calls, each prompt stored in `prompts/` beside its reply in `replies/`.

- Seeds 0-3 reuse the 44 replies from `../llm_pilot_v2` (ids in `reused_from_earlier_pilots.txt`); 178 replies are new.
- `heuristic_same_seeds/` holds the rule-based agents on the same 20 datasets per world, for a like-for-like comparison.
- Replies came from Claude subagents inside Claude Code, not the raw Messages API. Each agent is a single stateless call. One subagent wrote its ten falsifier replies in a shell loop; all ten were checked against their diagnostics.
- Variable names are shown. The anonymized contrast (`-anonymize`) has not been run.

## Result (perfect recovery / runs with a false claim, n = 100 per row)

| condition | Claude | rules (same datasets) |
|---|---|---|
| single pass | 59% / 22% | 38% / 49% |
| adversarial loop | 92% / 2% | 61% / 38% |
| loop + holdout gate | 98% / 0% | 96% / 2% |

Fisher exact: single pass p = .0045 (perfect), p = .0001 (false claims); loop p < .0001 for both; with the gate p = .68 and .50.
