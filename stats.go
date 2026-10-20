package theoryloop

import (
	"math"
	"math/rand/v2"
)

// this stats toolkit is hand-written to minimize dependencies 
// it is not intended to be a general-purpose statistics package.

// rng draws every random number in the benchmark. PCG is a fixed, published
// algorithm, and the transforms below are written out here rather than taken
// from the standard library, so a seed means the same data on any Go version.
type rng struct {
	src      *rand.PCG
	spare    float64
	hasSpare bool
}

func newRNG(seed, stream uint64) *rng { return &rng{src: rand.NewPCG(seed, stream)} }

// uniform returns a draw from [0, 1).
func (r *rng) uniform() float64 { return float64(r.src.Uint64()>>11) / (1 << 53) }

// normal returns a standard normal draw (Box-Muller).
func (r *rng) normal() float64 {
	if r.hasSpare {
		r.hasSpare = false
		return r.spare
	}
	u1 := 1 - r.uniform() // (0, 1]
	u2 := r.uniform()
	rad := math.Sqrt(-2 * math.Log(u1))
	r.spare, r.hasSpare = rad*math.Sin(2*math.Pi*u2), true
	return rad * math.Cos(2*math.Pi*u2)
}

func (r *rng) exponential() float64 { return -math.Log(1 - r.uniform()) }

func (r *rng) intn(n int) int { return int(r.uniform() * float64(n)) }

func (r *rng) bernoulli(p float64) float64 {
	if r.uniform() < p {
		return 1
	}
	return 0
}

// beta draws Beta(a, b) for integer shapes as a ratio of gamma sums.
func (r *rng) beta(a, b int) float64 {
	x, y := 0.0, 0.0
	for i := 0; i < a; i++ {
		x += r.exponential()
	}
	for i := 0; i < b; i++ {
		y += r.exponential()
	}
	return x / (x + y)
}

// perm returns a random permutation of 0..n-1 (Fisher-Yates).
func (r *rng) perm(n int) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := r.intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	return p
}

// ---------------------------------------------------------------- helpers

func meanOf(x []float64) float64 {
	if len(x) == 0 {
		return math.NaN()
	}
	s := 0.0
	for _, v := range x {
		s += v
	}
	return s / float64(len(x))
}

// sdOf is the sample standard deviation (n-1 denominator).
func sdOf(x []float64) float64 {
	if len(x) < 2 {
		return 0
	}
	m, ss := meanOf(x), 0.0
	for _, v := range x {
		ss += (v - m) * (v - m)
	}
	return math.Sqrt(ss / float64(len(x)-1))
}

func scale(x []float64, by float64) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = v / by
	}
	return out
}

func standardize(x []float64) []float64 {
	s := sdOf(x)
	if s == 0 {
		return append([]float64(nil), x...)
	}
	return scale(x, s)
}

func mul(a, b []float64) []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i] * b[i]
	}
	return out
}

func sign(v float64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// ------------------------------------------------------------------- OLS

// ols regresses y on an intercept plus xs. It returns the slope coefficients
// and their two-sided p-values, aligned with xs. ok is false when the design
// matrix is singular, in which case every p-value is 1.
func ols(y []float64, xs ...[]float64) (coef, p []float64, ok bool) {
	n, k := len(y), len(xs)+1
	at := func(i, j int) float64 {
		if j == 0 {
			return 1
		}
		return xs[j-1][i]
	}
	xtx := make([][]float64, k)
	for j := range xtx {
		xtx[j] = make([]float64, k)
	}
	xty := make([]float64, k)
	for i := 0; i < n; i++ {
		for a := 0; a < k; a++ {
			va := at(i, a)
			xty[a] += va * y[i]
			for b := a; b < k; b++ {
				xtx[a][b] += va * at(i, b)
			}
		}
	}
	for a := 0; a < k; a++ {
		for b := 0; b < a; b++ {
			xtx[a][b] = xtx[b][a]
		}
	}
	coef, p = make([]float64, k-1), make([]float64, k-1)
	inv, ok := invert(xtx)
	if !ok || n <= k {
		for j := range p {
			p[j] = 1
		}
		return coef, p, false
	}
	beta := make([]float64, k)
	for a := 0; a < k; a++ {
		for b := 0; b < k; b++ {
			beta[a] += inv[a][b] * xty[b]
		}
	}
	rss := 0.0
	for i := 0; i < n; i++ {
		fit := 0.0
		for a := 0; a < k; a++ {
			fit += beta[a] * at(i, a)
		}
		r := y[i] - fit
		rss += r * r
	}
	dof := float64(n - k)
	s2 := rss / dof
	for j := 1; j < k; j++ {
		se := math.Sqrt(math.Max(s2*inv[j][j], 1e-300))
		coef[j-1] = beta[j]
		p[j-1] = tTwoSided(beta[j]/se, dof)
	}
	return coef, p, true
}

// invert returns the inverse of a small square matrix (Gauss-Jordan with
// partial pivoting), or false if it is numerically singular.
func invert(a [][]float64) ([][]float64, bool) {
	k := len(a)
	m := make([][]float64, k)
	maxDiag := 0.0
	for i := range a {
		m[i] = make([]float64, 2*k)
		copy(m[i], a[i])
		m[i][k+i] = 1
		maxDiag = math.Max(maxDiag, math.Abs(a[i][i]))
	}
	tol := 1e-12 * maxDiag
	for c := 0; c < k; c++ {
		piv := c
		for r := c + 1; r < k; r++ {
			if math.Abs(m[r][c]) > math.Abs(m[piv][c]) {
				piv = r
			}
		}
		if math.Abs(m[piv][c]) <= tol {
			return nil, false
		}
		m[c], m[piv] = m[piv], m[c]
		d := m[c][c]
		for j := range m[c] {
			m[c][j] /= d
		}
		for r := 0; r < k; r++ {
			if r == c || m[r][c] == 0 {
				continue
			}
			f := m[r][c]
			for j := range m[r] {
				m[r][j] -= f * m[c][j]
			}
		}
	}
	inv := make([][]float64, k)
	for i := range inv {
		inv[i] = m[i][k:]
	}
	return inv, true
}

// tTwoSided is the two-sided p-value of a t statistic with dof degrees of freedom.
func tTwoSided(t, dof float64) float64 {
	if math.IsNaN(t) {
		return 1
	}
	if math.IsInf(t, 0) {
		return 0
	}
	return regIncBeta(dof/2, 0.5, dof/(dof+t*t))
}

// regIncBeta is the regularized incomplete beta function I_x(a, b).
func regIncBeta(a, b, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	front := math.Exp(lab - la - lb + a*math.Log(x) + b*math.Log(1-x))
	if x < (a+1)/(a+b+2) {
		return front * betaCF(a, b, x) / a
	}
	return 1 - front*betaCF(b, a, 1-x)/b
}

// betaCF evaluates the continued fraction for the incomplete beta function (Lentz).
func betaCF(a, b, x float64) float64 {
	const maxIter, eps, tiny = 500, 3e-16, 1e-300
	qab, qap, qam := a+b, a+1, a-1
	c, d := 1.0, 1-qab*x/qap
	if math.Abs(d) < tiny {
		d = tiny
	}
	d = 1 / d
	h := d
	for m := 1; m <= maxIter; m++ {
		mf := float64(m)
		m2 := 2 * mf
		aa := mf * (b - mf) * x / ((qam + m2) * (a + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		h *= d * c
		aa = -(a + mf) * (qab + mf) * x / ((a + m2) * (qap + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < eps {
			break
		}
	}
	return h
}
