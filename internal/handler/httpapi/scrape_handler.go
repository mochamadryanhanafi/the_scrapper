package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"the_scrapper/internal/adapter/detik"
	"the_scrapper/internal/adapter/httpclient"
	"the_scrapper/internal/adapter/kompas"
	"the_scrapper/internal/adapter/liputan6"
	"the_scrapper/internal/domain"
	"the_scrapper/internal/repository"
	"the_scrapper/internal/usecase"
)

// ScrapeRequest mendefinisikan body JSON untuk request API
type ScrapeRequest struct {
	Source    string `json:"source"`
	Query     string `json:"query"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// ScrapeHandler mengelola dependensi untuk handler API
type ScrapeHandler struct {
	db             *mongo.Database
	scraperFactory map[string]repository.Scraper
}

// NewScrapeHandler membuat handler baru dan pabrik scraper
func NewScrapeHandler(db *mongo.Database) *ScrapeHandler {
	httpClient := httpclient.NewHTTPClient()

	// Pabrik ini memetakan nama source ke implementasi scraper-nya
	factory := map[string]repository.Scraper{
		"detik":    detik.NewDetikScraper(httpClient),
		"kompas":   kompas.NewKompasScraper(httpClient),
		"liputan6": liputan6.NewLiputan6Scraper(httpClient),
	}

	return &ScrapeHandler{
		db:             db,
		scraperFactory: factory,
	}
}

// HandleScrape adalah method handler utama untuk endpoint /scrape
func (h *ScrapeHandler) HandleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Validasi Source
	scraper, ok := h.scraperFactory[req.Source]
	if !ok {
		http.Error(w, "Invalid source. Must be 'detik', 'kompas', or 'liputan6'", http.StatusBadRequest)
		return
	}

	// 2. Validasi Tanggal
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		http.Error(w, "Invalid start_date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		http.Error(w, "Invalid end_date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// 3. Validasi Query
	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// --- PERBAIKAN ---
	// Dapatkan nama koleksi secara dinamis berdasarkan sumber (source)
	// Ini akan membuat koleksi seperti "detik_articles" dan "kompas_articles"
	// alih-alih menyimpan semuanya di satu tempat.
	collectionName := fmt.Sprintf("%s_articles", req.Source)
	collection := h.db.Collection(collectionName)
	// --- AKHIR PERBAIKAN ---

	// 4. Eksekusi Usecase
	service := usecase.NewSearchService(scraper)

	// Beri timeout 5 menit untuk setiap request scraping
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	log.Printf("🚀 Memulai scraping: Source=%s, Query=%s, Range=%s to %s, Collection=%s",
		req.Source, req.Query, req.StartDate, req.EndDate, collectionName)

	// API akan scrape seluruh rentang tanggal sekaligus (bukan per hari)
	articles, err := service.Execute(ctx, req.Query, startDate, endDate)
	if err != nil {
		log.Printf("❌ Gagal scraping: %v", err)
		http.Error(w, fmt.Sprintf("Scraping failed: %v", err), http.StatusInternalServerError)
		return
	}

	if len(articles) == 0 {
		log.Printf("ℹ️ Tidak ada artikel ditemukan untuk query: %s", req.Query)
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message":  "Scraping successful, 0 articles found.",
			"articles": []domain.Article{},
		})
		return
	}

	log.Printf("✅ %d artikel ditemukan, menyimpan ke MongoDB...", len(articles))

	// 5. Simpan ke DB
	if err := saveArticles(ctx, collection, articles); err != nil {
		log.Printf("❌ Gagal menyimpan artikel: %v", err)
		http.Error(w, "Failed to save articles to DB", http.StatusInternalServerError)
		return
	}

	log.Printf("💾 Artikel berhasil disimpan.")
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":  fmt.Sprintf("Scraping successful, %d articles saved.", len(articles)),
		"articles": articles,
	})
}

// saveArticles menyimpan daftar artikel ke MongoDB (dipindahkan dari main.go lama)
func saveArticles(ctx context.Context, collection *mongo.Collection, articles []domain.Article) error {
	docs := make([]interface{}, len(articles))
	for i, article := range articles {
		docs[i] = article
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := collection.InsertMany(ctx, docs, opts)
	if err != nil {
		// Ini menangani error jika ada duplikat (bukan masalah besar)
		// Tapi jika error lain, kita kembalikan
		if mongo.IsDuplicateKeyError(err) {
			log.Println("ℹ️  Beberapa artikel duplikat dilewati.")
			return nil
		}
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}

// writeJSONResponse adalah helper untuk mengirim balasan JSON
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
