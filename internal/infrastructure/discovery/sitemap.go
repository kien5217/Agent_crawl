package discovery

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/repository"

	"github.com/rs/zerolog/log"
)

type SitemapDiscovery struct {
	cfg     config.Config
	sources config.SourcesFile
	client  *http.Client
	q       repository.QueueRepository
}

func NewSitemapDiscovery(cfg config.Config, sources config.SourcesFile, q repository.QueueRepository) *SitemapDiscovery {
	to := cfg.Sitemap.HTTPTimeoutSeconds
	if to <= 0 {
		to = cfg.HTTP.TimeoutSeconds
	}
	return &SitemapDiscovery{
		cfg:     cfg,
		sources: sources,
		client:  &http.Client{Timeout: time.Duration(to) * time.Second},
		q:       q,
	}
}

func (d *SitemapDiscovery) Name() string {
	return "sitemap"
}

func (d *SitemapDiscovery) Enqueue(ctx context.Context) (int, error) {
	if !d.cfg.Sitemap.Enabled {
		return 0, nil
	}

	total := 0
	for _, s := range d.sources.Sources {
		if !s.Enabled || len(s.SitemapURLs) == 0 {
			continue
		}
		log.Info().Str("source", s.ID).Int("sitemaps", len(s.SitemapURLs)).Msg("sitemap discovery start")
		added, err := d.enqueueSource(ctx, s)
		if err != nil {
			log.Warn().Err(err).Str("source", s.ID).Msg("sitemap discovery failed")
			continue
		}
		log.Info().Str("source", s.ID).Int("enqueued", added).Msg("sitemap discovery done")
		total += added
	}
	return total, nil
}

// enqueueSource thực hiện việc enqueue các URL từ sitemap của một nguồn tin cụ thể. Nó giới hạn số lượng URL được enqueue dựa trên cấu hình để tránh quá tải hệ thống. Hàm này gọi processSitemapAny để xử lý từng URL trong sitemap, bao gồm cả việc phân tích cú pháp và chuẩn hóa URL trước khi enqueue vào hàng đợi crawl. Nếu có lỗi trong quá trình xử lý sitemap, nó sẽ ghi log cảnh báo và tiếp tục với URL tiếp theo.
func (d *SitemapDiscovery) enqueueSource(ctx context.Context, s config.Source) (int, error) {
	limit := d.cfg.Sitemap.MaxURLsPerSourcePerRun
	if limit <= 0 {
		limit = 20000
	}

	added := 0
	for _, smURL := range s.SitemapURLs {
		if added >= limit {
			break
		}
		n, err := d.processSitemapAny(ctx, s, smURL, limit-added, 0)
		if err != nil {
			log.Warn().Err(err).Str("sitemap", smURL).Msg("process sitemap failed")
			continue
		}
		added += n
	}
	return added, nil
}

// processSitemapAny thực hiện việc xử lý một URL sitemap, bao gồm cả việc phân tích cú pháp và chuẩn hóa URL trước khi enqueue vào hàng đợi crawl. Hàm này có thể xử lý cả sitemap index và url set, và nó sẽ gọi đệ quy nếu gặp sitemap index để xử lý các sitemap con. Nó cũng giới hạn độ sâu của sitemap để tránh vòng lặp vô hạn và giới hạn số lượng URL được enqueue dựa trên cấu hình.
func (d *SitemapDiscovery) processSitemapAny(ctx context.Context, s config.Source, sitemapURL string, remaining int, depth int) (int, error) {
	if remaining <= 0 {
		return 0, nil
	}
	if depth > 5 {
		return 0, errors.New("sitemap depth too deep")
	}

	b, err := d.httpGetBytes(ctx, sitemapURL)
	if err != nil {
		return 0, err
	}

	// detect which root: sitemapindex or urlset
	root := detectRootTag(b)
	switch root {
	case "sitemapindex":
		var idx sitemapIndex
		if err := xml.Unmarshal(b, &idx); err != nil {
			return 0, err
		}
		maxChildren := d.cfg.Sitemap.MaxSitemapsPerIndex
		if maxChildren <= 0 {
			maxChildren = 200
		}

		added := 0
		for i, child := range idx.Sitemaps {
			if i >= maxChildren || added >= remaining {
				break
			}
			loc := strings.TrimSpace(child.Loc)
			if loc == "" {
				continue
			}
			n, err := d.processSitemapAny(ctx, s, loc, remaining-added, depth+1)
			if err != nil {
				log.Warn().Err(err).Str("child", loc).Msg("child sitemap failed")
				continue
			}
			added += n
		}
		return added, nil

	case "urlset":
		var us urlSet
		if err := xml.Unmarshal(b, &us); err != nil {
			return 0, err
		}
		added := 0
		for _, u := range us.URLs {
			if added >= remaining {
				break
			}
			loc := strings.TrimSpace(u.Loc)
			if loc == "" {
				continue
			}
			// Lọc CVE theo URL (2A)
			if !LooksLikeCVEByURL(loc) {
				continue
			}
			norm, domain, ok := NormalizeURL(loc)
			if !ok {
				continue
			}
			ins, err := d.q.EnqueueURL(ctx, "cve", s.ID, norm, domain, 0) // priority thấp hơn RSS
			if err != nil {
				log.Warn().Err(err).Str("url", norm).Msg("enqueue failed")
				continue
			}
			if ins {
				added++
			}
		}
		return added, nil

	default:
		return 0, errors.New("unknown sitemap root")
	}
}

// httpGetBytes thực hiện việc gửi yêu cầu HTTP GET đến URL được cung cấp và trả về nội dung của phản hồi dưới dạng byte slice. Nó thiết lập User-Agent trong header của yêu cầu và giới hạn kích thước của phản hồi để tránh quá tải bộ nhớ. Nếu phản hồi có mã trạng thái từ 400 trở lên, nó sẽ trả về lỗi tương ứng.
func (d *SitemapDiscovery) httpGetBytes(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", d.cfg.HTTP.UserAgent)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, errors.New("http status " + resp.Status)
	}
	// sitemap có thể lớn; MVP đọc vừa phải (10MB)
	r := io.LimitReader(resp.Body, 10*1024*1024)
	return io.ReadAll(r)
}

// detectRootTag thực hiện việc phát hiện thẻ gốc của một tài liệu XML dựa trên nội dung của nó. Hàm này chuyển đổi byte slice thành chuỗi và kiểm tra xem nó có chứa thẻ <sitemapindex> hoặc <urlset> hay không. Nếu tìm thấy thẻ <sitemapindex>, nó trả về "sitemapindex". Nếu tìm thấy thẻ <urlset>, nó trả về "urlset". Nếu không tìm thấy bất kỳ thẻ nào trong số đó, nó trả về một chuỗi rỗng.
func detectRootTag(b []byte) string {
	s := strings.ToLower(string(b))
	// crude but effective MVP
	if strings.Contains(s, "<sitemapindex") {
		return "sitemapindex"
	}
	if strings.Contains(s, "<urlset") {
		return "urlset"
	}
	return ""
}

type sitemapIndex struct {
	XMLName  xml.Name      `xml:"sitemapindex"`
	Sitemaps []sitemapItem `xml:"sitemap"`
}
type sitemapItem struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	URLs    []urlItem `xml:"url"`
}
type urlItem struct {
	Loc string `xml:"loc"`
}
