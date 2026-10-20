# Benchmark summary (llm, 4 seeds per world)

- perfect: recovered every planted mechanism with correct scope and nothing else
- any_false: share of runs asserting at least one mechanism that does not exist
- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)
- tests: mean number of statistical tests in the ledger behind the final theory
- distinct_theories: how many different final theories across seeds (grows with the number of seeds)
- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)

| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| main | single_pass | 1.00 | 0.00 | 1.00 | 0.00 | 10.0 | 1 | 1.00 |
| main | loop | 1.00 | 0.00 | 1.00 | 0.00 | 23.0 | 1 | 1.00 |
| main | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 24.0 | 1 | 1.00 |
| scope | single_pass | 0.00 | 0.00 | 0.00 | 1.50 | 10.0 | 2 | 0.50 |
| scope | loop | 1.00 | 0.00 | 1.00 | 0.00 | 43.0 | 2 | 0.50 |
| scope | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 46.0 | 2 | 0.50 |
| confound | single_pass | 0.00 | 1.00 | 1.00 | 0.00 | 10.0 | 2 | 0.75 |
| confound | loop | 1.00 | 0.00 | 1.00 | 0.00 | 58.2 | 1 | 1.00 |
| confound | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 59.2 | 1 | 1.00 |
| null | single_pass | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| null | loop | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| null | loop_holdout | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| multi | single_pass | 1.00 | 0.00 | 1.00 | 0.00 | 10.0 | 1 | 1.00 |
| multi | loop | 1.00 | 0.00 | 1.00 | 0.00 | 42.0 | 1 | 1.00 |
| multi | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 44.0 | 1 | 1.00 |

## Overall

| condition | perfect | any_false |
|---|---:|---:|
| single_pass | 0.60 | 0.20 |
| loop | 1.00 | 0.00 |
| loop_holdout | 1.00 | 0.00 |
