package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles embedding generation and vector search operations
type Service struct {
	geminiAPIKey string
	geminiURL    string
	httpClient   *http.Client
	mongodb      *mongo.Database
	redis        *redis.Client
	logger       logger.Logger
}

// Config holds the configuration for the embedding service
type Config struct {
	GeminiAPIKey string
	GeminiURL    string
	MongoDB      *mongo.Database
	Redis        *redis.Client
	Logger       logger.Logger
}

// GeminiEmbeddingRequest represents the request structure for Gemini embedding API
type GeminiEmbeddingRequest struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

// GeminiEmbeddingResponse represents the response structure from Gemini embedding API
type GeminiEmbeddingResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
}

// EmbeddingResult represents the result of an embedding operation
type EmbeddingResult struct {
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	Embeddings []float64 `json:"embeddings"`
	CreatedAt  time.Time `json:"created_at"`
}

// SearchResult represents a vector search result
type SearchResult struct {
	ID        string                 `json:"id"`
	Score     float64                `json:"score"`
	Document  *models.Document       `json:"document,omitempty"`
	Knowledge *models.KnowledgeItem  `json:"knowledge,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SearchOptions defines options for vector search
type SearchOptions struct {
	Limit      int                    `json:"limit"`
	Threshold  float64                `json:"threshold"`
	Filters    map[string]interface{} `json:"filters"`
	Collection string                 `json:"collection"` // "documents" or "knowledge_items"
}

// NewService creates a new embedding service
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.GeminiAPIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	geminiURL := config.GeminiURL
	if geminiURL == "" {
		geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent"
	}

	return &Service{
		geminiAPIKey: config.GeminiAPIKey,
		geminiURL:    geminiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		mongodb: config.MongoDB,
		redis:   config.Redis,
		logger:  config.Logger,
	}, nil
}

// GenerateEmbedding generates embeddings for the given text using Gemini API
func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Check cache first
	if s.redis != nil {
		cacheKey := fmt.Sprintf("embedding:%x", text)
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var embeddings []float64
			if err := json.Unmarshal([]byte(cached), &embeddings); err == nil {
				s.logger.Debug("Retrieved embedding from cache", map[string]interface{}{
					"cache_key": cacheKey,
				})
				return embeddings, nil
			}
		}
	}

	// Prepare request
	request := GeminiEmbeddingRequest{}
	request.Content.Parts = []struct {
		Text string `json:"text"`
	}{
		{Text: text},
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
	var response GeminiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := response.Embedding.Values

	// Cache the result
	if s.redis != nil {
		cacheKey := fmt.Sprintf("embedding:%x", text)
		embeddingJSON, _ := json.Marshal(embeddings)
		s.redis.Set(ctx, cacheKey, embeddingJSON, 24*time.Hour) // Cache for 24 hours
	}

	s.logger.Debug("Generated embedding", map[string]interface{}{
		"text_length":         len(text),
		"embedding_dimension": len(embeddings),
	})
	return embeddings, nil
}

// GenerateDocumentEmbedding generates and stores embeddings for a document
func (s *Service) GenerateDocumentEmbedding(ctx context.Context, documentID primitive.ObjectID) error {
	// Retrieve document
	collection := s.mongodb.Collection("documents")
	var document models.Document
	err := collection.FindOne(ctx, bson.M{"_id": documentID}).Decode(&document)
	if err != nil {
		return fmt.Errorf("failed to find document: %w", err)
	}

	// Generate embedding for document content
	embeddings, err := s.GenerateEmbedding(ctx, document.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Update document with embeddings
	update := bson.M{
		"$set": bson.M{
			"embeddings":           embeddings,
			"processing_timestamp": time.Now(),
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
	if err != nil {
		return fmt.Errorf("failed to update document with embeddings: %w", err)
	}

	s.logger.Info("Generated document embedding", map[string]interface{}{
		"document_id":         documentID.Hex(),
		"embedding_dimension": len(embeddings),
	})
	return nil
}

// GenerateKnowledgeEmbedding generates and stores embeddings for a knowledge item
func (s *Service) GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID primitive.ObjectID) error {
	// Retrieve knowledge item
	collection := s.mongodb.Collection("knowledge_items")
	var knowledge models.KnowledgeItem
	err := collection.FindOne(ctx, bson.M{"_id": knowledgeID}).Decode(&knowledge)
	if err != nil {
		return fmt.Errorf("failed to find knowledge item: %w", err)
	}

	// Combine title and content for embedding
	text := knowledge.Title + "\n" + knowledge.Content
	if knowledge.Summary != nil {
		text += "\n" + *knowledge.Summary
	}

	// Generate embedding
	embeddings, err := s.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Update knowledge item with embeddings
	update := bson.M{
		"$set": bson.M{
			"embeddings": embeddings,
			"updated_at": time.Now(),
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": knowledgeID}, update)
	if err != nil {
		return fmt.Errorf("failed to update knowledge item with embeddings: %w", err)
	}

	s.logger.Info("Generated knowledge embedding", map[string]interface{}{
		"knowledge_id":        knowledgeID.Hex(),
		"embedding_dimension": len(embeddings),
	})
	return nil
}

// VectorSearch performs semantic similarity search across documents and knowledge items
func (s *Service) VectorSearch(ctx context.Context, query string, options *SearchOptions) ([]SearchResult, error) {
	if options == nil {
		options = &SearchOptions{
			Limit:     10,
			Threshold: 0.7,
			Filters:   make(map[string]interface{}),
		}
	}

	// Generate embedding for the query
	queryEmbedding, err := s.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var results []SearchResult

	// Search documents if collection is not specified or is "documents"
	if options.Collection == "" || options.Collection == "documents" {
		docResults, err := s.searchDocuments(ctx, queryEmbedding, options)
		if err != nil {
			s.logger.Error("Failed to search documents", err, nil)
		} else {
			results = append(results, docResults...)
		}
	}

	// Search knowledge items if collection is not specified or is "knowledge_items"
	if options.Collection == "" || options.Collection == "knowledge_items" {
		knowledgeResults, err := s.searchKnowledgeItems(ctx, queryEmbedding, options)
		if err != nil {
			s.logger.Error("Failed to search knowledge items", err, nil)
		} else {
			results = append(results, knowledgeResults...)
		}
	}

	// Sort results by score (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > options.Limit {
		results = results[:options.Limit]
	}

	s.logger.Debug("Vector search completed", map[string]interface{}{
		"query_length":  len(query),
		"results_count": len(results),
	})
	return results, nil
}

// searchDocuments performs vector search on documents collection
func (s *Service) searchDocuments(ctx context.Context, queryEmbedding []float64, options *SearchOptions) ([]SearchResult, error) {
	collection := s.mongodb.Collection("documents")

	// Build aggregation pipeline for vector search
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"embeddings":        bson.M{"$exists": true, "$ne": nil},
				"processing_status": "completed",
			},
		},
	}

	// Add filters if specified
	if len(options.Filters) > 0 {
		matchStage := pipeline[0]["$match"].(bson.M)
		for key, value := range options.Filters {
			matchStage[key] = value
		}
	}

	// Add vector similarity calculation
	pipeline = append(pipeline, bson.M{
		"$addFields": bson.M{
			"similarity": bson.M{
				"$let": bson.M{
					"vars": bson.M{
						"dotProduct": bson.M{
							"$reduce": bson.M{
								"input": bson.M{
									"$zip": bson.M{
										"inputs": []interface{}{"$embeddings", queryEmbedding},
									},
								},
								"initialValue": 0,
								"in": bson.M{
									"$add": []interface{}{
										"$$value",
										bson.M{"$multiply": []interface{}{"$$this.0", "$$this.1"}},
									},
								},
							},
						},
						"magnitude1": bson.M{
							"$sqrt": bson.M{
								"$reduce": bson.M{
									"input":        "$embeddings",
									"initialValue": 0,
									"in": bson.M{
										"$add": []interface{}{"$$value", bson.M{"$multiply": []interface{}{"$$this", "$$this"}}},
									},
								},
							},
						},
						"magnitude2": bson.M{
							"$sqrt": bson.M{
								"$reduce": bson.M{
									"input":        queryEmbedding,
									"initialValue": 0,
									"in": bson.M{
										"$add": []interface{}{"$$value", bson.M{"$multiply": []interface{}{"$$this", "$$this"}}},
									},
								},
							},
						},
					},
					"in": bson.M{
						"$divide": []interface{}{
							"$$dotProduct",
							bson.M{"$multiply": []interface{}{"$$magnitude1", "$$magnitude2"}},
						},
					},
				},
			},
		},
	})

	// Filter by similarity threshold
	pipeline = append(pipeline, bson.M{
		"$match": bson.M{
			"similarity": bson.M{"$gte": options.Threshold},
		},
	})

	// Sort by similarity (descending)
	pipeline = append(pipeline, bson.M{
		"$sort": bson.M{"similarity": -1},
	})

	// Limit results
	pipeline = append(pipeline, bson.M{
		"$limit": options.Limit,
	})

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute document search aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var results []SearchResult
	for cursor.Next(ctx) {
		var doc struct {
			models.Document `bson:",inline"`
			Similarity      float64 `bson:"similarity"`
		}

		if err := cursor.Decode(&doc); err != nil {
			s.logger.Error("Failed to decode document search result", err, nil)
			continue
		}

		result := SearchResult{
			ID:       doc.ID.Hex(),
			Score:    doc.Similarity,
			Document: &doc.Document,
			Metadata: map[string]interface{}{
				"type":     "document",
				"category": doc.Metadata.Category,
				"tags":     doc.Metadata.Tags,
			},
		}

		results = append(results, result)
	}

	return results, nil
}

// searchKnowledgeItems performs vector search on knowledge_items collection
func (s *Service) searchKnowledgeItems(ctx context.Context, queryEmbedding []float64, options *SearchOptions) ([]SearchResult, error) {
	collection := s.mongodb.Collection("knowledge_items")

	// Build aggregation pipeline for vector search
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"embeddings": bson.M{"$exists": true, "$ne": nil},
				"is_active":  true,
			},
		},
	}

	// Add filters if specified
	if len(options.Filters) > 0 {
		matchStage := pipeline[0]["$match"].(bson.M)
		for key, value := range options.Filters {
			matchStage[key] = value
		}
	}

	// Add vector similarity calculation (same as documents)
	pipeline = append(pipeline, bson.M{
		"$addFields": bson.M{
			"similarity": bson.M{
				"$let": bson.M{
					"vars": bson.M{
						"dotProduct": bson.M{
							"$reduce": bson.M{
								"input": bson.M{
									"$zip": bson.M{
										"inputs": []interface{}{"$embeddings", queryEmbedding},
									},
								},
								"initialValue": 0,
								"in": bson.M{
									"$add": []interface{}{
										"$$value",
										bson.M{"$multiply": []interface{}{"$$this.0", "$$this.1"}},
									},
								},
							},
						},
						"magnitude1": bson.M{
							"$sqrt": bson.M{
								"$reduce": bson.M{
									"input":        "$embeddings",
									"initialValue": 0,
									"in": bson.M{
										"$add": []interface{}{"$$value", bson.M{"$multiply": []interface{}{"$$this", "$$this"}}},
									},
								},
							},
						},
						"magnitude2": bson.M{
							"$sqrt": bson.M{
								"$reduce": bson.M{
									"input":        queryEmbedding,
									"initialValue": 0,
									"in": bson.M{
										"$add": []interface{}{"$$value", bson.M{"$multiply": []interface{}{"$$this", "$$this"}}},
									},
								},
							},
						},
					},
					"in": bson.M{
						"$divide": []interface{}{
							"$$dotProduct",
							bson.M{"$multiply": []interface{}{"$$magnitude1", "$$magnitude2"}},
						},
					},
				},
			},
		},
	})

	// Filter by similarity threshold
	pipeline = append(pipeline, bson.M{
		"$match": bson.M{
			"similarity": bson.M{"$gte": options.Threshold},
		},
	})

	// Sort by similarity (descending)
	pipeline = append(pipeline, bson.M{
		"$sort": bson.M{"similarity": -1},
	})

	// Limit results
	pipeline = append(pipeline, bson.M{
		"$limit": options.Limit,
	})

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute knowledge search aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var results []SearchResult
	for cursor.Next(ctx) {
		var knowledge struct {
			models.KnowledgeItem `bson:",inline"`
			Similarity           float64 `bson:"similarity"`
		}

		if err := cursor.Decode(&knowledge); err != nil {
			s.logger.Error("Failed to decode knowledge search result", err, nil)
			continue
		}

		result := SearchResult{
			ID:        knowledge.ID.Hex(),
			Score:     knowledge.Similarity,
			Knowledge: &knowledge.KnowledgeItem,
			Metadata: map[string]interface{}{
				"type":       "knowledge",
				"category":   knowledge.Category,
				"tags":       knowledge.Tags,
				"confidence": knowledge.Confidence,
			},
		}

		results = append(results, result)
	}

	return results, nil
}

// BatchGenerateEmbeddings generates embeddings for multiple texts in batch
func (s *Service) BatchGenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		embedding, err := s.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embedding

		// Add small delay to avoid rate limiting
		if i < len(texts)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return embeddings, nil
}

// GetSimilarDocuments finds documents similar to a given document
func (s *Service) GetSimilarDocuments(ctx context.Context, documentID primitive.ObjectID, limit int) ([]SearchResult, error) {
	// Get the document and its embeddings
	collection := s.mongodb.Collection("documents")
	var document models.Document
	err := collection.FindOne(ctx, bson.M{"_id": documentID}).Decode(&document)
	if err != nil {
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	if len(document.Embeddings) == 0 {
		return nil, fmt.Errorf("document has no embeddings")
	}

	// Search for similar documents
	options := &SearchOptions{
		Limit:      limit,
		Threshold:  0.5, // Lower threshold for similarity search
		Collection: "documents",
		Filters: map[string]interface{}{
			"_id": bson.M{"$ne": documentID}, // Exclude the source document
		},
	}

	return s.searchDocuments(ctx, document.Embeddings, options)
}

// GetSimilarKnowledge finds knowledge items similar to a given knowledge item
func (s *Service) GetSimilarKnowledge(ctx context.Context, knowledgeID primitive.ObjectID, limit int) ([]SearchResult, error) {
	// Get the knowledge item and its embeddings
	collection := s.mongodb.Collection("knowledge_items")
	var knowledge models.KnowledgeItem
	err := collection.FindOne(ctx, bson.M{"_id": knowledgeID}).Decode(&knowledge)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge item: %w", err)
	}

	if len(knowledge.Embeddings) == 0 {
		return nil, fmt.Errorf("knowledge item has no embeddings")
	}

	// Search for similar knowledge items
	options := &SearchOptions{
		Limit:      limit,
		Threshold:  0.5, // Lower threshold for similarity search
		Collection: "knowledge_items",
		Filters: map[string]interface{}{
			"_id": bson.M{"$ne": knowledgeID}, // Exclude the source knowledge item
		},
	}

	return s.searchKnowledgeItems(ctx, knowledge.Embeddings, options)
}

// ClearCache clears the embedding cache
func (s *Service) ClearCache(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}

	pattern := "embedding:*"
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) > 0 {
		err = s.redis.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
	}

	s.logger.Info("Cleared embedding cache", map[string]interface{}{
		"keys_deleted": len(keys),
	})
	return nil
}
