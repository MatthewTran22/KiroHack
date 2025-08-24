package speech

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// SpeechService provides a unified interface for all speech-related operations
type SpeechService struct {
	ttsService        TextToSpeechService
	sttService        SpeechToTextService
	sessionManager    SpeechSessionManager
	voiceAuth         VoiceAuthenticator
	audioProcessor    AudioProcessor
	config            SpeechServiceConfig
}

// SpeechServiceConfig contains configuration for the speech service
type SpeechServiceConfig struct {
	EnableTTS         bool                      `json:"enable_tts"`
	EnableSTT         bool                      `json:"enable_stt"`
	EnableVoiceAuth   bool                      `json:"enable_voice_auth"`
	DefaultLanguage   string                    `json:"default_language"`
	MaxSessionTime    time.Duration             `json:"max_session_time"`
	CleanupInterval   time.Duration             `json:"cleanup_interval"`
	ElevenLabs        ElevenLabsConfig          `json:"elevenlabs"`
	ElevenLabsSTT     ElevenLabsSTTConfig       `json:"elevenlabs_stt"`
	ElevenLabsService ElevenLabsServiceConfig   `json:"elevenlabs_service"`
	VoiceAuth         VoiceAuthConfig           `json:"voice_auth"`
}

// NewSpeechService creates a new speech service with all components
func NewSpeechService(db *mongo.Database, config SpeechServiceConfig) (*SpeechService, error) {
	// Initialize audio processor
	audioProcessor := NewDefaultAudioProcessor()

	// Initialize TTS service
	var ttsService TextToSpeechService
	if config.EnableTTS {
		ttsService = NewElevenLabsTTSService(config.ElevenLabs)
	}

	// Initialize STT service
	var sttService SpeechToTextService
	if config.EnableSTT {
		var err error
		if config.ElevenLabsService.BaseURL != "" {
			// Use ElevenLabs microservice
			sttService, err = NewElevenLabsServiceSTT(config.ElevenLabsService)
		} else {
			// Use direct ElevenLabs API
			sttService, err = NewElevenLabsSTTService(config.ElevenLabsSTT)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to initialize STT service: %w", err)
		}
	}

	// Initialize session manager
	sessionManager := NewMongoSpeechSessionManager(db)

	// Initialize voice authentication
	var voiceAuth VoiceAuthenticator
	if config.EnableVoiceAuth {
		embeddingModel := NewSimpleVoiceEmbeddingModel(256) // 256-dimensional embeddings
		voiceAuth = NewMongoVoiceAuthenticator(db, audioProcessor, embeddingModel)
	}

	service := &SpeechService{
		ttsService:     ttsService,
		sttService:     sttService,
		sessionManager: sessionManager,
		voiceAuth:      voiceAuth,
		audioProcessor: audioProcessor,
		config:         config,
	}

	// Start cleanup routine
	go service.startCleanupRoutine()

	return service, nil
}

// CreateConsultationSession creates a new consultation session with speech capabilities
func (s *SpeechService) CreateConsultationSession(ctx context.Context, userID string) (*SpeechSession, error) {
	if s.sessionManager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	session, err := s.sessionManager.CreateSession(ctx, userID, SessionTypeConsultation)
	if err != nil {
		return nil, fmt.Errorf("failed to create consultation session: %w", err)
	}

	return session, nil
}

// ProcessVoiceQuery processes a voice query in a consultation session
func (s *SpeechService) ProcessVoiceQuery(ctx context.Context, sessionID string, audioData []byte, options STTOptions) (*VoiceQueryResult, error) {
	if s.sttService == nil {
		return nil, fmt.Errorf("speech-to-text service not enabled")
	}

	// Get session
	session, err := s.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Type != SessionTypeConsultation {
		return nil, fmt.Errorf("session is not a consultation session")
	}

	// Set default language if not specified
	if options.Language == "" {
		options.Language = s.config.DefaultLanguage
	}

	// Transcribe audio
	transcription, err := s.sttService.TranscribeAudio(ctx, audioData, options)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Record interaction
	interaction := SpeechInteraction{
		Type:           InteractionTypeSTT,
		Input:          audioData,
		Text:           transcription.Text,
		Confidence:     transcription.Confidence,
		ProcessingTime: transcription.ProcessingTime,
		Options: map[string]interface{}{
			"language": options.Language,
			"model":    options.Model,
		},
	}

	if err := s.sessionManager.AddInteraction(ctx, sessionID, interaction); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to record STT interaction: %v\n", err)
	}

	result := &VoiceQueryResult{
		SessionID:      sessionID,
		TranscribedText: transcription.Text,
		Confidence:     transcription.Confidence,
		Language:       transcription.Language,
		ProcessingTime: transcription.ProcessingTime,
		Timestamps:     transcription.Timestamps,
	}

	return result, nil
}

// GenerateVoiceResponse generates a voice response for consultation results
func (s *SpeechService) GenerateVoiceResponse(ctx context.Context, sessionID string, text string, options TTSOptions) (*VoiceResponseResult, error) {
	if s.ttsService == nil {
		return nil, fmt.Errorf("text-to-speech service not enabled")
	}

	// Get session to validate it exists
	_, err := s.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Set default language if not specified
	if options.Language == "" {
		options.Language = s.config.DefaultLanguage
	}

	// Generate speech
	audioResult, err := s.ttsService.ConvertToSpeech(ctx, text, options)
	if err != nil {
		return nil, fmt.Errorf("speech generation failed: %w", err)
	}

	// Record interaction
	interaction := SpeechInteraction{
		Type:           InteractionTypeTTS,
		Output:         audioResult.AudioData,
		Text:           text,
		Confidence:     1.0, // TTS always has full confidence
		ProcessingTime: time.Since(audioResult.GeneratedAt).Seconds(),
		Options: map[string]interface{}{
			"voice":    options.Voice,
			"language": options.Language,
			"quality":  options.Quality,
		},
	}

	if err := s.sessionManager.AddInteraction(ctx, sessionID, interaction); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to record TTS interaction: %v\n", err)
	}

	result := &VoiceResponseResult{
		SessionID:    sessionID,
		AudioData:    audioResult.AudioData,
		Duration:     audioResult.Duration,
		Format:       audioResult.Format,
		Size:         audioResult.Size,
		Voice:        audioResult.Voice,
		Quality:      audioResult.Quality,
		GeneratedAt:  audioResult.GeneratedAt,
	}

	return result, nil
}

// AuthenticateVoice authenticates a user's voice
func (s *SpeechService) AuthenticateVoice(ctx context.Context, userID string, audioData []byte) (*AuthenticationResult, error) {
	if s.voiceAuth == nil {
		return nil, fmt.Errorf("voice authentication not enabled")
	}

	return s.voiceAuth.AuthenticateVoice(ctx, userID, audioData)
}

// EnrollUserVoice enrolls a user's voice for authentication
func (s *SpeechService) EnrollUserVoice(ctx context.Context, userID string, audioSamples [][]byte) (*VoiceProfile, error) {
	if s.voiceAuth == nil {
		return nil, fmt.Errorf("voice authentication not enabled")
	}

	return s.voiceAuth.EnrollVoice(ctx, userID, audioSamples)
}

// GetAvailableVoices returns available TTS voices
func (s *SpeechService) GetAvailableVoices(ctx context.Context) ([]Voice, error) {
	if s.ttsService == nil {
		return nil, fmt.Errorf("text-to-speech service not enabled")
	}

	return s.ttsService.GetAvailableVoices(ctx)
}

// GetSupportedLanguages returns supported STT languages
func (s *SpeechService) GetSupportedLanguages() ([]Language, error) {
	if s.sttService == nil {
		return nil, fmt.Errorf("speech-to-text service not enabled")
	}

	return s.sttService.GetSupportedLanguages()
}

// GetSessionHistory returns speech session history for a user
func (s *SpeechService) GetSessionHistory(ctx context.Context, userID string, limit int) ([]SpeechSession, error) {
	if s.sessionManager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	// This method would need to be added to the session manager interface
	if mongoManager, ok := s.sessionManager.(*MongoSpeechSessionManager); ok {
		return mongoManager.GetUserSessions(ctx, userID, limit)
	}

	return nil, fmt.Errorf("session history not supported by current session manager")
}

// ValidateAudioInput validates audio input for processing
func (s *SpeechService) ValidateAudioInput(audioData []byte, purpose AudioPurpose) (*AudioValidationResult, error) {
	validator := NewAudioValidator()
	
	// Basic format validation
	if err := validator.ValidateWAVFormat(audioData); err != nil {
		return &AudioValidationResult{
			IsValid: false,
			Issues:  []string{err.Error()},
		}, nil
	}

	// Get audio info
	info, err := validator.GetAudioInfo(audioData)
	if err != nil {
		return &AudioValidationResult{
			IsValid: false,
			Issues:  []string{"failed to extract audio information"},
		}, nil
	}

	result := &AudioValidationResult{
		IsValid:    true,
		Issues:     []string{},
		Format:     info.Format,
		SampleRate: info.SampleRate,
		Channels:   info.Channels,
		Duration:   info.Duration,
		Size:       info.Size,
	}

	// Purpose-specific validation
	switch purpose {
	case AudioPurposeSTT:
		if info.Duration < 0.5 {
			result.Issues = append(result.Issues, "audio too short for transcription")
		}
		if info.Duration > 300 { // 5 minutes
			result.Issues = append(result.Issues, "audio too long for transcription")
		}
		if info.SampleRate < 8000 {
			result.Issues = append(result.Issues, "sample rate too low for accurate transcription")
		}

	case AudioPurposeVoiceAuth:
		if info.Duration < 2.0 {
			result.Issues = append(result.Issues, "audio too short for voice authentication")
		}
		if info.Duration > 30.0 {
			result.Issues = append(result.Issues, "audio too long for voice authentication")
		}
		if info.SampleRate < 16000 {
			result.Issues = append(result.Issues, "sample rate too low for voice authentication")
		}
	}

	// Check for issues that make audio invalid
	for _, issue := range result.Issues {
		if containsCriticalKeyword(issue) {
			result.IsValid = false
			break
		}
	}

	return result, nil
}

// EndSession ends a speech session
func (s *SpeechService) EndSession(ctx context.Context, sessionID string) error {
	if s.sessionManager == nil {
		return fmt.Errorf("session manager not initialized")
	}

	return s.sessionManager.EndSession(ctx, sessionID)
}

// startCleanupRoutine starts the background cleanup routine
func (s *SpeechService) startCleanupRoutine() {
	if s.sessionManager == nil {
		return
	}

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := s.sessionManager.CleanupExpiredSessions(ctx); err != nil {
			fmt.Printf("Warning: session cleanup failed: %v\n", err)
		}
		cancel()
	}
}

// containsCriticalKeyword checks if an issue contains critical keywords
func containsCriticalKeyword(issue string) bool {
	criticalKeywords := []string{"too short", "too long", "missing", "invalid"}
	for _, keyword := range criticalKeywords {
		if len(issue) > len(keyword) && issue[:len(keyword)] == keyword {
			return true
		}
	}
	return false
}

// VoiceQueryResult represents the result of processing a voice query
type VoiceQueryResult struct {
	SessionID       string          `json:"session_id"`
	TranscribedText string          `json:"transcribed_text"`
	Confidence      float64         `json:"confidence"`
	Language        string          `json:"language"`
	ProcessingTime  float64         `json:"processing_time"`
	Timestamps      []WordTimestamp `json:"timestamps,omitempty"`
}

// VoiceResponseResult represents the result of generating a voice response
type VoiceResponseResult struct {
	SessionID   string    `json:"session_id"`
	AudioData   []byte    `json:"audio_data"`
	Duration    float64   `json:"duration"`
	Format      string    `json:"format"`
	Size        int64     `json:"size"`
	Voice       string    `json:"voice"`
	Quality     string    `json:"quality"`
	GeneratedAt time.Time `json:"generated_at"`
}

// AudioValidationResult represents the result of audio validation
type AudioValidationResult struct {
	IsValid    bool     `json:"is_valid"`
	Issues     []string `json:"issues,omitempty"`
	Format     string   `json:"format"`
	SampleRate int      `json:"sample_rate"`
	Channels   int      `json:"channels"`
	Duration   float64  `json:"duration"`
	Size       int64    `json:"size"`
}

// AudioPurpose represents the intended purpose of audio processing
type AudioPurpose string

const (
	AudioPurposeSTT       AudioPurpose = "stt"
	AudioPurposeTTS       AudioPurpose = "tts"
	AudioPurposeVoiceAuth AudioPurpose = "voice_auth"
	AudioPurposeGeneral   AudioPurpose = "general"
)