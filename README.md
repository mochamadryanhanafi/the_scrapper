# The Scrapper

This is a web scraping application that extracts articles from various news sources.

## Running the Application

1.  Install the dependencies:
    ```bash
    go mod tidy
    ```

2.  Run the application:
    ```bash
    go run cmd/scraper-cli/main.go
    ```

## API Endpoint

### POST /scrape

Scrapes articles from a specified source.

**Request Body:**

```json
{
  "source": "<source>",
  "query": "<query>",
  "start_date": "<YYYY-MM-DD>",
  "end_date": "<YYYY-MM-DD>"
}
```

**Parameters:**

*   `source`: The news source to scrape. Valid sources are `detik`, `kompas`, and `liputan6`.
*   `query`: The search query.
*   `start_date`: The start date for the search range (YYYY-MM-DD).
*   `end_date`: The end date for the search range (YYYY-MM-DD).

**Example:**

```bash
curl -X POST http://localhost:8080/scrape \
-H "Content-Type: application/json" \
-d 
{
  "source": "liputan6",
  "query": "ekonomi jokowi",
  "start_date": "2017-01-01",
  "end_date": "2017-01-30"
}
```
