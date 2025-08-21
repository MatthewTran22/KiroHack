package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Wav2Vec2STTService implements SpeechToTextService using Wav2Vec2 models
type Wav2Vec2STTService struct {
	config         Wav2Vec2Config
	audioProcessor AudioProcessor
	pythonPath     string
	scriptPath     string
}

// NewWav2Vec2STTService creates a new Wav2Vec2 STT service
func NewWav2Vec2STTService(config Wav2Vec2Config, audioProcessor AudioProcessor) (*Wav2Vec2STTService, error) {
	service := &Wav2Vec2STTService{
		config:         config,
		audioProcessor: audioProcessor,
		pythonPath:     "python3", // Default Python path
	}

	// Create the Python script for Wav2Vec2 inference
	if err := service.createInferenceScript(); err != nil {
		return nil, fmt.Errorf("failed to create inference script: %w", err)
	}

	// Validate model availability
	if err := service.validateModel(); err != nil {
		return nil, fmt.Errorf("model validation failed: %w", err)
	}

	return service, nil
}

// TranscribeAudio transcribes audio using Wav2Vec2 model
func (s *Wav2Vec2STTService) TranscribeAudio(ctx context.Context, audioData []byte, options STTOptions) (*TranscriptionResult, error) {
	startTime := time.Now()

	// Validate audio format
	validation := s.ValidateAudioFormat(audioData)
	if !validation.IsValid {
		return nil, fmt.Errorf("invalid audio format: %v", validation.Issues)
	}

	// Set default options
	if options.Model == "" {
		options.Model = "base"
	}
	if options.SampleRate == 0 {
		options.SampleRate = s.config.SampleRate
	}
	if options.ChunkSize == 0 {
		options.ChunkSize = 1024
	}
	if options.ConfidenceThreshold == 0 {
		options.ConfidenceThreshold = 0.5
	}

	// Preprocess audio for Wav2Vec2
	processedAudio, err := s.preprocessAudioForWav2Vec2(audioData, options)
	if err != nil {
		return nil, fmt.Errorf("audio preprocessing failed: %w", err)
	}

	// Create temporary file for audio
	tempDir, err := os.MkdirTemp("", "wav2vec2_audio_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	audioFile := filepath.Join(tempDir, "input.wav")
	if err := os.WriteFile(audioFile, processedAudio, 0644); err != nil {
		return nil, fmt.Errorf("failed to write audio file: %w", err)
	}

	// Prepare inference parameters
	params := map[string]interface{}{
		"audio_file":           audioFile,
		"model_path":          s.config.ModelPath,
		"processor_path":      s.config.ProcessorPath,
		"device":              s.config.DeviceType,
		"batch_size":          s.config.BatchSize,
		"chunk_duration":      s.config.ChunkDuration,
		"confidence_threshold": options.ConfidenceThreshold,
		"enable_timestamps":   true,
		"language":            options.Language,
	}

	// Execute Wav2Vec2 inference
	result, err := s.runInference(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Process results
	transcriptionResult := &TranscriptionResult{
		Text:           result.Text,
		Confidence:     result.Confidence,
		Language:       options.Language,
		Duration:       validation.Duration,
		Timestamps:     result.Timestamps,
		ProcessedAt:    time.Now(),
		ModelUsed:      fmt.Sprintf("wav2vec2-%s", options.Model),
		ProcessingTime: time.Since(startTime).Seconds(),
		Segments:       result.Segments,
	}

	// Filter results by confidence threshold
	if transcriptionResult.Confidence < options.ConfidenceThreshold {
		return nil, fmt.Errorf("transcription confidence (%.2f) below threshold (%.2f)", 
			transcriptionResult.Confidence, options.ConfidenceThreshold)
	}

	return transcriptionResult, nil
}

// GetSupportedLanguages returns supported languages for Wav2Vec2
func (s *Wav2Vec2STTService) GetSupportedLanguages() ([]Language, error) {
	// Common Wav2Vec2 supported languages
	languages := []Language{
		{Code: "en", Name: "English", Region: "US", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "es", Name: "Spanish", Region: "ES", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "fr", Name: "French", Region: "FR", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "de", Name: "German", Region: "DE", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "it", Name: "Italian", Region: "IT", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "pt", Name: "Portuguese", Region: "PT", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "ru", Name: "Russian", Region: "RU", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "zh", Name: "Chinese", Region: "CN", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "ja", Name: "Japanese", Region: "JP", IsSupported: true, ModelPath: s.config.ModelPath},
		{Code: "ko", Name: "Korean", Region: "KR", IsSupported: true, ModelPath: s.config.ModelPath},
	}

	return languages, nil
}

// ValidateAudioFormat validates audio format for Wav2Vec2 processing
func (s *Wav2Vec2STTService) ValidateAudioFormat(audioData []byte) *FormatValidation {
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

	// Basic WAV header validation
	if len(audioData) >= 44 {
		// Check RIFF header
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
			validation.Format = "unknown"
			validation.Issues = append(validation.Issues, "unsupported audio format, WAV required")
			validation.IsValid = false
		}
	} else {
		validation.Issues = append(validation.Issues, "invalid audio header")
		validation.IsValid = false
	}

	validation.Size = int64(len(audioData))

	// Validate sample rate for Wav2Vec2
	if validation.SampleRate != s.config.SampleRate {
		validation.Issues = append(validation.Issues, 
			fmt.Sprintf("sample rate %d Hz not optimal, %d Hz recommended", 
				validation.SampleRate, s.config.SampleRate))
	}

	// Check if mono audio (Wav2Vec2 works best with mono)
	if validation.Channels > 1 {
		validation.Issues = append(validation.Issues, "stereo audio detected, mono recommended for better accuracy")
	}

	return validation
}

// preprocessAudioForWav2Vec2 preprocesses audio for optimal Wav2Vec2 performance
func (s *Wav2Vec2STTService) preprocessAudioForWav2Vec2(audioData []byte, options STTOptions) ([]byte, error) {
	// Target format for Wav2Vec2
	targetFormat := AudioFormat{
		Codec:      "wav",
		SampleRate: s.config.SampleRate, // 16000 Hz
		Channels:   1,                    // Mono
		BitDepth:   16,                   // 16-bit
	}

	// Resample if necessary
	if options.SampleRate != s.config.SampleRate {
		resampled, err := s.audioProcessor.ResampleAudio(audioData, s.config.SampleRate)
		if err != nil {
			return nil, fmt.Errorf("resampling failed: %w", err)
		}
		audioData = resampled
	}

	// Convert to target format
	processed, err := s.audioProcessor.PreprocessAudio(audioData, targetFormat)
	if err != nil {
		return nil, fmt.Errorf("format conversion failed: %w", err)
	}

	// Normalize audio
	normalized, err := s.audioProcessor.NormalizeAudio(processed)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	return normalized, nil
}

// createInferenceScript creates the Python script for Wav2Vec2 inference
func (s *Wav2Vec2STTService) createInferenceScript() error {
	scriptContent := `#!/usr/bin/env python3
import json
import sys
import torch
import torchaudio
import argparse
from transformers import Wav2Vec2ForCTC, Wav2Vec2Processor
import numpy as np
from typing import List, Dict, Any

def load_model_and_processor(model_path: str, processor_path: str, device: str):
    """Load Wav2Vec2 model and processor"""
    try:
        processor = Wav2Vec2Processor.from_pretrained(processor_path)
        model = Wav2Vec2ForCTC.from_pretrained(model_path)
        model.to(device)
        model.eval()
        return model, processor
    except Exception as e:
        raise RuntimeError(f"Failed to load model: {e}")

def load_audio(audio_path: str, target_sample_rate: int = 16000):
    """Load and preprocess audio file"""
    try:
        waveform, sample_rate = torchaudio.load(audio_path)
        
        # Convert to mono if stereo
        if waveform.shape[0] > 1:
            waveform = torch.mean(waveform, dim=0, keepdim=True)
        
        # Resample if necessary
        if sample_rate != target_sample_rate:
            resampler = torchaudio.transforms.Resample(sample_rate, target_sample_rate)
            waveform = resampler(waveform)
        
        return waveform.squeeze().numpy()
    except Exception as e:
        raise RuntimeError(f"Failed to load audio: {e}")

def chunk_audio(audio: np.ndarray, chunk_duration: float, sample_rate: int):
    """Split audio into chunks for processing"""
    chunk_size = int(chunk_duration * sample_rate)
    chunks = []
    
    for i in range(0, len(audio), chunk_size):
        chunk = audio[i:i + chunk_size]
        if len(chunk) > 0:
            chunks.append(chunk)
    
    return chunks

def transcribe_chunk(model, processor, audio_chunk: np.ndarray, device: str):
    """Transcribe a single audio chunk"""
    try:
        # Preprocess audio
        inputs = processor(audio_chunk, sampling_rate=16000, return_tensors="pt", padding=True)
        input_values = inputs.input_values.to(device)
        
        # Get model predictions
        with torch.no_grad():
            logits = model(input_values).logits
        
        # Decode predictions
        predicted_ids = torch.argmax(logits, dim=-1)
        transcription = processor.batch_decode(predicted_ids)[0]
        
        # Calculate confidence (simplified)
        probs = torch.nn.functional.softmax(logits, dim=-1)
        confidence = torch.max(probs, dim=-1)[0].mean().item()
        
        return transcription, confidence
    except Exception as e:
        return "", 0.0

def main():
    parser = argparse.ArgumentParser(description='Wav2Vec2 Speech Recognition')
    parser.add_argument('--params', required=True, help='JSON parameters')
    args = parser.parse_args()
    
    try:
        # Parse parameters
        params = json.loads(args.params)
        
        # Load model and processor
        model, processor = load_model_and_processor(
            params['model_path'],
            params['processor_path'],
            params['device']
        )
        
        # Load audio
        audio = load_audio(params['audio_file'])
        
        # Process audio in chunks
        chunks = chunk_audio(audio, params['chunk_duration'], 16000)
        
        transcriptions = []
        confidences = []
        timestamps = []
        segments = []
        
        current_time = 0.0
        chunk_duration = params['chunk_duration']
        
        for i, chunk in enumerate(chunks):
            transcription, confidence = transcribe_chunk(model, processor, chunk, params['device'])
            
            if transcription.strip() and confidence >= params['confidence_threshold']:
                transcriptions.append(transcription.strip())
                confidences.append(confidence)
                
                # Create segment
                segment = {
                    'text': transcription.strip(),
                    'start_time': current_time,
                    'end_time': current_time + chunk_duration,
                    'confidence': confidence
                }
                segments.append(segment)
                
                # Create word timestamps (simplified)
                words = transcription.strip().split()
                word_duration = chunk_duration / len(words) if words else 0
                
                for j, word in enumerate(words):
                    timestamp = {
                        'word': word,
                        'start_time': current_time + (j * word_duration),
                        'end_time': current_time + ((j + 1) * word_duration),
                        'confidence': confidence
                    }
                    timestamps.append(timestamp)
            
            current_time += chunk_duration
        
        # Combine results
        full_text = ' '.join(transcriptions)
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0.0
        
        # Output results
        result = {
            'text': full_text,
            'confidence': avg_confidence,
            'timestamps': timestamps,
            'segments': segments,
            'success': True
        }
        
        print(json.dumps(result))
        
    except Exception as e:
        error_result = {
            'text': '',
            'confidence': 0.0,
            'timestamps': [],
            'segments': [],
            'success': False,
            'error': str(e)
        }
        print(json.dumps(error_result))
        sys.exit(1)

if __name__ == '__main__':
    main()
`

	// Create script directory
	scriptDir := filepath.Join("scripts", "speech")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script directory: %w", err)
	}

	// Write script file
	s.scriptPath = filepath.Join(scriptDir, "wav2vec2_inference.py")
	if err := os.WriteFile(s.scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	return nil
}

// validateModel validates that the Wav2Vec2 model is available
func (s *Wav2Vec2STTService) validateModel() error {
	// Check if model path exists
	if _, err := os.Stat(s.config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model path does not exist: %s", s.config.ModelPath)
	}

	// Check if processor path exists
	if _, err := os.Stat(s.config.ProcessorPath); os.IsNotExist(err) {
		return fmt.Errorf("processor path does not exist: %s", s.config.ProcessorPath)
	}

	// Test Python and dependencies
	cmd := exec.Command(s.pythonPath, "-c", "import torch, torchaudio, transformers; print('OK')")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Python dependencies not available: %w", err)
	}

	return nil
}

// runInference executes the Wav2Vec2 inference script
func (s *Wav2Vec2STTService) runInference(ctx context.Context, params map[string]interface{}) (*InferenceResult, error) {
	// Convert parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Create command
	cmd := exec.CommandContext(ctx, s.pythonPath, s.scriptPath, "--params", string(paramsJSON))
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("inference script failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse result
	var result InferenceResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse inference result: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("inference failed: %s", result.Error)
	}

	return &result, nil
}

// InferenceResult represents the result from Wav2Vec2 inference
type InferenceResult struct {
	Text       string          `json:"text"`
	Confidence float64         `json:"confidence"`
	Timestamps []WordTimestamp `json:"timestamps"`
	Segments   []TextSegment   `json:"segments"`
	Success    bool            `json:"success"`
	Error      string          `json:"error,omitempty"`
}