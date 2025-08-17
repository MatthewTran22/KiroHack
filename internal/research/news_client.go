package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ai-government-consultant/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewsAPIResponse represents the response from News API
type NewsAPIResponse struct {
	Status       string        `json:"status"`
	TotalResults int           `json:"totalResults"`
	Articles     []NewsArticle `json:"articles"`
	Code         string        `json:"code,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// NewsArticle represents a news article from the API
type NewsArticle struct {
	Source      NewsSource `json:"source"`
	Author      string     `json:"author"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	URL         string     `json:"url"`
	URLToImage  string     `json:"urlToImage"`
	PublishedAt string     `json:"publishedAt"`
	Content     string     `json:"content"`
}

// NewsSource represents the source of a news article
type NewsSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HTTPNewsAPIClient implements NewsAPIClient using HTTP requests
type HTTPNewsAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewHTTPNewsAPIClient creates a new HTTP-based News API client
func NewHTTPNewsAPIClient(apiKey, baseURL string) *HTTPNewsAPIClient {
	if baseURL == "" {
		baseURL = "https://newsapi.org/v2"
	}
	
	return &HTTPNewsAPIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchNews searches for news articles related to a query
func (c *HTTPNewsAPIClient) SearchNews(ctx context.Context, query string, options NewsSearchOptions) ([]models.CurrentEvent, error) {
	endpoint := fmt.Sprintf("%s/everything", c.baseURL)
	
	params := url.Values{}
	params.Set("q", query)
	params.Set("apiKey", c.apiKey)
	
	if options.Language != "" {
		params.Set("language", options.Language)
	}
	
	if options.SortBy != "" {
		params.Set("sortBy", options.SortBy)
	} else {
		params.Set("sortBy", "relevancy")
	}
	
	if options.PageSize > 0 {
		params.Set("pageSize", strconv.Itoa(options.PageSize))
	} else {
		params.Set("pageSize", "20")
	}
	
	if options.Page > 0 {
		params.Set("page", strconv.Itoa(options.Page))
	}
	
	if options.From != nil {
		params.Set("from", options.From.Format("2006-01-02"))
	}
	
	if options.To != nil {
		params.Set("to", options.To.Format("2006-01-02"))
	}
	
	if len(options.Domains) > 0 {
		params.Set("domains", strings.Join(options.Domains, ","))
	}
	
	if len(options.ExcludeDomains) > 0 {
		params.Set("excludeDomains", strings.Join(options.ExcludeDomains, ","))
	}
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse NewsAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if apiResponse.Status != "ok" {
		return nil, fmt.Errorf("API error: %s - %s", apiResponse.Code, apiResponse.Message)
	}
	
	return c.convertArticlesToEvents(apiResponse.Articles, query)
}

// GetTopHeadlines retrieves top headlines for a category
func (c *HTTPNewsAPIClient) GetTopHeadlines(ctx context.Context, category string, options NewsSearchOptions) ([]models.CurrentEvent, error) {
	endpoint := fmt.Sprintf("%s/top-headlines", c.baseURL)
	
	params := url.Values{}
	params.Set("apiKey", c.apiKey)
	
	if category != "" {
		params.Set("category", category)
	}
	
	if options.Country != "" {
		params.Set("country", options.Country)
	} else {
		params.Set("country", "us") // Default to US
	}
	
	if options.Language != "" {
		params.Set("language", options.Language)
	}
	
	if options.PageSize > 0 {
		params.Set("pageSize", strconv.Itoa(options.PageSize))
	} else {
		params.Set("pageSize", "20")
	}
	
	if options.Page > 0 {
		params.Set("page", strconv.Itoa(options.Page))
	}
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResponse NewsAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if apiResponse.Status != "ok" {
		return nil, fmt.Errorf("API error: %s - %s", apiResponse.Code, apiResponse.Message)
	}
	
	return c.convertArticlesToEvents(apiResponse.Articles, category)
}

// ValidateAPIKey validates the news API key
func (c *HTTPNewsAPIClient) ValidateAPIKey(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/top-headlines", c.baseURL)
	
	params := url.Values{}
	params.Set("apiKey", c.apiKey)
	params.Set("country", "us")
	params.Set("pageSize", "1")
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// convertArticlesToEvents converts news articles to current events
func (c *HTTPNewsAPIClient) convertArticlesToEvents(articles []NewsArticle, query string) ([]models.CurrentEvent, error) {
	events := make([]models.CurrentEvent, 0, len(articles))
	
	for _, article := range articles {
		// Skip articles with missing essential information
		if article.Title == "" || article.URL == "" {
			continue
		}
		
		publishedAt, err := time.Parse("2006-01-02T15:04:05Z", article.PublishedAt)
		if err != nil {
			// Try alternative format
			publishedAt, err = time.Parse("2006-01-02T15:04:05.000Z", article.PublishedAt)
			if err != nil {
				// Use current time if parsing fails
				publishedAt = time.Now()
			}
		}
		
		// Calculate relevance based on query match (simple implementation)
		relevance := c.calculateRelevance(article, query)
		
		// Extract category from source or use general
		category := c.extractCategory(article.Source.Name)
		
		// Generate tags from title and description
		tags := c.extractTags(article.Title + " " + article.Description)
		
		event := models.CurrentEvent{
			ID:          primitive.NewObjectID(),
			Title:       article.Title,
			Description: article.Description,
			Source:      article.Source.Name,
			URL:         article.URL,
			PublishedAt: publishedAt,
			Relevance:   relevance,
			Category:    category,
			Tags:        tags,
			Content:     article.Content,
			Author:      article.Author,
			Language:    "en", // Default to English, could be enhanced
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// calculateRelevance calculates the relevance of an article to a query
func (c *HTTPNewsAPIClient) calculateRelevance(article NewsArticle, query string) float64 {
	// Simple relevance calculation based on keyword matching
	queryWords := strings.Fields(strings.ToLower(query))
	titleWords := strings.Fields(strings.ToLower(article.Title))
	descWords := strings.Fields(strings.ToLower(article.Description))
	
	matches := 0
	totalWords := len(queryWords)
	
	for _, qWord := range queryWords {
		for _, tWord := range titleWords {
			if strings.Contains(tWord, qWord) {
				matches += 2 // Title matches are weighted more
				break
			}
		}
		for _, dWord := range descWords {
			if strings.Contains(dWord, qWord) {
				matches++
				break
			}
		}
	}
	
	if totalWords == 0 {
		return 0.5 // Default relevance
	}
	
	relevance := float64(matches) / float64(totalWords*2) // Normalize to 0-1
	if relevance > 1.0 {
		relevance = 1.0
	}
	
	return relevance
}

// extractCategory extracts category from source name
func (c *HTTPNewsAPIClient) extractCategory(sourceName string) string {
	sourceName = strings.ToLower(sourceName)
	
	// Simple category mapping based on source names
	if strings.Contains(sourceName, "tech") || strings.Contains(sourceName, "wired") {
		return "technology"
	}
	if strings.Contains(sourceName, "business") || strings.Contains(sourceName, "financial") {
		return "business"
	}
	if strings.Contains(sourceName, "health") || strings.Contains(sourceName, "medical") {
		return "health"
	}
	if strings.Contains(sourceName, "science") {
		return "science"
	}
	if strings.Contains(sourceName, "politics") || strings.Contains(sourceName, "government") {
		return "politics"
	}
	
	return "general"
}

// extractTags extracts tags from text
func (c *HTTPNewsAPIClient) extractTags(text string) []string {
	// Simple tag extraction - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(text))
	tagMap := make(map[string]bool)
	
	// Common government/policy related keywords
	keywords := []string{
		"policy", "government", "regulation", "law", "bill", "congress",
		"senate", "house", "federal", "state", "local", "public",
		"administration", "department", "agency", "budget", "tax",
		"healthcare", "education", "defense", "security", "environment",
		"economy", "trade", "immigration", "infrastructure", "technology",
	}
	
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")
		
		// Check if word is a relevant keyword
		for _, keyword := range keywords {
			if strings.Contains(word, keyword) {
				tagMap[keyword] = true
			}
		}
	}
	
	// Convert map to slice
	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	
	// Limit to 10 tags
	if len(tags) > 10 {
		tags = tags[:10]
	}
	
	return tags
}