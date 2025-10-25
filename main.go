package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"the_scrapper/internal/adapter/detik"
	"the_scrapper/internal/adapter/httpclient"
	mongoAdapter "the_scrapper/internal/adapter/mongo"
	"the_scrapper/internal/domain"
	"the_scrapper/internal/usecase"
)

// loadEnv memuat variabel lingkungan dari file .env
func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  File .env tidak ditemukan, menggunakan variabel lingkungan dari sistem.")
	}
}

func main() {
	loadEnv()

	// === Konfigurasi MongoDB ===
	mongoURI := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")
	collectionName := os.Getenv("COLLECTION_NAME")

	if mongoURI == "" || dbName == "" || collectionName == "" {
		log.Fatal("❌ Pastikan variabel MONGO_URI, DB_NAME, dan COLLECTION_NAME diatur di file .env")
	}

	// === Konfigurasi Pencarian ===
	query := "ekonomi jokowi"
	startDate := time.Date(2015, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2015, time.January, 30, 0, 0, 0, 0, time.UTC)

	// === Inisialisasi HTTP Client dan Scraper ===
	httpClient := httpclient.NewHTTPClient()
	scraper := detik.NewDetikScraper(httpClient)
	service := usecase.NewSearchService(scraper)

	// === Koneksi MongoDB ===
	ctx := context.Background()
	mongoClient, err := mongoAdapter.NewClient(ctx, mongoURI)
	if err != nil {
		log.Fatalf("❌ Gagal koneksi MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("⚠️  Gagal disconnect dari MongoDB: %v", err)
		}
	}()
	collection := mongoClient.Database(dbName).Collection(collectionName)

	fmt.Println("🚀 Memulai scraping otomatis untuk 1–30 Januari 2015...")

	// === Loop scraping per hari ===
	for current := startDate; !current.After(endDate); current = current.AddDate(0, 0, 1) {
		fmt.Printf("\n📅 [%s] Memulai scraping...\n", current.Format("02-01-2006"))

		dayCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)

		var articles []domain.Article
		var scrapeErr error

		// Retry ringan 2x jika gagal koneksi
		for attempt := 1; attempt <= 2; attempt++ {
			articles, scrapeErr = service.Execute(dayCtx, query, current, current)
			if scrapeErr == nil {
				break
			}
			log.Printf("⚠️  Percobaan %d gagal (%v), mencoba ulang...\n", attempt, scrapeErr)
			time.Sleep(3 * time.Second)
		}
		cancel()

		if scrapeErr != nil {
			log.Printf("❌ Gagal scraping tanggal %s setelah 2 percobaan.\n", current.Format("02-01-2006"))
			continue
		}

		if len(articles) == 0 {
			fmt.Printf("ℹ️ Tidak ada artikel ditemukan pada %s\n", current.Format("02-01-2006"))
			continue
		}

		fmt.Printf("✅ %d artikel ditemukan pada %s, menyimpan ke MongoDB...\n",
			len(articles), current.Format("02-01-2006"))

		if err := saveArticles(ctx, collection, articles); err != nil {
			log.Printf("❌ Gagal menyimpan artikel tanggal %s: %v\n", current.Format("02-01-2006"), err)
		} else {
			fmt.Printf("💾 Artikel tanggal %s berhasil disimpan.\n", current.Format("02-01-2006"))
		}

		// Delay kecil antar hari agar tidak dianggap bot agresif
		time.Sleep(5 * time.Second)
	}

	fmt.Println("\n🎉 Scraping selesai untuk periode 1–30 Januari 2015.")
}

// saveArticles menyimpan daftar artikel ke MongoDB
func saveArticles(ctx context.Context, collection *mongoDriver.Collection, articles []domain.Article) error {
	docs := make([]interface{}, len(articles))
	for i, article := range articles {
		docs[i] = article
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := collection.InsertMany(ctx, docs, opts)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}
