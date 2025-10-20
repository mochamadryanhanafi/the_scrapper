package repository

import (
	"context"
	"time"

	"the_scrapper/internal/domain"
)

type Scraper interface {
	Search(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error)
}
