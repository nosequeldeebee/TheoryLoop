# Benchmark summary (llm, 4 seeds per world)

- perfect: recovered every planted mechanism with correct scope and nothing else
- any_false: share of runs asserting at least one mechanism that does not exist
- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)
- tests: mean number of statistical tests in the ledger behind the final theory
- distinct_theories: how many different final theories across seeds (grows with the number of seeds)
- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)

| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| main | single_pass | 0.25 | 0.75 | 1.00 | 0.00 | 10.0 | 4 | 0.25 |
| main | loop | 0.50 | 0.50 | 1.00 | 0.00 | 62.2 | 3 | 0.50 |
| main | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 63.8 | 1 | 1.00 |
| scope | single_pass | 0.00 | 1.00 | 0.00 | 2.00 | 10.0 | 3 | 0.50 |
| scope | loop | 0.50 | 0.50 | 1.00 | 0.00 | 76.2 | 3 | 0.50 |
| scope | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 80.8 | 1 | 1.00 |
| confound | single_pass | 0.00 | 1.00 | 1.00 | 0.00 | 10.0 | 4 | 0.25 |
| confound | loop | 1.00 | 0.00 | 1.00 | 0.00 | 66.2 | 1 | 1.00 |
| confound | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 67.2 | 1 | 1.00 |
| null | single_pass | 0.00 | 1.00 | – | 0.00 | 10.0 | 4 | 0.25 |
| null | loop | 0.50 | 0.50 | – | 0.00 | 58.0 | 3 | 0.50 |
| null | loop_holdout | 1.00 | 0.00 | – | 0.00 | 58.5 | 1 | 1.00 |
| multi | single_pass | 0.50 | 0.50 | 1.00 | 0.00 | 10.0 | 3 | 0.50 |
| multi | loop | 0.75 | 0.25 | 1.00 | 0.00 | 57.2 | 2 | 0.75 |
| multi | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 59.5 | 1 | 1.00 |

## Overall

| condition | perfect | any_false |
|---|---:|---:|
| single_pass | 0.15 | 0.85 |
| loop | 0.65 | 0.35 |
| loop_holdout | 1.00 | 0.00 |
