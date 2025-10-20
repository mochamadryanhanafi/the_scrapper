package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"

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

	// === Konfigurasi Scraper ===
	query := "ekonomi jokowi"
	from := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC)

	// === Inisialisasi konteks ===
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// === Inisialisasi scraper ===
	httpClient := httpclient.NewHTTPClient()
	scraper := detik.NewDetikScraper(httpClient)
	service := usecase.NewSearchService(scraper)

	// === Koneksi MongoDB ===
	mongoClient, err := mongoAdapter.NewClient(ctx, mongoURI)
	if err != nil {
		log.Fatalf("❌ Gagal koneksi MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("⚠️  Gagal disconnect dari MongoDB: %v", err)
		}
	}()

	// === Jalankan scraping ===
	articles, err := service.Execute(ctx, query, from, to)
	if err != nil {
		log.Fatalf("❌ Scraping gagal: %v", err)
	}

	if len(articles) == 0 {
		fmt.Println("⚠️  Tidak ada artikel ditemukan.")
		return
	}

	fmt.Printf("✅ Ditemukan %d artikel. Menyimpan ke MongoDB...\n", len(articles))

	// === Simpan ke MongoDB ===
	collection := mongoClient.Database(dbName).Collection(collectionName)
	if err := saveArticles(ctx, collection, articles); err != nil {
		log.Fatalf("❌ Gagal menyimpan artikel: %v", err)
	}

	fmt.Println("🎉 Selesai! Semua artikel tersimpan di MongoDB.")
}

// saveArticles menyimpan daftar artikel ke MongoDB
func saveArticles(ctx context.Context, collection *mongoDriver.Collection, articles []domain.Article) error {
	docs := make([]interface{}, len(articles))
	for i, article := range articles {
		docs[i] = article
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	return nil
}
