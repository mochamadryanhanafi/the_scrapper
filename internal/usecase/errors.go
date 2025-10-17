package usecase

import "errors"

var (
	ErrInvalidDateRange = errors.New("invalid date range: 'to' date is before 'from' date")
)
