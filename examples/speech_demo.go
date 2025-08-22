package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"ai-government-consultant/internal/speech"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	fmt.Println("AI Government Consultant - Speech Services Demo")
	fmt.Println("===============================================")

	// Connect to MongoDB
	ctx := context.Background()
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("speech_demo")

	// Configure speech services
	config := speech.SpeechServiceConfig{
		EnableTTS:       true,
		EnableSTT:       true,
		EnableVoiceAuth: true,
		DefaultLanguage: "en",
		MaxSessionTime:  time.Hour,
		CleanupInterval: time.Minute,
		ElevenLabs: speech.ElevenLabsConfig{
			APIKey:      os.Getenv("ELEVENLABS_API_KEY"),
			BaseURL:     "https://api.elevenlabs.io",
			DefaultVoice: "21m00Tcm4TlvDq8ikWAM",
			MaxRetries:  3,
			Timeout:     30,
			RateLimit:   60,
		},
		Wav2Vec2: speech.Wav2Vec2Config{
			ModelPath:       os.Getenv("WAV2VEC2_MODEL_PATH"),
			ProcessorPath:   os.Getenv("WAV2VEC2_PROCESSOR_PATH"),
			DeviceType:      "cpu",
			BatchSize:       1,
			MaxAudioLength:  300,
			SampleRate:      16000,
			ChunkDuration:   5.0,
		},
	}

	// Disable services if credentials/models not available
	if config.ElevenLabs.APIKey == "" {
		config.EnableTTS = false
		fmt.Println("⚠️  TTS disabled: ELEVENLABS_API_KEY not set")
	}

	if config.Wav2Vec2.ModelPath == "" || config.Wav2Vec2.ProcessorPath == "" {
		config.EnableSTT = false
		fmt.Println("⚠️  STT disabled: WAV2VEC2_MODEL_PATH or WAV2VEC2_PROCESSOR_PATH not set")
	}

	// Create speech service
	speechService, err := speech.NewSpeechService(db, config)
	if err != nil {
		log.Fatalf("Failed to create speech service: %v", err)
	}

	fmt.Printf("✅ Speech service initialized (TTS: %v, STT: %v, VoiceAuth: %v)\n", 
		config.EnableTTS, config.EnableSTT, config.EnableVoiceAuth)

	// Demo 1: Session Management
	fmt.Println("\n🎯 Demo 1: Session Management")
	userID := "demo-user-123"
	session, err := speechService.CreateConsultationSession(ctx, userID)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
	} else {
		fmt.Printf("✅ Created session: %s\n", session.ID)
		fmt.Printf("   User: %s, Type: %s, Status: %s\n", 
			session.UserID, session.Type, session.Status)
	}

	// Demo 2: Text-to-Speech
	if config.EnableTTS {
		fmt.Println("\n🎯 Demo 2: Text-to-Speech")
		text := "Welcome to the AI Government Consultant platform. This is a demonstration of our text-to-speech capabilities."
		
		options := speech.TTSOptions{
			Language:     "en",
			OutputFormat: "mp3",
			Quality:      "medium",
		}

		fmt.Printf("🔊 Converting text to speech: \"%s\"\n", text)
		result, err := speechService.GenerateVoiceResponse(ctx, session.ID, text, options)
		if err != nil {
			log.Printf("TTS failed: %v", err)
		} else {
			fmt.Printf("✅ Generated %d bytes of audio (%.1fs duration, %s format)\n", 
				result.Size, result.Duration, result.Format)
			
			// Save audio file for testing
			audioFile := "demo_output.mp3"
			if err := os.WriteFile(audioFile, result.AudioData, 0644); err == nil {
				fmt.Printf("💾 Audio saved to: %s\n", audioFile)
			}
		}
	}

	// Demo 3: Speech-to-Text
	if config.EnableSTT {
		fmt.Println("\n🎯 Demo 3: Speech-to-Text")
		
		// Create a mock audio file for demonstration
		audioData := createMockAudioData()
		
		options := speech.STTOptions{
			Language:            "en",
			Model:              "base",
			ConfidenceThreshold: 0.5,
		}

		fmt.Printf("🎤 Transcribing audio (%d bytes)\n", len(audioData))
		result, err := speechService.ProcessVoiceQuery(ctx, session.ID, audioData, options)
		if err != nil {
			log.Printf("STT failed (expected in demo): %v", err)
		} else {
			fmt.Printf("✅ Transcription: \"%s\"\n", result.TranscribedText)
			fmt.Printf("   Confidence: %.2f, Language: %s, Processing time: %.2fs\n", 
				result.Confidence, result.Language, result.ProcessingTime)
		}
	}

	// Demo 4: Voice Authentication
	if config.EnableVoiceAuth {
		fmt.Println("\n🎯 Demo 4: Voice Authentication")
		
		// Create mock audio samples for enrollment
		audioSamples := [][]byte{
			createMockAudioData(),
			createMockAudioData(),
			createMockAudioData(),
		}

		fmt.Printf("👤 Enrolling voice profile with %d samples\n", len(audioSamples))
		profile, err := speechService.EnrollUserVoice(ctx, userID, audioSamples)
		if err != nil {
			log.Printf("Voice enrollment failed: %v", err)
		} else {
			fmt.Printf("✅ Voice profile created: %d samples, quality: %.2f\n", 
				profile.SampleCount, profile.Quality)

			// Test authentication
			testAudio := createMockAudioData()
			fmt.Printf("🔐 Testing voice authentication\n")
			authResult, err := speechService.AuthenticateVoice(ctx, userID, testAudio)
			if err != nil {
				log.Printf("Voice authentication failed: %v", err)
			} else {
				fmt.Printf("✅ Authentication result: %v (confidence: %.2f)\n", 
					authResult.IsAuthenticated, authResult.Confidence)
			}
		}
	}

	// Demo 5: Service Capabilities
	fmt.Println("\n🎯 Demo 5: Service Capabilities")
	
	if config.EnableTTS {
		voices, err := speechService.GetAvailableVoices(ctx)
		if err != nil {
			log.Printf("Failed to get voices: %v", err)
		} else {
			fmt.Printf("🎭 Available TTS voices: %d\n", len(voices))
			for i, voice := range voices {
				if i < 3 { // Show first 3 voices
					fmt.Printf("   - %s (%s, %s, %s)\n", 
						voice.Name, voice.Language, voice.Gender, voice.Provider)
				}
			}
			if len(voices) > 3 {
				fmt.Printf("   ... and %d more\n", len(voices)-3)
			}
		}
	}

	if config.EnableSTT {
		languages, err := speechService.GetSupportedLanguages()
		if err != nil {
			log.Printf("Failed to get languages: %v", err)
		} else {
			fmt.Printf("🌍 Supported STT languages: %d\n", len(languages))
			for i, lang := range languages {
				if i < 5 { // Show first 5 languages
					fmt.Printf("   - %s (%s)\n", lang.Name, lang.Code)
				}
			}
			if len(languages) > 5 {
				fmt.Printf("   ... and %d more\n", len(languages)-5)
			}
		}
	}

	// Demo 6: Audio Validation
	fmt.Println("\n🎯 Demo 6: Audio Validation")
	
	validAudio := createMockAudioData()
	validation, err := speechService.ValidateAudioInput(validAudio, speech.AudioPurposeSTT)
	if err != nil {
		log.Printf("Validation failed: %v", err)
	} else {
		fmt.Printf("🔍 Audio validation: %v\n", validation.IsValid)
		fmt.Printf("   Format: %s, Duration: %.1fs, Size: %d bytes\n", 
			validation.Format, validation.Duration, validation.Size)
		if len(validation.Issues) > 0 {
			fmt.Printf("   Issues: %v\n", validation.Issues)
		}
	}

	// Demo 7: Session History
	fmt.Println("\n🎯 Demo 7: Session History")
	
	history, err := speechService.GetSessionHistory(ctx, userID, 10)
	if err != nil {
		log.Printf("Failed to get session history: %v", err)
	} else {
		fmt.Printf("📚 Session history: %d sessions\n", len(history))
		for _, sess := range history {
			fmt.Printf("   - %s: %s (%s, %d interactions)\n", 
				sess.ID, sess.Type, sess.Status, len(sess.Interactions))
		}
	}

	// Cleanup
	if session != nil {
		fmt.Println("\n🧹 Cleaning up...")
		err = speechService.EndSession(ctx, session.ID)
		if err != nil {
			log.Printf("Failed to end session: %v", err)
		} else {
			fmt.Printf("✅ Session ended: %s\n", session.ID)
		}
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set up ElevenLabs API key for TTS functionality")
	fmt.Println("2. Download and configure Wav2Vec2 models for STT")
	fmt.Println("3. Install FFmpeg for audio processing")
	fmt.Println("4. Run integration tests with real audio data")
}

// createMockAudioData creates a simple WAV file for demonstration
func createMockAudioData() []byte {
	// Create a minimal WAV header for a 3-second, 16kHz, mono, 16-bit file
	sampleRate := 16000
	duration := 3.0
	channels := 1
	bitsPerSample := 16
	
	bytesPerSample := bitsPerSample / 8
	dataSize := int(duration * float64(sampleRate) * float64(channels) * float64(bytesPerSample))
	
	// WAV header (44 bytes)
	header := make([]byte, 44)
	
	// RIFF header
	copy(header[0:4], "RIFF")
	// File size (will be set later)
	copy(header[8:12], "WAVE")
	
	// fmt chunk
	copy(header[12:16], "fmt ")
	header[16] = 16 // chunk size
	header[20] = 1  // audio format (PCM)
	header[22] = byte(channels)
	
	// Sample rate
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	
	// Byte rate
	byteRate := sampleRate * channels * bytesPerSample
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	
	// Block align
	blockAlign := channels * bytesPerSample
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)
	
	// Bits per sample
	header[34] = byte(bitsPerSample)
	header[35] = byte(bitsPerSample >> 8)
	
	// data chunk
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)
	
	// Set file size in RIFF header
	fileSize := 44 + dataSize - 8
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)
	
	// Create audio data (simple sine wave)
	audioData := make([]byte, dataSize)
	for i := 0; i < dataSize/2; i++ {
		// Generate a simple sine wave at 440Hz (A note)
		t := float64(i) / float64(sampleRate)
		sample := int16(32767 * 0.1 * math.Sin(2*math.Pi*440*t)) // Low volume
		
		audioData[i*2] = byte(sample)
		audioData[i*2+1] = byte(sample >> 8)
	}
	
	// Combine header and data
	result := make([]byte, len(header)+len(audioData))
	copy(result, header)
	copy(result[len(header):], audioData)
	
	return result
}