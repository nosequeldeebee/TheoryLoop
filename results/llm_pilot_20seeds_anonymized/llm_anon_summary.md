# Benchmark summary (llm_anon, 20 seeds per world)

- perfect: recovered every planted mechanism with correct scope and nothing else
- any_false: share of runs asserting at least one mechanism that does not exist
- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)
- tests: mean number of statistical tests in the ledger behind the final theory
- distinct_theories: how many different final theories across seeds (grows with the number of seeds)
- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)

| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| main | single_pass | 1.00 | 0.00 | 1.00 | 0.00 | 10.0 | 1 | 1.00 |
| main | loop | 0.90 | 0.00 | 0.90 | 0.10 | 23.9 | 3 | 0.90 |
| main | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 25.1 | 1 | 1.00 |
| scope | single_pass | 0.00 | 0.00 | 0.00 | 1.35 | 10.0 | 3 | 0.45 |
| scope | loop | 0.90 | 0.00 | 0.90 | 0.00 | 39.7 | 3 | 0.45 |
| scope | loop_holdout | 0.90 | 0.00 | 0.90 | 0.00 | 42.4 | 3 | 0.45 |
| confound | single_pass | 0.00 | 1.00 | 1.00 | 0.00 | 10.0 | 1 | 1.00 |
| confound | loop | 1.00 | 0.00 | 1.00 | 0.00 | 55.0 | 1 | 1.00 |
| confound | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 56.0 | 1 | 1.00 |
| null | single_pass | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| null | loop | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| null | loop_holdout | 1.00 | 0.00 | – | 0.00 | 10.0 | 1 | 1.00 |
| multi | single_pass | 0.95 | 0.05 | 1.00 | 0.00 | 10.0 | 2 | 0.95 |
| multi | loop | 0.90 | 0.05 | 0.97 | 0.05 | 44.0 | 3 | 0.90 |
| multi | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 46.2 | 1 | 1.00 |

## Overall

| condition | perfect | any_false |
|---|---:|---:|
| single_pass | 0.59 | 0.21 |
| loop | 0.94 | 0.01 |
| loop_holdout | 0.98 | 0.00 |
