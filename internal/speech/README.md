# Speech Services

This package provides comprehensive speech-to-text (STT), text-to-speech (TTS), and voice authentication capabilities for the AI Government Consultant platform.

## Features

### Text-to-Speech (TTS)
- **ElevenLabs Integration**: High-quality speech synthesis using ElevenLabs API
- **Multiple Voices**: Support for various voice profiles and languages
- **Quality Control**: Configurable quality settings (low, medium, high)
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **Audio Formats**: Support for MP3, WAV, and OGG output formats

### Speech-to-Text (STT)
- **Wav2Vec2 Models**: Local processing using Facebook's Wav2Vec2 models
- **Privacy-First**: All transcription happens locally for data security
- **Multi-Language**: Support for 10+ languages
- **Real-time Processing**: Chunked audio processing for long recordings
- **High Accuracy**: State-of-the-art speech recognition accuracy

### Voice Authentication
- **Biometric Security**: Voice-based user authentication
- **Enrollment Process**: Multi-sample voice profile creation
- **Similarity Matching**: Cosine similarity-based authentication
- **Profile Management**: Voice profile updates and maintenance
- **Security Features**: Encrypted voice embeddings and secure storage

### Session Management
- **Speech Sessions**: Managed conversation sessions with audio history
- **Interaction Tracking**: Complete audit trail of speech interactions
- **Session Types**: Support for consultation, transcription, and authentication sessions
- **Automatic Cleanup**: Expired session cleanup and resource management

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TTS Service   │    │   STT Service   │    │  Voice Auth     │
│  (ElevenLabs)   │    │   (Wav2Vec2)    │    │   Service       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Speech Service  │
                    │   (Unified)     │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Session Manager │
                    │   (MongoDB)     │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Audio Processor │
                    │   (FFmpeg)      │
                    └─────────────────┘
```

## Components

### Core Services

#### `SpeechService`
Main service that orchestrates all speech-related operations:
- Session management
- Service coordination
- Audio validation
- Error handling

#### `ElevenLabsTTSService`
Text-to-speech implementation using ElevenLabs API:
- Voice synthesis
- Voice management
- Quality control
- Rate limiting

#### `Wav2Vec2STTService`
Speech-to-text implementation using Wav2Vec2 models:
- Local transcription
- Multi-language support
- Audio preprocessing
- Chunked processing

#### `MongoVoiceAuthenticator`
Voice authentication service:
- Voice enrollment
- Authentication
- Profile management
- Security features

### Supporting Components

#### `AudioProcessor`
Audio preprocessing and format conversion:
- Format conversion (WAV, MP3, OGG)
- Sample rate conversion
- Audio normalization
- Quality validation

#### `SpeechSessionManager`
Session lifecycle management:
- Session creation/deletion
- Interaction tracking
- Cleanup routines
- Statistics collection

## API Endpoints

### Session Management
```
POST   /api/v1/speech/sessions              # Create speech session
GET    /api/v1/speech/sessions/history      # Get session history
DELETE /api/v1/speech/sessions/:id          # End session
```

### Voice Processing
```
POST   /api/v1/speech/sessions/:id/query    # Process voice query
POST   /api/v1/speech/sessions/:id/response # Generate voice response
POST   /api/v1/speech/transcribe            # Direct transcription
POST   /api/v1/speech/synthesize            # Direct synthesis
```

### Voice Authentication
```
POST   /api/v1/speech/auth/enroll           # Enroll voice profile
POST   /api/v1/speech/auth/authenticate     # Authenticate voice
```

### Service Information
```
GET    /api/v1/speech/voices                # Available TTS voices
GET    /api/v1/speech/languages             # Supported STT languages
POST   /api/v1/speech/upload                # Upload and validate audio
```

## Configuration

### Environment Variables
```bash
# ElevenLabs Configuration
ELEVENLABS_API_KEY=your_api_key_here

# Wav2Vec2 Configuration
WAV2VEC2_MODEL_PATH=/path/to/wav2vec2/model
WAV2VEC2_PROCESSOR_PATH=/path/to/wav2vec2/processor

# MongoDB Configuration
MONGODB_URI=mongodb://localhost:27017
```

### Configuration File
See `configs/speech.yaml` for detailed configuration options.

## Setup Instructions

### 1. ElevenLabs Setup
1. Sign up for ElevenLabs account
2. Get API key from dashboard
3. Set `ELEVENLABS_API_KEY` environment variable

### 2. Wav2Vec2 Setup
1. Install Python dependencies:
   ```bash
   pip install torch torchaudio transformers
   ```

2. Download Wav2Vec2 models:
   ```python
   from transformers import Wav2Vec2ForCTC, Wav2Vec2Processor
   
   model = Wav2Vec2ForCTC.from_pretrained("facebook/wav2vec2-base-960h")
   processor = Wav2Vec2Processor.from_pretrained("facebook/wav2vec2-base-960h")
   
   model.save_pretrained("/path/to/model")
   processor.save_pretrained("/path/to/processor")
   ```

3. Set model paths in environment variables

### 3. Audio Processing Setup
1. Install FFmpeg:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install ffmpeg
   
   # macOS
   brew install ffmpeg
   
   # Windows
   # Download from https://ffmpeg.org/download.html
   ```

### 4. MongoDB Setup
Ensure MongoDB is running and accessible. The service will create necessary collections automatically.

## Usage Examples

### Creating a Speech Session
```go
speechService := speech.NewSpeechService(db, config)
session, err := speechService.CreateConsultationSession(ctx, userID)
```

### Processing Voice Query
```go
audioData := []byte{...} // WAV audio data
options := speech.STTOptions{
    Language: "en",
    Model: "base",
}

result, err := speechService.ProcessVoiceQuery(ctx, sessionID, audioData, options)
fmt.Printf("Transcribed: %s (confidence: %.2f)", result.TranscribedText, result.Confidence)
```

### Generating Voice Response
```go
text := "Thank you for your question. Based on the policy documents..."
options := speech.TTSOptions{
    Voice: "professional-female",
    Language: "en",
    Quality: "high",
}

result, err := speechService.GenerateVoiceResponse(ctx, sessionID, text, options)
// result.AudioData contains the synthesized speech
```

### Voice Authentication
```go
// Enrollment
audioSamples := [][]byte{sample1, sample2, sample3}
profile, err := speechService.EnrollUserVoice(ctx, userID, audioSamples)

// Authentication
testAudio := []byte{...}
authResult, err := speechService.AuthenticateVoice(ctx, userID, testAudio)
if authResult.IsAuthenticated {
    fmt.Printf("Voice authenticated with confidence: %.2f", authResult.Confidence)
}
```

## Testing

### Unit Tests
```bash
go test ./internal/speech/...
```

### Integration Tests
```bash
INTEGRATION_TEST=true go test ./internal/speech/...
```

### Performance Tests
```bash
PERFORMANCE_TEST=true go test ./internal/speech/...
```

### Load Tests
```bash
LOAD_TEST=true go test ./internal/speech/...
```

## Security Considerations

### Data Protection
- All audio data is encrypted at rest and in transit
- Voice embeddings are stored securely in MongoDB
- Session data includes audit trails for compliance

### Privacy
- Wav2Vec2 processing happens locally (no data sent to external services)
- ElevenLabs TTS only sends text (no audio data)
- Voice profiles use embeddings, not raw audio storage

### Access Control
- All endpoints require authentication
- Role-based access control for administrative functions
- Session isolation prevents cross-user data access

## Performance Characteristics

### TTS Performance
- **Latency**: ~2-5 seconds for typical responses
- **Throughput**: 60 requests/minute (API limit)
- **Quality**: High-quality neural voices

### STT Performance
- **Latency**: ~0.5-2 seconds per audio chunk
- **Accuracy**: >95% for clear English speech
- **Languages**: 10+ supported languages

### Voice Authentication
- **Enrollment**: 3-10 samples required
- **Authentication**: <1 second processing time
- **Accuracy**: >90% with quality audio samples

## Monitoring and Observability

### Metrics
- Request/response times
- Error rates
- Queue sizes
- Resource utilization

### Logging
- All speech interactions logged
- Performance metrics tracked
- Error conditions recorded

### Health Checks
- Service availability
- Model loading status
- External API connectivity

## Troubleshooting

### Common Issues

#### TTS Issues
- **API Key Invalid**: Check ElevenLabs API key
- **Rate Limiting**: Implement backoff strategies
- **Audio Quality**: Adjust quality settings

#### STT Issues
- **Model Not Found**: Verify model paths
- **Python Dependencies**: Install required packages
- **Audio Format**: Ensure WAV format with correct sample rate

#### Voice Authentication Issues
- **Low Accuracy**: Improve audio quality, add more samples
- **Enrollment Failures**: Check audio duration and format
- **Profile Not Found**: Verify user enrollment

### Debug Mode
Enable debug logging in configuration:
```yaml
monitoring:
  log_level: "debug"
  debug_audio_processing: true
  save_debug_audio: true
```

## Future Enhancements

### Planned Features
- Real-time streaming STT
- Custom voice cloning
- Multi-speaker recognition
- Emotion detection
- Language auto-detection

### Performance Improvements
- GPU acceleration for Wav2Vec2
- Audio compression optimization
- Caching strategies
- Load balancing

### Security Enhancements
- Advanced voice spoofing detection
- Continuous authentication
- Behavioral biometrics
- Zero-trust architecture

## Contributing

When contributing to the speech services:

1. Follow the existing code patterns
2. Add comprehensive tests
3. Update documentation
4. Consider security implications
5. Test with real audio data
6. Verify performance impact

## License

This speech services implementation is part of the AI Government Consultant platform and follows the same licensing terms.