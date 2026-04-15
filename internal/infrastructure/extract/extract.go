package extract

import (
	"bytes"
	"strings"

	util "Agent_Crawl/internal/platform"

	"github.com/PuerkitoBio/goquery"
)

type Result struct {
	CanonicalURL string
	Title        string
	Author       string
	PublishedAt  string // keep as raw first; parse later
	ContentText  string
	Lang         string
}

func FromHTML(url string, html []byte) (*Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	res := &Result{
		CanonicalURL: util.FirstNonEmpty(
			attr(doc, `link[rel="canonical"]`, "href"),
			meta(doc, `meta[property="og:url"]`, "content"),
			url,
		),
		Title: util.CleanSpace(util.FirstNonEmpty(
			meta(doc, `meta[property="og:title"]`, "content"),
			str(doc, "title"),
			str(doc, "h1"),
		)),
		Author: util.CleanSpace(util.FirstNonEmpty(
			meta(doc, `meta[name="author"]`, "content"),
			meta(doc, `meta[property="article:author"]`, "content"),
		)),
		PublishedAt: util.FirstNonEmpty(
			meta(doc, `meta[property="article:published_time"]`, "content"),
			meta(doc, `meta[name="pubdate"]`, "content"),
			meta(doc, `meta[name="date"]`, "content"),
			attr(doc, `time[datetime]`, "datetime"),
		),
		Lang: util.FirstNonEmpty(
			attr(doc, "html", "lang"),
			"vi",
		),
	}

	// Content: ưu tiên <article>, fallback các selector phổ biến, cuối cùng lấy body text.
	contentSel := firstExisting(doc,
		"article",
		".article__body",
		".article-content",
		".content-detail",
		".detail__content",
		".post-content",
		".entry-content",
	)
	contentText := textFromSelection(doc, contentSel)
	if contentText == "" {
		contentText = textFromSelection(doc, "body")
	}
	res.ContentText = util.CleanSpace(contentText)

	return res, nil
}

func firstExisting(doc *goquery.Document, sels ...string) string {
	for _, s := range sels {
		if doc.Find(s).Length() > 0 {
			return s
		}
	}
	return ""
}

func textFromSelection(doc *goquery.Document, sel string) string {
	if sel == "" {
		return ""
	}
	// lấy text các đoạn p để giảm menu/footer
	var b strings.Builder
	doc.Find(sel).Find("p").Each(func(i int, s *goquery.Selection) {
		t := util.CleanSpace(s.Text())
		if len(t) >= 40 {
			b.WriteString(t)
			b.WriteString("\n")
		}
	})
	out := strings.TrimSpace(b.String())
	if out != "" {
		return out
	}
	return strings.TrimSpace(doc.Find(sel).Text())
}

func str(doc *goquery.Document, sel string) string {
	return strings.TrimSpace(doc.Find(sel).First().Text())
}

func attr(doc *goquery.Document, sel, a string) string {
	v, _ := doc.Find(sel).First().Attr(a)
	return strings.TrimSpace(v)
}

func meta(doc *goquery.Document, sel, a string) string { return attr(doc, sel, a) }
