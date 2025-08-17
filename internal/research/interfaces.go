package research

import (
	"context"
	"time"

	"ai-government-consultant/internal/models"
)

// LangChainResearchService defines the interface for LangChain-based research operations
type LangChainResearchService interface {
	// ResearchPolicyContext performs research on current events related to a policy document
	ResearchPolicyContext(ctx context.Context, document *models.Document) (*models.ResearchResult, error)
	
	// GeneratePolicySuggestions generates policy suggestions based on research results
	GeneratePolicySuggestions(ctx context.Context, researchResult *models.ResearchResult) ([]models.PolicySuggestion, error)
	
	// ValidateResearchSources validates the credibility and relevance of research sources
	ValidateResearchSources(ctx context.Context, sources []models.ResearchSource) (*models.ValidationResult, error)
	
	// GetCurrentEvents retrieves current events related to a specific topic within a timeframe
	GetCurrentEvents(ctx context.Context, topic string, timeframe time.Duration) ([]models.CurrentEvent, error)
	
	// AnalyzePolicyImpact analyzes the potential impact of current events on policy areas
	AnalyzePolicyImpact(ctx context.Context, events []models.CurrentEvent, policyArea string) ([]models.PolicyImpact, error)
	
	// GenerateResearchQuery creates an optimized research query based on document content
	GenerateResearchQuery(ctx context.Context, document *models.Document) (string, error)
}

// ResearchRepository defines the interface for research data persistence
type ResearchRepository interface {
	// SaveResearchResult saves a research result to the database
	SaveResearchResult(ctx context.Context, result *models.ResearchResult) error
	
	// GetResearchResult retrieves a research result by ID
	GetResearchResult(ctx context.Context, id string) (*models.ResearchResult, error)
	
	// GetResearchResultsByDocument retrieves research results for a specific document
	GetResearchResultsByDocument(ctx context.Context, documentID string) ([]models.ResearchResult, error)
	
	// SavePolicySuggestion saves a policy suggestion to the database
	SavePolicySuggestion(ctx context.Context, suggestion *models.PolicySuggestion) error
	
	// GetPolicySuggestion retrieves a policy suggestion by ID
	GetPolicySuggestion(ctx context.Context, id string) (*models.PolicySuggestion, error)
	
	// GetPolicySuggestionsByCategory retrieves policy suggestions by category
	GetPolicySuggestionsByCategory(ctx context.Context, category models.DocumentCategory) ([]models.PolicySuggestion, error)
	
	// SaveCurrentEvent saves a current event to the database
	SaveCurrentEvent(ctx context.Context, event *models.CurrentEvent) error
	
	// GetCurrentEvents retrieves current events with optional filters
	GetCurrentEvents(ctx context.Context, filters CurrentEventFilters) ([]models.CurrentEvent, error)
	
	// SaveResearchSource saves a research source to the database
	SaveResearchSource(ctx context.Context, source *models.ResearchSource) error
	
	// GetResearchSources retrieves research sources with optional filters
	GetResearchSources(ctx context.Context, filters ResearchSourceFilters) ([]models.ResearchSource, error)
	
	// UpdatePolicySuggestionStatus updates the status of a policy suggestion
	UpdatePolicySuggestionStatus(ctx context.Context, id string, status string, reviewNotes *string) error
}

// NewsAPIClient defines the interface for news API integration
type NewsAPIClient interface {
	// SearchNews searches for news articles related to a query
	SearchNews(ctx context.Context, query string, options NewsSearchOptions) ([]models.CurrentEvent, error)
	
	// GetTopHeadlines retrieves top headlines for a category
	GetTopHeadlines(ctx context.Context, category string, options NewsSearchOptions) ([]models.CurrentEvent, error)
	
	// ValidateAPIKey validates the news API key
	ValidateAPIKey(ctx context.Context) error
}

// LLMClient defines the interface for LLM integration
type LLMClient interface {
	// GenerateText generates text based on a prompt
	GenerateText(ctx context.Context, prompt string, options LLMOptions) (string, error)
	
	// AnalyzeText analyzes text and extracts insights
	AnalyzeText(ctx context.Context, text string, analysisType string) (map[string]interface{}, error)
	
	// SummarizeText creates a summary of the provided text
	SummarizeText(ctx context.Context, text string, maxLength int) (string, error)
	
	// ExtractKeywords extracts keywords from text
	ExtractKeywords(ctx context.Context, text string, maxKeywords int) ([]string, error)
}

// CurrentEventFilters defines filters for querying current events
type CurrentEventFilters struct {
	Category    *string
	Tags        []string
	DateFrom    *time.Time
	DateTo      *time.Time
	MinRelevance *float64
	Language    *string
	Source      *string
	Limit       int
	Offset      int
}

// ResearchSourceFilters defines filters for querying research sources
type ResearchSourceFilters struct {
	Type            *models.ResearchSourceType
	MinCredibility  *float64
	MinRelevance    *float64
	DateFrom        *time.Time
	DateTo          *time.Time
	Keywords        []string
	Language        *string
	Limit           int
	Offset          int
}

// NewsSearchOptions defines options for news API searches
type NewsSearchOptions struct {
	Language    string
	Country     string
	Category    string
	SortBy      string // "relevancy", "popularity", "publishedAt"
	PageSize    int
	Page        int
	From        *time.Time
	To          *time.Time
	Domains     []string
	ExcludeDomains []string
}

// LLMOptions defines options for LLM requests
type LLMOptions struct {
	Temperature   float64
	MaxTokens     int
	TopP          float64
	FrequencyPenalty float64
	PresencePenalty  float64
	Model         string
	SystemPrompt  *string
}

// ResearchConfig holds configuration for the research service
type ResearchConfig struct {
	NewsAPIKey          string
	NewsAPIBaseURL      string
	LLMAPIKey           string
	LLMModel            string
	MaxConcurrentRequests int
	RequestTimeout      time.Duration
	CacheEnabled        bool
	CacheTTL            time.Duration
	DefaultLanguage     string
	MaxSourcesPerQuery  int
	MinCredibilityScore float64
	MinRelevanceScore   float64
}