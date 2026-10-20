package theoryloop

// Data is a column-oriented panel: one slice per variable, all of length N.
type Data struct {
	N    int
	Cols map[string][]float64
}

// Col returns a variable's column.
func (d Data) Col(name string) []float64 { return d.Cols[name] }

// Rows returns the rows at idx, in that order.
func (d Data) Rows(idx []int) Data {
	out := Data{N: len(idx), Cols: make(map[string][]float64, len(d.Cols))}
	for name, c := range d.Cols {
		nc := make([]float64, len(idx))
		for i, j := range idx {
			nc[i] = c[j]
		}
		out.Cols[name] = nc
	}
	return out
}

// Where returns the rows satisfying a scope condition (all rows if unscoped).
func (d Data) Where(c Cond) Data {
	if c.Var == "" {
		return d
	}
	var idx []int
	for i, v := range d.Cols[c.Var] {
		if v == float64(c.Val) {
			idx = append(idx, i)
		}
	}
	return d.Rows(idx)
}

// SplitIndex divides n cases into a discovery half and a holdout half. The
// split depends only on the seed, so every condition sees the same halves.
func SplitIndex(n int, seed uint64) (discovery, holdout []int) {
	p := newRNG(seed+10_000, 2).perm(n)
	cut := n / 2
	return p[:cut], p[cut:]
}

// Split returns the discovery and holdout halves of d.
func Split(d Data, seed uint64) (discovery, holdout Data) {
	a, b := SplitIndex(d.N, seed)
	return d.Rows(a), d.Rows(b)
}
