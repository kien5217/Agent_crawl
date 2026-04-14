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

	// --- AI (anchors) ---
	if strings.Contains(s, "large language model") || strings.Contains(s, "llm") ||
		strings.Contains(s, "fine-tuning") || strings.Contains(s, "transformer") {
		return "ai", 0.85, "anchor:ai_llm_transformer", true
	}
	if strings.Contains(s, "diffusion") && (strings.Contains(s, "image") || strings.Contains(s, "text-to-image")) {
		return "ai", 0.80, "anchor:ai_diffusion", true
	}

	// --- Cloud (anchors) ---
	if strings.Contains(s, "kubernetes") || strings.Contains(s, "k8s") {
		return "cloud", 0.85, "anchor:cloud_k8s", true
	}
	if strings.Contains(s, "terraform") {
		return "cloud", 0.80, "anchor:cloud_terraform", true
	}
	if strings.Contains(s, "aws") || strings.Contains(s, "gcp") || strings.Contains(s, "azure") {
		// cẩn thận: từ này có thể xuất hiện trong nhiều bài; giữ confidence vừa thôi
		return "cloud", 0.75, "anchor:cloud_hyperscaler", true
	}

	// --- Programming (anchors) ---
	if strings.Contains(s, "golang") || strings.Contains(s, "go ") || strings.Contains(s, "rust") ||
		strings.Contains(s, "typescript") || strings.Contains(s, "python") {
		// "go " dễ dính tiếng Anh (go to...), nên nếu dùng thì cần rule chặt hơn theo ngữ cảnh
		return "programming", 0.75, "anchor:prog_lang", true
	}
	if strings.Contains(s, "github repository") || strings.Contains(s, "release v") {
		return "programming", 0.70, "anchor:prog_release", true
	}

	// --- Blockchain (anchors) ---
	if strings.Contains(s, "ethereum") || strings.Contains(s, "solidity") ||
		strings.Contains(s, "smart contract") || strings.Contains(s, "defi") {
		return "blockchain", 0.85, "anchor:blockchain_eth_sc", true
	}

	// --- Security (anchors, không phải CVE) ---
	// Nếu bạn muốn tách "security" với "cve", giữ security cho các bài ransomware/phishing/incident…
	if strings.Contains(s, "ransomware") || strings.Contains(s, "phishing") ||
		strings.Contains(s, "malware") || strings.Contains(s, "data breach") {
		return "security", 0.85, "anchor:security_incident", true
	}
	// Strong-ish CVE signals
	if strings.Contains(s, "cvss") && (strings.Contains(s, "vulnerability") || strings.Contains(s, "lỗ hổng")) {
		return "cve", 0.85, "anchor:cvss+vuln", true
	}
	if strings.Contains(s, "cve-") {
		return "cve", 0.90, "anchor:cve_dash", true
	}

	return "", 0, "", false
}

// ApplyWeakLabels thực hiện việc áp dụng weak labeling cho một slice các DocForLearning. Nó sử dụng một WeakLabeler để gán nhãn cho từng tài liệu dựa trên tiêu đề và nội dung của nó. Kết quả trả về là một slice các struct chứa DocID, TopicID, Confidence và RuleID cho những tài liệu đã được gán nhãn thành công. Nếu một tài liệu không khớp với bất kỳ quy tắc weak labeling nào, nó sẽ bị bỏ qua.
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
