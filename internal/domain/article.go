package domain

import "time"

type Article struct {
	Title   string
	URL     string
	Summary string
	Content string
	Date    time.Time
}
