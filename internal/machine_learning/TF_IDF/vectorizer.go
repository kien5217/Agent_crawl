package TF_IDF

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type Vectorizer struct {
	MinDF int

	// vocab: token -> index
	Vocab map[string]int
	// idf[index]
	IDF []float64
	// df[token]
	df map[string]int
	// N docs fitted
	N int
}

type SparseVector map[int]float64

func New(minDF int) *Vectorizer {
	if minDF <= 0 {
		minDF = 3
	}
	return &Vectorizer{
		MinDF: minDF,
		Vocab: map[string]int{},
		df:    map[string]int{},
	}
}

// Fit builds DF and vocab, then computes IDF.
func (v *Vectorizer) Fit(docs []string) {
	v.N = len(docs)
	v.df = map[string]int{}
	seen := make(map[string]struct{}, 1024)

	for _, doc := range docs {
		for k := range seen {
			delete(seen, k)
		}
		toks := TokenizeUnigram(doc)
		for _, t := range toks {
			seen[t] = struct{}{}
		}
		for t := range seen {
			v.df[t]++
		}
	}

	// Build vocab with min_df
	type kv struct {
		t  string
		df int
	}
	var items []kv
	for t, df := range v.df {
		if df >= v.MinDF {
			items = append(items, kv{t: t, df: df})
		}
	}
	// stable order by df desc then token asc (deterministic model)
	sort.Slice(items, func(i, j int) bool {
		if items[i].df == items[j].df {
			return items[i].t < items[j].t
		}
		return items[i].df > items[j].df
	})

	v.Vocab = map[string]int{}
	v.IDF = make([]float64, len(items))
	for i, it := range items {
		v.Vocab[it.t] = i
		// smoothed idf: log((N+1)/(df+1)) + 1
		v.IDF[i] = math.Log(float64(v.N+1)/float64(it.df+1)) + 1.0
	}
}

// Transform converts doc into L2-normalized TF-IDF sparse vector.
func (v *Vectorizer) Transform(doc string) SparseVector {
	tf := map[int]float64{}
	toks := TokenizeUnigram(doc)
	for _, t := range toks {
		idx, ok := v.Vocab[t]
		if !ok {
			continue
		}
		tf[idx]++
	}

	// tf-idf
	var norm2 float64
	for idx, cnt := range tf {
		val := cnt * v.IDF[idx]
		tf[idx] = val
		norm2 += val * val
	}
	if norm2 == 0 {
		return SparseVector{}
	}
	norm := math.Sqrt(norm2)
	for idx, val := range tf {
		tf[idx] = val / norm
	}
	return SparseVector(tf)
}

func TokenizeUnigram(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	fields := strings.Fields(b.String())
	// optional: drop too short tokens
	out := fields[:0]
	for _, t := range fields {
		if len(t) >= 2 {
			out = append(out, t)
		}
	}
	return out
}
