package classify

import (
	"sort"
	"strings"

	"Agent_Crawl/internal/config"
	util "Agent_Crawl/internal/utils"
)

type Result struct {
	TopicID string
	Scores  map[string]int
	Max     int
}

type KeywordClassifier struct {
	topics       []config.Topic
	minAccept    int
	normalizedKW map[string]map[string]int // topicID -> term -> weight
}

// khởi tạo KeywordClassifier từ config.TopicsFile, chuẩn hóa các keyword để so khớp dễ dàng hơn
func NewKeywordClassifier(tf config.TopicsFile, minScoreToAccept int) *KeywordClassifier {
	nkw := map[string]map[string]int{} // topicID -> keyword -> weight
	for _, t := range tf.Topics {
		m := map[string]int{}
		for _, kw := range t.Keywords {
			term := util.NormalizeText(kw.Term)
			if term == "" {
				continue
			}
			m[term] = kw.Weight
		}
		nkw[t.ID] = m
	}
	return &KeywordClassifier{
		topics:       tf.Topics,
		minAccept:    minScoreToAccept,
		normalizedKW: nkw,
	}
}

func (c *KeywordClassifier) Classify(title, content string) Result {
	titleN := util.NormalizeText(title)
	bodyN := util.NormalizeText(content)

	scores := map[string]int{} //tạo map để lưu điểm số của từng topic, key là topicID, value là điểm số
	for _, t := range c.topics {
		score := 0
		for term, w := range c.normalizedKW[t.ID] {
			// phrase match (contains)
			if strings.Contains(titleN, term) {
				score += w * 3 // tăng trọng số nếu keyword xuất hiện trong title vì thường title sẽ cô đọng nội dung chính
			}
			if strings.Contains(bodyN, term) {
				score += w // keyword xuất hiện trong content cũng cộng điểm nhưng ít hơn title vì content có thể dài và có nhiều thông tin phụ
			}
		}
		scores[t.ID] = score
	}

	type kv struct {
		k string
		v int
	}
	var arr []kv //chuyển map scores thành slice để sắp xếp theo điểm số, vì map không đảm bảo thứ tự
	for k, v := range scores {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })

	best := "unknown"
	max := 0 // sau khi tính điểm cho tất cả topic, sắp xếp để tìm topic có điểm cao nhất. Nếu điểm cao nhất thấp hơn ngưỡng chấp nhận thì trả về "unknown"
	if len(arr) > 0 {
		best = arr[0].k
		max = arr[0].v
	}
	if max < c.minAccept {
		best = "unknown"
	}
	return Result{TopicID: best, Scores: scores, Max: max}
}
