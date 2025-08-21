package speech

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

// ElevenLabsTTSService implements TextToSpeechService using ElevenLabs API
type ElevenLabsTTSService struct {
	config     ElevenLabsConfig
	httpClient *http.Client
	rateLimiter *RateLimiter
}

// NewElevenLabsTTSService creates a new ElevenLabs TTS service
func NewElevenLabsTTSService(config ElevenLabsConfig) *ElevenLabsTTSService {
	return &ElevenLabsTTSService{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		rateLimiter: NewRateLimiter(config.RateLimit, time.Minute),
	}
}

// ConvertToSpeech converts text to speech using ElevenLabs API
func (s *ElevenLabsTTSService) ConvertToSpeech(ctx context.Context, text string, options TTSOptions) (*AudioResult, error) {
	// Validate text content
	validation := s.ValidateTextContent(text)
	if !validation.IsValid {
		return nil, fmt.Errorf("invalid text content: %v", validation.Issues)
	}

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Set default options
	if options.Voice == "" {
		options.Voice = s.config.DefaultVoice
	}
	if options.Speed == 0 {
		options.Speed = 1.0
	}
	if options.OutputFormat == "" {
		options.OutputFormat = "mp3"
	}
	if options.Quality == "" {
		options.Quality = "medium"
	}
	if options.Stability == 0 {
		options.Stability = 0.5
	}
	if options.Clarity == 0 {
		options.Clarity = 0.75
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"text": text,
		"model_id": s.getModelID(options.Quality),
		"voice_settings": map[string]interface{}{
			"stability":        options.Stability,
			"similarity_boost": options.Clarity,
			"style":           0.0,
			"use_speaker_boost": true,
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/text-to-speech/%s", s.config.BaseURL, options.Voice)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", s.config.APIKey)
	req.Header.Set("Accept", fmt.Sprintf("audio/%s", options.OutputFormat))

	// Execute request with retries
	var resp *http.Response
	var lastErr error
	
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		resp, lastErr = s.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		if attempt < s.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to execute request after %d attempts: %w", s.config.MaxRetries+1, lastErr)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Read audio data
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Create result
	result := &AudioResult{
		AudioData:   audioData,
		Duration:    validation.EstimatedDuration,
		Format:      options.OutputFormat,
		Size:        int64(len(audioData)),
		GeneratedAt: time.Now(),
		Voice:       options.Voice,
		Quality:     options.Quality,
	}

	return result, nil
}

// GetAvailableVoices retrieves available voices from ElevenLabs
func (s *ElevenLabsTTSService) GetAvailableVoices(ctx context.Context) ([]Voice, error) {
	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/voices", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("xi-api-key", s.config.APIKey)

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Voices []struct {
			VoiceID   string `json:"voice_id"`
			Name      string `json:"name"`
			Category  string `json:"category"`
			Labels    map[string]string `json:"labels"`
		} `json:"voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Voice structs
	voices := make([]Voice, len(response.Voices))
	for i, v := range response.Voices {
		voices[i] = Voice{
			ID:        v.VoiceID,
			Name:      v.Name,
			Language:  s.extractLanguage(v.Labels),
			Gender:    s.extractGender(v.Labels),
			Age:       s.extractAge(v.Labels),
			Style:     v.Category,
			SampleRate: 22050, // ElevenLabs default
			Formats:   []string{"mp3", "wav", "ogg"},
			IsDefault: v.VoiceID == s.config.DefaultVoice,
			Provider:  "elevenlabs",
			Quality:   "high",
		}
	}

	return voices, nil
}

// ValidateTextContent validates text content for TTS conversion
func (s *ElevenLabsTTSService) ValidateTextContent(text string) *ContentValidation {
	validation := &ContentValidation{
		IsValid:   true,
		Issues:    []string{},
		WordCount: len(strings.Fields(text)),
		CharCount: len(text),
	}

	// Check text length
	if len(text) == 0 {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "text is empty")
		return validation
	}

	if len(text) > 5000 { // ElevenLabs character limit
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "text exceeds maximum length of 5000 characters")
	}

	// Check for invalid characters
	if strings.Contains(text, "\x00") {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "text contains null characters")
	}

	// Estimate duration (approximately 150 words per minute)
	wordsPerMinute := 150.0
	validation.EstimatedDuration = float64(validation.WordCount) / wordsPerMinute * 60.0

	return validation
}

// getModelID returns the appropriate model ID based on quality setting
func (s *ElevenLabsTTSService) getModelID(quality string) string {
	switch quality {
	case "low":
		return "eleven_turbo_v2"
	case "high":
		return "eleven_multilingual_v2"
	default: // medium
		return "eleven_monolingual_v1"
	}
}

// extractLanguage extracts language from voice labels
func (s *ElevenLabsTTSService) extractLanguage(labels map[string]string) string {
	if lang, ok := labels["language"]; ok {
		return lang
	}
	return "en" // default to English
}

// extractGender extracts gender from voice labels
func (s *ElevenLabsTTSService) extractGender(labels map[string]string) string {
	if gender, ok := labels["gender"]; ok {
		return gender
	}
	return "unknown"
}

// extractAge extracts age from voice labels
func (s *ElevenLabsTTSService) extractAge(labels map[string]string) string {
	if age, ok := labels["age"]; ok {
		return age
	}
	return "unknown"
}

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	tokens   chan struct{}
	interval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:   make(chan struct{}, limit),
		interval: interval,
	}

	// Fill the bucket initially
	for i := 0; i < limit; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens periodically
	go func() {
		ticker := time.NewTicker(interval / time.Duration(limit))
		defer ticker.Stop()
		
		for range ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Bucket is full
			}
		}
	}()

	return rl
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}