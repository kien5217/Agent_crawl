package learning

import (
	"encoding/json"

	"Agent_Crawl/internal/db"
	tfidf "Agent_Crawl/internal/machine_learning/TF_IDF"
	logreg "Agent_Crawl/internal/machine_learning/logistic_regression"
	modelbundle "Agent_Crawl/internal/machine_learning/model_bundle"
)

type TrainResult struct {
	NumSamples int
	NumClasses int
	VocabSize  int
}

func TrainFromDocs(train []db.DocForLearning, classes []string, minDF int) (*modelbundle.Bundle, *TrainResult) {
	// map topic -> class index
	classIndex := map[string]int{}
	for i, c := range classes {
		classIndex[c] = i
	}

	texts := make([]string, 0, len(train))
	ys := make([]int, 0, len(train))
	for _, d := range train {
		ci, ok := classIndex[d.TopicID]
		if !ok {
			continue
		}
		texts = append(texts, d.Title+"\n"+d.Content)
		ys = append(ys, ci)
	}

	vec := tfidf.New(minDF)
	vec.Fit(texts)

	xs := make([]tfidf.SparseVector, len(texts))
	for i, t := range texts {
		xs[i] = vec.Transform(t)
	}

	m := logreg.NewModel(len(classes), len(vec.IDF), classes)
	m.Lambda = 1e-4
	m.TrainSGD(xs, ys, logreg.TrainOptions{Epochs: 6, LR: 0.2, Shuffle: true})

	return &modelbundle.Bundle{Vectorizer: vec, Model: m}, &TrainResult{
		NumSamples: len(xs),
		NumClasses: len(classes),
		VocabSize:  len(vec.IDF),
	}
}

func ClassesJSON(classes []string) []byte {
	b, _ := json.Marshal(classes)
	return b
}
