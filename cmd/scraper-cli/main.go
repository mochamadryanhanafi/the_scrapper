package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	mongoAdapter "the_scrapper/internal/adapter/mongo"
	"the_scrapper/internal/handler/httpapi" // Paket handler baru

	"github.com/joho/godotenv"
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

	if mongoURI == "" || dbName == "" {
		log.Fatal("❌ Pastikan variabel MONGO_URI dan DB_NAME diatur di file .env")
	}

	// === Koneksi MongoDB ===
	// Menggunakan konteks dengan timeout untuk koneksi awal
	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongoAdapter.NewClient(connectCtx, mongoURI)
	if err != nil {
		log.Fatalf("❌ Gagal koneksi MongoDB: %v", err)
	}

	// Konteks utama untuk aplikasi
	appCtx := context.Background()
	defer func() {
		if err := mongoClient.Disconnect(appCtx); err != nil {
			log.Printf("⚠️  Gagal disconnect dari MongoDB: %v", err)
		}
	}()

	// Dapatkan database
	db := mongoClient.Database(dbName)

	// === Inisialisasi Handler API ===
	// Handler akan mengelola scraper factory dan dependensi lainnya
	scrapeHandler := httpapi.NewScrapeHandler(db)

	// === Routes ===
	http.HandleFunc("/scrape", scrapeHandler.HandleScrape)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Menjalankan API server di http://localhost:%s", port)

	// === Mulai Server ===
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Server gagal berjalan: %v", err)
	}
}
