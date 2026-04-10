package logistic_regression

import (
	"encoding/json"
	"math"
	"math/rand"
	"time"

	tfidf "Agent_Crawl/internal/machine_learning/TF_IDF"
)

type Model struct {
	// dims
	K int // num classes
	V int // vocab size

	// weights: K x V
	W [][]float64
	B []float64

	// label mapping: class index -> topic_id
	Classes []string

	// hyperparams
	Lambda float64 // L2
}

func NewModel(k, v int, classes []string) *Model {
	m := &Model{
		K:       k,
		V:       v,
		W:       make([][]float64, k),
		B:       make([]float64, k),
		Classes: append([]string(nil), classes...),
		Lambda:  1e-4,
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < k; i++ {
		m.W[i] = make([]float64, v)
		// small init
		for j := 0; j < v; j++ {
			m.W[i][j] = (rng.Float64()*2 - 1) * 0.001
		}
		m.B[i] = 0
	}
	return m
}

func (m *Model) PredictProba(x tfidf.SparseVector) []float64 {
	z := make([]float64, m.K)
	var maxz float64 = -1e30
	for k := 0; k < m.K; k++ {
		sum := m.B[k]
		for j, xj := range x {
			sum += m.W[k][j] * xj
		}
		z[k] = sum
		if sum > maxz {
			maxz = sum
		}
	}
	// softmax stable
	var denom float64
	for k := 0; k < m.K; k++ {
		z[k] = math.Exp(z[k] - maxz)
		denom += z[k]
	}
	if denom == 0 {
		p := make([]float64, m.K)
		for k := range p {
			p[k] = 1 / float64(m.K)
		}
		return p
	}
	for k := 0; k < m.K; k++ {
		z[k] /= denom
	}
	return z
}

// One SGD step for one sample (x, yIndex)
func (m *Model) SGDStep(x tfidf.SparseVector, y int, lr float64) {
	p := m.PredictProba(x)

	// gradient for each class: (p_k - y_k)*x + 2*lambda*w
	for k := 0; k < m.K; k++ {
		yk := 0.0
		if k == y {
			yk = 1.0
		}
		err := (p[k] - yk)

		// update bias
		m.B[k] -= lr * err

		// sparse update for weights
		row := m.W[k]
		for j, xj := range x {
			grad := err*xj + 2*m.Lambda*row[j]
			row[j] -= lr * grad
		}
	}
}

type TrainOptions struct {
	Epochs  int
	LR      float64
	Shuffle bool
}

func (m *Model) TrainSGD(xs []tfidf.SparseVector, ys []int, opt TrainOptions) {
	if opt.Epochs <= 0 {
		opt.Epochs = 5
	}
	if opt.LR <= 0 {
		opt.LR = 0.1
	}
	n := len(xs)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for ep := 0; ep < opt.Epochs; ep++ {
		if opt.Shuffle {
			rng.Shuffle(n, func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })
		}
		lr := opt.LR / (1.0 + 0.2*float64(ep)) // simple decay

		for _, ii := range idx {
			m.SGDStep(xs[ii], ys[ii], lr)
		}
	}
}

func (m *Model) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func FromJSON(b []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
