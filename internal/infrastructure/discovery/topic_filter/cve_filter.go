package topicfilter

import "strings"

type CVETopicFilter struct{}

func NewCVETopicFilter() TopicHeuristicFilter {
	return CVETopicFilter{}
}

func (CVETopicFilter) TopicID() string {
	return "cve"
}

func (CVETopicFilter) MatchURL(u string) bool {
	return looksLikeCVEByURL(u)
}

func (CVETopicFilter) MatchText(title, desc string) bool {
	return looksLikeCVEByText(title, desc)
}

func looksLikeCVEByURL(u string) bool {
	s := strings.ToLower(u)
	if strings.Contains(s, "cve-") {
		return true
	}
	if strings.Contains(s, "/cve/") || strings.Contains(s, "-cve-") || strings.Contains(s, "cve") {
		return true
	}
	if strings.Contains(s, "vuln") || strings.Contains(s, "vulnerability") {
		return true
	}
	return false
}

func looksLikeCVEByText(title, desc string) bool {
	s := strings.ToLower(title + " " + desc)
	return strings.Contains(s, "cve-") ||
		strings.Contains(s, "cvss") ||
		strings.Contains(s, "vulnerability") ||
		strings.Contains(s, "lỗ hổng") ||
		strings.Contains(s, "security advisory") ||
		strings.Contains(s, "rce") ||
		strings.Contains(s, "proof of concept") ||
		strings.Contains(s, "poc")
}
