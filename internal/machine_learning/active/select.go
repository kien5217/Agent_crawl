package active

import (
	"math"
	"sort"

	tfidf "Agent_Crawl/internal/machine_learning/TF_IDF"
	logreg "Agent_Crawl/internal/machine_learning/logistic_regression"
)

type Candidate struct {
	DocID  int64
	X      tfidf.SparseVector
	Proba  []float64
	Margin float64 // p1 - p2 (smaller = more uncertain)
}

// Compute margin for a probability distribution
func margin(p []float64) float64 {
	if len(p) == 0 {
		return 1
	}
	p1, p2 := 0.0, 0.0
	for _, v := range p {
		if v >= p1 {
			p2 = p1
			p1 = v
		} else if v > p2 {
			p2 = v
		}
	}
	return p1 - p2
}

// Cosine similarity for L2-normalized sparse vectors (TF-IDF output is normalized)
func cosine(a, b tfidf.SparseVector) float64 {
	// dot product (since already L2 norm ~1)
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for i, av := range a {
		if bv, ok := b[i]; ok {
			dot += av * bv
		}
	}
	// clamp
	if dot < 0 {
		return 0
	}
	if dot > 1 {
		return 1
	}
	return dot
}

// SelectBatch selects N docs using: smallest margin first, then enforce diversity by cosine threshold.
func SelectBatch(model *logreg.Model, docIDs []int64, xs []tfidf.SparseVector, batchSize int, diversityThreshold float64) []int64 {
	if batchSize <= 0 {
		batchSize = 50
	}
	if diversityThreshold <= 0 {
		diversityThreshold = 0.80
	}

	cands := make([]Candidate, 0, len(xs))
	for i := range xs {
		p := model.PredictProba(xs[i])
		m := margin(p)
		cands = append(cands, Candidate{
			DocID:  docIDs[i],
			X:      xs[i],
			Proba:  p,
			Margin: m,
		})
	}

	// sort by margin asc (most uncertain first)
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].Margin == cands[j].Margin {
			return cands[i].DocID < cands[j].DocID
		}
		return cands[i].Margin < cands[j].Margin
	})

	var picked []Candidate
	for _, c := range cands {
		if len(picked) >= batchSize {
			break
		}
		ok := true
		for _, p := range picked {
			if cosine(c.X, p.X) >= diversityThreshold {
				ok = false
				break
			}
		}
		if ok {
			picked = append(picked, c)
		}
	}

	out := make([]int64, 0, len(picked))
	for _, p := range picked {
		out = append(out, p.DocID)
	}
	return out
}

// Optional: entropy (if you prefer), kept here for reference.
func Entropy(p []float64) float64 {
	var h float64
	for _, v := range p {
		if v <= 0 {
			continue
		}
		h -= v * math.Log(v)
	}
	return h
}
