package discovery

import (
	"net/url"
	"strings"
)

func NormalizeURL(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw) // Xóa khoảng trắng ở đầu và cuối URL để tránh lỗi khi phân tích cú pháp. Nếu URL rỗng sau khi trim, trả về false để báo lỗi.
	if raw == "" {
		return "", "", false
	}
	u, err := url.Parse(raw) // Phân tích cú pháp URL
	if err != nil {
		return "", "", false
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		return "", "", false
	}
	u.Fragment = ""
	// Remove common tracking params (MVP)
	q := u.Query() // Xóa các tham số theo dõi phổ biến như utm_source, utm_medium, fbclid, gclid để tránh trùng lặp do các URL khác nhau chỉ vì có thêm tham số này. Điều này giúp giảm số lượng URL cần crawl và lưu trữ.
	for _, k := range []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "fbclid", "gclid"} {
		q.Del(k)
	}
	u.RawQuery = q.Encode() // Chuẩn hóa host thành chữ thường để tránh trùng lặp do khác biệt về chữ hoa chữ thường. Trả về URL đã chuẩn hóa, host và true để báo thành công.

	host := strings.ToLower(u.Host)
	normalized := u.String()
	return normalized, host, true
}
