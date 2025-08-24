package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// ElevenLabsSTTService implements SpeechToTextService using ElevenLabs API
type ElevenLabsSTTService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// ElevenLabsSTTConfig contains configuration for ElevenLabs STT service
type ElevenLabsSTTConfig struct {
	APIKey  string        `json:"api_key"`
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout"`
}

// ElevenLabsSTTResponse represents the response from ElevenLabs STT API
type ElevenLabsSTTResponse struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// NewElevenLabsSTTService creates a new ElevenLabs STT service
func NewElevenLabsSTTService(config ElevenLabsSTTConfig) (*ElevenLabsSTTService, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.elevenlabs.io"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &ElevenLabsSTTService{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// TranscribeAudio transcribes audio using ElevenLabs STT API
func (s *ElevenLabsSTTService) TranscribeAudio(ctx context.Context, audioData []byte, options STTOptions) (*TranscriptionResult, error) {
	startTime := time.Now()

	// Validate audio format
	validation := s.ValidateAudioFormat(audioData)
	if !validation.IsValid {
		return nil, fmt.Errorf("invalid audio format: %v", validation.Issues)
	}

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	part, err := writer.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model parameter if specified
	if options.Model != "" {
		if err := writer.WriteField("model", options.Model); err != nil {
			return nil, fmt.Errorf("failed to write model field: %w", err)
		}
	}

	// Add language parameter if specified
	if options.Language != "" {
		if err := writer.WriteField("language", options.Language); err != nil {
			return nil, fmt.Errorf("failed to write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/speech-to-text", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", s.apiKey)

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ElevenLabs API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var sttResponse ElevenLabsSTTResponse
	if err := json.Unmarshal(responseBody, &sttResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Create transcription result
	result := &TranscriptionResult{
		Text:           sttResponse.Text,
		Confidence:     sttResponse.Confidence,
		Language:       options.Language,
		Duration:       validation.Duration,
		Timestamps:     []WordTimestamp{}, // ElevenLabs doesn't provide word-level timestamps
		ProcessedAt:    time.Now(),
		ModelUsed:      "elevenlabs-stt",
		ProcessingTime: time.Since(startTime).Seconds(),
		Segments:       []TextSegment{}, // ElevenLabs doesn't provide segments
	}

	// If confidence is not provided, set a default high confidence
	if result.Confidence == 0 {
		result.Confidence = 0.95 // ElevenLabs typically has high accuracy
	}

	return result, nil
}

// GetSupportedLanguages returns supported languages for ElevenLabs STT
func (s *ElevenLabsSTTService) GetSupportedLanguages() ([]Language, error) {
	// ElevenLabs supports many languages
	languages := []Language{
		{Code: "en", Name: "English", Region: "US", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "es", Name: "Spanish", Region: "ES", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "fr", Name: "French", Region: "FR", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "de", Name: "German", Region: "DE", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "it", Name: "Italian", Region: "IT", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "pt", Name: "Portuguese", Region: "PT", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "pl", Name: "Polish", Region: "PL", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "tr", Name: "Turkish", Region: "TR", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "ru", Name: "Russian", Region: "RU", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "nl", Name: "Dutch", Region: "NL", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "cs", Name: "Czech", Region: "CZ", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "ar", Name: "Arabic", Region: "SA", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "zh", Name: "Chinese", Region: "CN", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "ja", Name: "Japanese", Region: "JP", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "hu", Name: "Hungarian", Region: "HU", IsSupported: true, ModelPath: "elevenlabs-stt"},
		{Code: "ko", Name: "Korean", Region: "KR", IsSupported: true, ModelPath: "elevenlabs-stt"},
	}

	return languages, nil
}

// ValidateAudioFormat validates audio format for ElevenLabs STT processing
func (s *ElevenLabsSTTService) ValidateAudioFormat(audioData []byte) *FormatValidation {
	validation := &FormatValidation{
		IsValid: true,
		Issues:  []string{},
	}

	// Check minimum size
	if len(audioData) < 1024 {
		validation.IsValid = false
		validation.Issues = append(validation.Issues, "audio data too small")
		return validation
	}

	// ElevenLabs supports various formats, but let's validate WAV
	if len(audioData) >= 44 {
		// Check WAV header
		if string(audioData[0:4]) == "RIFF" && string(audioData[8:12]) == "WAVE" {
			validation.Format = "wav"
			
			// Extract sample rate from WAV header
			sampleRateBytes := audioData[24:28]
			sampleRate := int(sampleRateBytes[0]) | int(sampleRateBytes[1])<<8 | 
						 int(sampleRateBytes[2])<<16 | int(sampleRateBytes[3])<<24
			validation.SampleRate = sampleRate
			
			// Extract channels
			channelBytes := audioData[22:24]
			channels := int(channelBytes[0]) | int(channelBytes[1])<<8
			validation.Channels = channels
			
			// Calculate duration (approximate)
			dataSize := len(audioData) - 44 // Subtract header size
			bytesPerSample := 2 // Assuming 16-bit
			validation.Duration = float64(dataSize) / float64(sampleRate*channels*bytesPerSample)
		} else {
			// Check for MP3 header
			if len(audioData) >= 3 && string(audioData[0:3]) == "ID3" {
				validation.Format = "mp3"
			} else if len(audioData) >= 2 && audioData[0] == 0xFF && (audioData[1]&0xE0) == 0xE0 {
				validation.Format = "mp3"
			} else {
				validation.Format = "unknown"
				validation.Issues = append(validation.Issues, "unsupported audio format")
			}
		}
	} else {
		validation.Issues = append(validation.Issues, "invalid audio header")
		validation.IsValid = false
	}

	validation.Size = int64(len(audioData))

	// ElevenLabs has generous limits
	if validation.Duration > 600 { // 10 minutes
		validation.Issues = append(validation.Issues, "audio too long for transcription")
	}

	// Check for critical issues
	for _, issue := range validation.Issues {
		if containsCriticalKeyword(issue) {
			validation.IsValid = false
			break
		}
	}

	return validation
}

// GetServiceInfo returns information about the ElevenLabs STT service
func (s *ElevenLabsSTTService) GetServiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":           "ElevenLabs",
		"service":           "Speech-to-Text",
		"base_url":          s.baseURL,
		"max_audio_length":  600, // 10 minutes
		"supported_formats": []string{"wav", "mp3", "ogg", "flac"},
		"features": []string{
			"Multi-language support",
			"High accuracy",
			"Fast processing",
			"Cloud-based",
		},
	}
}