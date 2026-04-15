package ml

import "encoding/json"

type Bundle struct {
	Vectorizer *Vectorizer `json:"vectorizer"`
	Model      *Model      `json:"model"`
}

func (b *Bundle) Marshal() ([]byte, error) { return json.Marshal(b) }
func Unmarshal(data []byte) (*Bundle, error) {
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
