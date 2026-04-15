package discovery

import "strings"

// Lọc nhanh theo URL (dùng cho sitemap backfill).
func LooksLikeCVEByURL(u string) bool {
	s := strings.ToLower(u)
	// Những pattern hay gặp trên site security:
	// - chứa CVE ID
	if strings.Contains(s, "cve-") {
		return true
	}
	// Một số site đặt đường dẫn có "cve" hoặc "vulnerability"
	if strings.Contains(s, "/cve/") || strings.Contains(s, "-cve-") || strings.Contains(s, "cve") {
		return true
	}
	if strings.Contains(s, "vuln") || strings.Contains(s, "vulnerability") {
		return true
	}
	return false
}

// Lọc theo text (dùng cho RSS: title+desc).
func LooksLikeCVEByText(title, desc string) bool {
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
