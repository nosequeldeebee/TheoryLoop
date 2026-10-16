# Benchmark summary (heuristic, 50 seeds per world)

perfect = recovered every planted mechanism with correct scope and nothing else
any_false = share of runs asserting at least one mechanism that does not exist
tests = mean number of statistical tests in the ledger behind the final theory
distinct_theories = how many different final theories across seeds

|                              |   perfect |   any_false |   recall_exact |   wrong_scope |   tests |   distinct_theories |
|:-----------------------------|----------:|------------:|---------------:|--------------:|--------:|--------------------:|
| ('main', 'single_pass')      |      0.64 |        0.36 |           1    |          0    |   10    |                  15 |
| ('main', 'loop')             |      0.64 |        0.36 |           1    |          0    |   29.98 |                  15 |
| ('main', 'loop_holdout')     |      1    |        0    |           1    |          0    |   31.4  |                   1 |
| ('scope', 'single_pass')     |      0    |        0.34 |           0    |          1.66 |   10    |                  20 |
| ('scope', 'loop')            |      0.54 |        0.34 |           0.82 |          0.22 |   55.22 |                  24 |
| ('scope', 'loop_holdout')    |      0.74 |        0    |           0.76 |          0.24 |   58.8  |                   9 |
| ('confound', 'single_pass')  |      0    |        1    |           1    |          0    |   10    |                  10 |
| ('confound', 'loop')         |      0.68 |        0.32 |           1    |          0    |   61.68 |                  12 |
| ('confound', 'loop_holdout') |      0.9  |        0.1  |           1    |          0    |   63.04 |                   3 |
| ('null', 'single_pass')      |      0.54 |        0.46 |         nan    |          0    |   10    |                  18 |
| ('null', 'loop')             |      0.54 |        0.46 |         nan    |          0    |   18.6  |                  18 |
| ('null', 'loop_holdout')     |      1    |        0    |         nan    |          0    |   19.16 |                   1 |
| ('multi', 'single_pass')     |      0.64 |        0.36 |           1    |          0    |   10    |                  15 |
| ('multi', 'loop')            |      0.62 |        0.36 |           0.99 |          0.02 |   47.78 |                  16 |
| ('multi', 'loop_holdout')    |      0.98 |        0    |           0.99 |          0.02 |   50.16 |                   3 |

## Overall

| condition    |   perfect |   any_false |
|:-------------|----------:|------------:|
| single_pass  |      0.36 |        0.5  |
| loop         |      0.6  |        0.37 |
| loop_holdout |      0.92 |        0.02 |
