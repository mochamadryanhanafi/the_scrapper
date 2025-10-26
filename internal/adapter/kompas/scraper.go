package kompas

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

// KompasScraper adalah implementasi untuk Kompas
type KompasScraper struct {
	client *http.Client
}

// NewKompasScraper membuat instance scraper Kompas baru
func NewKompasScraper(client *http.Client) *KompasScraper {
	return &KompasScraper{client: client}
}

// Search mengimplementasikan logika scraping untuk search.kompas.com
func (k *KompasScraper) Search(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error) {
	// Format tanggal sesuai URL: YYYY-MM-DD
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	// Buat parameter URL
	params := url.Values{}
	params.Set("q", query)
	params.Set("site_id", "all")
	params.Set("start_date", fromStr)
	params.Set("end_date", toStr)

	urlSearch := fmt.Sprintf("https://search.kompas.com/search?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlSearch, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ScraperBot/1.0; +http://yourwebsite.com/bot)")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gagal mengambil halaman pencarian kompas: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var articles []domain.Article

	// --- PERUBAHAN ---
	// Selector diperbarui untuk mencocokkan struktur Google Custom Search (gsc)
	// yang digunakan oleh Kompas.
	doc.Find("div.gsc-webResult").Each(func(i int, s *goquery.Selection) {
		titleEl := s.Find("a.gs-title")
		title := strings.TrimSpace(titleEl.Text())
		link, _ := titleEl.Attr("href") // URL artikel

		// Mengambil ringkasan/snippet
		summary := strings.TrimSpace(s.Find("div.gs-bidi-start-align").Text())

		if title != "" && link != "" {
			articles = append(articles, domain.Article{
				Title:   title,
				URL:     link,
				Summary: summary,
			})
		}
	})
	// --- AKHIR PERUBAHAN ---

	if len(articles) == 0 {
		return []domain.Article{}, nil // Tidak ada artikel ditemukan, tapi bukan error
	}

	// Ambil konten lengkap untuk setiap artikel
	for i, a := range articles {
		// Beri sedikit jeda agar tidak membanjiri server Kompas
		time.Sleep(300 * time.Millisecond)

		content, err := k.scrapeArticleContent(ctx, a.URL)
		if err != nil {
			fmt.Printf("[warn] kompas: gagal ambil konten %s: %v\n", a.URL, err)
			continue // Lanjut ke artikel berikutnya, konten akan kosong
		}
		articles[i].Content = content
	}

	return articles, nil
}

// scrapeArticleContent mengambil konten teks dari halaman artikel Kompas
func (k *KompasScraper) scrapeArticleContent(ctx context.Context, articleURL string) (string, error) {
	if strings.Contains(articleURL, "video.kompas.com") || strings.Contains(articleURL, "foto.kompas.com") {
		return "", fmt.Errorf("link adalah video/foto, bukan artikel teks")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ScraperBot/1.0)")

	resp, err := k.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gagal mengambil artikel: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// --- PERUBAHAN ---
	// Selector untuk konten artikel diperbarui dan dibuat lebih tangguh
	// dengan menggabungkan paragraf dan membersihkan "Baca juga:".

	contentContainer := doc.Find("div.read__content")

	// Hapus elemen yang tidak diinginkan seperti "Baca juga"
	contentContainer.Find("strong:contains('Baca juga:')").Each(func(i int, s *goquery.Selection) {
		s.Parent().Remove() // Hapus <p> yang berisi "Baca juga"
	})
	contentContainer.Find("strong:contains('Baca juga :')").Each(func(i int, s *goquery.Selection) {
		s.Parent().Remove()
	})

	// Gabungkan teks dari semua tag <p> di dalam div konten
	var contentBuilder strings.Builder
	contentContainer.Find("p").Each(func(i int, s *goquery.Selection) {
		contentBuilder.WriteString(strings.TrimSpace(s.Text()) + "\n")
	})

	content := strings.TrimSpace(contentBuilder.String())
	// --- AKHIR PERUBAHAN ---

	if content == "" {
		return "", fmt.Errorf("tidak dapat menemukan konten artikel (selector 'div.read__content' tidak cocok atau kosong)")
	}

	return content, nil
}
