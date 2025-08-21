package speech

import (
	"context"
	"time"
)

// TextToSpeechService defines the interface for text-to-speech conversion
type TextToSpeechService interface {
	ConvertToSpeech(ctx context.Context, text string, options TTSOptions) (*AudioResult, error)
	GetAvailableVoices(ctx context.Context) ([]Voice, error)
	ValidateTextContent(text string) *ContentValidation
}

// SpeechToTextService defines the interface for speech-to-text transcription
type SpeechToTextService interface {
	TranscribeAudio(ctx context.Context, audioData []byte, options STTOptions) (*TranscriptionResult, error)
	GetSupportedLanguages() ([]Language, error)
	ValidateAudioFormat(audioData []byte) *FormatValidation
}

// AudioProcessor defines the interface for audio preprocessing
type AudioProcessor interface {
	PreprocessAudio(audioData []byte, targetFormat AudioFormat) ([]byte, error)
	ResampleAudio(audioData []byte, targetSampleRate int) ([]byte, error)
	NormalizeAudio(audioData []byte) ([]byte, error)
	ConvertFormat(audioData []byte, sourceFormat, targetFormat AudioFormat) ([]byte, error)
}

// SpeechSessionManager defines the interface for managing speech sessions
type SpeechSessionManager interface {
	CreateSession(ctx context.Context, userID string, sessionType SessionType) (*SpeechSession, error)
	GetSession(ctx context.Context, sessionID string) (*SpeechSession, error)
	UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error
	EndSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context) error
	AddInteraction(ctx context.Context, sessionID string, interaction SpeechInteraction) error
}

// VoiceAuthenticator defines the interface for voice authentication
type VoiceAuthenticator interface {
	EnrollVoice(ctx context.Context, userID string, audioSamples [][]byte) (*VoiceProfile, error)
	AuthenticateVoice(ctx context.Context, userID string, audioSample []byte) (*AuthenticationResult, error)
	UpdateVoiceProfile(ctx context.Context, userID string, audioSample []byte) error
	DeleteVoiceProfile(ctx context.Context, userID string) error
}

// TTSOptions contains options for text-to-speech conversion
type TTSOptions struct {
	Voice        string  `json:"voice"`
	Speed        float64 `json:"speed"`
	Language     string  `json:"language"`
	OutputFormat string  `json:"output_format"` // "mp3", "wav", "ogg"
	Quality      string  `json:"quality"`       // "low", "medium", "high"
	Stability    float64 `json:"stability"`     // ElevenLabs stability setting
	Clarity      float64 `json:"clarity"`       // ElevenLabs clarity setting
}

// STTOptions contains options for speech-to-text transcription
type STTOptions struct {
	Language            string  `json:"language"`
	Model              string  `json:"model"`                // Wav2Vec2 model variant (base, large, etc.)
	EnablePunctuation  bool    `json:"enable_punctuation"`
	FilterProfanity    bool    `json:"filter_profanity"`
	SampleRate         int     `json:"sample_rate"`          // Audio sample rate for Wav2Vec2
	ChunkSize          int     `json:"chunk_size"`           // Audio chunk size for processing
	ConfidenceThreshold float64 `json:"confidence_threshold"` // Minimum confidence for accepting results
}

// AudioResult represents the result of text-to-speech conversion
type AudioResult struct {
	AudioData   []byte    `json:"audio_data" bson:"audio_data"`
	Duration    float64   `json:"duration" bson:"duration"`
	Format      string    `json:"format" bson:"format"`
	Size        int64     `json:"size" bson:"size"`
	GeneratedAt time.Time `json:"generated_at" bson:"generated_at"`
	Voice       string    `json:"voice" bson:"voice"`
	Quality     string    `json:"quality" bson:"quality"`
}

// TranscriptionResult represents the result of speech-to-text transcription
type TranscriptionResult struct {
	Text           string          `json:"text" bson:"text"`
	Confidence     float64         `json:"confidence" bson:"confidence"`
	Language       string          `json:"language" bson:"language"`
	Duration       float64         `json:"duration" bson:"duration"`
	Timestamps     []WordTimestamp `json:"timestamps,omitempty" bson:"timestamps,omitempty"`
	ProcessedAt    time.Time       `json:"processed_at" bson:"processed_at"`
	ModelUsed      string          `json:"model_used" bson:"model_used"`
	ProcessingTime float64         `json:"processing_time" bson:"processing_time"`
	Segments       []TextSegment   `json:"segments,omitempty" bson:"segments,omitempty"`
}

// Voice represents an available voice for TTS
type Voice struct {
	ID          string   `json:"id" bson:"_id"`
	Name        string   `json:"name" bson:"name"`
	Language    string   `json:"language" bson:"language"`
	Gender      string   `json:"gender" bson:"gender"`
	Age         string   `json:"age" bson:"age"`
	Style       string   `json:"style" bson:"style"`
	SampleRate  int      `json:"sample_rate" bson:"sample_rate"`
	Formats     []string `json:"formats" bson:"formats"`
	IsDefault   bool     `json:"is_default" bson:"is_default"`
	Provider    string   `json:"provider" bson:"provider"` // "elevenlabs", "local", etc.
	Quality     string   `json:"quality" bson:"quality"`
}

// Language represents a supported language for STT
type Language struct {
	Code        string `json:"code" bson:"code"`
	Name        string `json:"name" bson:"name"`
	Region      string `json:"region" bson:"region"`
	IsSupported bool   `json:"is_supported" bson:"is_supported"`
	ModelPath   string `json:"model_path,omitempty" bson:"model_path,omitempty"`
}

// WordTimestamp represents timing information for transcribed words
type WordTimestamp struct {
	Word       string  `json:"word" bson:"word"`
	StartTime  float64 `json:"start_time" bson:"start_time"`
	EndTime    float64 `json:"end_time" bson:"end_time"`
	Confidence float64 `json:"confidence" bson:"confidence"`
}

// TextSegment represents a segment of transcribed text
type TextSegment struct {
	Text       string  `json:"text" bson:"text"`
	StartTime  float64 `json:"start_time" bson:"start_time"`
	EndTime    float64 `json:"end_time" bson:"end_time"`
	Confidence float64 `json:"confidence" bson:"confidence"`
}

// ContentValidation represents validation result for text content
type ContentValidation struct {
	IsValid      bool     `json:"is_valid"`
	Issues       []string `json:"issues,omitempty"`
	WordCount    int      `json:"word_count"`
	CharCount    int      `json:"char_count"`
	EstimatedDuration float64 `json:"estimated_duration"`
}

// FormatValidation represents validation result for audio format
type FormatValidation struct {
	IsValid      bool     `json:"is_valid"`
	Format       string   `json:"format"`
	SampleRate   int      `json:"sample_rate"`
	Channels     int      `json:"channels"`
	Duration     float64  `json:"duration"`
	Size         int64    `json:"size"`
	Issues       []string `json:"issues,omitempty"`
}

// AudioFormat represents audio format specifications
type AudioFormat struct {
	Codec      string `json:"codec"`      // "wav", "mp3", "ogg", "flac"
	SampleRate int    `json:"sample_rate"` // 16000, 22050, 44100, 48000
	Channels   int    `json:"channels"`    // 1 (mono), 2 (stereo)
	BitDepth   int    `json:"bit_depth"`   // 16, 24, 32
}

// SpeechSession represents an active speech interaction session
type SpeechSession struct {
	ID          string                 `json:"id" bson:"_id"`
	UserID      string                 `json:"user_id" bson:"user_id"`
	Type        SessionType            `json:"type" bson:"type"`
	Status      SessionStatus          `json:"status" bson:"status"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
	ExpiresAt   time.Time              `json:"expires_at" bson:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata" bson:"metadata"`
	Interactions []SpeechInteraction   `json:"interactions" bson:"interactions"`
}

// SpeechInteraction represents a single speech interaction within a session
type SpeechInteraction struct {
	ID            string              `json:"id" bson:"_id"`
	Type          InteractionType     `json:"type" bson:"type"` // "stt", "tts"
	Timestamp     time.Time           `json:"timestamp" bson:"timestamp"`
	Input         []byte              `json:"input,omitempty" bson:"input,omitempty"`
	Output        []byte              `json:"output,omitempty" bson:"output,omitempty"`
	Text          string              `json:"text,omitempty" bson:"text,omitempty"`
	Confidence    float64             `json:"confidence" bson:"confidence"`
	ProcessingTime float64            `json:"processing_time" bson:"processing_time"`
	Options       map[string]interface{} `json:"options" bson:"options"`
}

// VoiceProfile represents a user's voice authentication profile
type VoiceProfile struct {
	UserID      string    `json:"user_id" bson:"user_id"`
	Embeddings  [][]float64 `json:"embeddings" bson:"embeddings"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
	SampleCount int       `json:"sample_count" bson:"sample_count"`
	Quality     float64   `json:"quality" bson:"quality"`
	IsActive    bool      `json:"is_active" bson:"is_active"`
}

// AuthenticationResult represents the result of voice authentication
type AuthenticationResult struct {
	IsAuthenticated bool    `json:"is_authenticated"`
	Confidence      float64 `json:"confidence"`
	Threshold       float64 `json:"threshold"`
	ProcessingTime  float64 `json:"processing_time"`
	Reason          string  `json:"reason,omitempty"`
}

// Wav2Vec2Config contains configuration for Wav2Vec2 model
type Wav2Vec2Config struct {
	ModelPath       string  `json:"model_path"`       // Path to local Wav2Vec2 model
	ProcessorPath   string  `json:"processor_path"`   // Path to audio processor
	DeviceType      string  `json:"device_type"`      // "cpu" or "cuda"
	BatchSize       int     `json:"batch_size"`       // Batch size for processing
	MaxAudioLength  int     `json:"max_audio_length"` // Maximum audio length in seconds
	SampleRate      int     `json:"sample_rate"`      // Required sample rate (16000 Hz)
	ChunkDuration   float64 `json:"chunk_duration"`   // Duration of audio chunks in seconds
}

// ElevenLabsConfig contains configuration for ElevenLabs API
type ElevenLabsConfig struct {
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	DefaultVoice string `json:"default_voice"`
	MaxRetries  int     `json:"max_retries"`
	Timeout     int     `json:"timeout"` // seconds
	RateLimit   int     `json:"rate_limit"` // requests per minute
}

// SessionType represents the type of speech session
type SessionType string

const (
	SessionTypeConsultation SessionType = "consultation"
	SessionTypeTranscription SessionType = "transcription"
	SessionTypeVoiceAuth    SessionType = "voice_auth"
	SessionTypeGeneral      SessionType = "general"
)

// SessionStatus represents the status of a speech session
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusInactive SessionStatus = "inactive"
	SessionStatusExpired  SessionStatus = "expired"
	SessionStatusEnded    SessionStatus = "ended"
)

// InteractionType represents the type of speech interaction
type InteractionType string

const (
	InteractionTypeSTT InteractionType = "stt"
	InteractionTypeTTS InteractionType = "tts"
)