package speech

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// ElevenLabsServiceSTT implements SpeechToTextService using ElevenLabs microservice
type ElevenLabsServiceSTT struct {
	baseURL    string
	httpClient *http.Client
}

// ElevenLabsServiceConfig contains configuration for ElevenLabs microservice
type ElevenLabsServiceConfig struct {
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout"`
}

// ElevenLabsServiceSTTResponse represents the response from ElevenLabs microservice
type ElevenLabsServiceSTTResponse struct {
	Text           string  `json:"text"`
	Confidence     float64 `json:"confidence"`
	Language       string  `json:"language"`
	ProcessingTime float64 `json:"processing_time"`
	ModelID        string  `json:"model_id"`
}

// ElevenLabsServiceTTSRequest represents TTS request to microservice
type ElevenLabsServiceTTSRequest struct {
	Text          string                 `json:"text"`
	VoiceID       string                 `json:"voice_id,omitempty"`
	ModelID       string                 `json:"model_id,omitempty"`
	VoiceSettings map[string]interface{} `json:"voice_settings,omitempty"`
}

// ElevenLabsServiceTTSResponse represents TTS response from microservice
type ElevenLabsServiceTTSResponse struct {
	AudioData   string `json:"audio_data"`
	VoiceID     string `json:"voice_id"`
	ModelID     string `json:"model_id"`
	Duration    float64 `json:"duration"`
	Size        int    `json:"size"`
	GeneratedAt string `json:"generated_at"`
}

// NewElevenLabsServiceSTT creates a new ElevenLabs microservice STT client
func NewElevenLabsServiceSTT(config ElevenLabsServiceConfig) (*ElevenLabsServiceSTT, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	service := &ElevenLabsServiceSTT{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Test connection
	if err := service.healthCheck(); err != nil {
		return nil, fmt.Errorf("ElevenLabs microservice health check failed: %w", err)
	}

	return service, nil
}

// TranscribeAudio transcribes audio using ElevenLabs microservice
func (s *ElevenLabsServiceSTT) TranscribeAudio(ctx context.Context, audioData []byte, options STTOptions) (*TranscriptionResult, error) {
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
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add optional parameters
	if options.Model != "" {
		if err := writer.WriteField("model_id", options.Model); err != nil {
			return nil, fmt.Errorf("failed to write model field: %w", err)
		}
	}

	if options.Language != "" {
		if err := writer.WriteField("language", options.Language); err != nil {
			return nil, fmt.Errorf("failed to write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/stt-file", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

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
		return nil, fmt.Errorf("ElevenLabs microservice returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var sttResponse ElevenLabsServiceSTTResponse
	if err := json.Unmarshal(responseBody, &sttResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Create transcription result
	result := &TranscriptionResult{
		Text:           sttResponse.Text,
		Confidence:     sttResponse.Confidence,
		Language:       sttResponse.Language,
		Duration:       validation.Duration,
		Timestamps:     []WordTimestamp{}, // ElevenLabs doesn't provide word-level timestamps
		ProcessedAt:    time.Now(),
		ModelUsed:      sttResponse.ModelID,
		ProcessingTime: time.Since(startTime).Seconds(),
		Segments:       []TextSegment{}, // ElevenLabs doesn't provide segments
	}

	return result, nil
}

// GetSupportedLanguages returns supported languages for ElevenLabs STT
func (s *ElevenLabsServiceSTT) GetSupportedLanguages() ([]Language, error) {
	// ElevenLabs supports many languages
	languages := []Language{
		{Code: "en", Name: "English", Region: "US", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "es", Name: "Spanish", Region: "ES", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "fr", Name: "French", Region: "FR", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "de", Name: "German", Region: "DE", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "it", Name: "Italian", Region: "IT", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "pt", Name: "Portuguese", Region: "PT", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "pl", Name: "Polish", Region: "PL", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "tr", Name: "Turkish", Region: "TR", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "ru", Name: "Russian", Region: "RU", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "nl", Name: "Dutch", Region: "NL", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "cs", Name: "Czech", Region: "CZ", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "ar", Name: "Arabic", Region: "SA", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "zh", Name: "Chinese", Region: "CN", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "ja", Name: "Japanese", Region: "JP", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "hu", Name: "Hungarian", Region: "HU", IsSupported: true, ModelPath: "whisper-1"},
		{Code: "ko", Name: "Korean", Region: "KR", IsSupported: true, ModelPath: "whisper-1"},
	}

	return languages, nil
}

// ValidateAudioFormat validates audio format for ElevenLabs processing
func (s *ElevenLabsServiceSTT) ValidateAudioFormat(audioData []byte) *FormatValidation {
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

	// ElevenLabs supports various formats
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

// healthCheck checks if the ElevenLabs microservice is healthy
func (s *ElevenLabsServiceSTT) healthCheck() error {
	url := fmt.Sprintf("%s/health", s.baseURL)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetServiceInfo returns information about the ElevenLabs microservice
func (s *ElevenLabsServiceSTT) GetServiceInfo(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/models", s.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return info, nil
}

// GenerateSpeech generates speech using ElevenLabs microservice TTS
func (s *ElevenLabsServiceSTT) GenerateSpeech(ctx context.Context, text string, voiceID string, options map[string]interface{}) ([]byte, error) {
	// Prepare TTS request
	request := ElevenLabsServiceTTSRequest{
		Text:          text,
		VoiceID:       voiceID,
		ModelID:       "eleven_monolingual_v1",
		VoiceSettings: options,
	}

	// Convert to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/tts", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("ElevenLabs microservice returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var ttsResponse ElevenLabsServiceTTSResponse
	if err := json.Unmarshal(responseBody, &ttsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(ttsResponse.AudioData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	return audioData, nil
}