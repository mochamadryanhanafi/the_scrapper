package usecase

import "errors"

var (
	ErrInvalidDateRange = errors.New("invalid date range: 'to' date must be after 'from' date")
)
