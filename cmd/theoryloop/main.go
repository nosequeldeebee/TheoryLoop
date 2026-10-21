// Command theoryloop runs the planted-mechanism benchmark and exports its data.
//
//	theoryloop bench                                  reference agents, 5 worlds x 500 seeds
//	theoryloop bench -backend llm -seeds 10           Claude subagents (needs ANTHROPIC_API_KEY)
//	theoryloop bench -backend llm -seeds 10 -anonymize   same, variable names hidden
//	theoryloop export -seeds 3                        write the generated datasets to data/
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tl "theoryloop"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "bench":
		err = bench(os.Args[2:])
	case "export":
		err = export(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  theoryloop bench  [-backend heuristic|llm] [-seeds N] [-worlds a,b] [-anonymize] [-model M] [-workers N] [-out DIR]
  theoryloop export [-seeds N] [-worlds a,b] [-out DIR]
run "theoryloop bench -h" or "theoryloop export -h" for details`)
	os.Exit(2)
}

func worldList(s string) ([]string, error) {
	var out []string
	for _, w := range strings.Split(s, ",") {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if _, err := tl.MakeWorld(w, 0, 10); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

// ------------------------------------------------------------------ bench

func bench(args []string) error {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	backend := fs.String("backend", "heuristic", "agents to use: heuristic or llm")
	seeds := fs.Int("seeds", 500, "seeds per world")
	worldsFlag := fs.String("worlds", strings.Join(tl.WorldNames, ","), "comma-separated worlds")
	model := fs.String("model", tl.DefaultModel, "model for the llm backend")
	anon := fs.Bool("anonymize", false, "hide variable names from llm agents (tests reliance on priors)")
	out := fs.String("out", "results", "output directory")
	workers := fs.Int("workers", 0, "parallel runs (default: number of CPUs for heuristic, 4 for llm)")
	n := fs.Int("n", 2000, "projects per world")
	v1 := fs.Bool("prompt-v1", false, "llm backend: use the original proposer prompt (no 'empty answer is fine')")
	replay := fs.String("replay", "", "llm backend: answer model calls from files in this directory instead of the API (see FileCaller)")
	fs.Parse(args)

	worlds, err := worldList(*worldsFlag)
	if err != nil {
		return err
	}
	if *backend != "heuristic" && *backend != "llm" {
		return fmt.Errorf("unknown backend %q", *backend)
	}
	if *backend == "llm" && *replay == "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
		return fmt.Errorf("the llm backend needs ANTHROPIC_API_KEY (or -replay DIR)")
	}
	if *workers <= 0 {
		*workers = runtime.NumCPU()
		if *backend == "llm" {
			*workers = 4
		}
	}
	tag := *backend
	if *anon {
		tag += "_anon"
	}
	ledgerDir := filepath.Join(*out, tag+"_ledgers")
	if err := os.MkdirAll(ledgerDir, 0o755); err != nil {
		return err
	}
	cfg := tl.Config{Backend: *backend, Model: *model, Anonymize: *anon, PromptV1: *v1}
	if *replay != "" {
		cfg.Call = (&tl.FileCaller{Dir: *replay}).Call
	}

	type job struct {
		world string
		seed  int
	}
	var jobs []job
	for _, w := range worlds {
		for s := 0; s < *seeds; s++ {
			jobs = append(jobs, job{w, s})
		}
	}
	results := make([][]tl.Result, len(jobs))
	idx := make(chan int)
	var done atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()
	for k := 0; k < *workers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				j := jobs[i]
				w, err := tl.MakeWorld(j.world, uint64(j.seed), *n)
				for _, c := range tl.Conditions {
					var r tl.Result
					if err == nil {
						var runErr error
						r, runErr = tl.Run(w, c, uint64(j.seed), cfg)
						if runErr != nil {
							r.Err = runErr.Error()
						}
					} else {
						r = tl.Result{World: j.world, Condition: c, Seed: uint64(j.seed), Backend: *backend, Err: err.Error()}
					}
					if j.seed < 3 && r.Ledger != nil {
						name := fmt.Sprintf("%s_%s_seed%d.jsonl", j.world, c, j.seed)
						if err := r.Ledger.Dump(filepath.Join(ledgerDir, name)); err != nil {
							fmt.Fprintln(os.Stderr, "ledger:", err)
						}
					}
					results[i] = append(results[i], r)
				}
				if k := done.Add(1); *backend == "llm" {
					fmt.Fprintf(os.Stderr, "\r%d/%d world-seeds done", k, len(jobs))
				}
			}
		}()
	}
	for i := range jobs {
		idx <- i
	}
	close(idx)
	wg.Wait()
	if *backend == "llm" {
		fmt.Fprintln(os.Stderr)
	}

	var flat []tl.Result
	pending := 0
	for _, rs := range results {
		flat = append(flat, rs...)
		for _, r := range rs {
			if strings.Contains(r.Err, "reply pending") {
				pending++
			}
		}
	}
	if pending > 0 {
		fmt.Printf("%d of %d runs are waiting for replies; prompts are in %s/pending. Supply replies in %s/replies and run again.\n",
			pending, len(flat), *replay, *replay)
		return nil
	}
	if err := writeRuns(filepath.Join(*out, tag+"_runs.csv"), flat); err != nil {
		return err
	}
	agg := aggregate(flat, worlds)
	summary := summaryMarkdown(agg, flat, worlds, tag, *seeds)
	if err := os.WriteFile(filepath.Join(*out, tag+"_summary.md"), []byte(summary), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, tag+"_chart.svg"), []byte(chartSVG(agg, worlds, tag)), 0o644); err != nil {
		return err
	}
	fmt.Print(summary)
	fmt.Printf("\n%d runs in %.1fs, written to %s/\n", len(flat), time.Since(start).Seconds(), *out)
	return nil
}

func fnum(v float64) string {
	if v != v { // NaN
		return ""
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func theoryString(ps []tl.Proposition) string {
	var parts []string
	for _, p := range ps {
		parts = append(parts, p.Label())
	}
	return strings.Join(parts, " ; ")
}

func writeRuns(path string, rs []tl.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"world", "condition", "seed", "backend", "theory", "iterations", "n_tests", "n_props",
		"recall_exact", "recall_variable", "false_props", "wrong_scope", "perfect", "any_false", "error"})
	for _, r := range rs {
		w.Write([]string{r.World, r.Condition, strconv.FormatUint(r.Seed, 10), r.Backend, theoryString(r.Theory),
			strconv.Itoa(r.Iterations), strconv.Itoa(r.NTests), strconv.Itoa(r.NProps),
			fnum(r.RecallExact), fnum(r.RecallVariable), strconv.Itoa(r.FalseProps), strconv.Itoa(r.WrongScope),
			strconv.Itoa(r.Perfect), strconv.Itoa(r.AnyFalse), r.Err})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ----------------------------------------------------------------- export

func export(args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	seeds := fs.Int("seeds", 3, "seeds per world")
	worldsFlag := fs.String("worlds", strings.Join(tl.WorldNames, ","), "comma-separated worlds")
	out := fs.String("out", "data", "output directory")
	n := fs.Int("n", 2000, "projects per world")
	fs.Parse(args)

	worlds, err := worldList(*worldsFlag)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	type truthOut struct {
		World       string     `json:"world"`
		Description string     `json:"description"`
		Mechanisms  [][]string `json:"mechanisms"` // one list per mechanism: its equivalent statements
	}
	var truth []truthOut
	cols := append(append([]string{}, tl.Predictors...), tl.Outcome)
	for _, name := range worlds {
		for s := 0; s < *seeds; s++ {
			w, err := tl.MakeWorld(name, uint64(s), *n)
			if err != nil {
				return err
			}
			if s == 0 {
				t := truthOut{World: name, Description: w.Description, Mechanisms: [][]string{}}
				for _, slot := range w.Truth {
					var ls []string
					for _, p := range slot {
						ls = append(ls, p.Label())
					}
					t.Mechanisms = append(t.Mechanisms, ls)
				}
				truth = append(truth, t)
			}
			_, hold := tl.SplitIndex(w.Data.N, uint64(s))
			isHold := make([]bool, w.Data.N)
			for _, i := range hold {
				isHold[i] = true
			}
			f, err := os.Create(filepath.Join(*out, fmt.Sprintf("%s_seed%d.csv", name, s)))
			if err != nil {
				return err
			}
			cw := csv.NewWriter(f)
			cw.Write(append([]string{"split"}, cols...))
			for i := 0; i < w.Data.N; i++ {
				row := []string{"discovery"}
				if isHold[i] {
					row[0] = "holdout"
				}
				for _, c := range cols {
					row = append(row, fnum(w.Data.Col(c)[i]))
				}
				cw.Write(row)
			}
			cw.Flush()
			if err := cw.Error(); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
	b, _ := json.MarshalIndent(truth, "", "  ")
	if err := os.WriteFile(filepath.Join(*out, "truth.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %d datasets and truth.json to %s/\n", len(worlds)**seeds, *out)
	return nil
}
