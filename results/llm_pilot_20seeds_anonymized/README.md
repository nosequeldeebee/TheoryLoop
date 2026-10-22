# LLM pilot, variables anonymized (x1-x10): 20 datasets per world

Same datasets, prompts, model and procedure as `../llm_pilot_20seeds`, except the proposer and falsifier see x1..x10 instead of construct names.
219 model calls, each prompt in `prompts/` beside its reply in `replies/`. Nothing was reused from earlier pilots.

| condition | named: perfect / false | anonymized: perfect / false | identical final theory |
|---|---|---|---|
| single pass | 59% / 22% | 59% / 21% | 90 of 100 |
| adversarial loop | 92% / 2% | 94% / 1% | 89 of 100 |
| loop + holdout gate | 98% / 0% | 98% / 0% | 94 of 100 |

No difference is significant (Fisher p > .78). The decoy ("verified") was asserted in all 20 decoy datasets in both runs.

Caveats: x-labels keep the same order as the named variables; the system prompt still describes NFT projects and legitimacy; replies came from Claude subagents inside Claude Code, one stateless call each.
