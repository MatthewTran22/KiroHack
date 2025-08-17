package research

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPNewsAPIClient(t *testing.T) {
	client := NewHTTPNewsAPIClient("test-api-key", "")
	
	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "https://newsapi.org/v2", client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestHTTPNewsAPIClient_SearchNews(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/everything")
		assert.Equal(t, "healthcare policy", r.URL.Query().Get("q"))
		assert.Equal(t, "test-api-key", r.URL.Query().Get("apiKey"))
		assert.Equal(t, "en", r.URL.Query().Get("language"))
		assert.Equal(t, "relevancy", r.URL.Query().Get("sortBy"))
		assert.Equal(t, "10", r.URL.Query().Get("pageSize"))
		
		// Mock response
		response := NewsAPIResponse{
			Status:       "ok",
			TotalResults: 2,
			Articles: []NewsArticle{
				{
					Source: NewsSource{
						ID:   "reuters",
						Name: "Reuters",
					},
					Author:      "John Doe",
					Title:       "Healthcare Policy Reform Announced",
					Description: "Government announces new healthcare policy reforms",
					URL:         "https://example.com/article1",
					URLToImage:  "https://example.com/image1.jpg",
					PublishedAt: "2024-01-15T10:30:00Z",
					Content:     "Full article content about healthcare policy reforms...",
				},
				{
					Source: NewsSource{
						ID:   "bbc-news",
						Name: "BBC News",
					},
					Author:      "Jane Smith",
					Title:       "Policy Implementation Challenges",
					Description: "Analysis of challenges in implementing new policies",
					URL:         "https://example.com/article2",
					URLToImage:  "https://example.com/image2.jpg",
					PublishedAt: "2024-01-14T15:45:00Z",
					Content:     "Full article content about policy implementation...",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewHTTPNewsAPIClient("test-api-key", server.URL)
	
	options := NewsSearchOptions{
		Language: "en",
		SortBy:   "relevancy",
		PageSize: 10,
	}
	
	ctx := context.Background()
	events, err := client.SearchNews(ctx, "healthcare policy", options)
	
	require.NoError(t, err)
	assert.Len(t, events, 2)
	
	// Verify first event
	assert.Equal(t, "Healthcare Policy Reform Announced", events[0].Title)
	assert.Equal(t, "Government announces new healthcare policy reforms", events[0].Description)
	assert.Equal(t, "Reuters", events[0].Source)
	assert.Equal(t, "https://example.com/article1", events[0].URL)
	assert.Equal(t, "John Doe", events[0].Author)
	assert.Equal(t, "Full article content about healthcare policy reforms...", events[0].Content)
	assert.True(t, events[0].Relevance > 0)
	assert.Contains(t, events[0].Tags, "policy")
	
	// Verify second event
	assert.Equal(t, "Policy Implementation Challenges", events[1].Title)
	assert.Equal(t, "BBC News", events[1].Source)
	assert.Equal(t, "https://example.com/article2", events[1].URL)
}

func TestHTTPNewsAPIClient_GetTopHeadlines(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/top-headlines")
		assert.Equal(t, "business", r.URL.Query().Get("category"))
		assert.Equal(t, "us", r.URL.Query().Get("country"))
		assert.Equal(t, "test-api-key", r.URL.Query().Get("apiKey"))
		
		// Mock response
		response := NewsAPIResponse{
			Status:       "ok",
			TotalResults: 1,
			Articles: []NewsArticle{
				{
					Source: NewsSource{
						ID:   "financial-times",
						Name: "Financial Times",
					},
					Author:      "Business Reporter",
					Title:       "Economic Policy Update",
					Description: "Latest updates on economic policy changes",
					URL:         "https://example.com/business-article",
					PublishedAt: "2024-01-15T12:00:00Z",
					Content:     "Economic policy content...",
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewHTTPNewsAPIClient("test-api-key", server.URL)
	
	options := NewsSearchOptions{
		Country: "us",
	}
	
	ctx := context.Background()
	events, err := client.GetTopHeadlines(ctx, "business", options)
	
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "Economic Policy Update", events[0].Title)
	assert.Equal(t, "Financial Times", events[0].Source)
	assert.Equal(t, "business", events[0].Category)
}

func TestHTTPNewsAPIClient_ValidateAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  bool
		errorContains  string
	}{
		{
			name:          "Valid API Key",
			statusCode:    http.StatusOK,
			responseBody:  `{"status":"ok","totalResults":1,"articles":[]}`,
			expectedError: false,
		},
		{
			name:          "Invalid API Key",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"status":"error","code":"apiKeyInvalid","message":"Your API key is invalid"}`,
			expectedError: true,
			errorContains: "invalid API key",
		},
		{
			name:          "Rate Limited",
			statusCode:    http.StatusTooManyRequests,
			responseBody:  `{"status":"error","code":"rateLimited","message":"You have made too many requests"}`,
			expectedError: true,
			errorContains: "API validation failed with status 429",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()
			
			client := NewHTTPNewsAPIClient("test-api-key", server.URL)
			
			ctx := context.Background()
			err := client.ValidateAPIKey(ctx)
			
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPNewsAPIClient_SearchNewsWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all query parameters
		query := r.URL.Query()
		assert.Equal(t, "climate policy", query.Get("q"))
		assert.Equal(t, "en", query.Get("language"))
		assert.Equal(t, "publishedAt", query.Get("sortBy"))
		assert.Equal(t, "5", query.Get("pageSize"))
		assert.Equal(t, "2", query.Get("page"))
		assert.Equal(t, "2024-01-01", query.Get("from"))
		assert.Equal(t, "2024-01-31", query.Get("to"))
		assert.Equal(t, "reuters.com,bbc.com", query.Get("domains"))
		assert.Equal(t, "tabloid.com", query.Get("excludeDomains"))
		
		response := NewsAPIResponse{
			Status:       "ok",
			TotalResults: 0,
			Articles:     []NewsArticle{},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewHTTPNewsAPIClient("test-api-key", server.URL)
	
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	
	options := NewsSearchOptions{
		Language:       "en",
		SortBy:         "publishedAt",
		PageSize:       5,
		Page:           2,
		From:           &from,
		To:             &to,
		Domains:        []string{"reuters.com", "bbc.com"},
		ExcludeDomains: []string{"tabloid.com"},
	}
	
	ctx := context.Background()
	events, err := client.SearchNews(ctx, "climate policy", options)
	
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestHTTPNewsAPIClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		statusCode    int
		expectedError string
	}{
		{
			name:          "API Error Response",
			responseBody:  `{"status":"error","code":"apiKeyInvalid","message":"Your API key is invalid"}`,
			statusCode:    http.StatusUnauthorized,
			expectedError: "API request failed with status 401",
		},
		{
			name:          "Invalid JSON Response",
			responseBody:  `invalid json`,
			statusCode:    http.StatusOK,
			expectedError: "failed to parse response",
		},
		{
			name:          "HTTP Error Status",
			responseBody:  `Server Error`,
			statusCode:    http.StatusInternalServerError,
			expectedError: "API request failed with status 500",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()
			
			client := NewHTTPNewsAPIClient("test-api-key", server.URL)
			
			ctx := context.Background()
			_, err := client.SearchNews(ctx, "test query", NewsSearchOptions{})
			
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCalculateRelevance(t *testing.T) {
	client := NewHTTPNewsAPIClient("test-api-key", "")
	
	tests := []struct {
		name        string
		article     NewsArticle
		query       string
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "High Relevance - Title Match",
			article: NewsArticle{
				Title:       "Healthcare Policy Reform",
				Description: "New healthcare policies announced",
			},
			query:       "healthcare policy",
			expectedMin: 0.8,
			expectedMax: 1.0,
		},
		{
			name: "Medium Relevance - Description Match",
			article: NewsArticle{
				Title:       "Government Announcement",
				Description: "Healthcare policy changes discussed",
			},
			query:       "healthcare policy",
			expectedMin: 0.3,
			expectedMax: 0.8,
		},
		{
			name: "Low Relevance - No Match",
			article: NewsArticle{
				Title:       "Sports News",
				Description: "Football game results",
			},
			query:       "healthcare policy",
			expectedMin: 0.0,
			expectedMax: 0.3,
		},
		{
			name: "Empty Query",
			article: NewsArticle{
				Title:       "Any Title",
				Description: "Any description",
			},
			query:       "",
			expectedMin: 0.5,
			expectedMax: 0.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relevance := client.calculateRelevance(tt.article, tt.query)
			assert.GreaterOrEqual(t, relevance, tt.expectedMin)
			assert.LessOrEqual(t, relevance, tt.expectedMax)
		})
	}
}

func TestExtractCategory(t *testing.T) {
	client := NewHTTPNewsAPIClient("test-api-key", "")
	
	tests := []struct {
		sourceName string
		expected   string
	}{
		{"TechCrunch", "technology"},
		{"Wired Magazine", "technology"},
		{"Business Insider", "business"},
		{"Financial Times", "business"},
		{"Health News", "health"},
		{"Medical Daily", "health"},
		{"Science Magazine", "science"},
		{"Politico", "general"},
		{"Government Executive", "politics"},
		{"Unknown Source", "general"},
		{"", "general"},
	}
	
	for _, tt := range tests {
		t.Run(tt.sourceName, func(t *testing.T) {
			category := client.extractCategory(tt.sourceName)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestExtractTags(t *testing.T) {
	client := NewHTTPNewsAPIClient("test-api-key", "")
	
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Government Policy Text",
			text:     "The government announced new healthcare policy regulations for federal agencies",
			expected: []string{"policy", "government", "regulation", "healthcare", "federal"},
		},
		{
			name:     "Technology Policy Text",
			text:     "Congress debates new technology infrastructure bill for public education",
			expected: []string{"technology", "infrastructure", "public", "education"},
		},
		{
			name:     "No Keywords",
			text:     "The quick brown fox jumps over the lazy dog",
			expected: []string{},
		},
		{
			name:     "Empty Text",
			text:     "",
			expected: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := client.extractTags(tt.text)
			
			// Check that all expected tags are present
			for _, expectedTag := range tt.expected {
				assert.Contains(t, tags, expectedTag, "Expected tag '%s' not found in %v", expectedTag, tags)
			}
			
			// Check that we don't have more than 10 tags
			assert.LessOrEqual(t, len(tags), 10)
		})
	}
}

func TestConvertArticlesToEvents(t *testing.T) {
	client := NewHTTPNewsAPIClient("test-api-key", "")
	
	articles := []NewsArticle{
		{
			Source: NewsSource{
				ID:   "reuters",
				Name: "Reuters",
			},
			Author:      "John Doe",
			Title:       "Healthcare Policy Update",
			Description: "Government announces healthcare policy changes",
			URL:         "https://example.com/article1",
			PublishedAt: "2024-01-15T10:30:00Z",
			Content:     "Full article content...",
		},
		{
			Source: NewsSource{
				Name: "Unknown Source",
			},
			Title:       "", // Empty title should be skipped
			Description: "Description without title",
			URL:         "https://example.com/article2",
			PublishedAt: "2024-01-15T11:00:00Z",
		},
		{
			Source: NewsSource{
				Name: "BBC News",
			},
			Author:      "Jane Smith",
			Title:       "Policy Implementation",
			Description: "Analysis of policy implementation challenges",
			URL:         "", // Empty URL should be skipped
			PublishedAt: "2024-01-15T12:00:00Z",
		},
		{
			Source: NewsSource{
				Name: "TechCrunch",
			},
			Title:       "Technology Policy",
			Description: "New technology regulations announced",
			URL:         "https://example.com/article3",
			PublishedAt: "invalid-date", // Invalid date format
		},
	}
	
	events, err := client.convertArticlesToEvents(articles, "healthcare policy")
	
	require.NoError(t, err)
	assert.Len(t, events, 2) // Only valid articles should be converted
	
	// Check first event
	assert.Equal(t, "Healthcare Policy Update", events[0].Title)
	assert.Equal(t, "Reuters", events[0].Source)
	assert.Equal(t, "John Doe", events[0].Author)
	assert.Equal(t, "https://example.com/article1", events[0].URL)
	assert.True(t, events[0].Relevance > 0)
	assert.Equal(t, "general", events[0].Category)
	
	// Check second event (with invalid date)
	assert.Equal(t, "Technology Policy", events[1].Title)
	assert.Equal(t, "TechCrunch", events[1].Source)
	assert.Equal(t, "technology", events[1].Category)
	assert.False(t, events[1].CreatedAt.IsZero()) // Should have a valid created time
}