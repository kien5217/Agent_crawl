package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"Agent_Crawl/internal/config"
)

type Fetcher struct {
	client    *http.Client
	userAgent string
	maxBytes  int64
}

func New(cfg config.Config) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
		},
		userAgent: cfg.HTTP.UserAgent,
		maxBytes:  cfg.HTTP.MaxBytes,
	}
}

// Get thực hiện việc gửi yêu cầu HTTP GET đến URL được cung cấp, trả về nội dung của phản hồi dưới dạng byte slice, loại nội dung (Content-Type), mã trạng thái HTTP và lỗi nếu có. Nó thiết lập User-Agent trong header của yêu cầu và giới hạn kích thước của phản hồi để tránh quá tải bộ nhớ. Nếu phản hồi có mã trạng thái từ 400 trở lên, nó sẽ trả về lỗi tương ứng.
func (f *Fetcher) Get(ctx context.Context, url string) ([]byte, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", resp.StatusCode, fmt.Errorf("http status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	r := io.LimitReader(resp.Body, f.maxBytes)
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ct, resp.StatusCode, err
	}
	return b, ct, resp.StatusCode, nil
}
