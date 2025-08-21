package speech

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTextToSpeechService is a mock implementation of TextToSpeechService
type MockTextToSpeechService struct {
	mock.Mock
}

func (m *MockTextToSpeechService) ConvertToSpeech(ctx context.Context, text string, options TTSOptions) (*AudioResult, error) {
	args := m.Called(ctx, text, options)
	return args.Get(0).(*AudioResult), args.Error(1)
}

func (m *MockTextToSpeechService) GetAvailableVoices(ctx context.Context) ([]Voice, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Voice), args.Error(1)
}

func (m *MockTextToSpeechService) ValidateTextContent(text string) *ContentValidation {
	args := m.Called(text)
	return args.Get(0).(*ContentValidation)
}

// MockSpeechToTextService is a mock implementation of SpeechToTextService
type MockSpeechToTextService struct {
	mock.Mock
}

func (m *MockSpeechToTextService) TranscribeAudio(ctx context.Context, audioData []byte, options STTOptions) (*TranscriptionResult, error) {
	args := m.Called(ctx, audioData, options)
	return args.Get(0).(*TranscriptionResult), args.Error(1)
}

func (m *MockSpeechToTextService) GetSupportedLanguages() ([]Language, error) {
	args := m.Called()
	return args.Get(0).([]Language), args.Error(1)
}

func (m *MockSpeechToTextService) ValidateAudioFormat(audioData []byte) *FormatValidation {
	args := m.Called(audioData)
	return args.Get(0).(*FormatValidation)
}

// TestSpeechService_CreateConsultationSession tests session creation
func TestSpeechService_CreateConsultationSession(t *testing.T) {
	// Setup
	sessionManager := NewInMemorySpeechSessionManager()
	
	service := &SpeechService{
		sessionManager: sessionManager,
		config: SpeechServiceConfig{
			DefaultLanguage: "en",
		},
	}

	ctx := context.Background()
	userID := "test-user-123"

	// Test
	session, err := service.CreateConsultationSession(ctx, userID)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, SessionTypeConsultation, session.Type)
	assert.Equal(t, SessionStatusActive, session.Status)
	assert.NotEmpty(t, session.ID)
}

// TestSpeechService_ProcessVoiceQuery tests voice query processing
func TestSpeechService_ProcessVoiceQuery(t *testing.T) {
	// Setup mocks
	mockSTT := new(MockSpeechToTextService)
	sessionManager := NewInMemorySpeechSessionManager()

	service := &SpeechService{
		sttService:     mockSTT,
		sessionManager: sessionManager,
		config: SpeechServiceConfig{
			DefaultLanguage: "en",
		},
	}

	// Create test session
	ctx := context.Background()
	userID := "test-user-123"
	session, err := service.CreateConsultationSession(ctx, userID)
	require.NoError(t, err)

	// Mock data
	audioData := []byte("fake-audio-data")
	options := STTOptions{Language: "en"}
	
	expectedTranscription := &TranscriptionResult{
		Text:           "Hello, this is a test transcription",
		Confidence:     0.95,
		Language:       "en",
		Duration:       3.5,
		ProcessingTime: 0.5,
		Timestamps: []WordTimestamp{
			{Word: "Hello", StartTime: 0.0, EndTime: 0.5, Confidence: 0.98},
			{Word: "this", StartTime: 0.6, EndTime: 0.8, Confidence: 0.95},
		},
	}

	// Set up mock expectations
	mockSTT.On("TranscribeAudio", ctx, audioData, options).Return(expectedTranscription, nil)

	// Test
	result, err := service.ProcessVoiceQuery(ctx, session.ID, audioData, options)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.SessionID)
	assert.Equal(t, expectedTranscription.Text, result.TranscribedText)
	assert.Equal(t, expectedTranscription.Confidence, result.Confidence)
	assert.Equal(t, expectedTranscription.Language, result.Language)
	assert.Equal(t, expectedTranscription.ProcessingTime, result.ProcessingTime)
	assert.Equal(t, expectedTranscription.Timestamps, result.Timestamps)

	// Verify mock was called
	mockSTT.AssertExpectations(t)
}

// TestSpeechService_GenerateVoiceResponse tests voice response generation
func TestSpeechService_GenerateVoiceResponse(t *testing.T) {
	// Setup mocks
	mockTTS := new(MockTextToSpeechService)
	sessionManager := NewInMemorySpeechSessionManager()

	service := &SpeechService{
		ttsService:     mockTTS,
		sessionManager: sessionManager,
		config: SpeechServiceConfig{
			DefaultLanguage: "en",
		},
	}

	// Create test session
	ctx := context.Background()
	userID := "test-user-123"
	session, err := service.CreateConsultationSession(ctx, userID)
	require.NoError(t, err)

	// Mock data
	text := "This is a test response"
	options := TTSOptions{Voice: "test-voice", Language: "en"}
	
	expectedAudio := &AudioResult{
		AudioData:   []byte("fake-audio-data"),
		Duration:    2.5,
		Format:      "mp3",
		Size:        1024,
		GeneratedAt: time.Now(),
		Voice:       "test-voice",
		Quality:     "high",
	}

	// Set up mock expectations
	mockTTS.On("ConvertToSpeech", ctx, text, options).Return(expectedAudio, nil)

	// Test
	result, err := service.GenerateVoiceResponse(ctx, session.ID, text, options)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.SessionID)
	assert.Equal(t, expectedAudio.AudioData, result.AudioData)
	assert.Equal(t, expectedAudio.Duration, result.Duration)
	assert.Equal(t, expectedAudio.Format, result.Format)
	assert.Equal(t, expectedAudio.Size, result.Size)
	assert.Equal(t, expectedAudio.Voice, result.Voice)
	assert.Equal(t, expectedAudio.Quality, result.Quality)

	// Verify mock was called
	mockTTS.AssertExpectations(t)
}

// TestSpeechService_ValidateAudioInput tests audio validation
func TestSpeechService_ValidateAudioInput(t *testing.T) {
	service := &SpeechService{}

	tests := []struct {
		name        string
		audioData   []byte
		purpose     AudioPurpose
		expectValid bool
		expectError bool
	}{
		{
			name:        "Valid WAV file",
			audioData:   createMockWAVData(16000, 1, 3.0), // 16kHz, mono, 3 seconds
			purpose:     AudioPurposeSTT,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "Too short for STT",
			audioData:   createMockWAVData(16000, 1, 0.3), // 0.3 seconds
			purpose:     AudioPurposeSTT,
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Invalid audio data",
			audioData:   []byte("not-audio-data"),
			purpose:     AudioPurposeSTT,
			expectValid: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateAudioInput(tt.audioData, tt.purpose)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectValid, result.IsValid)
			}
		})
	}
}

// TestSpeechService_GetAvailableVoices tests getting available voices
func TestSpeechService_GetAvailableVoices(t *testing.T) {
	// Setup mock
	mockTTS := new(MockTextToSpeechService)
	service := &SpeechService{
		ttsService: mockTTS,
	}

	ctx := context.Background()
	expectedVoices := []Voice{
		{ID: "voice1", Name: "Test Voice 1", Language: "en"},
		{ID: "voice2", Name: "Test Voice 2", Language: "es"},
	}

	// Set up mock expectations
	mockTTS.On("GetAvailableVoices", ctx).Return(expectedVoices, nil)

	// Test
	voices, err := service.GetAvailableVoices(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, expectedVoices, voices)
	mockTTS.AssertExpectations(t)
}

// TestSpeechService_GetSupportedLanguages tests getting supported languages
func TestSpeechService_GetSupportedLanguages(t *testing.T) {
	// Setup mock
	mockSTT := new(MockSpeechToTextService)
	service := &SpeechService{
		sttService: mockSTT,
	}

	expectedLanguages := []Language{
		{Code: "en", Name: "English", IsSupported: true},
		{Code: "es", Name: "Spanish", IsSupported: true},
	}

	// Set up mock expectations
	mockSTT.On("GetSupportedLanguages").Return(expectedLanguages, nil)

	// Test
	languages, err := service.GetSupportedLanguages()

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, expectedLanguages, languages)
	mockSTT.AssertExpectations(t)
}

// TestSpeechService_EndSession tests ending a session
func TestSpeechService_EndSession(t *testing.T) {
	// Setup
	sessionManager := NewInMemorySpeechSessionManager()
	service := &SpeechService{
		sessionManager: sessionManager,
	}

	// Create test session
	ctx := context.Background()
	userID := "test-user-123"
	session, err := service.CreateConsultationSession(ctx, userID)
	require.NoError(t, err)

	// Test ending session
	err = service.EndSession(ctx, session.ID)
	require.NoError(t, err)

	// Verify session is ended
	retrievedSession, err := sessionManager.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, SessionStatusEnded, retrievedSession.Status)
}

// Helper function to create mock WAV data
func createMockWAVData(sampleRate, channels int, duration float64) []byte {
	// Create a minimal WAV header
	header := make([]byte, 44)
	
	// RIFF header
	copy(header[0:4], "RIFF")
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	
	// Format chunk size (16 for PCM)
	header[16] = 16
	
	// Audio format (1 for PCM)
	header[20] = 1
	
	// Number of channels
	header[22] = byte(channels)
	
	// Sample rate
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	
	// Bits per sample
	header[34] = 16
	
	// Data chunk
	copy(header[36:40], "data")
	
	// Calculate data size
	bytesPerSample := 2 // 16-bit
	dataSize := int(duration * float64(sampleRate) * float64(channels) * float64(bytesPerSample))
	
	// Set data size in header
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)
	
	// Set file size in header (total size - 8)
	totalSize := 44 + dataSize - 8
	header[4] = byte(totalSize)
	header[5] = byte(totalSize >> 8)
	header[6] = byte(totalSize >> 16)
	header[7] = byte(totalSize >> 24)
	
	// Create fake audio data
	audioData := make([]byte, dataSize)
	for i := range audioData {
		audioData[i] = byte(i % 256) // Simple pattern
	}
	
	// Combine header and data
	result := make([]byte, len(header)+len(audioData))
	copy(result, header)
	copy(result[len(header):], audioData)
	
	return result
}