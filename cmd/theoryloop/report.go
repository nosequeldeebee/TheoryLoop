package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	tl "theoryloop"
)

type cell struct {
	runs, failed                                      int
	perfect, anyFalse, recallExact, wrongScope, tests float64
	distinct                                          int
	modalShare                                        float64 // share of runs ending in the most common final theory
}

type key struct{ world, cond string }

func aggregate(rs []tl.Result, worlds []string) map[key]cell {
	type acc struct {
		cell
		recallN  int
		theories map[string]int
	}
	m := map[key]*acc{}
	for _, w := range worlds {
		for _, c := range tl.Conditions {
			m[key{w, c}] = &acc{theories: map[string]int{}}
		}
	}
	for _, r := range rs {
		a := m[key{r.World, r.Condition}]
		if a == nil {
			continue
		}
		if r.Err != "" {
			a.failed++
			continue
		}
		a.runs++
		a.perfect += float64(r.Perfect)
		a.anyFalse += float64(r.AnyFalse)
		a.wrongScope += float64(r.WrongScope)
		a.tests += float64(r.NTests)
		if !math.IsNaN(r.RecallExact) {
			a.recallExact += r.RecallExact
			a.recallN++
		}
		a.theories[canonical(r.Theory)]++
	}
	out := map[key]cell{}
	for k, a := range m {
		c := a.cell
		if c.runs > 0 {
			n := float64(c.runs)
			c.perfect /= n
			c.anyFalse /= n
			c.wrongScope /= n
			c.tests /= n
		}
		if a.recallN > 0 {
			c.recallExact /= float64(a.recallN)
		} else {
			c.recallExact = math.NaN()
		}
		c.distinct = len(a.theories)
		for _, k := range a.theories {
			if c.runs > 0 && float64(k)/float64(c.runs) > c.modalShare {
				c.modalShare = float64(k) / float64(c.runs)
			}
		}
		out[k] = c
	}
	return out
}

// canonical identifies a theory regardless of the order its propositions were
// listed in, so stability measures count the same theory once.
func canonical(ps []tl.Proposition) string {
	var ls []string
	for _, p := range ps {
		ls = append(ls, p.Label())
	}
	sort.Strings(ls)
	return strings.Join(ls, " ; ")
}

func f2(v float64) string {
	if math.IsNaN(v) {
		return "–"
	}
	return fmt.Sprintf("%.2f", v)
}

func summaryMarkdown(agg map[key]cell, rs []tl.Result, worlds []string, tag string, seeds int) string {
	failed := 0
	for _, c := range agg {
		failed += c.failed
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Benchmark summary (%s, %d seeds per world)\n\n", tag, seeds)
	b.WriteString("- perfect: recovered every planted mechanism with correct scope and nothing else\n")
	b.WriteString("- any_false: share of runs asserting at least one mechanism that does not exist\n")
	b.WriteString("- recall_exact: share of planted mechanisms recovered with the right scope (– in the null world)\n")
	b.WriteString("- tests: mean number of statistical tests in the ledger behind the final theory\n")
	b.WriteString("- distinct_theories: how many different final theories across seeds (grows with the number of seeds)\n")
	b.WriteString("- modal_share: share of runs ending in the single most common final theory (stability; does not grow with seeds)\n\n")
	b.WriteString("| world | condition | perfect | any_false | recall_exact | wrong_scope | tests | distinct_theories | modal_share |")
	if failed > 0 {
		b.WriteString(" failed |")
	}
	b.WriteString("\n|---|---|---:|---:|---:|---:|---:|---:|---:|")
	if failed > 0 {
		b.WriteString("---:|")
	}
	b.WriteString("\n")
	for _, w := range worlds {
		for _, c := range tl.Conditions {
			x := agg[key{w, c}]
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %.1f | %d | %s |", w, c,
				f2(x.perfect), f2(x.anyFalse), f2(x.recallExact), f2(x.wrongScope), x.tests, x.distinct, f2(x.modalShare))
			if failed > 0 {
				fmt.Fprintf(&b, " %d |", x.failed)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Overall\n\n| condition | perfect | any_false |\n|---|---:|---:|\n")
	for _, c := range tl.Conditions {
		var n, perfect, anyFalse float64
		for _, r := range rs {
			if r.Condition == c && r.Err == "" {
				n++
				perfect += float64(r.Perfect)
				anyFalse += float64(r.AnyFalse)
			}
		}
		if n == 0 {
			n = math.NaN()
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c, f2(perfect/n), f2(anyFalse/n))
	}
	if failed > 0 {
		fmt.Fprintf(&b, "\n%d runs failed and are excluded; see the error column in the runs CSV.\n", failed)
	}
	return b.String()
}

// chartSVG draws two panels of grouped bars: exact recovery and false-mechanism rate.
func chartSVG(agg map[key]cell, worlds []string, tag string) string {
	const W, H = 1100.0, 440.0
	const pw, top, bot = 460.0, 90.0, 350.0
	colors := map[string]string{"single_pass": "#9aa5b1", "loop": "#3b6ea8", "loop_holdout": "#1f3a5f"}
	names := map[string]string{"single_pass": "Single pass", "loop": "Adversarial loop", "loop_holdout": "Loop + holdout gate"}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" font-family="Helvetica, Arial, sans-serif" font-size="13">`, W, H)
	b.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	fmt.Fprintf(&b, `<text x="20" y="34" font-size="17" fill="#111">Planted-mechanism benchmark (%s agents)</text>`, tag)
	panels := []struct {
		x0     float64
		title  string
		metric func(c cell) float64
	}{
		{80, "Recovered the planted theory exactly", func(c cell) float64 { return c.perfect }},
		{620, "Asserted a mechanism that does not exist", func(c cell) float64 { return c.anyFalse }},
	}
	for _, pn := range panels {
		fmt.Fprintf(&b, `<text x="%.0f" y="72" font-size="14" fill="#111">%s</text>`, pn.x0, pn.title)
		for _, t := range []float64{0, 0.25, 0.5, 0.75, 1} {
			y := bot - t*(bot-top)
			fmt.Fprintf(&b, `<line x1="%.0f" x2="%.0f" y1="%.1f" y2="%.1f" stroke="#e5e7eb"/>`, pn.x0, pn.x0+pw, y, y)
			fmt.Fprintf(&b, `<text x="%.0f" y="%.1f" text-anchor="end" fill="#666">%.0f%%</text>`, pn.x0-8, y+4, t*100)
		}
		band := pw / float64(len(worlds))
		bw := band * 0.24
		for i, w := range worlds {
			cx := pn.x0 + band*(float64(i)+0.5)
			for j, c := range tl.Conditions {
				v := pn.metric(agg[key{w, c}])
				if math.IsNaN(v) {
					v = 0
				}
				h := v * (bot - top)
				x := cx + (float64(j)-1)*bw - bw/2
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"><title>%s, %s: %.0f%%</title></rect>`,
					x, bot-h, bw, h, colors[c], w, names[c], v*100)
			}
			fmt.Fprintf(&b, `<text x="%.1f" y="%.0f" text-anchor="middle" fill="#333">%s</text>`, cx, bot+20, w)
		}
		fmt.Fprintf(&b, `<line x1="%.0f" x2="%.0f" y1="%.0f" y2="%.0f" stroke="#333"/>`, pn.x0, pn.x0+pw, bot, bot)
	}
	fmt.Fprintf(&b, `<text transform="translate(26 %.0f) rotate(-90)" text-anchor="middle" fill="#666">share of runs</text>`, (top+bot)/2)
	lx := W/2 - 250
	for i, c := range tl.Conditions {
		x := lx + float64(i)*180
		fmt.Fprintf(&b, `<rect x="%.0f" y="402" width="14" height="14" fill="%s"/><text x="%.0f" y="414" fill="#333">%s</text>`,
			x, colors[c], x+20, names[c])
	}
	b.WriteString(`</svg>`)
	return b.String()
}
