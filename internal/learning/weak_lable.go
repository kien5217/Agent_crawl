package learning

import (
	"regexp"
	"strings"

	"Agent_Crawl/internal/db"
)

var reCVE = regexp.MustCompile(`\bCVE-\d{4}-\d{4,7}\b`)

type WeakLabeler struct{}

func NewWeakLabeler() *WeakLabeler { return &WeakLabeler{} }

// Trả về (topic_id, confidence, rule_id, ok)
func (wl *WeakLabeler) Label(title, content string) (string, float32, string, bool) {
	txt := title + "\n" + content

	// Anchor rule: CVE id
	if reCVE.MatchString(txt) {
		return "cve", 0.95, "anchor:cve_id_regex", true
	}

	s := strings.ToLower(txt)
	// Strong-ish CVE signals
	if strings.Contains(s, "cvss") && (strings.Contains(s, "vulnerability") || strings.Contains(s, "lỗ hổng")) {
		return "cve", 0.85, "anchor:cvss+vuln", true
	}
	if strings.Contains(s, "cve-") {
		return "cve", 0.90, "anchor:cve_dash", true
	}

	// Nếu bạn muốn bootstrapping cho các topic khác, thêm rules tương tự.
	return "", 0, "", false
}

func ApplyWeakLabels(docs []db.DocForLearning, wl *WeakLabeler) []struct {
	DocID      int64
	TopicID    string
	Confidence float32
	RuleID     string
} {
	out := make([]struct {
		DocID      int64
		TopicID    string
		Confidence float32
		RuleID     string
	}, 0, len(docs))

	for _, d := range docs {
		topic, conf, rule, ok := wl.Label(d.Title, d.Content)
		if !ok {
			continue
		}
		out = append(out, struct {
			DocID      int64
			TopicID    string
			Confidence float32
			RuleID     string
		}{d.ID, topic, conf, rule})
	}
	return out
}
