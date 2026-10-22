# Benchmark summary (code/rules-based, not LLM, 500 seeds per world)

- perfect: recovered every planted mechanism with correct scope and nothing else
- any_false: share of runs asserting at least one mechanism that does not exist
- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)
- tests: mean number of statistical tests in the ledger behind the final theory
- distinct_theories: how many different final theories across seeds (grows with the number of seeds)
- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)

| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| main | single_pass | 0.62 | 0.38 | 1.00 | 0.00 | 10.0 | 52 | 0.62 |
| main | loop | 0.61 | 0.38 | 0.98 | 0.01 | 30.6 | 62 | 0.61 |
| main | loop_holdout | 0.97 | 0.02 | 0.99 | 0.00 | 32.0 | 8 | 0.97 |
| scope | single_pass | 0.00 | 0.34 | 0.00 | 1.74 | 10.0 | 46 | 0.48 |
| scope | loop | 0.54 | 0.34 | 0.82 | 0.30 | 55.9 | 73 | 0.40 |
| scope | loop_holdout | 0.77 | 0.01 | 0.79 | 0.32 | 59.5 | 14 | 0.58 |
| confound | single_pass | 0.00 | 1.00 | 1.00 | 0.00 | 10.0 | 17 | 0.65 |
| confound | loop | 0.62 | 0.38 | 1.00 | 0.00 | 64.6 | 31 | 0.62 |
| confound | loop_holdout | 0.93 | 0.07 | 1.00 | 0.00 | 66.0 | 7 | 0.93 |
| null | single_pass | 0.58 | 0.42 | – | 0.00 | 10.0 | 57 | 0.58 |
| null | loop | 0.58 | 0.42 | – | 0.00 | 18.0 | 64 | 0.58 |
| null | loop_holdout | 0.98 | 0.02 | – | 0.00 | 18.5 | 7 | 0.98 |
| multi | single_pass | 0.65 | 0.35 | 1.00 | 0.00 | 10.0 | 17 | 0.65 |
| multi | loop | 0.63 | 0.35 | 0.99 | 0.02 | 48.4 | 30 | 0.63 |
| multi | loop_holdout | 0.99 | 0.01 | 1.00 | 0.00 | 50.8 | 5 | 0.99 |

## Overall

| condition | perfect | any_false |
|---|---:|---:|
| single_pass | 0.37 | 0.50 |
| loop | 0.60 | 0.37 |
| loop_holdout | 0.93 | 0.03 |
