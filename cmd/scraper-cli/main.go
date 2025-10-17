package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"yourmodule/internal/adapter/detik"
	"yourmodule/internal/adapter/httpclient"
	"yourmodule/internal/usecase"
)

func main() {
	query := "ekonomi jokowi"
	from := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC)

	client := httpclient.NewClient(10 * time.Second)
	scraper := detik.NewDetikScraper(client)
	service := usecase.NewSearchService(scraper)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	articles, err := service.Execute(ctx, query, from, to)
	if err != nil {
		log.Fatalf("scraping failed: %v", err)
	}

	if len(articles) == 0 {
		fmt.Println("Tidak ada artikel ditemukan atau gagal scraping.")
		return
	}

	fmt.Println("=== Hasil Pencarian ===")
	for _, a := range articles {
		fmt.Printf("[%s] %s\n%s\n\n", a.Date.Format("02 Jan 2006"), a.Title, a.URL)
	}
}
