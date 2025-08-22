package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ai-government-consultant/internal/speech"

	"github.com/gin-gonic/gin"
)

// SpeechHandler handles speech-related API endpoints
type SpeechHandler struct {
	speechService *speech.SpeechService
}

// NewSpeechHandler creates a new speech handler
func NewSpeechHandler(speechService *speech.SpeechService) *SpeechHandler {
	return &SpeechHandler{
		speechService: speechService,
	}
}

// CreateSpeechSession creates a new speech session
// POST /api/v1/speech/sessions
func (h *SpeechHandler) CreateSpeechSession(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	session, err := h.speechService.CreateConsultationSession(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create session: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session": session,
		"message": "speech session created successfully",
	})
}

// ProcessVoiceQuery processes a voice query
// POST /api/v1/speech/sessions/:sessionId/query
func (h *SpeechHandler) ProcessVoiceQuery(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session ID is required"})
		return
	}

	// Parse request body
	var request struct {
		AudioData string                `json:"audio_data"` // Base64 encoded audio
		Options   speech.STTOptions     `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Decode audio data
	audioData, err := base64.StdEncoding.DecodeString(request.AudioData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audio data encoding"})
		return
	}

	// Process voice query
	result, err := h.speechService.ProcessVoiceQuery(c.Request.Context(), sessionID, audioData, request.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to process voice query: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": result,
		"message": "voice query processed successfully",
	})
}

// GenerateVoiceResponse generates a voice response
// POST /api/v1/speech/sessions/:sessionId/response
func (h *SpeechHandler) GenerateVoiceResponse(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session ID is required"})
		return
	}

	// Parse request body
	var request struct {
		Text    string                `json:"text"`
		Options speech.TTSOptions     `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if request.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	// Generate voice response
	result, err := h.speechService.GenerateVoiceResponse(c.Request.Context(), sessionID, request.Text, request.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate voice response: %v", err)})
		return
	}

	// Encode audio data as base64 for JSON response
	audioDataB64 := base64.StdEncoding.EncodeToString(result.AudioData)

	c.JSON(http.StatusOK, gin.H{
		"result": gin.H{
			"session_id":   result.SessionID,
			"audio_data":   audioDataB64,
			"duration":     result.Duration,
			"format":       result.Format,
			"size":         result.Size,
			"voice":        result.Voice,
			"quality":      result.Quality,
			"generated_at": result.GeneratedAt,
		},
		"message": "voice response generated successfully",
	})
}

// UploadAudioFile handles audio file uploads
// POST /api/v1/speech/upload
func (h *SpeechHandler) UploadAudioFile(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
		return
	}
	defer file.Close()

	// Read file data
	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audio file"})
		return
	}

	// Get purpose from form data
	purpose := c.PostForm("purpose")
	if purpose == "" {
		purpose = "general"
	}

	// Validate audio
	audioPurpose := speech.AudioPurpose(purpose)
	validation, err := h.speechService.ValidateAudioInput(audioData, audioPurpose)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("validation failed: %v", err)})
		return
	}

	response := gin.H{
		"filename":   header.Filename,
		"size":       header.Size,
		"validation": validation,
		"audio_data": base64.StdEncoding.EncodeToString(audioData),
	}

	if validation.IsValid {
		c.JSON(http.StatusOK, gin.H{
			"result":  response,
			"message": "audio file uploaded and validated successfully",
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  response,
			"message": "audio file validation failed",
		})
	}
}

// AuthenticateVoice authenticates a user's voice
// POST /api/v1/speech/auth/authenticate
func (h *SpeechHandler) AuthenticateVoice(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse request body
	var request struct {
		AudioData string `json:"audio_data"` // Base64 encoded audio
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Decode audio data
	audioData, err := base64.StdEncoding.DecodeString(request.AudioData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audio data encoding"})
		return
	}

	// Authenticate voice
	result, err := h.speechService.AuthenticateVoice(c.Request.Context(), userID, audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("voice authentication failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result":  result,
		"message": "voice authentication completed",
	})
}

// EnrollVoice enrolls a user's voice for authentication
// POST /api/v1/speech/auth/enroll
func (h *SpeechHandler) EnrollVoice(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse request body
	var request struct {
		AudioSamples []string `json:"audio_samples"` // Array of base64 encoded audio samples
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if len(request.AudioSamples) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least 3 audio samples are required for enrollment"})
		return
	}

	// Decode audio samples
	var audioSamples [][]byte
	for i, sample := range request.AudioSamples {
		audioData, err := base64.StdEncoding.DecodeString(sample)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid audio data encoding in sample %d", i)})
			return
		}
		audioSamples = append(audioSamples, audioData)
	}

	// Enroll voice
	profile, err := h.speechService.EnrollUserVoice(c.Request.Context(), userID, audioSamples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("voice enrollment failed: %v", err)})
		return
	}

	// Return profile without embeddings for security
	responseProfile := gin.H{
		"user_id":      profile.UserID,
		"created_at":   profile.CreatedAt,
		"updated_at":   profile.UpdatedAt,
		"sample_count": profile.SampleCount,
		"quality":      profile.Quality,
		"is_active":    profile.IsActive,
	}

	c.JSON(http.StatusCreated, gin.H{
		"profile": responseProfile,
		"message": "voice enrollment completed successfully",
	})
}

// GetAvailableVoices returns available TTS voices
// GET /api/v1/speech/voices
func (h *SpeechHandler) GetAvailableVoices(c *gin.Context) {
	voices, err := h.speechService.GetAvailableVoices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get voices: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"voices": voices,
		"count":  len(voices),
	})
}

// GetSupportedLanguages returns supported STT languages
// GET /api/v1/speech/languages
func (h *SpeechHandler) GetSupportedLanguages(c *gin.Context) {
	languages, err := h.speechService.GetSupportedLanguages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get languages: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"languages": languages,
		"count":     len(languages),
	})
}

// GetSessionHistory returns speech session history for a user
// GET /api/v1/speech/sessions/history
func (h *SpeechHandler) GetSessionHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Get session history
	sessions, err := h.speechService.GetSessionHistory(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get session history: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// EndSession ends a speech session
// DELETE /api/v1/speech/sessions/:sessionId
func (h *SpeechHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session ID is required"})
		return
	}

	err := h.speechService.EndSession(c.Request.Context(), sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to end session: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "session ended successfully",
	})
}

// TranscribeAudio transcribes audio without creating a session
// POST /api/v1/speech/transcribe
func (h *SpeechHandler) TranscribeAudio(c *gin.Context) {
	// Parse request body
	var request struct {
		AudioData string            `json:"audio_data"` // Base64 encoded audio
		Options   speech.STTOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Decode audio data
	audioData, err := base64.StdEncoding.DecodeString(request.AudioData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audio data encoding"})
		return
	}

	// Create temporary session for transcription
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	session, err := h.speechService.CreateConsultationSession(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create transcription session"})
		return
	}

	// Process transcription
	result, err := h.speechService.ProcessVoiceQuery(c.Request.Context(), session.ID, audioData, request.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("transcription failed: %v", err)})
		return
	}

	// End the temporary session
	h.speechService.EndSession(c.Request.Context(), session.ID)

	c.JSON(http.StatusOK, gin.H{
		"transcription": result.TranscribedText,
		"confidence":    result.Confidence,
		"language":      result.Language,
		"processing_time": result.ProcessingTime,
		"timestamps":    result.Timestamps,
	})
}

// SynthesizeSpeech synthesizes speech without creating a session
// POST /api/v1/speech/synthesize
func (h *SpeechHandler) SynthesizeSpeech(c *gin.Context) {
	// Parse request body
	var request struct {
		Text    string            `json:"text"`
		Options speech.TTSOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if request.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	// Create temporary session for synthesis
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	session, err := h.speechService.CreateConsultationSession(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create synthesis session"})
		return
	}

	// Generate speech
	result, err := h.speechService.GenerateVoiceResponse(c.Request.Context(), session.ID, request.Text, request.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("speech synthesis failed: %v", err)})
		return
	}

	// End the temporary session
	h.speechService.EndSession(c.Request.Context(), session.ID)

	// Return audio as base64
	audioDataB64 := base64.StdEncoding.EncodeToString(result.AudioData)

	c.JSON(http.StatusOK, gin.H{
		"audio_data":   audioDataB64,
		"duration":     result.Duration,
		"format":       result.Format,
		"size":         result.Size,
		"voice":        result.Voice,
		"quality":      result.Quality,
		"generated_at": result.GeneratedAt,
	})
}