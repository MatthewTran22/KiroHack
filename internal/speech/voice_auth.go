package speech

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoVoiceAuthenticator implements VoiceAuthenticator using MongoDB
type MongoVoiceAuthenticator struct {
	collection      *mongo.Collection
	audioProcessor  AudioProcessor
	embeddingModel  VoiceEmbeddingModel
	config          VoiceAuthConfig
}

// VoiceAuthConfig contains configuration for voice authentication
type VoiceAuthConfig struct {
	MinSamples          int     `json:"min_samples"`           // Minimum samples for enrollment
	MaxSamples          int     `json:"max_samples"`           // Maximum samples to store
	SimilarityThreshold float64 `json:"similarity_threshold"`  // Threshold for authentication
	MinAudioDuration    float64 `json:"min_audio_duration"`    // Minimum audio duration in seconds
	MaxAudioDuration    float64 `json:"max_audio_duration"`    // Maximum audio duration in seconds
	UpdateThreshold     float64 `json:"update_threshold"`      // Threshold for profile updates
}

// VoiceEmbeddingModel defines interface for voice embedding generation
type VoiceEmbeddingModel interface {
	GenerateEmbedding(audioData []byte) ([]float64, error)
	GetEmbeddingDimension() int
}

// NewMongoVoiceAuthenticator creates a new MongoDB-based voice authenticator
func NewMongoVoiceAuthenticator(db *mongo.Database, audioProcessor AudioProcessor, embeddingModel VoiceEmbeddingModel) *MongoVoiceAuthenticator {
	config := VoiceAuthConfig{
		MinSamples:          3,
		MaxSamples:          10,
		SimilarityThreshold: 0.85,
		MinAudioDuration:    2.0,  // 2 seconds
		MaxAudioDuration:    30.0, // 30 seconds
		UpdateThreshold:     0.90,
	}

	return &MongoVoiceAuthenticator{
		collection:     db.Collection("voice_profiles"),
		audioProcessor: audioProcessor,
		embeddingModel: embeddingModel,
		config:         config,
	}
}

// EnrollVoice enrolls a user's voice for authentication
func (v *MongoVoiceAuthenticator) EnrollVoice(ctx context.Context, userID string, audioSamples [][]byte) (*VoiceProfile, error) {
	if len(audioSamples) < v.config.MinSamples {
		return nil, fmt.Errorf("insufficient audio samples: need at least %d, got %d", v.config.MinSamples, len(audioSamples))
	}

	// Validate and preprocess audio samples
	var embeddings [][]float64
	var validSamples [][]byte

	for i, sample := range audioSamples {
		// Validate audio duration
		validation := v.validateAudioSample(sample)
		if !validation.IsValid {
			return nil, fmt.Errorf("invalid audio sample %d: %v", i, validation.Issues)
		}

		// Preprocess audio
		processed, err := v.preprocessVoiceAudio(sample)
		if err != nil {
			return nil, fmt.Errorf("failed to preprocess audio sample %d: %w", i, err)
		}

		// Generate embedding
		embedding, err := v.embeddingModel.GenerateEmbedding(processed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for sample %d: %w", i, err)
		}

		embeddings = append(embeddings, embedding)
		validSamples = append(validSamples, processed)
	}

	// Calculate quality score based on embedding consistency
	quality := v.calculateEmbeddingQuality(embeddings)
	if quality < 0.7 {
		return nil, fmt.Errorf("voice samples quality too low: %.2f (minimum 0.7)", quality)
	}

	// Create voice profile
	profile := &VoiceProfile{
		UserID:      userID,
		Embeddings:  embeddings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		SampleCount: len(embeddings),
		Quality:     quality,
		IsActive:    true,
	}

	// Store in database
	_, err := v.collection.InsertOne(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to store voice profile: %w", err)
	}

	return profile, nil
}

// AuthenticateVoice authenticates a user's voice
func (v *MongoVoiceAuthenticator) AuthenticateVoice(ctx context.Context, userID string, audioSample []byte) (*AuthenticationResult, error) {
	startTime := time.Now()

	// Validate audio sample
	validation := v.validateAudioSample(audioSample)
	if !validation.IsValid {
		return &AuthenticationResult{
			IsAuthenticated: false,
			Confidence:      0.0,
			Threshold:       v.config.SimilarityThreshold,
			ProcessingTime:  time.Since(startTime).Seconds(),
			Reason:          fmt.Sprintf("invalid audio: %v", validation.Issues),
		}, nil
	}

	// Get user's voice profile
	profile, err := v.getVoiceProfile(ctx, userID)
	if err != nil {
		return &AuthenticationResult{
			IsAuthenticated: false,
			Confidence:      0.0,
			Threshold:       v.config.SimilarityThreshold,
			ProcessingTime:  time.Since(startTime).Seconds(),
			Reason:          "voice profile not found",
		}, nil
	}

	if !profile.IsActive {
		return &AuthenticationResult{
			IsAuthenticated: false,
			Confidence:      0.0,
			Threshold:       v.config.SimilarityThreshold,
			ProcessingTime:  time.Since(startTime).Seconds(),
			Reason:          "voice profile inactive",
		}, nil
	}

	// Preprocess audio
	processed, err := v.preprocessVoiceAudio(audioSample)
	if err != nil {
		return &AuthenticationResult{
			IsAuthenticated: false,
			Confidence:      0.0,
			Threshold:       v.config.SimilarityThreshold,
			ProcessingTime:  time.Since(startTime).Seconds(),
			Reason:          "audio preprocessing failed",
		}, nil
	}

	// Generate embedding for the sample
	sampleEmbedding, err := v.embeddingModel.GenerateEmbedding(processed)
	if err != nil {
		return &AuthenticationResult{
			IsAuthenticated: false,
			Confidence:      0.0,
			Threshold:       v.config.SimilarityThreshold,
			ProcessingTime:  time.Since(startTime).Seconds(),
			Reason:          "embedding generation failed",
		}, nil
	}

	// Calculate similarity with stored embeddings
	maxSimilarity := 0.0
	for _, storedEmbedding := range profile.Embeddings {
		similarity := v.calculateCosineSimilarity(sampleEmbedding, storedEmbedding)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}
	}

	// Determine authentication result
	isAuthenticated := maxSimilarity >= v.config.SimilarityThreshold
	
	result := &AuthenticationResult{
		IsAuthenticated: isAuthenticated,
		Confidence:      maxSimilarity,
		Threshold:       v.config.SimilarityThreshold,
		ProcessingTime:  time.Since(startTime).Seconds(),
	}

	if isAuthenticated {
		result.Reason = "voice authenticated successfully"
		
		// Update profile if similarity is very high
		if maxSimilarity >= v.config.UpdateThreshold {
			go v.updateVoiceProfileAsync(ctx, userID, sampleEmbedding)
		}
	} else {
		result.Reason = fmt.Sprintf("similarity %.3f below threshold %.3f", maxSimilarity, v.config.SimilarityThreshold)
	}

	return result, nil
}

// UpdateVoiceProfile updates a user's voice profile with new sample
func (v *MongoVoiceAuthenticator) UpdateVoiceProfile(ctx context.Context, userID string, audioSample []byte) error {
	// Validate audio sample
	validation := v.validateAudioSample(audioSample)
	if !validation.IsValid {
		return fmt.Errorf("invalid audio sample: %v", validation.Issues)
	}

	// Get existing profile
	profile, err := v.getVoiceProfile(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get voice profile: %w", err)
	}

	// Preprocess audio
	processed, err := v.preprocessVoiceAudio(audioSample)
	if err != nil {
		return fmt.Errorf("failed to preprocess audio: %w", err)
	}

	// Generate embedding
	newEmbedding, err := v.embeddingModel.GenerateEmbedding(processed)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Add new embedding to profile
	profile.Embeddings = append(profile.Embeddings, newEmbedding)
	
	// Remove oldest embedding if we exceed max samples
	if len(profile.Embeddings) > v.config.MaxSamples {
		profile.Embeddings = profile.Embeddings[1:]
	}

	// Recalculate quality
	profile.Quality = v.calculateEmbeddingQuality(profile.Embeddings)
	profile.SampleCount = len(profile.Embeddings)
	profile.UpdatedAt = time.Now()

	// Update in database
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"embeddings":   profile.Embeddings,
			"quality":      profile.Quality,
			"sample_count": profile.SampleCount,
			"updated_at":   profile.UpdatedAt,
		},
	}

	_, err = v.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update voice profile: %w", err)
	}

	return nil
}

// DeleteVoiceProfile deletes a user's voice profile
func (v *MongoVoiceAuthenticator) DeleteVoiceProfile(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}
	result, err := v.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete voice profile: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("voice profile not found")
	}

	return nil
}

// getVoiceProfile retrieves a user's voice profile
func (v *MongoVoiceAuthenticator) getVoiceProfile(ctx context.Context, userID string) (*VoiceProfile, error) {
	var profile VoiceProfile
	filter := bson.M{"user_id": userID}
	
	err := v.collection.FindOne(ctx, filter).Decode(&profile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("voice profile not found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to retrieve voice profile: %w", err)
	}

	return &profile, nil
}

// validateAudioSample validates an audio sample for voice authentication
func (v *MongoVoiceAuthenticator) validateAudioSample(audioData []byte) *FormatValidation {
	validation := &FormatValidation{
		IsValid: true,
		Issues:  []string{},
	}

	// Check minimum size
	if len(audioData) < 1024 {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "audio sample too small")
		return validation
	}

	// Validate WAV format
	validator := NewAudioValidator()
	if err := validator.ValidateWAVFormat(audioData); err != nil {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, err.Error())
		return validation
	}

	// Get audio info
	info, err := validator.GetAudioInfo(audioData)
	if err != nil {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "failed to extract audio info")
		return validation
	}

	validation.Format = info.Format
	validation.SampleRate = info.SampleRate
	validation.Channels = info.Channels
	validation.Duration = info.Duration
	validation.Size = info.Size

	// Check duration constraints
	if info.Duration < v.config.MinAudioDuration {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, 
			fmt.Sprintf("audio too short: %.1fs (minimum %.1fs)", info.Duration, v.config.MinAudioDuration))
	}

	if info.Duration > v.config.MaxAudioDuration {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, 
			fmt.Sprintf("audio too long: %.1fs (maximum %.1fs)", info.Duration, v.config.MaxAudioDuration))
	}

	// Check sample rate (16kHz is optimal for voice)
	if info.SampleRate < 8000 {
		validation.Issues = append(validation.Issues, "sample rate too low for voice recognition")
	}

	return validation
}

// preprocessVoiceAudio preprocesses audio for voice authentication
func (v *MongoVoiceAuthenticator) preprocessVoiceAudio(audioData []byte) ([]byte, error) {
	// Target format for voice authentication
	targetFormat := AudioFormat{
		Codec:      "wav",
		SampleRate: 16000, // 16kHz is optimal for voice
		Channels:   1,     // Mono
		BitDepth:   16,    // 16-bit
	}

	// Preprocess audio
	processed, err := v.audioProcessor.PreprocessAudio(audioData, targetFormat)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	// Normalize audio
	normalized, err := v.audioProcessor.NormalizeAudio(processed)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	return normalized, nil
}

// calculateEmbeddingQuality calculates the quality of voice embeddings
func (v *MongoVoiceAuthenticator) calculateEmbeddingQuality(embeddings [][]float64) float64 {
	if len(embeddings) < 2 {
		return 0.5 // Default quality for single embedding
	}

	// Calculate pairwise similarities
	var similarities []float64
	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			similarity := v.calculateCosineSimilarity(embeddings[i], embeddings[j])
			similarities = append(similarities, similarity)
		}
	}

	// Calculate average similarity (consistency)
	sum := 0.0
	for _, sim := range similarities {
		sum += sim
	}
	avgSimilarity := sum / float64(len(similarities))

	// Quality is based on consistency (high similarity between samples)
	// but penalized if too high (might indicate identical samples)
	if avgSimilarity > 0.95 {
		return 0.8 // Penalize for potentially identical samples
	}

	return avgSimilarity
}

// calculateCosineSimilarity calculates cosine similarity between two embeddings
func (v *MongoVoiceAuthenticator) calculateCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// updateVoiceProfileAsync updates voice profile asynchronously
func (v *MongoVoiceAuthenticator) updateVoiceProfileAsync(ctx context.Context, userID string, embedding []float64) {
	// This would run in a goroutine to avoid blocking authentication
	// Implementation would be similar to UpdateVoiceProfile but with the embedding directly
}

// SimpleVoiceEmbeddingModel provides a simple embedding model for testing
type SimpleVoiceEmbeddingModel struct {
	dimension int
}

// NewSimpleVoiceEmbeddingModel creates a simple voice embedding model
func NewSimpleVoiceEmbeddingModel(dimension int) *SimpleVoiceEmbeddingModel {
	return &SimpleVoiceEmbeddingModel{
		dimension: dimension,
	}
}

// GenerateEmbedding generates a simple embedding based on audio characteristics
func (m *SimpleVoiceEmbeddingModel) GenerateEmbedding(audioData []byte) ([]float64, error) {
	// This is a simplified implementation for testing
	// In production, you would use a proper voice embedding model
	
	embedding := make([]float64, m.dimension)
	
	// Generate pseudo-random embedding based on audio data
	seed := int64(0)
	for i := 0; i < len(audioData) && i < 1000; i++ {
		seed += int64(audioData[i])
	}
	
	// Use seed to generate consistent but varied embeddings
	for i := 0; i < m.dimension; i++ {
		seed = seed*1103515245 + 12345 // Linear congruential generator
		embedding[i] = float64((seed>>16)&0x7fff) / 32768.0 - 0.5
	}
	
	// Normalize embedding
	norm := 0.0
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)
	
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}
	
	return embedding, nil
}

// GetEmbeddingDimension returns the embedding dimension
func (m *SimpleVoiceEmbeddingModel) GetEmbeddingDimension() int {
	return m.dimension
}