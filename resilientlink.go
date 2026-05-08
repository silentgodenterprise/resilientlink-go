// Package resilientlink provides the official Go client for the ResilientLink
// Web Scraping API.
//
// Usage:
//
//	client := resilientlink.New("YOUR_API_KEY")
//	result, err := client.Scrape("https://example.com", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Data.Title)
package resilientlink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://resilientlink-api.vercel.app"
	defaultTimeout = 60 * time.Second
	sdkVersion     = "1.0.0"
)

// ── ERROR ─────────────────────────────────────────────────────────────────────

// APIError is returned when the ResilientLink API responds with an error status.
type APIError struct {
	Message    string
	StatusCode int
	Body       map[string]any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("resilientlink: HTTP %d — %s", e.StatusCode, e.Message)
}

// ── OPTIONS ───────────────────────────────────────────────────────────────────

// ScrapeOptions configures an individual scrape request.
// All fields are optional. Zero values are omitted from the request.
type ScrapeOptions struct {
	// ReturnHTML returns raw HTML in the response.
	ReturnHTML bool `json:"return_html,omitempty"`

	// Screenshot returns a base64-encoded PNG screenshot (Pro/Enterprise only).
	Screenshot bool `json:"screenshot,omitempty"`

	// FullPage captures a full-page screenshot (default: true when Screenshot=true).
	FullPage *bool `json:"full_page,omitempty"`

	// PDF returns a base64-encoded PDF (Pro/Enterprise only).
	PDF bool `json:"pdf,omitempty"`

	// PDFFormat sets the PDF page format: "A4", "Letter", etc. (default: "A4").
	PDFFormat string `json:"pdf_format,omitempty"`

	// PDFBackground includes page background in the PDF (default: true).
	PDFBackground *bool `json:"pdf_background,omitempty"`

	// PDFLandscape renders the PDF in landscape orientation.
	PDFLandscape bool `json:"pdf_landscape,omitempty"`

	// BypassCache forces a fresh scrape, ignoring cached results.
	BypassCache bool `json:"bypass_cache,omitempty"`

	// JSRender enables JavaScript rendering via headless Chrome (Pro/Enterprise).
	JSRender *bool `json:"js_render,omitempty"`

	// WaitForSelector waits for a CSS selector to appear before scraping.
	WaitForSelector string `json:"wait_for_selector,omitempty"`

	// WaitUntil sets the Puppeteer page.waitUntil condition:
	// "networkidle0", "networkidle2", "load", "domcontentloaded".
	WaitUntil string `json:"wait_until,omitempty"`

	// WaitMs adds an extra delay in milliseconds before scraping.
	WaitMs int `json:"wait_ms,omitempty"`

	// CustomHeaders forwards these HTTP headers with the outbound scrape request.
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// CustomJS executes this JavaScript on the page and returns the result (Enterprise).
	CustomJS string `json:"custom_js,omitempty"`

	// ReturnCookies includes page cookies in the response.
	ReturnCookies bool `json:"return_cookies,omitempty"`

	// BlockResources sets resource types to block: ["media", "font", "image", ...].
	BlockResources []string `json:"block_resources,omitempty"`

	// Timeout is the per-request timeout in milliseconds (max 60000).
	Timeout int `json:"timeout,omitempty"`
}

// ── RESPONSE ─────────────────────────────────────────────────────────────────

// ScrapeResult is the structured response from a successful scrape.
type ScrapeResult struct {
	Success      bool       `json:"success"`
	Cached       bool       `json:"cached"`
	Tier         string     `json:"tier"`
	Engine       string     `json:"engine"`
	ResponseTime int        `json:"responseTime"`
	StatusCode   int        `json:"statusCode"`
	Data         *ScrapData `json:"data"`
	HTML         string     `json:"html,omitempty"`
	Screenshot   string     `json:"screenshot,omitempty"`
	PDF          string     `json:"pdf,omitempty"`
	Error        string     `json:"error,omitempty"`
	Status       string     `json:"status,omitempty"`
}

// ScrapData contains the extracted metadata from the scraped page.
type ScrapData struct {
	URL         string         `json:"url"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Image       string         `json:"image"`
	Domain      string         `json:"domain"`
	Language    string         `json:"language"`
	OG          map[string]any `json:"og"`
	Twitter     map[string]any `json:"twitter"`
	Content     map[string]any `json:"content"`
	SEO         map[string]any `json:"seo"`
	JsonLD      []any          `json:"jsonLd"`
	Images      []any          `json:"images"`
	ScrapedAt   string         `json:"scrapedAt"`
}

// ── CLIENT ────────────────────────────────────────────────────────────────────

// Client is the ResilientLink API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New creates a new ResilientLink client with the given API key.
// Optional functional options can override defaults.
func New(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		panic("resilientlink: apiKey is required")
	}

	c := &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(defaultBaseURL, "/"),
		http:    &http.Client{Timeout: defaultTimeout},
	}

	for _, o := range opts {
		o(c)
	}
	return c
}

// Option is a functional option for Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// ── SCRAPE ────────────────────────────────────────────────────────────────────

// Scrape scrapes the given URL and returns structured metadata.
// opts may be nil to use all defaults.
//
// Example:
//
//	result, err := client.Scrape("https://example.com", nil)
//	result, err := client.Scrape("https://example.com", &resilientlink.ScrapeOptions{
//	    Screenshot: true,
//	    BypassCache: true,
//	})
func (c *Client) Scrape(url string, opts *ScrapeOptions) (*ScrapeResult, error) {
	return c.ScrapeWithContext(context.Background(), url, opts)
}

// ScrapeWithContext is the context-aware version of Scrape.
func (c *Client) ScrapeWithContext(ctx context.Context, url string, opts *ScrapeOptions) (*ScrapeResult, error) {
	if url == "" {
		return nil, fmt.Errorf("resilientlink: url is required")
	}

	type requestBody struct {
		URL string `json:"url"`
		*ScrapeOptions
	}

	payload := requestBody{URL: url, ScrapeOptions: opts}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("resilientlink: marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/scrape", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("resilientlink: build request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "resilientlink-go/"+sdkVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resilientlink: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("resilientlink: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		_ = json.Unmarshal(body, &errBody)
		msg, _ := errBody["error"].(string)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, &APIError{Message: msg, StatusCode: resp.StatusCode, Body: errBody}
	}

	var result ScrapeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("resilientlink: decode response: %w", err)
	}

	return &result, nil
}
