package ml

import (
	"math"
	"sort"
)

type Pred struct {
	DocID  int64
	X      SparseVector
	Proba  []float64
	Top1   int
	Top2   int
	P1     float64
	P2     float64
	Margin float64 // p1 - p2 (smaller = more uncertain)
}

func argTop2(p []float64) (top1, top2 int, p1, p2 float64) {
	top1, top2 = -1, -1
	p1, p2 = -1, -1
	for i, v := range p {
		if v > p1 {
			top2, p2 = top1, p1
			top1, p1 = i, v
		} else if v > p2 {
			top2, p2 = i, v
		}
	}
	if top2 == -1 {
		top2, p2 = top1, p1
	}
	return
}

// Cosine similarity for L2-normalized sparse vectors (TF-IDF output is normalized)
func cosine(a, b SparseVector) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for i, av := range a {
		if bv, ok := b[i]; ok {
			dot += av * bv
		}
	}
	if dot < 0 {
		return 0
	}
	if dot > 1 {
		return 1
	}
	return dot
}

func SelectBatchBalanced(
	model *Model,
	docIDs []int64,
	xs []SparseVector,
	batchSize int,
	diversityThreshold float64,
) []int64 {
	if batchSize <= 0 {
		batchSize = 50
	}
	if diversityThreshold <= 0 {
		diversityThreshold = 0.80
	}

	// 1) Predict all
	preds := make([]Pred, 0, len(xs))
	for i := range xs {
		p := model.PredictProba(xs[i])
		t1, t2, p1, p2 := argTop2(p)
		preds = append(preds, Pred{
			DocID:  docIDs[i],
			X:      xs[i],
			Proba:  p,
			Top1:   t1,
			Top2:   t2,
			P1:     p1,
			P2:     p2,
			Margin: p1 - p2,
		})
	}

	// 2) Group by predicted class (Top1)
	byClass := make([][]Pred, model.K)
	for _, pr := range preds {
		if pr.Top1 < 0 || pr.Top1 >= model.K {
			continue
		}
		byClass[pr.Top1] = append(byClass[pr.Top1], pr)
	}

	// sort each class by margin asc (most uncertain first)
	for k := 0; k < model.K; k++ {
		sort.Slice(byClass[k], func(i, j int) bool {
			if byClass[k][i].Margin == byClass[k][j].Margin {
				return byClass[k][i].DocID < byClass[k][j].DocID
			}
			return byClass[k][i].Margin < byClass[k][j].Margin
		})
	}

	// 3) Compute quotas
	// base quota: ceil(batch/K)
	base := int(math.Ceil(float64(batchSize) / float64(maxInt(1, model.K))))

	type pickItem struct {
		DocID  int64
		X      SparseVector
		Margin float64
		Top1   int
	}
	var picked []pickItem

	// helper to check diversity against already picked
	canPick := func(x SparseVector) bool {
		for _, p := range picked {
			if cosine(x, p.X) >= diversityThreshold {
				return false
			}
		}
		return true
	}

	// 4) First pass: pick up to base quota per class
	for k := 0; k < model.K; k++ {
		need := base
		for _, pr := range byClass[k] {
			if need == 0 || len(picked) >= batchSize {
				break
			}
			if !canPick(pr.X) {
				continue
			}
			picked = append(picked, pickItem{DocID: pr.DocID, X: pr.X, Margin: pr.Margin, Top1: k})
			need--
		}
		if len(picked) >= batchSize {
			break
		}
	}

	// 5) Fill remaining globally by margin (still diversity)
	if len(picked) < batchSize {
		// global sort by margin asc
		sort.Slice(preds, func(i, j int) bool {
			if preds[i].Margin == preds[j].Margin {
				return preds[i].DocID < preds[j].DocID
			}
			return preds[i].Margin < preds[j].Margin
		})

		pickedSet := map[int64]struct{}{}
		for _, p := range picked {
			pickedSet[p.DocID] = struct{}{}
		}

		for _, pr := range preds {
			if len(picked) >= batchSize {
				break
			}
			if _, ok := pickedSet[pr.DocID]; ok {
				continue
			}
			if !canPick(pr.X) {
				continue
			}
			picked = append(picked, pickItem{DocID: pr.DocID, X: pr.X, Margin: pr.Margin, Top1: pr.Top1})
			pickedSet[pr.DocID] = struct{}{}
		}
	}

	out := make([]int64, 0, len(picked))
	for _, p := range picked {
		out = append(out, p.DocID)
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
