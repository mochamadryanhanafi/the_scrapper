package liputan6

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"the_scrapper/internal/domain"

	"github.com/PuerkitoBio/goquery"
)

type Liputan6Scraper struct {
	client *http.Client
}

func NewLiputan6Scraper(client *http.Client) *Liputan6Scraper {
	return &Liputan6Scraper{client: client}
}

func (l *Liputan6Scraper) Search(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error) {
	fromStr := from.Format("02/01/2006")
	toStr := to.Format("02/01/2006")

	params := url.Values{}
	params.Set("q", query)
	params.Set("from_date", fromStr)
	params.Set("to_date", toStr)
	params.Set("order", "latest")
	params.Set("type", "all")

	urlSearch := fmt.Sprintf("https://www.liputan6.com/search?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlSearch, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Liputan6Scraper/1.0)")

	resp, err := l.client.Do(req)
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
	doc.Find("article.articles--iridescent-list--item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h4.articles--iridescent-list--text-item__title").Text())
		link, _ := s.Find("a").Attr("href")
		summary := strings.TrimSpace(s.Find("p.articles--iridescent-list--text-item__summary").Text())

		if title != "" && link != "" {
			articles = append(articles, domain.Article{
				Title:   title,
				URL:     link,
				Summary: summary,
			})
		}
	})

	for i, a := range articles {
		content, err := l.scrapeArticleContent(ctx, a.URL)
		if err != nil {
			fmt.Printf("[warn] gagal ambil konten %s: %v\n", a.URL, err)
			continue
		}
		articles[i].Content = content
	}

	return articles, nil
}

func (l *Liputan6Scraper) scrapeArticleContent(ctx context.Context, articleURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-agent", "Mozilla/5.0 (compatible; Liputan6Scraper/1.0)")

	resp, err := l.client.Do(req)
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

	content := strings.TrimSpace(doc.Find("div.article-content-body__item-content").Text())
	return content, nil
}
