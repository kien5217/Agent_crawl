package modelbundle

import (
	"encoding/json"

	tfidf "Agent_Crawl/internal/machine_learning/TF_IDF"
	logreg "Agent_Crawl/internal/machine_learning/logistic_regression"
)

type Bundle struct {
	Vectorizer *tfidf.Vectorizer `json:"vectorizer"`
	Model      *logreg.Model     `json:"model"`
}

func (b *Bundle) Marshal() ([]byte, error) { return json.Marshal(b) }
func Unmarshal(data []byte) (*Bundle, error) {
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
