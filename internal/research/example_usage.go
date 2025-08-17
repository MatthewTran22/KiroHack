package research

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-government-consultant/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExampleUsage demonstrates how to use the LangChain research service
func ExampleUsage() {
	// This is an example of how to use the research service
	// In a real application, you would get these from configuration
	
	ctx := context.Background()
	
	// 1. Setup MongoDB connection
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(ctx)
	
	db := client.Database("ai_government_consultant")
	
	// 2. Create repository
	repo := NewMongoResearchRepository(db)
	
	// Create indexes (only needed once)
	if err := repo.CreateIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}
	
	// 3. Create API clients
	newsClient := NewHTTPNewsAPIClient("your-news-api-key", "")
	llmClient := NewGeminiLLMClient("your-gemini-api-key", "gemini-1.5-flash")
	
	// 4. Create research service with configuration
	config := &ResearchConfig{
		NewsAPIKey:            "your-news-api-key",
		NewsAPIBaseURL:        "https://newsapi.org/v2",
		LLMModel:              "gemini-1.5-flash",
		MaxConcurrentRequests: 5,
		RequestTimeout:        30,
		CacheEnabled:          true,
		CacheTTL:              3600,
		DefaultLanguage:       "en",
		MaxSourcesPerQuery:    20,
		MinCredibilityScore:   0.6,
		MinRelevanceScore:     0.5,
	}
	
	service := NewLangChainResearchService(repo, newsClient, llmClient, config)
	
	// 5. Create a sample document for research
	document := &models.Document{
		ID:      primitive.NewObjectID(),
		Name:    "Healthcare Policy Reform Proposal",
		Content: `This document proposes comprehensive healthcare policy reforms to improve patient outcomes, 
		reduce costs, and enhance accessibility. Key areas include expanding coverage, implementing 
		digital health records, establishing quality metrics, and improving rural healthcare access.`,
		Metadata: models.DocumentMetadata{
			Category: models.DocumentCategoryPolicy,
			Tags:     []string{"healthcare", "policy", "reform", "accessibility"},
		},
	}
	
	// 6. Perform research on the document
	fmt.Println("Starting policy research...")
	
	researchResult, err := service.ResearchPolicyContext(ctx, document)
	if err != nil {
		log.Printf("Research failed: %v", err)
		return
	}
	
	fmt.Printf("Research completed with confidence: %.2f\n", researchResult.Confidence)
	fmt.Printf("Found %d current events\n", len(researchResult.CurrentEvents))
	fmt.Printf("Identified %d policy impacts\n", len(researchResult.PolicyImpacts))
	fmt.Printf("Analyzed %d sources\n", len(researchResult.Sources))
	
	// 7. Generate policy suggestions based on research
	fmt.Println("\nGenerating policy suggestions...")
	
	suggestions, err := service.GeneratePolicySuggestions(ctx, researchResult)
	if err != nil {
		log.Printf("Policy suggestion generation failed: %v", err)
		return
	}
	
	fmt.Printf("Generated %d policy suggestions:\n", len(suggestions))
	for i, suggestion := range suggestions {
		fmt.Printf("%d. %s (Priority: %s, Confidence: %.2f)\n", 
			i+1, suggestion.Title, suggestion.Priority, suggestion.Confidence)
		fmt.Printf("   Description: %s\n", suggestion.Description)
		fmt.Printf("   Status: %s\n\n", suggestion.Status)
	}
	
	// 8. Validate research sources
	fmt.Println("Validating research sources...")
	
	validation, err := service.ValidateResearchSources(ctx, researchResult.Sources)
	if err != nil {
		log.Printf("Source validation failed: %v", err)
		return
	}
	
	fmt.Printf("Source validation - Overall Score: %.2f, Valid: %t\n", 
		validation.CredibilityScore, validation.IsValid)
	
	if len(validation.Issues) > 0 {
		fmt.Println("Issues found:")
		for _, issue := range validation.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}
	
	if len(validation.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for _, rec := range validation.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}
	
	// 9. Query current events directly
	fmt.Println("\nQuerying current events...")
	
	events, err := service.GetCurrentEvents(ctx, "healthcare policy", 7*24*time.Hour)
	if err != nil {
		log.Printf("Current events query failed: %v", err)
		return
	}
	
	fmt.Printf("Found %d recent events:\n", len(events))
	for i, event := range events {
		if i >= 3 { // Limit output
			break
		}
		fmt.Printf("%d. %s (Relevance: %.2f)\n", i+1, event.Title, event.Relevance)
		fmt.Printf("   Source: %s, Published: %s\n", event.Source, event.PublishedAt.Format("2006-01-02"))
		fmt.Printf("   URL: %s\n\n", event.URL)
	}
	
	// 10. Update policy suggestion status (example workflow)
	if len(suggestions) > 0 {
		fmt.Println("Updating policy suggestion status...")
		
		suggestionID := suggestions[0].ID.Hex()
		reviewNotes := "Approved after stakeholder review and impact assessment"
		
		err = repo.UpdatePolicySuggestionStatus(ctx, suggestionID, "approved", &reviewNotes)
		if err != nil {
			log.Printf("Failed to update suggestion status: %v", err)
		} else {
			fmt.Println("Policy suggestion approved successfully")
		}
	}
	
	fmt.Println("\nResearch workflow completed successfully!")
}

// ExampleQueryResearchData demonstrates how to query existing research data
func ExampleQueryResearchData() {
	ctx := context.Background()
	
	// Setup (same as above)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(ctx)
	
	db := client.Database("ai_government_consultant")
	repo := NewMongoResearchRepository(db)
	
	// Query research results by document
	documentID := "your-document-id-here" // Replace with actual document ID
	results, err := repo.GetResearchResultsByDocument(ctx, documentID)
	if err != nil {
		log.Printf("Failed to query research results: %v", err)
		return
	}
	
	fmt.Printf("Found %d research results for document\n", len(results))
	
	// Query policy suggestions by category
	suggestions, err := repo.GetPolicySuggestionsByCategory(ctx, models.DocumentCategoryPolicy)
	if err != nil {
		log.Printf("Failed to query policy suggestions: %v", err)
		return
	}
	
	fmt.Printf("Found %d policy suggestions\n", len(suggestions))
	
	// Query current events with filters
	filters := CurrentEventFilters{
		Category:     stringPtr("healthcare"),
		MinRelevance: float64Ptr(0.7),
		Limit:        10,
		Offset:       0,
	}
	
	events, err := repo.GetCurrentEvents(ctx, filters)
	if err != nil {
		log.Printf("Failed to query current events: %v", err)
		return
	}
	
	fmt.Printf("Found %d relevant current events\n", len(events))
	
	// Query research sources with filters
	sourceFilters := ResearchSourceFilters{
		Type:           &[]models.ResearchSourceType{models.ResearchSourceTypeNews}[0],
		MinCredibility: float64Ptr(0.8),
		Limit:          10,
		Offset:         0,
	}
	
	sources, err := repo.GetResearchSources(ctx, sourceFilters)
	if err != nil {
		log.Printf("Failed to query research sources: %v", err)
		return
	}
	
	fmt.Printf("Found %d high-credibility sources\n", len(sources))
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

// ExampleConfiguration shows different ways to configure the research service
func ExampleConfiguration() {
	// Default configuration
	service1 := NewLangChainResearchService(nil, nil, nil, nil)
	fmt.Printf("Default config - Max sources: %d, Language: %s\n", 
		service1.config.MaxSourcesPerQuery, service1.config.DefaultLanguage)
	
	// Custom configuration for high-volume research
	highVolumeConfig := &ResearchConfig{
		MaxConcurrentRequests: 10,
		RequestTimeout:        60,
		MaxSourcesPerQuery:    50,
		MinCredibilityScore:   0.8,
		MinRelevanceScore:     0.7,
		DefaultLanguage:       "en",
		CacheEnabled:          true,
		CacheTTL:              7200, // 2 hours
	}
	
	service2 := NewLangChainResearchService(nil, nil, nil, highVolumeConfig)
	fmt.Printf("High-volume config - Max sources: %d, Min credibility: %.1f\n", 
		service2.config.MaxSourcesPerQuery, service2.config.MinCredibilityScore)
	
	// Configuration for international research
	internationalConfig := &ResearchConfig{
		DefaultLanguage:     "es", // Spanish
		MaxSourcesPerQuery:  30,
		MinCredibilityScore: 0.5,  // Lower threshold for diverse sources
		MinRelevanceScore:   0.4,
	}
	
	service3 := NewLangChainResearchService(nil, nil, nil, internationalConfig)
	fmt.Printf("International config - Language: %s, Min credibility: %.1f\n", 
		service3.config.DefaultLanguage, service3.config.MinCredibilityScore)
}