package document

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"ai-government-consultant/internal/models"
)

// extractText extracts text content from a document based on its type
func (s *Service) extractText(doc *models.Document) (string, error) {
	ext := strings.ToLower(filepath.Ext(doc.Name))

	switch ext {
	case ".txt":
		return s.extractTextFromTXT(doc.Content)
	case ".pdf":
		return s.extractTextFromPDF(doc.Content)
	case ".doc", ".docx":
		return s.extractTextFromDOC(doc.Content)
	default:
		return "", fmt.Errorf("unsupported file format: %s", ext)
	}
}

// extractTextFromTXT extracts text from plain text files
func (s *Service) extractTextFromTXT(content string) (string, error) {
	// For plain text files, just clean up the content
	cleaned := strings.TrimSpace(content)

	// Ensure the content is valid UTF-8
	if !utf8.ValidString(cleaned) {
		cleaned = strings.ToValidUTF8(cleaned, "�")
	}

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	return cleaned, nil
}

// extractTextFromPDF extracts text from PDF files
// Note: This is a placeholder implementation. In production, you would use a library like unidoc or pdfcpu
func (s *Service) extractTextFromPDF(content string) (string, error) {
	// Placeholder implementation - in reality, you'd use a PDF parsing library
	// For now, we'll assume the content is already text (for testing purposes)

	// Basic PDF text extraction simulation
	if strings.Contains(content, "%PDF") {
		// This is actual PDF binary content - in production, use proper PDF library
		return "PDF content extraction not implemented - use proper PDF library", nil
	}

	// If it's already text content (for testing), return as is
	return s.extractTextFromTXT(content)
}

// extractTextFromDOC extracts text from DOC/DOCX files
// Note: This is a placeholder implementation. In production, you would use a library like unioffice
func (s *Service) extractTextFromDOC(content string) (string, error) {
	// Placeholder implementation - in reality, you'd use a DOC/DOCX parsing library
	// For now, we'll assume the content is already text (for testing purposes)

	// Basic DOC/DOCX text extraction simulation
	if strings.Contains(content, "PK") || strings.Contains(content, "Microsoft") {
		// This might be actual DOC/DOCX binary content - in production, use proper library
		return "DOC/DOCX content extraction not implemented - use proper library", nil
	}

	// If it's already text content (for testing), return as is
	return s.extractTextFromTXT(content)
}

// extractMetadata extracts metadata from document content
func (s *Service) extractMetadata(doc *models.Document) (models.DocumentMetadata, error) {
	metadata := models.DocumentMetadata{
		Tags:         []string{},
		Language:     "en", // Default to English
		CustomFields: make(map[string]interface{}),
	}

	content := strings.ToLower(doc.Content)

	// Extract title from first line or filename
	lines := strings.Split(doc.Content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		title := strings.TrimSpace(lines[0])
		if len(title) > 0 && len(title) < 200 { // Reasonable title length
			metadata.Title = &title
		}
	}

	// If no title found, use filename without extension
	if metadata.Title == nil {
		name := strings.TrimSuffix(doc.Name, filepath.Ext(doc.Name))
		metadata.Title = &name
	}

	// Categorize document based on content keywords
	metadata.Category = s.categorizeDocument(content)

	// Extract tags based on content
	metadata.Tags = s.extractTags(content)

	// Set creation date to current time if not provided
	now := time.Now()
	metadata.CreatedDate = &now
	metadata.LastModified = &now

	// Extract language (basic implementation)
	metadata.Language = s.detectLanguage(content)

	return metadata, nil
}

// categorizeDocument categorizes a document based on its content
func (s *Service) categorizeDocument(content string) models.DocumentCategory {
	content = strings.ToLower(content)

	// Policy keywords
	policyKeywords := []string{"policy", "regulation", "compliance", "governance", "law", "legal", "statute", "ordinance"}
	for _, keyword := range policyKeywords {
		if strings.Contains(content, keyword) {
			return models.DocumentCategoryPolicy
		}
	}

	// Strategy keywords
	strategyKeywords := []string{"strategy", "strategic", "planning", "roadmap", "vision", "mission", "objectives", "goals"}
	for _, keyword := range strategyKeywords {
		if strings.Contains(content, keyword) {
			return models.DocumentCategoryStrategy
		}
	}

	// Operations keywords
	operationsKeywords := []string{"operations", "process", "procedure", "workflow", "efficiency", "optimization", "performance"}
	for _, keyword := range operationsKeywords {
		if strings.Contains(content, keyword) {
			return models.DocumentCategoryOperations
		}
	}

	// Technology keywords
	technologyKeywords := []string{"technology", "technical", "system", "software", "hardware", "digital", "it", "cyber"}
	for _, keyword := range technologyKeywords {
		if strings.Contains(content, keyword) {
			return models.DocumentCategoryTechnology
		}
	}

	return models.DocumentCategoryGeneral
}

// extractTags extracts relevant tags from document content
func (s *Service) extractTags(content string) []string {
	tags := []string{}
	content = strings.ToLower(content)

	// Common government/business tags
	tagKeywords := map[string]string{
		"budget":         "budget",
		"finance":        "finance",
		"security":       "security",
		"privacy":        "privacy",
		"audit":          "audit",
		"compliance":     "compliance",
		"risk":           "risk",
		"management":     "management",
		"analysis":       "analysis",
		"report":         "report",
		"assessment":     "assessment",
		"evaluation":     "evaluation",
		"review":         "review",
		"proposal":       "proposal",
		"recommendation": "recommendation",
		"implementation": "implementation",
		"training":       "training",
		"personnel":      "personnel",
		"resource":       "resource",
		"project":        "project",
	}

	for keyword, tag := range tagKeywords {
		if strings.Contains(content, keyword) {
			tags = append(tags, tag)
		}
	}

	// Remove duplicates
	uniqueTags := make(map[string]bool)
	var result []string
	for _, tag := range tags {
		if !uniqueTags[tag] {
			uniqueTags[tag] = true
			result = append(result, tag)
		}
	}

	return result
}

// detectLanguage detects the language of the document content
func (s *Service) detectLanguage(content string) string {
	// Basic language detection - in production, use a proper language detection library
	content = strings.ToLower(content)

	// Check for common English words
	englishWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	englishCount := 0

	words := strings.Fields(content)
	if len(words) == 0 {
		return "en" // Default to English
	}

	for _, word := range words {
		for _, englishWord := range englishWords {
			if word == englishWord {
				englishCount++
				break
			}
		}
	}

	// If more than 5% of words are common English words, assume English
	if float64(englishCount)/float64(len(words)) > 0.05 {
		return "en"
	}

	return "unknown"
}

// extractEntities extracts entities from document text
func (s *Service) extractEntities(content string) ([]models.Entity, error) {
	entities := []models.Entity{}

	// Extract email addresses
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emailMatches := emailRegex.FindAllStringIndex(content, -1)
	for _, match := range emailMatches {
		entities = append(entities, models.Entity{
			Type:       "email",
			Value:      content[match[0]:match[1]],
			Confidence: 0.95,
			StartPos:   match[0],
			EndPos:     match[1],
		})
	}

	// Extract phone numbers (basic US format)
	phoneRegex := regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	phoneMatches := phoneRegex.FindAllStringIndex(content, -1)
	for _, match := range phoneMatches {
		entities = append(entities, models.Entity{
			Type:       "phone",
			Value:      content[match[0]:match[1]],
			Confidence: 0.85,
			StartPos:   match[0],
			EndPos:     match[1],
		})
	}

	// Extract dates (basic format: MM/DD/YYYY or MM-DD-YYYY)
	dateRegex := regexp.MustCompile(`\b\d{1,2}[/-]\d{1,2}[/-]\d{4}\b`)
	dateMatches := dateRegex.FindAllStringIndex(content, -1)
	for _, match := range dateMatches {
		entities = append(entities, models.Entity{
			Type:       "date",
			Value:      content[match[0]:match[1]],
			Confidence: 0.80,
			StartPos:   match[0],
			EndPos:     match[1],
		})
	}

	// Extract monetary amounts
	moneyRegex := regexp.MustCompile(`\$\d{1,3}(?:,\d{3})*(?:\.\d{2})?`)
	moneyMatches := moneyRegex.FindAllStringIndex(content, -1)
	for _, match := range moneyMatches {
		entities = append(entities, models.Entity{
			Type:       "money",
			Value:      content[match[0]:match[1]],
			Confidence: 0.90,
			StartPos:   match[0],
			EndPos:     match[1],
		})
	}

	return entities, nil
}

// mergeMetadata merges extracted metadata with existing metadata
func (s *Service) mergeMetadata(existing *models.DocumentMetadata, extracted models.DocumentMetadata) {
	// Only update fields that are not already set
	if existing.Title == nil && extracted.Title != nil {
		existing.Title = extracted.Title
	}

	if existing.Category == "" {
		existing.Category = extracted.Category
	}

	if len(existing.Tags) == 0 {
		existing.Tags = extracted.Tags
	} else {
		// Merge tags
		tagMap := make(map[string]bool)
		for _, tag := range existing.Tags {
			tagMap[tag] = true
		}
		for _, tag := range extracted.Tags {
			if !tagMap[tag] {
				existing.Tags = append(existing.Tags, tag)
			}
		}
	}

	if existing.Language == "" {
		existing.Language = extracted.Language
	}

	if existing.CreatedDate == nil && extracted.CreatedDate != nil {
		existing.CreatedDate = extracted.CreatedDate
	}

	if existing.LastModified == nil && extracted.LastModified != nil {
		existing.LastModified = extracted.LastModified
	}

	// Merge custom fields
	if existing.CustomFields == nil {
		existing.CustomFields = make(map[string]interface{})
	}
	for key, value := range extracted.CustomFields {
		if _, exists := existing.CustomFields[key]; !exists {
			existing.CustomFields[key] = value
		}
	}
}
