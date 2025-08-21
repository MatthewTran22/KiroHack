package speech

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

// TestElevenLabsTTSService_ConvertToSpeech tests text-to-speech conversion
func TestElevenLabsTTSService_ConvertToSpeech(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/text-to-speech/")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("xi-api-key"))

		// Parse request body
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)

		// Verify request content
		assert.Equal(t, "Hello, this is a test", requestBody["text"])
		assert.NotNil(t, requestBody["voice_settings"])

		// Return mock audio data
		w.Header().Set("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-mp3-audio-data"))
	}))
	defer server.Close()

	// Create service
	config := ElevenLabsConfig{
		APIKey:      "test-api-key",
		BaseURL:     server.URL,
		DefaultVoice: "test-voice",
		MaxRetries:  1,
		Timeout:     10,
		RateLimit:   60,
	}
	service := NewElevenLabsTTSService(config)

	// Test data
	ctx := context.Background()
	text := "Hello, this is a test"
	options := TTSOptions{
		Voice:        "test-voice",
		Speed:        1.0,
		Language:     "en",
		OutputFormat: "mp3",
		Quality:      "medium",
		Stability:    0.5,
		Clarity:      0.75,
	}

	// Test
	result, err := service.ConvertToSpeech(ctx, text, options)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("fake-mp3-audio-data"), result.AudioData)
	assert.Equal(t, "mp3", result.Format)
	assert.Equal(t, "test-voice", result.Voice)
	assert.Equal(t, "medium", result.Quality)
	assert.Equal(t, int64(len("fake-mp3-audio-data")), result.Size)
	assert.True(t, result.Duration > 0)
}

// TestElevenLabsTTSService_ConvertToSpeech_APIError tests API error handling
func TestElevenLabsTTSService_ConvertToSpeech_APIError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail": "Invalid voice ID"}`))
	}))
	defer server.Close()

	// Create service
	config := ElevenLabsConfig{
		APIKey:      "test-api-key",
		BaseURL:     server.URL,
		DefaultVoice: "invalid-voice",
		MaxRetries:  1,
		Timeout:     10,
		RateLimit:   60,
	}
	service := NewElevenLabsTTSService(config)

	// Test data
	ctx := context.Background()
	text := "Hello, this is a test"
	options := TTSOptions{}

	// Test
	result, err := service.ConvertToSpeech(ctx, text, options)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ElevenLabs API error")
}

// TestElevenLabsTTSService_GetAvailableVoices tests getting available voices
func TestElevenLabsTTSService_GetAvailableVoices(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/voices", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("xi-api-key"))

		// Return mock voices
		response := map[string]interface{}{
			"voices": []map[string]interface{}{
				{
					"voice_id": "voice1",
					"name":     "Test Voice 1",
					"category": "professional",
					"labels": map[string]string{
						"language": "en",
						"gender":   "female",
						"age":      "young",
					},
				},
				{
					"voice_id": "voice2",
					"name":     "Test Voice 2",
					"category": "conversational",
					"labels": map[string]string{
						"language": "es",
						"gender":   "male",
						"age":      "middle",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create service
	config := ElevenLabsConfig{
		APIKey:      "test-api-key",
		BaseURL:     server.URL,
		DefaultVoice: "voice1",
		MaxRetries:  1,
		Timeout:     10,
		RateLimit:   60,
	}
	service := NewElevenLabsTTSService(config)

	// Test
	ctx := context.Background()
	voices, err := service.GetAvailableVoices(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, voices, 2)

	// Check first voice
	assert.Equal(t, "voice1", voices[0].ID)
	assert.Equal(t, "Test Voice 1", voices[0].Name)
	assert.Equal(t, "en", voices[0].Language)
	assert.Equal(t, "female", voices[0].Gender)
	assert.Equal(t, "young", voices[0].Age)
	assert.Equal(t, "professional", voices[0].Style)
	assert.True(t, voices[0].IsDefault)
	assert.Equal(t, "elevenlabs", voices[0].Provider)

	// Check second voice
	assert.Equal(t, "voice2", voices[1].ID)
	assert.Equal(t, "Test Voice 2", voices[1].Name)
	assert.Equal(t, "es", voices[1].Language)
	assert.Equal(t, "male", voices[1].Gender)
	assert.Equal(t, "middle", voices[1].Age)
	assert.Equal(t, "conversational", voices[1].Style)
	assert.False(t, voices[1].IsDefault)
}

// TestElevenLabsTTSService_ValidateTextContent tests text validation
func TestElevenLabsTTSService_ValidateTextContent(t *testing.T) {
	service := &ElevenLabsTTSService{}

	tests := []struct {
		name        string
		text        string
		expectValid bool
		expectIssues []string
	}{
		{
			name:        "Valid text",
			text:        "This is a valid text for TTS conversion.",
			expectValid: true,
			expectIssues: []string{},
		},
		{
			name:        "Empty text",
			text:        "",
			expectValid: false,
			expectIssues: []string{"text is empty"},
		},
		{
			name:        "Text too long",
			text:        string(make([]byte, 5001)), // 5001 characters
			expectValid: false,
			expectIssues: []string{"text exceeds maximum length of 5000 characters"},
		},
		{
			name:        "Text with null characters",
			text:        "Hello\x00World",
			expectValid: false,
			expectIssues: []string{"text contains null characters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := service.ValidateTextContent(tt.text)

			assert.Equal(t, tt.expectValid, validation.IsValid)
			
			if len(tt.expectIssues) > 0 {
				for _, expectedIssue := range tt.expectIssues {
					assert.Contains(t, validation.Issues, expectedIssue)
				}
			}

			if tt.expectValid {
				assert.True(t, validation.WordCount > 0)
				assert.True(t, validation.CharCount > 0)
				assert.True(t, validation.EstimatedDuration > 0)
			}
		})
	}
}

// TestElevenLabsTTSService_RateLimiting tests rate limiting functionality
func TestElevenLabsTTSService_RateLimiting(t *testing.T) {
	// Create a rate limiter with very low limit for testing
	rateLimiter := NewRateLimiter(2, time.Second) // 2 requests per second

	ctx := context.Background()

	// First two requests should succeed immediately
	err1 := rateLimiter.Wait(ctx)
	assert.NoError(t, err1)

	err2 := rateLimiter.Wait(ctx)
	assert.NoError(t, err2)

	// Third request should block (we'll use a timeout to test this)
	ctx3, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err3 := rateLimiter.Wait(ctx3)
	duration := time.Since(start)

	// Should timeout because rate limit is exceeded
	assert.Error(t, err3)
	assert.True(t, duration >= 100*time.Millisecond)
}

// TestElevenLabsTTSService_DefaultOptions tests default option handling
func TestElevenLabsTTSService_DefaultOptions(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to check defaults
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		// Check that defaults are applied
		voiceSettings := requestBody["voice_settings"].(map[string]interface{})
		assert.Equal(t, 0.5, voiceSettings["stability"])
		assert.Equal(t, 0.75, voiceSettings["similarity_boost"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-audio"))
	}))
	defer server.Close()

	// Create service
	config := ElevenLabsConfig{
		APIKey:      "test-api-key",
		BaseURL:     server.URL,
		DefaultVoice: "default-voice",
		MaxRetries:  1,
		Timeout:     10,
		RateLimit:   60,
	}
	service := NewElevenLabsTTSService(config)

	// Test with empty options (should use defaults)
	ctx := context.Background()
	text := "Test text"
	options := TTSOptions{} // Empty options

	result, err := service.ConvertToSpeech(ctx, text, options)

	// Should succeed with defaults applied
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestElevenLabsTTSService_Retry tests retry functionality
func TestElevenLabsTTSService_Retry(t *testing.T) {
	callCount := 0
	
	// Mock server that fails first time, succeeds second time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call fails with server error
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else {
			// Second call succeeds
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success-audio"))
		}
	}))
	defer server.Close()

	// Create service with retry enabled
	config := ElevenLabsConfig{
		APIKey:      "test-api-key",
		BaseURL:     server.URL,
		DefaultVoice: "test-voice",
		MaxRetries:  2, // Allow 2 retries
		Timeout:     10,
		RateLimit:   60,
	}
	service := NewElevenLabsTTSService(config)

	// Test
	ctx := context.Background()
	text := "Test text"
	options := TTSOptions{Voice: "test-voice"}

	result, err := service.ConvertToSpeech(ctx, text, options)

	// Should succeed after retry
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("success-audio"), result.AudioData)
	assert.Equal(t, 2, callCount) // Should have been called twice
}