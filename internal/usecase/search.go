package usecase

import (
	"context"
	"time"

	"the_scrapper/internal/domain"
	"the_scrapper/internal/repository"
)

type SearchService struct {
	scraper repository.Scraper
}

func NewSearchService(scraper repository.Scraper) *SearchService {
	return &SearchService{scraper: scraper}
}

func (s *SearchService) Execute(ctx context.Context, query string, from, to time.Time) ([]domain.Article, error) {
	if to.Before(from) {
		return nil, ErrInvalidDateRange
	}

	results, err := s.scraper.Search(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	return results, nil
}
