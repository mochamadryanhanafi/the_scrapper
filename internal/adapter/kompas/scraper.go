package kompas

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"

	"the_scrapper/internal/domain"
)

type KompasScraper struct {
	client *http.Client
}

func NewKompasScraper(client *http.Client) *KompasScraper {
	return &KompasScraper{client: client}
}

func findFirstExecutable(executables ...string) string {
	for _, executable := range executables {
		path, err := exec.LookPath(executable)
		if err == nil {
			return path
		}
	}
	return ""
}

func (k *KompasScraper) Search(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error) {
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	params := url.Values{}
	params.Set("q", query)
	params.Set("site_id", "all")
	params.Set("start_date", fromStr)
	params.Set("end_date", toStr)

	urlSearch := fmt.Sprintf("https://search.kompas.com/search?%s", params.Encode())

	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.Flag("headless", true))

	if execPath := findFirstExecutable("brave", "brave-browser", "chromium-browser"); execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	var htmlBody string
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(urlSearch),
		chromedp.WaitVisible("div.gsc-webResult", chromedp.ByQuery),
		chromedp.OuterHTML("body", &htmlBody),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp failed to execute search task: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var articles []domain.Article

	doc.Find("div.gsc-webResult").Each(func(i int, s *goquery.Selection) {
		titleEl := s.Find("a.gs-title")
		title := strings.TrimSpace(titleEl.Text())
		link, _ := titleEl.Attr("href")

		summary := strings.TrimSpace(s.Find("div.gs-bidi-start-align").Text())

		if title != "" && link != "" {
			articles = append(articles, domain.Article{
				Title:   title,
				URL:     link,
				Summary: summary,
			})
		}
	})

	if len(articles) == 0 {
		log.Println("[INFO] kompas: No articles found after chromedp execution.")
		return []domain.Article{}, nil
	}

	for i, a := range articles {
		time.Sleep(300 * time.Millisecond)

		content, err := k.scrapeArticleContent(taskCtx, a.URL)
		if err != nil {
			fmt.Printf("[warn] kompas: failed to fetch content for %s: %v\n", a.URL, err)
			continue
		}
		articles[i].Content = content
	}

	return articles, nil
}

func (k *KompasScraper) scrapeArticleContent(ctx context.Context, articleURL string) (string, error) {
	if strings.Contains(articleURL, "video.kompas.com") || strings.Contains(articleURL, "foto.kompas.com") {
		return "", fmt.Errorf("link is a video/photo, not a text article")
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.Flag("headless", true))

	if execPath := findFirstExecutable("brave", "brave-browser", "chromium-browser"); execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var contentHTML string
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(articleURL),
		chromedp.WaitVisible("div.read__content", chromedp.ByQuery),
		chromedp.OuterHTML("div.read__content", &contentHTML, chromedp.ByQuery),
	)

	if err != nil {
		return "", fmt.Errorf("chromedp failed to retrieve article content: %w", err)
	}

	if contentHTML == "" {
		return "", fmt.Errorf("article content is empty (selector 'div.read__content' not found by chromedp)")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(contentHTML))
	if err != nil {
		return "", fmt.Errorf("failed to parse article content HTML: %w", err)
	}

	doc.Find("strong:contains('Baca juga:')").Each(func(i int, s *goquery.Selection) {
		s.Parent().Remove()
	})
	doc.Find("strong:contains('Baca juga :')").Each(func(i int, s *goquery.Selection) {
		s.Parent().Remove()
	})

	var contentBuilder strings.Builder
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		contentBuilder.WriteString(strings.TrimSpace(s.Text()) + "\n")
	})

	content := strings.TrimSpace(contentBuilder.String())

	if content == "" {
		return "", fmt.Errorf("could not find article content text after goquery parsing")
	}

	return content, nil
}
