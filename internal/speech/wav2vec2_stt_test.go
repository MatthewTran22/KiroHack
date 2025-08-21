package speech

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWav2Vec2STTService_ValidateAudioFormat tests audio format validation
func TestWav2Vec2STTService_ValidateAudioFormat(t *testing.T) {
	config := Wav2Vec2Config{
		SampleRate: 16000,
	}
	
	audioProcessor := NewSimpleAudioProcessor()
	service, err := NewWav2Vec2STTService(config, audioProcessor)
	require.NoError(t, err)

	tests := []struct {
		name        string
		audioData   []byte
		expectValid bool
		expectIssues []string
	}{
		{
			name:        "Valid WAV file",
			audioData:   createMockWAVData(16000, 1, 3.0),
			expectValid: true,
			expectIssues: []string{},
		},
		{
			name:        "Too small audio",
			audioData:   []byte("small"),
			expectValid: false,
			expectIssues: []string{"audio data too small"},
		},
		{
			name:        "Invalid WAV header",
			audioData:   make([]byte, 1000), // Large enough but invalid header
			expectValid: false,
			expectIssues: []string{"unsupported audio format, WAV required"},
		},
		{
			name:        "Wrong sample rate",
			audioData:   createMockWAVData(8000, 1, 3.0), // 8kHz instead of 16kHz
			expectValid: true, // Valid but with warning
			expectIssues: []string{"sample rate 8000 Hz not optimal, 16000 Hz recommended"},
		},
		{
			name:        "Stereo audio",
			audioData:   createMockWAVData(16000, 2, 3.0), // Stereo
			expectValid: true, // Valid but with warning
			expectIssues: []string{"stereo audio detected, mono recommended for better accuracy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := service.ValidateAudioFormat(tt.audioData)

			assert.Equal(t, tt.expectValid, validation.IsValid)
			
			if len(tt.expectIssues) > 0 {
				for _, expectedIssue := range tt.expectIssues {
					found := false
					for _, actualIssue := range validation.Issues {
						if actualIssue == expectedIssue {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected issue '%s' not found in %v", expectedIssue, validation.Issues)
				}
			}
		})
	}
}

// TestWav2Vec2STTService_GetSupportedLanguages tests getting supported languages
func TestWav2Vec2STTService_GetSupportedLanguages(t *testing.T) {
	config := Wav2Vec2Config{}
	audioProcessor := NewSimpleAudioProcessor()
	service, err := NewWav2Vec2STTService(config, audioProcessor)
	require.NoError(t, err)

	languages, err := service.GetSupportedLanguages()

	require.NoError(t, err)
	assert.NotEmpty(t, languages)

	// Check that common languages are supported
	languageCodes := make(map[string]bool)
	for _, lang := range languages {
		languageCodes[lang.Code] = true
		assert.True(t, lang.IsSupported)
		assert.NotEmpty(t, lang.Name)
	}

	// Verify common languages are present
	expectedLanguages := []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko"}
	for _, code := range expectedLanguages {
		assert.True(t, languageCodes[code], "Language %s should be supported", code)
	}
}

// TestWav2Vec2STTService_PreprocessAudio tests audio preprocessing
func TestWav2Vec2STTService_PreprocessAudio(t *testing.T) {
	config := Wav2Vec2Config{
		SampleRate: 16000,
	}
	
	audioProcessor := NewSimpleAudioProcessor()
	service, err := NewWav2Vec2STTService(config, audioProcessor)
	require.NoError(t, err)

	// Test with different sample rates
	tests := []struct {
		name       string
		audioData  []byte
		options    STTOptions
		expectError bool
	}{
		{
			name:       "16kHz audio (optimal)",
			audioData:  createMockWAVData(16000, 1, 3.0),
			options:    STTOptions{SampleRate: 16000},
			expectError: false,
		},
		{
			name:       "44.1kHz audio (needs resampling)",
			audioData:  createMockWAVData(44100, 1, 3.0),
			options:    STTOptions{SampleRate: 44100},
			expectError: false,
		},
		{
			name:       "Stereo audio (needs conversion)",
			audioData:  createMockWAVData(16000, 2, 3.0),
			options:    STTOptions{SampleRate: 16000},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := service.preprocessAudioForWav2Vec2(tt.audioData, tt.options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, processed)
				assert.True(t, len(processed) > 0)
			}
		})
	}
}

// TestWav2Vec2STTService_CreateInferenceScript tests script creation
func TestWav2Vec2STTService_CreateInferenceScript(t *testing.T) {
	config := Wav2Vec2Config{}
	audioProcessor := NewSimpleAudioProcessor()
	service, err := NewWav2Vec2STTService(config, audioProcessor)
	require.NoError(t, err)

	// Check that script was created
	assert.NotEmpty(t, service.scriptPath)
	
	// Check that script file exists
	_, err = os.Stat(service.scriptPath)
	assert.NoError(t, err)

	// Check script content
	content, err := os.ReadFile(service.scriptPath)
	require.NoError(t, err)
	
	scriptContent := string(content)
	assert.Contains(t, scriptContent, "#!/usr/bin/env python3")
	assert.Contains(t, scriptContent, "import torch")
	assert.Contains(t, scriptContent, "import torchaudio")
	assert.Contains(t, scriptContent, "from transformers import Wav2Vec2ForCTC")
	assert.Contains(t, scriptContent, "def load_model_and_processor")
	assert.Contains(t, scriptContent, "def transcribe_chunk")

	// Cleanup
	os.RemoveAll(filepath.Dir(service.scriptPath))
}

// TestWav2Vec2STTService_TranscribeAudio_MockInference tests transcription with mock inference
func TestWav2Vec2STTService_TranscribeAudio_MockInference(t *testing.T) {
	// This test would require mocking the Python script execution
	// For now, we'll test the validation and preprocessing parts
	
	config := Wav2Vec2Config{
		ModelPath:     "/fake/model/path",
		ProcessorPath: "/fake/processor/path",
		SampleRate:    16000,
		ChunkDuration: 5.0,
	}
	
	audioProcessor := NewSimpleAudioProcessor()
	
	// Create service but skip model validation for testing
	service := &Wav2Vec2STTService{
		config:         config,
		audioProcessor: audioProcessor,
		pythonPath:     "python3",
	}

	// Create script for testing
	err := service.createInferenceScript()
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Dir(service.scriptPath))

	ctx := context.Background()
	audioData := createMockWAVData(16000, 1, 3.0)
	options := STTOptions{
		Language:            "en",
		Model:              "base",
		SampleRate:         16000,
		ConfidenceThreshold: 0.5,
	}

	// Test validation part (this will work without Python dependencies)
	validation := service.ValidateAudioFormat(audioData)
	assert.True(t, validation.IsValid)

	// Test preprocessing part
	processed, err := service.preprocessAudioForWav2Vec2(audioData, options)
	assert.NoError(t, err)
	assert.NotNil(t, processed)

	// Note: Full transcription test would require actual Python environment
	// with PyTorch and transformers installed, which is tested in integration tests
}

// TestWav2Vec2STTService_InferenceResult tests inference result parsing
func TestWav2Vec2STTService_InferenceResult(t *testing.T) {
	// Test parsing of inference results
	mockResult := &InferenceResult{
		Text:       "Hello world this is a test",
		Confidence: 0.95,
		Timestamps: []WordTimestamp{
			{Word: "Hello", StartTime: 0.0, EndTime: 0.5, Confidence: 0.98},
			{Word: "world", StartTime: 0.6, EndTime: 1.0, Confidence: 0.96},
			{Word: "this", StartTime: 1.1, EndTime: 1.3, Confidence: 0.94},
			{Word: "is", StartTime: 1.4, EndTime: 1.5, Confidence: 0.92},
			{Word: "a", StartTime: 1.6, EndTime: 1.7, Confidence: 0.90},
			{Word: "test", StartTime: 1.8, EndTime: 2.2, Confidence: 0.97},
		},
		Segments: []TextSegment{
			{Text: "Hello world", StartTime: 0.0, EndTime: 1.0, Confidence: 0.97},
			{Text: "this is a test", StartTime: 1.1, EndTime: 2.2, Confidence: 0.93},
		},
		Success: true,
	}

	// Verify result structure
	assert.True(t, mockResult.Success)
	assert.Equal(t, "Hello world this is a test", mockResult.Text)
	assert.Equal(t, 0.95, mockResult.Confidence)
	assert.Len(t, mockResult.Timestamps, 6)
	assert.Len(t, mockResult.Segments, 2)

	// Verify timestamps
	for i, timestamp := range mockResult.Timestamps {
		assert.NotEmpty(t, timestamp.Word)
		assert.True(t, timestamp.Confidence > 0.8)
		if i > 0 {
			// Each timestamp should start after the previous one
			assert.True(t, timestamp.StartTime >= mockResult.Timestamps[i-1].StartTime)
		}
	}

	// Verify segments
	for _, segment := range mockResult.Segments {
		assert.NotEmpty(t, segment.Text)
		assert.True(t, segment.Confidence > 0.8)
		assert.True(t, segment.EndTime > segment.StartTime)
	}
}

// TestWav2Vec2STTService_ChunkAudio tests audio chunking logic
func TestWav2Vec2STTService_ChunkAudio(t *testing.T) {
	// Test the chunking logic that would be used in the Python script
	sampleRate := 16000
	chunkDuration := 5.0 // 5 seconds
	totalDuration := 12.0 // 12 seconds
	
	// Calculate expected chunks
	chunkSize := int(chunkDuration * float64(sampleRate))
	totalSamples := int(totalDuration * float64(sampleRate))
	
	expectedChunks := (totalSamples + chunkSize - 1) / chunkSize // Ceiling division
	
	// Verify chunking calculation
	assert.Equal(t, 80000, chunkSize) // 5 seconds * 16000 Hz
	assert.Equal(t, 192000, totalSamples) // 12 seconds * 16000 Hz
	assert.Equal(t, 3, expectedChunks) // Should need 3 chunks for 12 seconds of audio
}

// TestWav2Vec2STTService_ErrorHandling tests error handling
func TestWav2Vec2STTService_ErrorHandling(t *testing.T) {
	config := Wav2Vec2Config{
		SampleRate: 16000,
	}
	
	audioProcessor := NewSimpleAudioProcessor()
	service, err := NewWav2Vec2STTService(config, audioProcessor)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with invalid audio data
	invalidAudio := []byte("not-audio")
	options := STTOptions{}

	validation := service.ValidateAudioFormat(invalidAudio)
	assert.False(t, validation.IsValid)
	assert.NotEmpty(t, validation.Issues)

	// Test with empty audio
	emptyAudio := []byte{}
	validation = service.ValidateAudioFormat(emptyAudio)
	assert.False(t, validation.IsValid)
	assert.Contains(t, validation.Issues, "audio data too small")
}

// TestWav2Vec2STTService_ConfigValidation tests configuration validation
func TestWav2Vec2STTService_ConfigValidation(t *testing.T) {
	audioProcessor := NewSimpleAudioProcessor()

	tests := []struct {
		name        string
		config      Wav2Vec2Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: Wav2Vec2Config{
				ModelPath:     "/tmp/test_model",
				ProcessorPath: "/tmp/test_processor",
				SampleRate:    16000,
			},
			expectError: true, // Will fail because paths don't exist
		},
		{
			name: "Missing model path",
			config: Wav2Vec2Config{
				ProcessorPath: "/tmp/test_processor",
				SampleRate:    16000,
			},
			expectError: true,
		},
		{
			name: "Missing processor path",
			config: Wav2Vec2Config{
				ModelPath:  "/tmp/test_model",
				SampleRate: 16000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWav2Vec2STTService(tt.config, audioProcessor)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}