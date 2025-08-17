package consultation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EmbeddingServiceInterface defines the interface for embedding operations
type EmbeddingServiceInterface interface {
	VectorSearch(ctx context.Context, query string, options *embedding.SearchOptions) ([]embedding.SearchResult, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}

// Service handles AI consultation operations
type Service struct {
	geminiAPIKey     string
	geminiURL        string
	httpClient       *http.Client
	mongodb          *mongo.Database
	redis            *redis.Client
	embeddingService EmbeddingServiceInterface
	logger           logger.Logger
	rateLimiter      *RateLimiter
}

// Config holds the configuration for the consultation service
type Config struct {
	GeminiAPIKey     string
	GeminiURL        string
	MongoDB          *mongo.Database
	Redis            *redis.Client
	EmbeddingService EmbeddingServiceInterface
	Logger           logger.Logger
	RateLimit        RateLimitConfig
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	SafetySettings []SafetySetting `json:"safetySettings,omitempty"`
	GenerationConfig GenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents content in Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of content in Gemini request
type GeminiPart struct {
	Text string `json:"text"`
}

// SafetySetting represents safety settings for Gemini API
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GenerationConfig represents generation configuration for Gemini API
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	UsageMetadata UsageMetadata   `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a candidate response from Gemini
type GeminiCandidate struct {
	Content        GeminiContent `json:"content"`
	FinishReason   string        `json:"finishReason"`
	SafetyRatings  []SafetyRating `json:"safetyRatings,omitempty"`
}

// SafetyRating represents safety rating from Gemini
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// UsageMetadata represents usage metadata from Gemini
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// ConsultationRequest represents a consultation request
type ConsultationRequest struct {
	Query            string                 `json:"query"`
	Type             models.ConsultationType `json:"type"`
	UserID           primitive.ObjectID     `json:"user_id"`
	Context          models.ConsultationContext `json:"context"`
	MaxSources       int                    `json:"max_sources,omitempty"`
	ConfidenceThreshold float64             `json:"confidence_threshold,omitempty"`
}

// NewService creates a new consultation service
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.GeminiAPIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	if config.EmbeddingService == nil {
		return nil, fmt.Errorf("embedding service is required")
	}

	geminiURL := config.GeminiURL
	if geminiURL == "" {
		geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"
	}

	// Default rate limit configuration
	rateLimit := config.RateLimit
	if rateLimit.RequestsPerMinute == 0 {
		rateLimit.RequestsPerMinute = 60
	}
	if rateLimit.BurstSize == 0 {
		rateLimit.BurstSize = 10
	}

	return &Service{
		geminiAPIKey:     config.GeminiAPIKey,
		geminiURL:        geminiURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		mongodb:          config.MongoDB,
		redis:            config.Redis,
		embeddingService: config.EmbeddingService,
		logger:           config.Logger,
		rateLimiter:      NewRateLimiter(rateLimit),
	}, nil
}

// ConsultPolicy provides policy consultation
func (s *Service) ConsultPolicy(ctx context.Context, request *ConsultationRequest) (*models.ConsultationResponse, error) {
	if request.Type != models.ConsultationTypePolicy {
		return nil, fmt.Errorf("invalid consultation type for policy consultation")
	}

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Retrieve context from documents and knowledge base
	contextData, err := s.retrieveContext(ctx, request.Query, request.MaxSources)
	if err != nil {
		s.logger.Error("Failed to retrieve context", err, map[string]interface{}{
			"query": request.Query,
		})
		// Continue with empty context rather than failing
		contextData = &ContextData{}
	}

	// Generate policy consultation prompt
	prompt := s.generatePolicyPrompt(request.Query, contextData)

	// Call Gemini API
	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}

	// Parse and validate response
	consultationResponse, err := s.parseConsultationResponse(response, contextData, models.ConsultationTypePolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consultation response: %w", err)
	}

	// Validate response for safety and accuracy
	if err := s.validateResponse(consultationResponse); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return consultationResponse, nil
}

// ConsultStrategy provides strategy consultation
func (s *Service) ConsultStrategy(ctx context.Context, request *ConsultationRequest) (*models.ConsultationResponse, error) {
	if request.Type != models.ConsultationTypeStrategy {
		return nil, fmt.Errorf("invalid consultation type for strategy consultation")
	}

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Retrieve context
	contextData, err := s.retrieveContext(ctx, request.Query, request.MaxSources)
	if err != nil {
		s.logger.Error("Failed to retrieve context", err, map[string]interface{}{
			"query": request.Query,
		})
		contextData = &ContextData{}
	}

	// Generate strategy consultation prompt
	prompt := s.generateStrategyPrompt(request.Query, contextData)

	// Call Gemini API
	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}

	// Parse and validate response
	consultationResponse, err := s.parseConsultationResponse(response, contextData, models.ConsultationTypeStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consultation response: %w", err)
	}

	// Validate response
	if err := s.validateResponse(consultationResponse); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return consultationResponse, nil
}

// ConsultOperations provides operations consultation
func (s *Service) ConsultOperations(ctx context.Context, request *ConsultationRequest) (*models.ConsultationResponse, error) {
	if request.Type != models.ConsultationTypeOperations {
		return nil, fmt.Errorf("invalid consultation type for operations consultation")
	}

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Retrieve context
	contextData, err := s.retrieveContext(ctx, request.Query, request.MaxSources)
	if err != nil {
		s.logger.Error("Failed to retrieve context", err, map[string]interface{}{
			"query": request.Query,
		})
		contextData = &ContextData{}
	}

	// Generate operations consultation prompt
	prompt := s.generateOperationsPrompt(request.Query, contextData)

	// Call Gemini API
	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}

	// Parse and validate response
	consultationResponse, err := s.parseConsultationResponse(response, contextData, models.ConsultationTypeOperations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consultation response: %w", err)
	}

	// Validate response
	if err := s.validateResponse(consultationResponse); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return consultationResponse, nil
}

// ConsultTechnology provides technology consultation
func (s *Service) ConsultTechnology(ctx context.Context, request *ConsultationRequest) (*models.ConsultationResponse, error) {
	if request.Type != models.ConsultationTypeTechnology {
		return nil, fmt.Errorf("invalid consultation type for technology consultation")
	}

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Retrieve context
	contextData, err := s.retrieveContext(ctx, request.Query, request.MaxSources)
	if err != nil {
		s.logger.Error("Failed to retrieve context", err, map[string]interface{}{
			"query": request.Query,
		})
		contextData = &ContextData{}
	}

	// Generate technology consultation prompt
	prompt := s.generateTechnologyPrompt(request.Query, contextData)

	// Call Gemini API
	response, err := s.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}

	// Parse and validate response
	consultationResponse, err := s.parseConsultationResponse(response, contextData, models.ConsultationTypeTechnology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consultation response: %w", err)
	}

	// Validate response
	if err := s.validateResponse(consultationResponse); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return consultationResponse, nil
}

// callGeminiAPI makes a request to the Gemini API
func (s *Service) callGeminiAPI(ctx context.Context, prompt string) (*GeminiResponse, error) {
	// Prepare request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		SafetySettings: []SafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
		GenerationConfig: GenerationConfig{
			Temperature:     0.3, // Lower temperature for more consistent responses
			TopK:            40,
			TopP:            0.8,
			MaxOutputTokens: 4096,
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API request
	url := fmt.Sprintf("%s?key=%s", s.geminiURL, s.geminiAPIKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	s.logger.Debug("Gemini API call completed", map[string]interface{}{
		"prompt_tokens":     response.UsageMetadata.PromptTokenCount,
		"response_tokens":   response.UsageMetadata.CandidatesTokenCount,
		"total_tokens":      response.UsageMetadata.TotalTokenCount,
		"finish_reason":     response.Candidates[0].FinishReason,
	})

	return &response, nil
}

// ContextData holds retrieved context information
type ContextData struct {
	Documents     []embedding.SearchResult `json:"documents"`
	Knowledge     []embedding.SearchResult `json:"knowledge"`
	TotalSources  int                      `json:"total_sources"`
	QueryEmbedding []float64               `json:"query_embedding,omitempty"`
}

// retrieveContext retrieves relevant context from documents and knowledge base
func (s *Service) retrieveContext(ctx context.Context, query string, maxSources int) (*ContextData, error) {
	if maxSources == 0 {
		maxSources = 10
	}

	// Split sources between documents and knowledge
	docLimit := maxSources / 2
	knowledgeLimit := maxSources - docLimit

	// Search documents
	docOptions := &embedding.SearchOptions{
		Limit:      docLimit,
		Threshold:  0.7,
		Collection: "documents",
	}
	documents, err := s.embeddingService.VectorSearch(ctx, query, docOptions)
	if err != nil {
		s.logger.Error("Failed to search documents", err, nil)
		documents = []embedding.SearchResult{}
	}

	// Search knowledge base
	knowledgeOptions := &embedding.SearchOptions{
		Limit:      knowledgeLimit,
		Threshold:  0.7,
		Collection: "knowledge_items",
	}
	knowledge, err := s.embeddingService.VectorSearch(ctx, query, knowledgeOptions)
	if err != nil {
		s.logger.Error("Failed to search knowledge", err, nil)
		knowledge = []embedding.SearchResult{}
	}

	contextData := &ContextData{
		Documents:    documents,
		Knowledge:    knowledge,
		TotalSources: len(documents) + len(knowledge),
	}

	s.logger.Debug("Retrieved context", map[string]interface{}{
		"documents_found": len(documents),
		"knowledge_found": len(knowledge),
		"total_sources":   contextData.TotalSources,
	})

	return contextData, nil
}

// generatePolicyPrompt creates a prompt for policy consultation
func (s *Service) generatePolicyPrompt(query string, context *ContextData) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert government policy consultant with deep knowledge of public administration, regulatory frameworks, and policy development. ")
	prompt.WriteString("Provide comprehensive policy analysis and recommendations based on the user's query and the provided context.\n\n")

	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Analyze the policy implications of the query\n")
	prompt.WriteString("2. Consider regulatory compliance and legal requirements\n")
	prompt.WriteString("3. Identify potential risks and mitigation strategies\n")
	prompt.WriteString("4. Provide specific, actionable recommendations\n")
	prompt.WriteString("5. Include implementation considerations and timelines\n")
	prompt.WriteString("6. Reference relevant precedents and best practices\n")
	prompt.WriteString("7. Assess stakeholder impact and engagement needs\n\n")

	// Add context from documents and knowledge base
	if context.TotalSources > 0 {
		prompt.WriteString("RELEVANT CONTEXT:\n")
		
		if len(context.Documents) > 0 {
			prompt.WriteString("Documents:\n")
			for i, doc := range context.Documents {
				if doc.Document != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, doc.Document.Name, doc.Score))
					if len(doc.Document.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", doc.Document.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", doc.Document.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}

		if len(context.Knowledge) > 0 {
			prompt.WriteString("Knowledge Base:\n")
			for i, knowledge := range context.Knowledge {
				if knowledge.Knowledge != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, knowledge.Knowledge.Title, knowledge.Score))
					if len(knowledge.Knowledge.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", knowledge.Knowledge.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", knowledge.Knowledge.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("USER QUERY:\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\n")

	prompt.WriteString("RESPONSE FORMAT:\n")
	prompt.WriteString("Provide a structured response with the following sections:\n")
	prompt.WriteString("1. Executive Summary\n")
	prompt.WriteString("2. Policy Analysis\n")
	prompt.WriteString("3. Recommendations (prioritized)\n")
	prompt.WriteString("4. Risk Assessment\n")
	prompt.WriteString("5. Implementation Plan\n")
	prompt.WriteString("6. Compliance Considerations\n")
	prompt.WriteString("7. Next Steps\n\n")

	prompt.WriteString("Ensure all recommendations are specific, actionable, and include confidence levels.")

	return prompt.String()
}

// generateStrategyPrompt creates a prompt for strategy consultation
func (s *Service) generateStrategyPrompt(query string, context *ContextData) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert government strategy consultant with extensive experience in strategic planning, organizational development, and public sector transformation. ")
	prompt.WriteString("Provide comprehensive strategic analysis and guidance based on the user's query and the provided context.\n\n")

	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Analyze the strategic implications and opportunities\n")
	prompt.WriteString("2. Consider organizational capabilities and constraints\n")
	prompt.WriteString("3. Identify strategic options and trade-offs\n")
	prompt.WriteString("4. Provide comparative analysis of alternatives\n")
	prompt.WriteString("5. Include resource requirements and timeline considerations\n")
	prompt.WriteString("6. Assess risks and success factors\n")
	prompt.WriteString("7. Consider stakeholder alignment and change management\n\n")

	// Add context
	if context.TotalSources > 0 {
		prompt.WriteString("RELEVANT CONTEXT:\n")
		
		if len(context.Documents) > 0 {
			prompt.WriteString("Documents:\n")
			for i, doc := range context.Documents {
				if doc.Document != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, doc.Document.Name, doc.Score))
					if len(doc.Document.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", doc.Document.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", doc.Document.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}

		if len(context.Knowledge) > 0 {
			prompt.WriteString("Knowledge Base:\n")
			for i, knowledge := range context.Knowledge {
				if knowledge.Knowledge != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, knowledge.Knowledge.Title, knowledge.Score))
					if len(knowledge.Knowledge.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", knowledge.Knowledge.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", knowledge.Knowledge.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("USER QUERY:\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\n")

	prompt.WriteString("RESPONSE FORMAT:\n")
	prompt.WriteString("Provide a structured response with the following sections:\n")
	prompt.WriteString("1. Strategic Assessment\n")
	prompt.WriteString("2. Strategic Options Analysis\n")
	prompt.WriteString("3. Recommended Strategy\n")
	prompt.WriteString("4. Implementation Roadmap\n")
	prompt.WriteString("5. Resource Requirements\n")
	prompt.WriteString("6. Risk Management\n")
	prompt.WriteString("7. Success Metrics\n\n")

	prompt.WriteString("Ensure all strategic recommendations include rationale, expected outcomes, and confidence levels.")

	return prompt.String()
}

// generateOperationsPrompt creates a prompt for operations consultation
func (s *Service) generateOperationsPrompt(query string, context *ContextData) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert government operations consultant with deep expertise in process optimization, operational efficiency, and public service delivery. ")
	prompt.WriteString("Provide comprehensive operational analysis and improvement recommendations based on the user's query and the provided context.\n\n")

	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Analyze current operational processes and inefficiencies\n")
	prompt.WriteString("2. Identify bottlenecks and improvement opportunities\n")
	prompt.WriteString("3. Provide specific process optimization recommendations\n")
	prompt.WriteString("4. Include cost-benefit analysis and ROI projections\n")
	prompt.WriteString("5. Consider compliance and regulatory requirements\n")
	prompt.WriteString("6. Address change management and implementation challenges\n")
	prompt.WriteString("7. Provide performance metrics and monitoring approaches\n\n")

	// Add context
	if context.TotalSources > 0 {
		prompt.WriteString("RELEVANT CONTEXT:\n")
		
		if len(context.Documents) > 0 {
			prompt.WriteString("Documents:\n")
			for i, doc := range context.Documents {
				if doc.Document != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, doc.Document.Name, doc.Score))
					if len(doc.Document.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", doc.Document.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", doc.Document.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}

		if len(context.Knowledge) > 0 {
			prompt.WriteString("Knowledge Base:\n")
			for i, knowledge := range context.Knowledge {
				if knowledge.Knowledge != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, knowledge.Knowledge.Title, knowledge.Score))
					if len(knowledge.Knowledge.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", knowledge.Knowledge.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", knowledge.Knowledge.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("USER QUERY:\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\n")

	prompt.WriteString("RESPONSE FORMAT:\n")
	prompt.WriteString("Provide a structured response with the following sections:\n")
	prompt.WriteString("1. Current State Analysis\n")
	prompt.WriteString("2. Identified Inefficiencies\n")
	prompt.WriteString("3. Process Improvement Recommendations\n")
	prompt.WriteString("4. Cost-Benefit Analysis\n")
	prompt.WriteString("5. Implementation Plan\n")
	prompt.WriteString("6. Performance Metrics\n")
	prompt.WriteString("7. Risk Mitigation\n\n")

	prompt.WriteString("Ensure all recommendations include specific metrics, timelines, and confidence levels.")

	return prompt.String()
}

// generateTechnologyPrompt creates a prompt for technology consultation
func (s *Service) generateTechnologyPrompt(query string, context *ContextData) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert government technology consultant with extensive knowledge of digital transformation, cybersecurity, and government IT systems. ")
	prompt.WriteString("Provide comprehensive technology analysis and recommendations based on the user's query and the provided context.\n\n")

	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Analyze technology requirements and current capabilities\n")
	prompt.WriteString("2. Evaluate technology options and alternatives\n")
	prompt.WriteString("3. Consider security, compliance, and integration requirements\n")
	prompt.WriteString("4. Provide specific technology recommendations with rationale\n")
	prompt.WriteString("5. Include implementation approach and timeline\n")
	prompt.WriteString("6. Address cybersecurity and data protection considerations\n")
	prompt.WriteString("7. Consider scalability, maintainability, and total cost of ownership\n\n")

	// Add context
	if context.TotalSources > 0 {
		prompt.WriteString("RELEVANT CONTEXT:\n")
		
		if len(context.Documents) > 0 {
			prompt.WriteString("Documents:\n")
			for i, doc := range context.Documents {
				if doc.Document != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, doc.Document.Name, doc.Score))
					if len(doc.Document.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", doc.Document.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", doc.Document.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}

		if len(context.Knowledge) > 0 {
			prompt.WriteString("Knowledge Base:\n")
			for i, knowledge := range context.Knowledge {
				if knowledge.Knowledge != nil {
					prompt.WriteString(fmt.Sprintf("%d. %s (Relevance: %.2f)\n", i+1, knowledge.Knowledge.Title, knowledge.Score))
					if len(knowledge.Knowledge.Content) > 500 {
						prompt.WriteString(fmt.Sprintf("   Content: %s...\n", knowledge.Knowledge.Content[:500]))
					} else {
						prompt.WriteString(fmt.Sprintf("   Content: %s\n", knowledge.Knowledge.Content))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("USER QUERY:\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\n")

	prompt.WriteString("RESPONSE FORMAT:\n")
	prompt.WriteString("Provide a structured response with the following sections:\n")
	prompt.WriteString("1. Technology Assessment\n")
	prompt.WriteString("2. Solution Options Analysis\n")
	prompt.WriteString("3. Recommended Technology Stack\n")
	prompt.WriteString("4. Security and Compliance Considerations\n")
	prompt.WriteString("5. Implementation Strategy\n")
	prompt.WriteString("6. Integration Requirements\n")
	prompt.WriteString("7. Ongoing Support and Maintenance\n\n")

	prompt.WriteString("Ensure all technology recommendations include security assessments, compliance validation, and confidence levels.")

	return prompt.String()
}