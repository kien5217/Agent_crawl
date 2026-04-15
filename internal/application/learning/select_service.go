package learning

import (
	"sort"

	ml "Agent_Crawl/internal/infrastructure/machine_learning"
)

type Pick struct {
	DocID  int64
	Margin float64
}

func SelectBatchForLabeling(bundle *ml.Bundle, docIDs []int64, titles []string, contents []string, batchSize int) []int64 {
	vec := bundle.Vectorizer
	model := bundle.Model

	xs := make([]ml.SparseVector, len(docIDs))
	for i := range docIDs {
		xs[i] = vec.Transform(titles[i] + "\n" + contents[i])
	}
	return ml.SelectBatchBalanced(model, docIDs, xs, batchSize, 0.80)
}

// compute margin for saving in label_queue (optional)
func ComputeMargins(bundle *ml.Bundle, docIDs []int64, titles []string, contents []string) []Pick {
	vec := bundle.Vectorizer
	model := bundle.Model
	out := make([]Pick, 0, len(docIDs))

	for i := range docIDs {
		x := vec.Transform(titles[i] + "\n" + contents[i])
		p := model.PredictProba(x)

		// margin p1-p2
		p1, p2 := 0.0, 0.0
		for _, v := range p {
			if v >= p1 {
				p2 = p1
				p1 = v
			} else if v > p2 {
				p2 = v
			}
		}
		out = append(out, Pick{DocID: docIDs[i], Margin: p1 - p2})
	}
	// sort by margin ascending (uncertainty sampling)
	sort.Slice(out, func(i, j int) bool { return out[i].Margin < out[j].Margin })
	return out
}
