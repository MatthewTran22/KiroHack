package research

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-government-consultant/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LangChainResearchServiceImpl implements LangChainResearchService
type LangChainResearchServiceImpl struct {
	repository  ResearchRepository
	newsClient  NewsAPIClient
	llmClient   LLMClient
	config      *ResearchConfig
}

// NewLangChainResearchService creates a new LangChain research service
func NewLangChainResearchService(
	repository ResearchRepository,
	newsClient NewsAPIClient,
	llmClient LLMClient,
	config *ResearchConfig,
) *LangChainResearchServiceImpl {
	if config == nil {
		config = &ResearchConfig{
			MaxConcurrentRequests: 5,
			RequestTimeout:        30 * time.Second,
			CacheEnabled:          true,
			CacheTTL:              1 * time.Hour,
			DefaultLanguage:       "en",
			MaxSourcesPerQuery:    20,
			MinCredibilityScore:   0.6,
			MinRelevanceScore:     0.5,
		}
	}
	
	return &LangChainResearchServiceImpl{
		repository: repository,
		newsClient: newsClient,
		llmClient:  llmClient,
		config:     config,
	}
}

// ResearchPolicyContext performs research on current events related to a policy document
func (s *LangChainResearchServiceImpl) ResearchPolicyContext(ctx context.Context, document *models.Document) (*models.ResearchResult, error) {
	startTime := time.Now()
	
	// Generate research query from document content
	researchQuery, err := s.GenerateResearchQuery(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to generate research query: %w", err)
	}
	
	// Create initial research result
	result := &models.ResearchResult{
		ID:            primitive.NewObjectID(),
		DocumentID:    document.ID,
		ResearchQuery: researchQuery,
		Status:        models.ResearchStatusProcessing,
		GeneratedAt:   time.Now(),
		Metadata:      make(map[string]interface{}),
	}
	
	// Save initial result
	if err := s.repository.SaveResearchResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save initial research result: %w", err)
	}
	
	// Get current events related to the research query
	events, err := s.GetCurrentEvents(ctx, researchQuery, 30*24*time.Hour) // Last 30 days
	if err != nil {
		result.Status = models.ResearchStatusFailed
		result.ErrorMessage = &[]string{err.Error()}[0]
		s.repository.SaveResearchResult(ctx, result)
		return nil, fmt.Errorf("failed to get current events: %w", err)
	}
	
	// Analyze policy impacts
	policyArea := s.extractPolicyArea(document)
	policyImpacts, err := s.AnalyzePolicyImpact(ctx, events, policyArea)
	if err != nil {
		result.Status = models.ResearchStatusFailed
		result.ErrorMessage = &[]string{err.Error()}[0]
		s.repository.SaveResearchResult(ctx, result)
		return nil, fmt.Errorf("failed to analyze policy impact: %w", err)
	}
	
	// Convert events to research sources
	sources := s.convertEventsToSources(events)
	
	// Validate and score sources
	validationResult, err := s.ValidateResearchSources(ctx, sources)
	if err != nil {
		// Log warning but continue
		result.Metadata["validation_warning"] = err.Error()
	} else {
		result.Metadata["validation_result"] = validationResult
	}
	
	// Calculate overall confidence based on source quality and relevance
	confidence := s.calculateResearchConfidence(sources, policyImpacts)
	
	// Update result with findings
	result.CurrentEvents = events
	result.PolicyImpacts = policyImpacts
	result.Sources = sources
	result.Confidence = confidence
	result.Status = models.ResearchStatusCompleted
	result.ProcessingTime = time.Since(startTime)
	
	// Save final result
	if err := s.repository.SaveResearchResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save final research result: %w", err)
	}
	
	// Save individual events and sources to database
	for _, event := range events {
		s.repository.SaveCurrentEvent(ctx, &event)
	}
	
	for _, source := range sources {
		s.repository.SaveResearchSource(ctx, &source)
	}
	
	return result, nil
}

// GeneratePolicySuggestions generates policy suggestions based on research results
func (s *LangChainResearchServiceImpl) GeneratePolicySuggestions(ctx context.Context, researchResult *models.ResearchResult) ([]models.PolicySuggestion, error) {
	if !researchResult.IsCompleted() {
		return nil, fmt.Errorf("research result is not completed")
	}
	
	// Create context for policy suggestion generation
	contextText := s.buildPolicySuggestionContext(researchResult)
	
	// Generate suggestions using LLM
	prompt := fmt.Sprintf(`Based on the following research findings and current events, generate 3-5 specific policy suggestions that address the identified issues and opportunities.

Research Context:
%s

For each policy suggestion, provide:
1. A clear, actionable title
2. Detailed description of the proposed policy
3. Rationale explaining why this policy is needed
4. Implementation plan with specific steps
5. Risk assessment with potential challenges
6. Priority level (low, medium, high, critical)

Please format the response as a JSON array of policy suggestions with the following structure:
[
  {
    "title": "Policy Title",
    "description": "Detailed description",
    "rationale": "Why this policy is needed",
    "priority": "high",
    "implementation": {
      "steps": ["step1", "step2"],
      "timeline": "6-12 months",
      "resources": ["resource1", "resource2"],
      "stakeholders": ["stakeholder1", "stakeholder2"]
    },
    "risk_assessment": {
      "overall_risk": "medium",
      "risk_factors": [
        {
          "description": "Risk description",
          "probability": 0.3,
          "impact": "medium",
          "mitigation": "Mitigation strategy"
        }
      ]
    }
  }
]`, contextText)
	
	response, err := s.llmClient.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.7,
		MaxTokens:   4000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate policy suggestions: %w", err)
	}
	
	// Parse the response and convert to PolicySuggestion models
	suggestions, err := s.parsePolicySuggestions(response, researchResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy suggestions: %w", err)
	}
	
	// Save suggestions to database
	for i := range suggestions {
		if err := s.repository.SavePolicySuggestion(ctx, &suggestions[i]); err != nil {
			return nil, fmt.Errorf("failed to save policy suggestion: %w", err)
		}
	}
	
	return suggestions, nil
}

// ValidateResearchSources validates the credibility and relevance of research sources
func (s *LangChainResearchServiceImpl) ValidateResearchSources(ctx context.Context, sources []models.ResearchSource) (*models.ValidationResult, error) {
	if len(sources) == 0 {
		return &models.ValidationResult{
			IsValid:          false,
			CredibilityScore: 0.0,
			Issues:           []string{"No sources provided"},
			ValidatedAt:      time.Now(),
			ValidatedBy:      "system",
		}, nil
	}
	
	var totalCredibility float64
	var totalRelevance float64
	var issues []string
	var recommendations []string
	validSources := 0
	
	for _, source := range sources {
		// Validate source URL
		if source.URL == "" {
			issues = append(issues, fmt.Sprintf("Source '%s' has no URL", source.Title))
			continue
		}
		
		// Check credibility score
		if source.Credibility < s.config.MinCredibilityScore {
			issues = append(issues, fmt.Sprintf("Source '%s' has low credibility score: %.2f", source.Title, source.Credibility))
		}
		
		// Check relevance score
		if source.Relevance < s.config.MinRelevanceScore {
			issues = append(issues, fmt.Sprintf("Source '%s' has low relevance score: %.2f", source.Title, source.Relevance))
		}
		
		// Check publication date (sources older than 1 year get flagged)
		if time.Since(source.PublishedAt) > 365*24*time.Hour {
			issues = append(issues, fmt.Sprintf("Source '%s' is older than 1 year", source.Title))
		}
		
		totalCredibility += source.Credibility
		totalRelevance += source.Relevance
		validSources++
	}
	
	if validSources == 0 {
		return &models.ValidationResult{
			IsValid:          false,
			CredibilityScore: 0.0,
			Issues:           append(issues, "No valid sources found"),
			ValidatedAt:      time.Now(),
			ValidatedBy:      "system",
		}, nil
	}
	
	avgCredibility := totalCredibility / float64(validSources)
	avgRelevance := totalRelevance / float64(validSources)
	
	// Generate recommendations
	if avgCredibility < 0.7 {
		recommendations = append(recommendations, "Consider seeking additional sources with higher credibility")
	}
	if avgRelevance < 0.6 {
		recommendations = append(recommendations, "Consider refining search terms to find more relevant sources")
	}
	if len(sources) < 5 {
		recommendations = append(recommendations, "Consider expanding the search to include more diverse sources")
	}
	
	// Overall validation score combines credibility and relevance
	overallScore := (avgCredibility + avgRelevance) / 2
	isValid := overallScore >= 0.6 && len(issues) < len(sources)/2
	
	return &models.ValidationResult{
		IsValid:          isValid,
		CredibilityScore: overallScore,
		Issues:           issues,
		Recommendations:  recommendations,
		ValidatedAt:      time.Now(),
		ValidatedBy:      "system",
	}, nil
}

// GetCurrentEvents retrieves current events related to a specific topic within a timeframe
func (s *LangChainResearchServiceImpl) GetCurrentEvents(ctx context.Context, topic string, timeframe time.Duration) ([]models.CurrentEvent, error) {
	// Calculate date range
	to := time.Now()
	from := to.Add(-timeframe)
	
	// Search for news articles
	options := NewsSearchOptions{
		Language: s.config.DefaultLanguage,
		SortBy:   "relevancy",
		PageSize: s.config.MaxSourcesPerQuery,
		From:     &from,
		To:       &to,
	}
	
	events, err := s.newsClient.SearchNews(ctx, topic, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search news: %w", err)
	}
	
	// Filter events by relevance
	filteredEvents := make([]models.CurrentEvent, 0)
	for _, event := range events {
		if event.Relevance >= s.config.MinRelevanceScore {
			filteredEvents = append(filteredEvents, event)
		}
	}
	
	return filteredEvents, nil
}

// AnalyzePolicyImpact analyzes the potential impact of current events on policy areas
func (s *LangChainResearchServiceImpl) AnalyzePolicyImpact(ctx context.Context, events []models.CurrentEvent, policyArea string) ([]models.PolicyImpact, error) {
	if len(events) == 0 {
		return []models.PolicyImpact{}, nil
	}
	
	// Build context from events
	eventContext := s.buildEventContext(events)
	
	prompt := fmt.Sprintf(`Analyze the potential policy impacts of the following current events on the %s policy area.

Current Events Context:
%s

For each significant impact, provide:
1. The specific policy area affected
2. Description of the impact
3. Severity level (low, medium, high, critical)
4. Expected timeframe for the impact
5. Key stakeholders affected
6. Potential mitigation strategies

Please format the response as a JSON array:
[
  {
    "area": "specific policy area",
    "impact": "description of impact",
    "severity": "medium",
    "timeframe": "short-term|medium-term|long-term",
    "stakeholders": ["stakeholder1", "stakeholder2"],
    "mitigation": ["strategy1", "strategy2"],
    "confidence": 0.8,
    "evidence": ["evidence1", "evidence2"]
  }
]`, policyArea, eventContext)
	
	response, err := s.llmClient.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.4,
		MaxTokens:   3000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to analyze policy impact: %w", err)
	}
	
	// Parse the response
	impacts, err := s.parsePolicyImpacts(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy impacts: %w", err)
	}
	
	return impacts, nil
}

// GenerateResearchQuery creates an optimized research query based on document content
func (s *LangChainResearchServiceImpl) GenerateResearchQuery(ctx context.Context, document *models.Document) (string, error) {
	// Extract key information from document
	content := document.Content
	if len(content) > 2000 {
		content = content[:2000] // Limit content length for processing
	}
	
	prompt := fmt.Sprintf(`Based on the following policy document, generate an optimized search query to find current events and news that would be relevant for policy analysis and decision-making.

Document Title: %s
Document Category: %s
Document Content (excerpt):
%s

Generate a search query that would help find:
1. Recent developments related to this policy area
2. Current events that might impact this policy
3. Similar policies or initiatives in other jurisdictions
4. Stakeholder reactions or opinions
5. Implementation challenges or successes

Provide only the search query, optimized for news APIs, without additional explanation.`, 
		document.Name, document.Metadata.Category, content)
	
	query, err := s.llmClient.GenerateText(ctx, prompt, LLMOptions{
		Temperature: 0.3,
		MaxTokens:   100,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate research query: %w", err)
	}
	
	// Clean up the query
	query = strings.TrimSpace(query)
	query = strings.Trim(query, "\"'")
	
	return query, nil
}

// Helper methods

func (s *LangChainResearchServiceImpl) extractPolicyArea(document *models.Document) string {
	switch document.Metadata.Category {
	case models.DocumentCategoryPolicy:
		return "policy development"
	case models.DocumentCategoryStrategy:
		return "strategic planning"
	case models.DocumentCategoryOperations:
		return "operational efficiency"
	case models.DocumentCategoryTechnology:
		return "technology implementation"
	default:
		return "general government operations"
	}
}

func (s *LangChainResearchServiceImpl) convertEventsToSources(events []models.CurrentEvent) []models.ResearchSource {
	sources := make([]models.ResearchSource, len(events))
	
	for i, event := range events {
		sources[i] = models.ResearchSource{
			ID:          primitive.NewObjectID(),
			Type:        models.ResearchSourceTypeNews,
			Title:       event.Title,
			URL:         event.URL,
			Author:      event.Author,
			PublishedAt: event.PublishedAt,
			Credibility: s.calculateSourceCredibility(event.Source),
			Relevance:   event.Relevance,
			Content:     event.Content,
			Summary:     event.Description,
			Keywords:    event.Tags,
			Language:    event.Language,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	
	return sources
}

func (s *LangChainResearchServiceImpl) calculateSourceCredibility(sourceName string) float64 {
	// Simple credibility scoring based on source reputation
	// In a real implementation, this would use a more sophisticated scoring system
	sourceName = strings.ToLower(sourceName)
	
	highCredibility := []string{
		"reuters", "associated press", "bbc", "npr", "pbs",
		"wall street journal", "new york times", "washington post",
		"financial times", "the economist", "government", "official",
	}
	
	mediumCredibility := []string{
		"cnn", "fox news", "abc news", "cbs news", "nbc news",
		"usa today", "bloomberg", "cnbc", "politico",
	}
	
	for _, source := range highCredibility {
		if strings.Contains(sourceName, source) {
			return 0.9
		}
	}
	
	for _, source := range mediumCredibility {
		if strings.Contains(sourceName, source) {
			return 0.7
		}
	}
	
	return 0.5 // Default credibility for unknown sources
}

func (s *LangChainResearchServiceImpl) calculateResearchConfidence(sources []models.ResearchSource, impacts []models.PolicyImpact) float64 {
	if len(sources) == 0 {
		return 0.0
	}
	
	var totalCredibility float64
	var totalRelevance float64
	
	for _, source := range sources {
		totalCredibility += source.Credibility
		totalRelevance += source.Relevance
	}
	
	avgCredibility := totalCredibility / float64(len(sources))
	avgRelevance := totalRelevance / float64(len(sources))
	
	// Factor in number of sources (more sources = higher confidence, up to a point)
	sourceCountFactor := float64(len(sources)) / 10.0
	if sourceCountFactor > 1.0 {
		sourceCountFactor = 1.0
	}
	
	// Factor in policy impact confidence
	var impactConfidence float64
	if len(impacts) > 0 {
		var totalImpactConfidence float64
		for _, impact := range impacts {
			totalImpactConfidence += impact.Confidence
		}
		impactConfidence = totalImpactConfidence / float64(len(impacts))
	} else {
		impactConfidence = 0.5 // Default if no impacts
	}
	
	// Weighted average of all factors
	confidence := (avgCredibility*0.4 + avgRelevance*0.3 + sourceCountFactor*0.2 + impactConfidence*0.1)
	
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

func (s *LangChainResearchServiceImpl) buildPolicySuggestionContext(result *models.ResearchResult) string {
	var context strings.Builder
	
	context.WriteString(fmt.Sprintf("Research Query: %s\n\n", result.ResearchQuery))
	
	if len(result.CurrentEvents) > 0 {
		context.WriteString("Current Events:\n")
		for i, event := range result.CurrentEvents {
			if i >= 5 { // Limit to top 5 events
				break
			}
			context.WriteString(fmt.Sprintf("- %s: %s\n", event.Title, event.Description))
		}
		context.WriteString("\n")
	}
	
	if len(result.PolicyImpacts) > 0 {
		context.WriteString("Policy Impacts:\n")
		for _, impact := range result.PolicyImpacts {
			context.WriteString(fmt.Sprintf("- %s (%s): %s\n", impact.Area, impact.Severity, impact.Impact))
		}
		context.WriteString("\n")
	}
	
	context.WriteString(fmt.Sprintf("Research Confidence: %.2f\n", result.Confidence))
	
	return context.String()
}

func (s *LangChainResearchServiceImpl) buildEventContext(events []models.CurrentEvent) string {
	var context strings.Builder
	
	for i, event := range events {
		if i >= 10 { // Limit to top 10 events
			break
		}
		context.WriteString(fmt.Sprintf("Event %d:\n", i+1))
		context.WriteString(fmt.Sprintf("Title: %s\n", event.Title))
		context.WriteString(fmt.Sprintf("Description: %s\n", event.Description))
		context.WriteString(fmt.Sprintf("Source: %s\n", event.Source))
		context.WriteString(fmt.Sprintf("Published: %s\n", event.PublishedAt.Format("2006-01-02")))
		context.WriteString(fmt.Sprintf("Relevance: %.2f\n\n", event.Relevance))
	}
	
	return context.String()
}

// Placeholder parsing methods - these would need proper JSON parsing implementation
func (s *LangChainResearchServiceImpl) parsePolicySuggestions(response string, researchResult *models.ResearchResult) ([]models.PolicySuggestion, error) {
	// This is a simplified implementation - in practice, you'd want robust JSON parsing
	// For now, return a basic policy suggestion structure
	suggestions := []models.PolicySuggestion{
		{
			ID:             primitive.NewObjectID(),
			Title:          "Generated Policy Suggestion",
			Description:    "This is a placeholder policy suggestion generated from research",
			Rationale:      "Based on current events analysis",
			Priority:       models.PolicyPriorityMedium,
			Category:       models.DocumentCategoryPolicy,
			Confidence:     researchResult.Confidence,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Status:         "draft",
			CurrentContext: researchResult.CurrentEvents,
		},
	}
	
	return suggestions, nil
}

func (s *LangChainResearchServiceImpl) parsePolicyImpacts(response string) ([]models.PolicyImpact, error) {
	// This is a simplified implementation - in practice, you'd want robust JSON parsing
	// For now, return a basic policy impact structure
	impacts := []models.PolicyImpact{
		{
			Area:         "General Policy Area",
			Impact:       "Potential impact identified from current events",
			Severity:     "medium",
			Timeframe:    "medium-term",
			Stakeholders: []string{"government agencies", "citizens"},
			Mitigation:   []string{"monitor developments", "engage stakeholders"},
			Confidence:   0.7,
			Evidence:     []string{"current events analysis"},
		},
	}
	
	return impacts, nil
}