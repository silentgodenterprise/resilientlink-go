# ResilientLink Go SDK

Official Go client for the [ResilientLink](https://resilientlink.silentgode.com) Web Scraping API.

## Installation

```bash
go get github.com/resilientlink/resilientlink-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/resilientlink/resilientlink-go/resilientlink"
)

func main() {
    client := resilientlink.New("YOUR_API_KEY")

    result, err := client.Scrape("https://example.com", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Data.Title)        // "Example Domain"
    fmt.Println(result.Data.Description)  // meta description
    fmt.Println(result.Data.Image)        // OG image URL
}
```

## Options

```go
boolTrue := true

result, err := client.Scrape("https://example.com", &resilientlink.ScrapeOptions{
    ReturnHTML:        true,
    Screenshot:        true,          // base64 PNG (Pro/Enterprise)
    PDF:               true,          // base64 PDF (Pro/Enterprise)
    PDFFormat:         "A4",
    BypassCache:       true,          // force fresh scrape
    JSRender:          &boolTrue,     // JS rendering (Pro/Enterprise)
    WaitForSelector:   "#app",
    WaitMs:            2000,
    CustomHeaders:     map[string]string{"Accept-Language": "en-US"},
    Timeout:           30000,         // ms (max 60000)
})
```

## With Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
defer cancel()

result, err := client.ScrapeWithContext(ctx, "https://example.com", nil)
```

## Response

```go
result.Success         // bool
result.Cached          // bool
result.Tier            // "free" | "starter" | "pro" | "enterprise"
result.ResponseTime    // int (ms)
result.Data.Title      // string
result.Data.Description
result.Data.Image
result.Data.Domain
result.Data.OG         // map[string]any
result.Data.Content    // map[string]any {"wordCount": 423, ...}
result.Data.ScrapedAt  // string (ISO 8601)
result.Screenshot      // base64 string (if requested)
result.PDF             // base64 string (if requested)
```

## Error Handling

```go
result, err := client.Scrape("https://example.com", nil)
if err != nil {
    var apiErr *resilientlink.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode) // 429 = rate limit, 401 = bad key
        fmt.Println(apiErr.Message)
        fmt.Println(apiErr.Body)
    } else {
        fmt.Println("network error:", err)
    }
}
```

## Options

```go
client := resilientlink.New("YOUR_API_KEY",
    resilientlink.WithBaseURL("https://your-api.example.com"), // override base URL
    resilientlink.WithTimeout(45 * time.Second),               // override timeout
)
```

## Get Your API Key

Sign up at [resilientlink](https://resilientlink.silentgode.com) → Dashboard → API Key.
