# Reference workflow summary (heuristic, 20 seeds per world)

- perfect: recovered every planted mechanism with correct scope and nothing else
- any_false: share of runs asserting at least one mechanism that does not exist
- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)
- tests: mean number of statistical tests in the ledger behind the final theory
- distinct_theories: how many different final theories across seeds (grows with the number of seeds)
- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)

| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| main | single_pass | 0.60 | 0.40 | 1.00 | 0.00 | 10.0 | 8 | 0.60 |
| main | loop | 0.60 | 0.40 | 1.00 | 0.00 | 30.9 | 8 | 0.60 |
| main | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 32.4 | 1 | 1.00 |
| scope | single_pass | 0.00 | 0.35 | 0.00 | 1.75 | 10.0 | 7 | 0.40 |
| scope | loop | 0.60 | 0.35 | 0.90 | 0.20 | 57.1 | 9 | 0.35 |
| scope | loop_holdout | 0.90 | 0.00 | 0.90 | 0.15 | 60.8 | 4 | 0.65 |
| confound | single_pass | 0.00 | 1.00 | 1.00 | 0.00 | 10.0 | 7 | 0.60 |
| confound | loop | 0.55 | 0.45 | 1.00 | 0.00 | 64.5 | 9 | 0.55 |
| confound | loop_holdout | 0.90 | 0.10 | 1.00 | 0.00 | 66.0 | 2 | 0.90 |
| null | single_pass | 0.65 | 0.35 | – | 0.00 | 10.0 | 8 | 0.65 |
| null | loop | 0.65 | 0.35 | – | 0.00 | 16.2 | 8 | 0.65 |
| null | loop_holdout | 1.00 | 0.00 | – | 0.00 | 16.6 | 1 | 1.00 |
| multi | single_pass | 0.65 | 0.35 | 1.00 | 0.00 | 10.0 | 7 | 0.65 |
| multi | loop | 0.65 | 0.35 | 1.00 | 0.00 | 47.3 | 7 | 0.65 |
| multi | loop_holdout | 1.00 | 0.00 | 1.00 | 0.00 | 49.6 | 1 | 1.00 |

## Overall

| condition | perfect | any_false |
|---|---:|---:|
| single_pass | 0.38 | 0.49 |
| loop | 0.61 | 0.38 |
| loop_holdout | 0.96 | 0.02 |
