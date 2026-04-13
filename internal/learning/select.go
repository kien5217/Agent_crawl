package learning

import (
	"sort"

	"Agent_Crawl/internal/db"
	tfidf "Agent_Crawl/internal/machine_learning/TF_IDF"
	"Agent_Crawl/internal/machine_learning/active"
	modelbundle "Agent_Crawl/internal/machine_learning/model_bundle"

	"github.com/jackc/pgx/v5"
)

type Pick struct {
	DocID  int64
	Margin float64
}

func SelectBatchForLabeling(conn *pgx.Conn, bundle *modelbundle.Bundle, docIDs []int64, titles []string, contents []string, batchSize int) []int64 {
	vec := bundle.Vectorizer
	model := bundle.Model

	xs := make([]tfidf.SparseVector, len(docIDs))
	for i := range docIDs {
		xs[i] = vec.Transform(titles[i] + "\n" + contents[i])
	}
	return active.SelectBatchBalanced(model, docIDs, xs, batchSize, 0.80)
}

// compute margin for saving in label_queue (optional)
func ComputeMargins(bundle *modelbundle.Bundle, docIDs []int64, titles []string, contents []string) []Pick {
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

	sort.Slice(out, func(i, j int) bool { return out[i].Margin < out[j].Margin })
	return out
}

func EnqueuePicked(ctx any, conn *pgx.Conn, picks []Pick) error {
	// ctx is unused here to keep snippet short; in your code pass context.Context.
	return nil
}

func SavePicksToQueue(ctx interface{}, conn *pgx.Conn, picks []Pick) error {
	return nil
}

// Helper: enqueue picked IDs (margin lookup from pick list)
func EnqueueLabelQueueByIDs(ctxContext interface{}, conn *pgx.Conn, picks []Pick, pickedIDs []int64) error {
	pickMap := map[int64]float32{}
	for _, p := range picks {
		pickMap[p.DocID] = float32(p.Margin)
	}
	for _, id := range pickedIDs {
		_ = db.EnqueueLabelQueue // use in your cmd with context
		_ = pickMap[id]
	}
	return nil
}
