package consultation

import (
	"fmt"
	"regexp"
	"strings"

	"ai-government-consultant/internal/models"
)

// ResponseValidator validates consultation responses for safety and quality
type ResponseValidator struct {
	// Patterns for detecting potentially harmful content
	harmfulPatterns []string
	// Patterns for detecting low-quality responses
	qualityPatterns []string
	// Minimum confidence threshold
	minConfidence float64
}

// NewResponseValidator creates a new response validator
func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{
		harmfulPatterns: []string{
			`(?i)\b(?:classified|secret|confidential)\s+(?:information|data|documents?)\b`,
			`(?i)\b(?:personal|private)\s+(?:information|data)\b`,
			`(?i)\b(?:social\s+security|ssn|credit\s+card)\b`,
			`(?i)\b(?:password|login|credentials)\b`,
			`(?i)\b(?:illegal|unlawful|criminal)\s+(?:activity|action)\b`,
			`(?i)\b(?:discriminat|bias|prejudic)\w*\b`,
		},
		qualityPatterns: []string{
			`(?i)\b(?:i\s+don't\s+know|i'm\s+not\s+sure|uncertain|unclear)\b`,
			`(?i)\b(?:maybe|perhaps|possibly|might\s+be)\b.*(?:maybe|perhaps|possibly|might\s+be)\b`, // Multiple uncertainty markers
			`(?i)\b(?:sorry|apologize)\b.*\b(?:cannot|can't|unable)\b`,
			`(?i)\b(?:generic|general|vague)\s+(?:recommendation|advice|guidance)\b`,
		},
		minConfidence: 0.3,
	}
}

// validateResponse validates a consultation response for safety and quality
func (s *Service) validateResponse(response *models.ConsultationResponse) error {
	validator := NewResponseValidator()
	
	// Check overall confidence score
	if response.ConfidenceScore < validator.minConfidence {
		return fmt.Errorf("response confidence score %.2f is below minimum threshold %.2f", 
			response.ConfidenceScore, validator.minConfidence)
	}

	// Validate each recommendation
	for i, rec := range response.Recommendations {
		if err := validator.validateRecommendation(&rec); err != nil {
			return fmt.Errorf("recommendation %d validation failed: %w", i+1, err)
		}
	}

	// Validate analysis
	if err := validator.validateAnalysis(&response.Analysis); err != nil {
		return fmt.Errorf("analysis validation failed: %w", err)
	}

	// Check for harmful content in the overall response
	if err := validator.checkHarmfulContent(response); err != nil {
		return fmt.Errorf("harmful content detected: %w", err)
	}

	// Check response quality
	if err := validator.checkResponseQuality(response); err != nil {
		return fmt.Errorf("quality check failed: %w", err)
	}

	return nil
}

// validateRecommendation validates a single recommendation
func (v *ResponseValidator) validateRecommendation(rec *models.Recommendation) error {
	// Check minimum content requirements
	if strings.TrimSpace(rec.Title) == "" {
		return fmt.Errorf("recommendation title is empty")
	}
	
	if strings.TrimSpace(rec.Description) == "" {
		return fmt.Errorf("recommendation description is empty")
	}

	if len(rec.Description) < 20 {
		return fmt.Errorf("recommendation description is too short (minimum 20 characters)")
	}

	// Check confidence score
	if rec.ConfidenceScore < v.minConfidence {
		return fmt.Errorf("recommendation confidence score %.2f is below minimum threshold %.2f", 
			rec.ConfidenceScore, v.minConfidence)
	}

	// Check for harmful patterns
	combinedText := rec.Title + " " + rec.Description
	for _, pattern := range v.harmfulPatterns {
		if matched, _ := regexp.MatchString(pattern, combinedText); matched {
			return fmt.Errorf("potentially harmful content detected in recommendation")
		}
	}

	// Check implementation guidance
	if len(rec.Implementation.Steps) == 0 {
		return fmt.Errorf("recommendation lacks implementation steps")
	}

	return nil
}

// validateAnalysis validates the analysis section
func (v *ResponseValidator) validateAnalysis(analysis *models.Analysis) error {
	if strings.TrimSpace(analysis.Summary) == "" {
		return fmt.Errorf("analysis summary is empty")
	}

	if len(analysis.Summary) < 50 {
		return fmt.Errorf("analysis summary is too short (minimum 50 characters)")
	}

	// Check for quality issues in summary
	for _, pattern := range v.qualityPatterns {
		if matched, _ := regexp.MatchString(pattern, analysis.Summary); matched {
			return fmt.Errorf("low-quality content detected in analysis summary")
		}
	}

	return nil
}

// checkHarmfulContent checks for potentially harmful content across the response
func (v *ResponseValidator) checkHarmfulContent(response *models.ConsultationResponse) error {
	// Collect all text content
	var allText strings.Builder
	
	allText.WriteString(response.Analysis.Summary)
	allText.WriteString(" ")
	
	for _, finding := range response.Analysis.KeyFindings {
		allText.WriteString(finding)
		allText.WriteString(" ")
	}
	
	for _, rec := range response.Recommendations {
		allText.WriteString(rec.Title)
		allText.WriteString(" ")
		allText.WriteString(rec.Description)
		allText.WriteString(" ")
	}

	content := allText.String()

	// Check against harmful patterns
	for _, pattern := range v.harmfulPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return fmt.Errorf("potentially harmful content pattern detected")
		}
	}

	// Additional safety checks
	if err := v.checkForDataLeakage(content); err != nil {
		return err
	}

	if err := v.checkForBiasedLanguage(content); err != nil {
		return err
	}

	return nil
}

// checkResponseQuality checks the overall quality of the response
func (v *ResponseValidator) checkResponseQuality(response *models.ConsultationResponse) error {
	// Check minimum number of recommendations
	if len(response.Recommendations) == 0 {
		return fmt.Errorf("response contains no recommendations")
	}

	// Check for excessive uncertainty
	uncertaintyCount := 0
	totalRecommendations := len(response.Recommendations)
	
	for _, rec := range response.Recommendations {
		combinedText := rec.Title + " " + rec.Description
		for _, pattern := range v.qualityPatterns {
			if matched, _ := regexp.MatchString(pattern, combinedText); matched {
				uncertaintyCount++
				break
			}
		}
	}

	// If more than 50% of recommendations show uncertainty, flag as low quality
	if float64(uncertaintyCount)/float64(totalRecommendations) > 0.5 {
		return fmt.Errorf("response shows excessive uncertainty (%.1f%% of recommendations)", 
			float64(uncertaintyCount)/float64(totalRecommendations)*100)
	}

	// Check for sufficient detail
	avgDescriptionLength := 0
	for _, rec := range response.Recommendations {
		avgDescriptionLength += len(rec.Description)
	}
	avgDescriptionLength /= totalRecommendations

	if avgDescriptionLength < 100 {
		return fmt.Errorf("recommendations lack sufficient detail (average %d characters)", avgDescriptionLength)
	}

	// Check for actionability
	actionableCount := 0
	actionWords := []string{"implement", "develop", "create", "establish", "conduct", "review", "update", "modify", "enhance"}
	
	for _, rec := range response.Recommendations {
		lowerDesc := strings.ToLower(rec.Description)
		for _, word := range actionWords {
			if strings.Contains(lowerDesc, word) {
				actionableCount++
				break
			}
		}
	}

	if float64(actionableCount)/float64(totalRecommendations) < 0.7 {
		return fmt.Errorf("insufficient actionable recommendations (%.1f%% actionable)", 
			float64(actionableCount)/float64(totalRecommendations)*100)
	}

	return nil
}

// checkForDataLeakage checks for potential data leakage
func (v *ResponseValidator) checkForDataLeakage(content string) error {
	// Patterns for potential PII or sensitive data
	sensitivePatterns := []string{
		`\b\d{3}-\d{2}-\d{4}\b`,                    // SSN pattern
		`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, // Credit card pattern
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email pattern
		`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,   // IP address pattern
		`\b(?:password|pwd|pass)[\s:=]+\S+\b`,       // Password pattern
	}

	for _, pattern := range sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return fmt.Errorf("potential sensitive data detected")
		}
	}

	return nil
}

// checkForBiasedLanguage checks for potentially biased or discriminatory language
func (v *ResponseValidator) checkForBiasedLanguage(content string) error {
	biasPatterns := []string{
		`(?i)\b(?:he|she)\s+(?:should|must|needs?\s+to)\b`, // Gender-specific language
		`(?i)\b(?:old|young)\s+(?:people|person|individuals?)\b`, // Age-based language
		`(?i)\b(?:normal|typical)\s+(?:people|person|individuals?)\b`, // Normative language
		`(?i)\b(?:obviously|clearly|everyone\s+knows)\b`, // Assumptive language
	}

	for _, pattern := range biasPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return fmt.Errorf("potentially biased language detected")
		}
	}

	return nil
}

// ValidateRequest validates a consultation request
func (v *ResponseValidator) ValidateRequest(request *ConsultationRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if strings.TrimSpace(request.Query) == "" {
		return fmt.Errorf("query is empty")
	}

	if len(request.Query) < 10 {
		return fmt.Errorf("query is too short (minimum 10 characters)")
	}

	if len(request.Query) > 10000 {
		return fmt.Errorf("query is too long (maximum 10000 characters)")
	}

	if request.UserID.IsZero() {
		return fmt.Errorf("user ID is required")
	}

	// Check for valid consultation type
	validTypes := []models.ConsultationType{
		models.ConsultationTypePolicy,
		models.ConsultationTypeStrategy,
		models.ConsultationTypeOperations,
		models.ConsultationTypeTechnology,
		models.ConsultationTypeGeneral,
	}

	validType := false
	for _, validT := range validTypes {
		if request.Type == validT {
			validType = true
			break
		}
	}

	if !validType {
		return fmt.Errorf("invalid consultation type: %s", request.Type)
	}

	// Check for harmful content in query
	for _, pattern := range v.harmfulPatterns {
		if matched, _ := regexp.MatchString(pattern, request.Query); matched {
			return fmt.Errorf("potentially harmful content detected in query")
		}
	}

	return nil
}