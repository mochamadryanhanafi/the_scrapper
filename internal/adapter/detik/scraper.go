package detik

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yourmodule/internal/domain"

	"github.com/PuerkitoBio/goquery"
)

type DetikScraper struct {
	client *http.Client
}

func NewDetikScraper(client *http.Client) *DetikScraper {
	return &DetikScraper{client: client}
}

func (d *DetikScraper) Search(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error) {
	fromStr := from.Format("02/01/2006")
	toStr := to.Format("02/01/2006")

	params := url.Values{}
	params.Set("query", query)
	params.Set("fromdatex", fromStr)
	params.Set("todatex", toStr)
	params.Set("result_type", "relevansi")

	urlSearch := fmt.Sprintf("https://www.detik.com/search/searchall?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlSearch, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DetikScraper/1.0)")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var articles []domain.Article

	doc.Find("article").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h3").Text())
		link, _ := s.Find("a").Attr("href")
		summary := strings.TrimSpace(s.Find("p").Text())

		if title != "" && link != "" {
			articles = append(articles, domain.Article{
				Title:   title,
				URL:     link,
				Summary: summary,
			})
		}
	})

	// 🔥 Ambil isi berita tiap artikel
	for i, a := range articles {
		content, err := d.scrapeArticleContent(ctx, a.URL)
		if err != nil {
			// Abaikan error, lanjut ke artikel lain
			fmt.Printf("[warn] gagal ambil konten %s: %v\n", a.URL, err)
			continue
		}
		articles[i].Content = content
	}

	return articles, nil
}

// 🔎 Fungsi untuk ambil isi berita dari halaman artikel
func (d *DetikScraper) scrapeArticleContent(ctx context.Context, articleURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DetikScraper/1.0)")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch article: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Selector umum untuk isi berita Detik
	content := doc.Find("div.detail__body-text").Text()
	content = strings.TrimSpace(content)

	// Jika tidak ditemukan, coba selector alternatif
	if content == "" {
		content = doc.Find("div.detail__body").Text()
		content = strings.TrimSpace(content)
	}

	return content, nil
}
