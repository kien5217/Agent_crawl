package discovery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/repository"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
)

type RSSDiscovery struct {
	cfg     config.Config
	sources config.SourcesFile
	client  *http.Client
	q       repository.QueueRepository
} // RSSDiscovery implements Discovery interface. Nó sử dụng gofeed để phân tích cú pháp RSS feed từ các nguồn tin được cấu hình, chuẩn hóa URL của từng mục trong feed và enqueue chúng vào hàng đợi crawl. Nó cũng xử lý lỗi khi phân tích cú pháp hoặc enqueue và ghi log cảnh báo nếu có vấn đề với một nguồn tin cụ thể.
// Khởi tạo RSSDiscovery với cấu hình và danh sách nguồn tin, đồng thời thiết lập một HTTP client với timeout được cấu hình. Hàm Enqueue sẽ được gọi để lấy các mục từ RSS feed, chuẩn hóa URL và enqueue chúng vào hàng đợi crawl. Nó giới hạn số lượng mục được enqueue từ mỗi nguồn tin dựa trên cấu hình để tránh quá tải hệ thống.
func NewRSSDiscovery(cfg config.Config, sources config.SourcesFile, q repository.QueueRepository) *RSSDiscovery {
	return &RSSDiscovery{
		cfg:     cfg,
		sources: sources,
		client:  &http.Client{Timeout: time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second},
		q:       q,
	}
}

func (d *RSSDiscovery) Name() string {
	return "rss"
}

// Enqueue thực hiện việc lấy các mục từ RSS feed của mỗi nguồn tin đã được bật, chuẩn hóa URL của từng mục và enqueue chúng vào hàng đợi crawl. Nó sử dụng gofeed để phân tích cú pháp RSS feed và gọi hàm NormalizeURL để chuẩn hóa URL. Nếu có lỗi khi phân tích cú pháp hoặc enqueue, nó sẽ ghi log cảnh báo và tiếp tục với nguồn tin tiếp theo. Hàm này trả về tổng số URL đã được enqueue thành công.
func (d *RSSDiscovery) Enqueue(ctx context.Context) (int, error) {
	fp := gofeed.NewParser()
	enqueued := 0

	for _, s := range d.sources.Sources {
		if !s.Enabled || s.RSSURL == "" {
			continue
		}
		log.Info().Str("source", s.ID).Str("rss", s.RSSURL).Msg("rss discovery start")
		feed, err := d.fetchFeed(ctx, fp, s.RSSURL)
		if err != nil {
			log.Warn().Err(err).Str("source", s.ID).Msg("rss parse failed")
			continue
		}

		limit := d.cfg.Scheduler.EnqueueLimitPerSource
		for i, item := range feed.Items {
			if i >= limit {
				break
			}
			if item == nil || item.Link == "" {
				continue
			}

			title := ""
			desc := ""
			if item.Title != "" {
				title = item.Title
			}
			if item.Description != "" {
				desc = item.Description
			}

			if !LooksLikeCVEByText(title, desc) {
				continue
			}

			norm, domain, ok := NormalizeURL(item.Link)
			if !ok {
				continue
			}

			inserted, err := d.q.EnqueueURL(ctx, "cve", s.ID, norm, domain, 10)
			if err != nil {
				log.Warn().Err(err).Str("url", norm).Msg("enqueue failed")
				continue
			}
			if inserted {
				enqueued++
			}
		}
		log.Info().Str("source", s.ID).Int("enqueued", enqueued).Msg("rss discovery done")
	}

	return enqueued, nil
}

// fetchFeed thực hiện việc gửi yêu cầu HTTP để lấy nội dung của RSS feed từ URL được cung cấp, sử dụng gofeed để phân tích cú pháp và trả về đối tượng Feed. Nó thiết lập User-Agent trong header của yêu cầu và giới hạn kích thước của phản hồi để tránh quá tải bộ nhớ. Nếu có lỗi trong quá trình gửi yêu cầu hoặc phân tích cú pháp, nó sẽ trả về lỗi tương ứng.
func (d *RSSDiscovery) fetchFeed(ctx context.Context, fp *gofeed.Parser, rssURL string) (*gofeed.Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rssURL, nil) // Tạo yêu cầu HTTP với context để có thể hủy bỏ nếu cần thiết
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", d.cfg.HTTP.UserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New("http status " + resp.Status)
	}

	return fp.Parse(io.LimitReader(resp.Body, d.cfg.HTTP.MaxBytes))
}
