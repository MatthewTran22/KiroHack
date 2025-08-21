package speech

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestSpeechService_Integration tests the complete speech service integration
func TestSpeechService_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Connect to MongoDB (assuming Docker container is running)
	ctx := context.Background()
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Create test database
	db := client.Database("speech_test_" + time.Now().Format("20060102150405"))
	defer db.Drop(ctx)

	// Create speech service configuration
	config := SpeechServiceConfig{
		EnableTTS:       true,
		EnableSTT:       true,
		EnableVoiceAuth: true,
		DefaultLanguage: "en",
		MaxSessionTime:  time.Hour,
		CleanupInterval: time.Minute,
		ElevenLabs: ElevenLabsConfig{
			APIKey:      os.Getenv("ELEVENLABS_API_KEY"),
			BaseURL:     "https://api.elevenlabs.io",
			DefaultVoice: "21m00Tcm4TlvDq8ikWAM", // Default ElevenLabs voice
			MaxRetries:  3,
			Timeout:     30,
			RateLimit:   60,
		},
		Wav2Vec2: Wav2Vec2Config{
			ModelPath:       os.Getenv("WAV2VEC2_MODEL_PATH"),
			ProcessorPath:   os.Getenv("WAV2VEC2_PROCESSOR_PATH"),
			DeviceType:      "cpu",
			BatchSize:       1,
			MaxAudioLength:  300,
			SampleRate:      16000,
			ChunkDuration:   5.0,
		},
	}

	// Skip TTS tests if API key is not provided
	if config.ElevenLabs.APIKey == "" {
		config.EnableTTS = false
		t.Log("Skipping TTS tests: ELEVENLABS_API_KEY not set")
	}

	// Skip STT tests if model paths are not provided
	if config.Wav2Vec2.ModelPath == "" || config.Wav2Vec2.ProcessorPath == "" {
		config.EnableSTT = false
		t.Log("Skipping STT tests: WAV2VEC2_MODEL_PATH or WAV2VEC2_PROCESSOR_PATH not set")
	}

	// Create speech service
	speechService, err := NewSpeechService(db, config)
	require.NoError(t, err)

	// Test session creation
	t.Run("CreateSession", func(t *testing.T) {
		userID := "test-user-integration"
		session, err := speechService.CreateConsultationSession(ctx, userID)
		
		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, SessionTypeConsultation, session.Type)
		assert.Equal(t, SessionStatusActive, session.Status)
	})

	// Test TTS if enabled
	if config.EnableTTS {
		t.Run("TextToSpeech", func(t *testing.T) {
			userID := "test-user-tts"
			session, err := speechService.CreateConsultationSession(ctx, userID)
			require.NoError(t, err)

			text := "Hello, this is a test of the text-to-speech functionality."
			options := TTSOptions{
				Language:     "en",
				OutputFormat: "mp3",
				Quality:      "medium",
			}

			result, err := speechService.GenerateVoiceResponse(ctx, session.ID, text, options)
			
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.AudioData)
			assert.Equal(t, "mp3", result.Format)
			assert.True(t, result.Size > 0)
			assert.True(t, result.Duration > 0)
		})
	}

	// Test STT if enabled
	if config.EnableSTT {
		t.Run("SpeechToText", func(t *testing.T) {
			userID := "test-user-stt"
			session, err := speechService.CreateConsultationSession(ctx, userID)
			require.NoError(t, err)

			// Create test audio data (mock WAV file)
			audioData := createTestAudioFile(t)
			options := STTOptions{
				Language:            "en",
				Model:              "base",
				ConfidenceThreshold: 0.5,
			}

			result, err := speechService.ProcessVoiceQuery(ctx, session.ID, audioData, options)
			
			if err != nil {
				// STT might fail due to model availability, log but don't fail test
				t.Logf("STT test failed (expected in CI): %v", err)
				return
			}

			assert.NotNil(t, result)
			assert.NotEmpty(t, result.TranscribedText)
			assert.True(t, result.Confidence >= 0.0)
			assert.Equal(t, "en", result.Language)
		})
	}

	// Test voice authentication if enabled
	if config.EnableVoiceAuth {
		t.Run("VoiceAuthentication", func(t *testing.T) {
			userID := "test-user-voice-auth"

			// Create multiple audio samples for enrollment
			audioSamples := [][]byte{
				createTestAudioFile(t),
				createTestAudioFile(t),
				createTestAudioFile(t),
			}

			// Enroll voice
			profile, err := speechService.EnrollUserVoice(ctx, userID, audioSamples)
			require.NoError(t, err)
			assert.NotNil(t, profile)
			assert.Equal(t, userID, profile.UserID)
			assert.True(t, profile.IsActive)
			assert.Equal(t, 3, profile.SampleCount)

			// Test authentication
			testAudio := createTestAudioFile(t)
			authResult, err := speechService.AuthenticateVoice(ctx, userID, testAudio)
			require.NoError(t, err)
			assert.NotNil(t, authResult)
			// Note: Authentication might fail with mock audio, but should not error
		})
	}

	// Test session management
	t.Run("SessionManagement", func(t *testing.T) {
		userID := "test-user-session-mgmt"
		
		// Create session
		session, err := speechService.CreateConsultationSession(ctx, userID)
		require.NoError(t, err)

		// Get session history
		history, err := speechService.GetSessionHistory(ctx, userID, 10)
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, session.ID, history[0].ID)

		// End session
		err = speechService.EndSession(ctx, session.ID)
		require.NoError(t, err)
	})

	// Test audio validation
	t.Run("AudioValidation", func(t *testing.T) {
		// Test valid audio
		validAudio := createTestAudioFile(t)
		result, err := speechService.ValidateAudioInput(validAudio, AudioPurposeSTT)
		require.NoError(t, err)
		assert.True(t, result.IsValid)

		// Test invalid audio
		invalidAudio := []byte("not-audio-data")
		result, err = speechService.ValidateAudioInput(invalidAudio, AudioPurposeSTT)
		require.NoError(t, err)
		assert.False(t, result.IsValid)
		assert.NotEmpty(t, result.Issues)
	})

	// Test service capabilities
	t.Run("ServiceCapabilities", func(t *testing.T) {
		if config.EnableTTS {
			voices, err := speechService.GetAvailableVoices(ctx)
			require.NoError(t, err)
			assert.NotEmpty(t, voices)
		}

		if config.EnableSTT {
			languages, err := speechService.GetSupportedLanguages()
			require.NoError(t, err)
			assert.NotEmpty(t, languages)
		}
	})
}

// TestSpeechService_Performance tests performance characteristics
func TestSpeechService_Performance(t *testing.T) {
	// Skip if not running performance tests
	if os.Getenv("PERFORMANCE_TEST") != "true" {
		t.Skip("Skipping performance test. Set PERFORMANCE_TEST=true to run.")
	}

	// Connect to MongoDB
	ctx := context.Background()
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("speech_perf_test_" + time.Now().Format("20060102150405"))
	defer db.Drop(ctx)

	// Create minimal config for performance testing
	config := SpeechServiceConfig{
		EnableTTS:       false, // Disable to avoid API costs
		EnableSTT:       false, // Disable to avoid model requirements
		EnableVoiceAuth: true,
		DefaultLanguage: "en",
		MaxSessionTime:  time.Hour,
		CleanupInterval: time.Minute,
	}

	speechService, err := NewSpeechService(db, config)
	require.NoError(t, err)

	// Test concurrent session creation
	t.Run("ConcurrentSessions", func(t *testing.T) {
		numSessions := 100
		sessionChan := make(chan *SpeechSession, numSessions)
		errorChan := make(chan error, numSessions)

		start := time.Now()

		// Create sessions concurrently
		for i := 0; i < numSessions; i++ {
			go func(id int) {
				userID := "perf-user-" + string(rune(id))
				session, err := speechService.CreateConsultationSession(ctx, userID)
				if err != nil {
					errorChan <- err
				} else {
					sessionChan <- session
				}
			}(i)
		}

		// Collect results
		var sessions []*SpeechSession
		var errors []error

		for i := 0; i < numSessions; i++ {
			select {
			case session := <-sessionChan:
				sessions = append(sessions, session)
			case err := <-errorChan:
				errors = append(errors, err)
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for session creation")
			}
		}

		duration := time.Since(start)

		// Assertions
		assert.Empty(t, errors, "Should have no errors creating sessions")
		assert.Len(t, sessions, numSessions)
		assert.Less(t, duration, 10*time.Second, "Should create 100 sessions in less than 10 seconds")

		t.Logf("Created %d sessions in %v (%.2f sessions/sec)", 
			numSessions, duration, float64(numSessions)/duration.Seconds())
	})

	// Test session cleanup performance
	t.Run("SessionCleanup", func(t *testing.T) {
		// Create many expired sessions
		numSessions := 1000
		for i := 0; i < numSessions; i++ {
			userID := "cleanup-user-" + string(rune(i))
			session, err := speechService.CreateConsultationSession(ctx, userID)
			require.NoError(t, err)

			// Manually expire the session
			sessionManager := speechService.sessionManager.(*MongoSpeechSessionManager)
			err = sessionManager.UpdateSession(ctx, session.ID, map[string]interface{}{
				"expires_at": time.Now().Add(-time.Hour), // Expired 1 hour ago
			})
			require.NoError(t, err)
		}

		// Measure cleanup performance
		start := time.Now()
		err := speechService.sessionManager.CleanupExpiredSessions(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 5*time.Second, "Should cleanup 1000 sessions in less than 5 seconds")

		t.Logf("Cleaned up %d expired sessions in %v", numSessions, duration)
	})
}

// TestSpeechService_ErrorRecovery tests error recovery scenarios
func TestSpeechService_ErrorRecovery(t *testing.T) {
	// Skip if not running error recovery tests
	if os.Getenv("ERROR_RECOVERY_TEST") != "true" {
		t.Skip("Skipping error recovery test. Set ERROR_RECOVERY_TEST=true to run.")
	}

	// Test with invalid MongoDB connection
	t.Run("InvalidDatabase", func(t *testing.T) {
		ctx := context.Background()
		
		// Try to connect to non-existent MongoDB
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://invalid-host:27017"))
		if err == nil {
			defer client.Disconnect(ctx)
		}

		// This should handle the connection error gracefully
		config := SpeechServiceConfig{
			EnableTTS:       false,
			EnableSTT:       false,
			EnableVoiceAuth: false,
			DefaultLanguage: "en",
		}

		// Service creation might succeed even with invalid DB (lazy connection)
		if client != nil {
			db := client.Database("invalid_test")
			speechService, err := NewSpeechService(db, config)
			
			if err == nil {
				// Operations should fail gracefully
				_, err = speechService.CreateConsultationSession(ctx, "test-user")
				assert.Error(t, err, "Should fail with invalid database connection")
			}
		}
	})

	// Test with invalid service configurations
	t.Run("InvalidConfigurations", func(t *testing.T) {
		ctx := context.Background()
		mongoURI := os.Getenv("MONGODB_URI")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
		}

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		require.NoError(t, err)
		defer client.Disconnect(ctx)

		db := client.Database("error_recovery_test")
		defer db.Drop(ctx)

		// Test with invalid TTS configuration
		invalidTTSConfig := SpeechServiceConfig{
			EnableTTS: true,
			ElevenLabs: ElevenLabsConfig{
				APIKey:  "invalid-key",
				BaseURL: "https://invalid-url",
			},
		}

		speechService, err := NewSpeechService(db, invalidTTSConfig)
		require.NoError(t, err) // Service creation should succeed

		// TTS operations should fail gracefully
		session, err := speechService.CreateConsultationSession(ctx, "test-user")
		require.NoError(t, err)

		_, err = speechService.GenerateVoiceResponse(ctx, session.ID, "test", TTSOptions{})
		assert.Error(t, err, "Should fail with invalid TTS configuration")
	})
}

// createTestAudioFile creates a test WAV file for testing
func createTestAudioFile(t *testing.T) []byte {
	// Create a longer, more realistic test audio file
	sampleRate := 16000
	duration := 3.0 // 3 seconds
	channels := 1   // Mono
	
	return createMockWAVData(sampleRate, channels, duration)
}

// TestSpeechService_LoadTesting performs basic load testing
func TestSpeechService_LoadTesting(t *testing.T) {
	// Skip if not running load tests
	if os.Getenv("LOAD_TEST") != "true" {
		t.Skip("Skipping load test. Set LOAD_TEST=true to run.")
	}

	ctx := context.Background()
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("speech_load_test_" + time.Now().Format("20060102150405"))
	defer db.Drop(ctx)

	config := SpeechServiceConfig{
		EnableTTS:       false,
		EnableSTT:       false,
		EnableVoiceAuth: true,
		DefaultLanguage: "en",
		MaxSessionTime:  time.Hour,
		CleanupInterval: time.Minute,
	}

	speechService, err := NewSpeechService(db, config)
	require.NoError(t, err)

	// Test sustained load
	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 30 * time.Second
		concurrency := 10
		
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		results := make(chan time.Duration, concurrency*100)
		errors := make(chan error, concurrency*100)

		// Start concurrent workers
		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for {
					select {
					case <-ctx.Done():
						return
					default:
						start := time.Now()
						userID := "load-user-" + string(rune(workerID))
						
						session, err := speechService.CreateConsultationSession(ctx, userID)
						if err != nil {
							errors <- err
							continue
						}

						err = speechService.EndSession(ctx, session.ID)
						if err != nil {
							errors <- err
							continue
						}

						results <- time.Since(start)
					}
				}
			}(i)
		}

		// Collect results
		var latencies []time.Duration
		var errorCount int

		timeout := time.After(duration + 5*time.Second)
		for {
			select {
			case latency := <-results:
				latencies = append(latencies, latency)
			case <-errors:
				errorCount++
			case <-timeout:
				goto done
			}
		}

	done:
		// Calculate statistics
		if len(latencies) > 0 {
			var total time.Duration
			for _, lat := range latencies {
				total += lat
			}
			avgLatency := total / time.Duration(len(latencies))
			
			t.Logf("Load test results:")
			t.Logf("  Operations: %d", len(latencies))
			t.Logf("  Errors: %d", errorCount)
			t.Logf("  Average latency: %v", avgLatency)
			t.Logf("  Operations/sec: %.2f", float64(len(latencies))/duration.Seconds())

			// Assertions
			assert.Less(t, errorCount, len(latencies)/10, "Error rate should be less than 10%")
			assert.Less(t, avgLatency, 100*time.Millisecond, "Average latency should be less than 100ms")
		}
	})
}