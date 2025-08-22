package speech

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultAudioProcessor implements AudioProcessor interface
type DefaultAudioProcessor struct {
	ffmpegPath string
}

// NewDefaultAudioProcessor creates a new default audio processor
func NewDefaultAudioProcessor() *DefaultAudioProcessor {
	return &DefaultAudioProcessor{
		ffmpegPath: "ffmpeg", // Assumes ffmpeg is in PATH
	}
}

// PreprocessAudio preprocesses audio data to target format
func (p *DefaultAudioProcessor) PreprocessAudio(audioData []byte, targetFormat AudioFormat) ([]byte, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audio_processing_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write input audio to temporary file
	inputFile := filepath.Join(tempDir, "input.wav")
	if err := os.WriteFile(inputFile, audioData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Create output file path
	outputFile := filepath.Join(tempDir, "output.wav")

	// Build ffmpeg command
	args := []string{
		"-i", inputFile,
		"-ar", fmt.Sprintf("%d", targetFormat.SampleRate),
		"-ac", fmt.Sprintf("%d", targetFormat.Channels),
		"-sample_fmt", p.getSampleFormat(targetFormat.BitDepth),
		"-f", "wav",
		"-y", // Overwrite output file
		outputFile,
	}

	// Execute ffmpeg
	cmd := exec.Command(p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg processing failed: %w", err)
	}

	// Read processed audio
	processedData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read processed audio: %w", err)
	}

	return processedData, nil
}

// ResampleAudio resamples audio to target sample rate
func (p *DefaultAudioProcessor) ResampleAudio(audioData []byte, targetSampleRate int) ([]byte, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audio_resample_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write input audio to temporary file
	inputFile := filepath.Join(tempDir, "input.wav")
	if err := os.WriteFile(inputFile, audioData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Create output file path
	outputFile := filepath.Join(tempDir, "resampled.wav")

	// Build ffmpeg command for resampling
	args := []string{
		"-i", inputFile,
		"-ar", fmt.Sprintf("%d", targetSampleRate),
		"-f", "wav",
		"-y", // Overwrite output file
		outputFile,
	}

	// Execute ffmpeg
	cmd := exec.Command(p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg resampling failed: %w", err)
	}

	// Read resampled audio
	resampledData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read resampled audio: %w", err)
	}

	return resampledData, nil
}

// NormalizeAudio normalizes audio amplitude
func (p *DefaultAudioProcessor) NormalizeAudio(audioData []byte) ([]byte, error) {
	// Parse WAV header to get audio parameters
	if len(audioData) < 44 {
		return nil, fmt.Errorf("invalid WAV file: too short")
	}

	// Extract audio parameters from WAV header
	channels := p.extractChannels(audioData)
	bitsPerSample := p.extractBitsPerSample(audioData)
	
	// Calculate bytes per sample
	bytesPerSample := bitsPerSample / 8
	
	// Extract audio samples (skip 44-byte header)
	audioSamples := audioData[44:]
	numSamples := len(audioSamples) / (bytesPerSample * channels)
	
	if numSamples == 0 {
		return audioData, nil // No samples to normalize
	}

	// Convert samples to float64 for processing
	samples := make([]float64, numSamples*channels)
	maxValue := float64(int32(1<<uint(bitsPerSample-1)) - 1)
	
	for i := 0; i < numSamples*channels; i++ {
		sampleBytes := audioSamples[i*bytesPerSample : (i+1)*bytesPerSample]
		var sample int32
		
		switch bytesPerSample {
		case 1:
			sample = int32(sampleBytes[0]) - 128 // Convert unsigned to signed
		case 2:
			sample = int32(int16(binary.LittleEndian.Uint16(sampleBytes)))
		case 4:
			sample = int32(binary.LittleEndian.Uint32(sampleBytes))
		}
		
		samples[i] = float64(sample) / maxValue
	}

	// Find peak amplitude
	peak := 0.0
	for _, sample := range samples {
		if abs := math.Abs(sample); abs > peak {
			peak = abs
		}
	}

	// Normalize if peak is above threshold
	if peak > 0.1 { // Avoid normalizing very quiet audio
		normalizationFactor := 0.95 / peak // Leave some headroom
		
		for i := range samples {
			samples[i] *= normalizationFactor
		}
	}

	// Convert back to original format
	normalizedAudioData := make([]byte, len(audioData))
	copy(normalizedAudioData[:44], audioData[:44]) // Copy header
	
	for i := 0; i < numSamples*channels; i++ {
		sample := int32(samples[i] * maxValue)
		sampleBytes := normalizedAudioData[44+i*bytesPerSample : 44+(i+1)*bytesPerSample]
		
		switch bytesPerSample {
		case 1:
			sampleBytes[0] = byte(sample + 128) // Convert signed to unsigned
		case 2:
			binary.LittleEndian.PutUint16(sampleBytes, uint16(sample))
		case 4:
			binary.LittleEndian.PutUint32(sampleBytes, uint32(sample))
		}
	}

	return normalizedAudioData, nil
}

// ConvertFormat converts audio from one format to another
func (p *DefaultAudioProcessor) ConvertFormat(audioData []byte, sourceFormat, targetFormat AudioFormat) ([]byte, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audio_convert_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write input audio to temporary file
	inputFile := filepath.Join(tempDir, fmt.Sprintf("input.%s", sourceFormat.Codec))
	if err := os.WriteFile(inputFile, audioData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Create output file path
	outputFile := filepath.Join(tempDir, fmt.Sprintf("output.%s", targetFormat.Codec))

	// Build ffmpeg command for format conversion
	args := []string{
		"-i", inputFile,
		"-ar", fmt.Sprintf("%d", targetFormat.SampleRate),
		"-ac", fmt.Sprintf("%d", targetFormat.Channels),
		"-sample_fmt", p.getSampleFormat(targetFormat.BitDepth),
		"-f", targetFormat.Codec,
		"-y", // Overwrite output file
		outputFile,
	}

	// Execute ffmpeg
	cmd := exec.Command(p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg format conversion failed: %w", err)
	}

	// Read converted audio
	convertedData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read converted audio: %w", err)
	}

	return convertedData, nil
}

// getSampleFormat returns ffmpeg sample format string
func (p *DefaultAudioProcessor) getSampleFormat(bitDepth int) string {
	switch bitDepth {
	case 8:
		return "u8"
	case 16:
		return "s16"
	case 24:
		return "s32" // ffmpeg doesn't have s24, use s32
	case 32:
		return "s32"
	default:
		return "s16" // Default to 16-bit
	}
}

// extractSampleRate extracts sample rate from WAV header
func (p *DefaultAudioProcessor) extractSampleRate(audioData []byte) int {
	if len(audioData) < 28 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(audioData[24:28]))
}

// extractChannels extracts number of channels from WAV header
func (p *DefaultAudioProcessor) extractChannels(audioData []byte) int {
	if len(audioData) < 24 {
		return 0
	}
	return int(binary.LittleEndian.Uint16(audioData[22:24]))
}

// extractBitsPerSample extracts bits per sample from WAV header
func (p *DefaultAudioProcessor) extractBitsPerSample(audioData []byte) int {
	if len(audioData) < 36 {
		return 0
	}
	return int(binary.LittleEndian.Uint16(audioData[34:36]))
}

// SimpleAudioProcessor provides basic audio processing without external dependencies
type SimpleAudioProcessor struct{}

// NewSimpleAudioProcessor creates a new simple audio processor
func NewSimpleAudioProcessor() *SimpleAudioProcessor {
	return &SimpleAudioProcessor{}
}

// PreprocessAudio provides basic preprocessing without external tools
func (p *SimpleAudioProcessor) PreprocessAudio(audioData []byte, targetFormat AudioFormat) ([]byte, error) {
	// For now, just return the original data
	// In a production environment, you would implement proper audio processing
	return audioData, nil
}

// ResampleAudio provides basic resampling (placeholder implementation)
func (p *SimpleAudioProcessor) ResampleAudio(audioData []byte, targetSampleRate int) ([]byte, error) {
	// Placeholder implementation - in production, implement proper resampling
	return audioData, nil
}

// NormalizeAudio provides basic normalization
func (p *SimpleAudioProcessor) NormalizeAudio(audioData []byte) ([]byte, error) {
	// Use the same normalization logic as DefaultAudioProcessor
	processor := &DefaultAudioProcessor{}
	return processor.NormalizeAudio(audioData)
}

// ConvertFormat provides basic format conversion (placeholder)
func (p *SimpleAudioProcessor) ConvertFormat(audioData []byte, sourceFormat, targetFormat AudioFormat) ([]byte, error) {
	// Placeholder implementation - in production, implement proper format conversion
	return audioData, nil
}

// AudioValidator provides audio validation utilities
type AudioValidator struct{}

// NewAudioValidator creates a new audio validator
func NewAudioValidator() *AudioValidator {
	return &AudioValidator{}
}

// ValidateWAVFormat validates WAV file format
func (v *AudioValidator) ValidateWAVFormat(audioData []byte) error {
	if len(audioData) < 44 {
		return fmt.Errorf("file too small to be a valid WAV file")
	}

	// Check RIFF header
	if string(audioData[0:4]) != "RIFF" {
		return fmt.Errorf("missing RIFF header")
	}

	// Check WAVE format
	if string(audioData[8:12]) != "WAVE" {
		return fmt.Errorf("not a WAVE file")
	}

	// Check fmt chunk
	if string(audioData[12:16]) != "fmt " {
		return fmt.Errorf("missing fmt chunk")
	}

	return nil
}

// GetAudioInfo extracts audio information from WAV file
func (v *AudioValidator) GetAudioInfo(audioData []byte) (*AudioInfo, error) {
	if err := v.ValidateWAVFormat(audioData); err != nil {
		return nil, err
	}

	info := &AudioInfo{
		Format:     "wav",
		SampleRate: int(binary.LittleEndian.Uint32(audioData[24:28])),
		Channels:   int(binary.LittleEndian.Uint16(audioData[22:24])),
		BitDepth:   int(binary.LittleEndian.Uint16(audioData[34:36])),
		Size:       int64(len(audioData)),
	}

	// Calculate duration
	bytesPerSample := info.BitDepth / 8
	dataSize := len(audioData) - 44 // Subtract header size
	samplesPerChannel := dataSize / (bytesPerSample * info.Channels)
	info.Duration = float64(samplesPerChannel) / float64(info.SampleRate)

	return info, nil
}

// AudioInfo contains information about an audio file
type AudioInfo struct {
	Format     string  `json:"format"`
	SampleRate int     `json:"sample_rate"`
	Channels   int     `json:"channels"`
	BitDepth   int     `json:"bit_depth"`
	Duration   float64 `json:"duration"`
	Size       int64   `json:"size"`
}