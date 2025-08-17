package consultation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// parseConsultationResponse parses the Gemini API response into a structured consultation response
func (s *Service) parseConsultationResponse(geminiResponse *GeminiResponse, context *ContextData, consultationType models.ConsultationType) (*models.ConsultationResponse, error) {
	if len(geminiResponse.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	candidate := geminiResponse.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in Gemini response")
	}

	responseText := candidate.Content.Parts[0].Text

	// Parse the structured response
	analysis := s.parseAnalysis(responseText)
	recommendations := s.parseRecommendations(responseText, consultationType)
	riskAssessment := s.parseRiskAssessment(responseText)
	nextSteps := s.parseNextSteps(responseText)
	
	// Calculate overall confidence score
	confidenceScore := s.calculateConfidenceScore(recommendations, context)

	// Build document references from context
	sources := s.buildDocumentReferences(context)

	response := &models.ConsultationResponse{
		Recommendations: recommendations,
		Analysis:        analysis,
		Sources:         sources,
		ConfidenceScore: confidenceScore,
		RiskAssessment:  riskAssessment,
		NextSteps:       nextSteps,
		GeneratedAt:     time.Now(),
		ProcessingTime:  time.Duration(geminiResponse.UsageMetadata.TotalTokenCount) * time.Millisecond, // Rough estimate
	}

	return response, nil
}

// parseAnalysis extracts analysis information from the response text
func (s *Service) parseAnalysis(responseText string) models.Analysis {
	analysis := models.Analysis{
		Summary:         s.extractSection(responseText, "Executive Summary", "Summary"),
		KeyFindings:     s.extractListItems(responseText, "Key Findings", "Findings"),
		Assumptions:     s.extractListItems(responseText, "Assumptions"),
		Limitations:     s.extractListItems(responseText, "Limitations"),
		MethodologyUsed: s.extractSection(responseText, "Methodology", "Method"),
		DataSourcesUsed: s.extractListItems(responseText, "Data Sources", "Sources"),
	}

	// If summary is empty, try to extract from other sections
	if analysis.Summary == "" {
		analysis.Summary = s.extractSection(responseText, "Assessment", "Analysis")
	}

	return analysis
}

// parseRecommendations extracts recommendations from the response text
func (s *Service) parseRecommendations(responseText string, consultationType models.ConsultationType) []models.Recommendation {
	var recommendations []models.Recommendation

	// Extract recommendations section
	recommendationsText := s.extractSection(responseText, "Recommendations", "Recommended")
	if recommendationsText == "" {
		// Fallback to looking for numbered recommendations in full text
		recommendationsText = responseText
	}

	// Parse recommendations using bullet points and numbered items
	lines := strings.Split(recommendationsText, "\n")
	var currentRec strings.Builder
	var recTitle string
	recNumber := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for different recommendation patterns
		var isNewRec bool
		var newTitle string

		// Pattern 1: * **High Confidence (90%):** **Phase 1: Title** Description
		if matched, _ := regexp.MatchString(`^\*\s*\*\*.*?\*\*.*?\*\*.*?\*\*`, line); matched {
			isNewRec = true
			// Extract title from the pattern
			re := regexp.MustCompile(`\*\*([^*]+)\*\*.*?\*\*([^*]+)\*\*`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				newTitle = strings.TrimSpace(matches[2])
				// Add the rest as description
				remaining := re.ReplaceAllString(line, "")
				remaining = strings.TrimSpace(strings.TrimPrefix(remaining, "*"))
				if remaining != "" {
					currentRec.WriteString(remaining)
				}
			} else {
				newTitle = "Recommendation"
				currentRec.WriteString(strings.TrimPrefix(line, "*"))
			}
		} else if matched, _ := regexp.MatchString(`^\*\s*\*\*.*?\*\*`, line); matched {
			// Pattern 2: * **Title:** Description
			isNewRec = true
			re := regexp.MustCompile(`^\*\s*\*\*([^*]+)\*\*:?\s*(.*)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				newTitle = strings.TrimSpace(matches[1])
				if len(matches) > 2 && matches[2] != "" {
					currentRec.WriteString(strings.TrimSpace(matches[2]))
				}
			}
		} else if matched, _ := regexp.MatchString(`^\*`, line); matched {
			// Pattern 3: * Simple bullet point
			isNewRec = true
			content := strings.TrimPrefix(line, "*")
			content = strings.TrimSpace(content)
			
			// Try to extract title from first part
			parts := strings.SplitN(content, ":", 2)
			if len(parts) == 2 {
				newTitle = strings.TrimSpace(parts[0])
				currentRec.WriteString(strings.TrimSpace(parts[1]))
			} else {
				// Use first few words as title
				words := strings.Fields(content)
				if len(words) > 3 {
					newTitle = strings.Join(words[:3], " ")
					currentRec.WriteString(strings.Join(words[3:], " "))
				} else {
					newTitle = content
				}
			}
		} else if matched, _ := regexp.MatchString(`^\d+\.`, line); matched {
			// Pattern 4: Numbered items
			isNewRec = true
			parts := strings.SplitN(line, ".", 2)
			if len(parts) > 1 {
				newTitle = strings.TrimSpace(parts[1])
				// If title is too long, split it
				if len(newTitle) > 100 {
					titleParts := strings.SplitN(newTitle, " ", 8)
					if len(titleParts) > 4 {
						newTitle = strings.Join(titleParts[:4], " ")
						currentRec.WriteString(strings.Join(titleParts[4:], " "))
					}
				}
			}
		}

		if isNewRec {
			// Save previous recommendation if exists
			if currentRec.Len() > 0 || recTitle != "" {
				s.addRecommendation(&recommendations, recTitle, currentRec.String(), recNumber)
				currentRec.Reset()
			}
			
			// Start new recommendation
			recNumber++
			recTitle = newTitle
			if recTitle == "" {
				recTitle = fmt.Sprintf("Recommendation %d", recNumber)
			}
		} else if recNumber > 0 {
			// Add to current recommendation description
			if currentRec.Len() > 0 {
				currentRec.WriteString(" ")
			}
			currentRec.WriteString(line)
		}
	}

	// Add the last recommendation
	if currentRec.Len() > 0 {
		s.addRecommendation(&recommendations, recTitle, currentRec.String(), recNumber)
	}

	// If no recommendations found, create a general one from the response
	if len(recommendations) == 0 {
		recommendation := models.Recommendation{
			ID:              primitive.NewObjectID(),
			Title:           fmt.Sprintf("General %s Guidance", consultationType),
			Description:     s.extractMainContent(responseText),
			Priority:        models.PriorityMedium,
			Impact:          models.ImpactAssessment{OverallImpact: "Medium"},
			Implementation:  models.ImplementationGuidance{Steps: []string{"Review the provided analysis", "Develop detailed implementation plan"}},
			Risks:           []models.Risk{},
			Benefits:        []models.Benefit{},
			Timeline:        models.Timeline{EstimatedDuration: "To be determined"},
			ConfidenceScore: 0.7,
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// addRecommendation helper function to add a recommendation to the list
func (s *Service) addRecommendation(recommendations *[]models.Recommendation, title, description string, number int) {
	if description == "" {
		return
	}

	if title == "" {
		title = fmt.Sprintf("Recommendation %d", number)
	}

	recommendation := models.Recommendation{
		ID:              primitive.NewObjectID(),
		Title:           title,
		Description:     description,
		Priority:        s.inferPriority(description),
		Impact:          s.parseImpactAssessment(description),
		Implementation:  s.parseImplementationGuidance(description),
		Risks:           s.parseRisks(description),
		Benefits:        s.parseBenefits(description),
		Timeline:        s.parseTimeline(description),
		ConfidenceScore: s.inferConfidenceScore(description),
	}
	*recommendations = append(*recommendations, recommendation)
}

// parseRiskAssessment extracts risk assessment from the response text
func (s *Service) parseRiskAssessment(responseText string) models.RiskAnalysis {
	riskSection := s.extractSection(responseText, "Risk Assessment", "Risk Management", "Risks")
	
	riskAnalysis := models.RiskAnalysis{
		OverallRiskLevel: s.inferRiskLevel(riskSection),
		RiskFactors:      s.parseRisks(riskSection),
		MitigationPlan:   s.extractSection(riskSection, "Mitigation", "Risk Mitigation"),
	}

	if riskAnalysis.MitigationPlan == "" {
		riskAnalysis.MitigationPlan = "Implement recommended actions with careful monitoring and regular review."
	}

	return riskAnalysis
}

// parseNextSteps extracts next steps from the response text
func (s *Service) parseNextSteps(responseText string) []models.ActionItem {
	var actionItems []models.ActionItem

	nextStepsText := s.extractSection(responseText, "Next Steps", "Action Items", "Implementation Plan")
	if nextStepsText == "" {
		return actionItems
	}

	// Extract action items using patterns
	actionPatterns := []string{
		`(?i)(?:^|\n)\s*(?:step\s*)?(\d+)[\.\)]\s*([^\n]+)`,
		`(?i)(?:^|\n)\s*[-•]\s*([^\n]+)`,
	}

	for _, pattern := range actionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(nextStepsText, -1)
		
		if len(matches) > 0 {
			for _, match := range matches {
				if len(match) >= 2 {
					description := strings.TrimSpace(match[len(match)-1])
					if description != "" {
						actionItem := models.ActionItem{
							Description: description,
							Priority:    s.inferActionPriority(description),
							Status:      "pending",
						}
						actionItems = append(actionItems, actionItem)
					}
				}
			}
			break
		}
	}

	return actionItems
}

// Helper functions for parsing specific elements

func (s *Service) extractSection(text string, sectionNames ...string) string {
	for _, sectionName := range sectionNames {
		// Try different header patterns
		patterns := []string{
			// Markdown bold headers with numbers: **1. Executive Summary**
			fmt.Sprintf(`(?i)\*\*\d+\.\s*%s\*\*`, regexp.QuoteMeta(sectionName)),
			// Markdown bold headers: **Executive Summary**
			fmt.Sprintf(`(?i)\*\*%s\*\*`, regexp.QuoteMeta(sectionName)),
			// Regular headers with colon: Executive Summary:
			fmt.Sprintf(`(?i)%s\s*:`, regexp.QuoteMeta(sectionName)),
			// Numbered headers: 1. Executive Summary
			fmt.Sprintf(`(?i)\d+\.\s*%s`, regexp.QuoteMeta(sectionName)),
		}
		
		for _, pattern := range patterns {
			headerRe := regexp.MustCompile(pattern)
			headerMatch := headerRe.FindStringIndex(text)
			
			if headerMatch != nil {
				// Find content after header
				startPos := headerMatch[1]
				
				// Skip any remaining asterisks or colons
				for startPos < len(text) && (text[startPos] == '*' || text[startPos] == ':' || text[startPos] == ' ' || text[startPos] == '\n') {
					startPos++
				}
				
				// Find next section header or end of text
				nextSectionPatterns := []string{
					`\*\*\d+\.\s*[A-Z]`, // Next numbered markdown header
					`\*\*[A-Z]`,         // Next markdown header
					`\n\d+\.\s*[A-Z]`,   // Next numbered header
					`\n[A-Z][^:\n]*:`,   // Next colon header
				}
				
				var endPos int = len(text)
				for _, nextPattern := range nextSectionPatterns {
					nextSectionRe := regexp.MustCompile(nextPattern)
					nextMatch := nextSectionRe.FindStringIndex(text[startPos:])
					if nextMatch != nil {
						candidateEndPos := startPos + nextMatch[0]
						if candidateEndPos < endPos {
							endPos = candidateEndPos
						}
					}
				}
				
				content := strings.TrimSpace(text[startPos:endPos])
				if content != "" {
					return content
				}
			}
		}
	}
	return ""
}

func (s *Service) extractListItems(text string, sectionNames ...string) []string {
	sectionText := s.extractSection(text, sectionNames...)
	if sectionText == "" {
		return []string{}
	}

	var items []string
	
	// Extract list items using various patterns
	patterns := []string{
		`(?i)(?:^|\n)\s*[-•]\s*([^\n]+)`,
		`(?i)(?:^|\n)\s*\d+[\.\)]\s*([^\n]+)`,
		`(?i)(?:^|\n)\s*[a-z][\.\)]\s*([^\n]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sectionText, -1)
		
		if len(matches) > 0 {
			for _, match := range matches {
				if len(match) > 1 {
					item := strings.TrimSpace(match[1])
					if item != "" {
						items = append(items, item)
					}
				}
			}
			break
		}
	}

	// If no list items found, split by sentences
	if len(items) == 0 {
		sentences := strings.Split(sectionText, ".")
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if len(sentence) > 10 { // Minimum length for meaningful content
				items = append(items, sentence)
			}
		}
	}

	return items
}

func (s *Service) extractMainContent(text string) string {
	// Remove common headers and footers
	cleanText := regexp.MustCompile(`(?i)^.*?(?:summary|analysis|assessment):\s*`).ReplaceAllString(text, "")
	cleanText = regexp.MustCompile(`(?i)\n\s*(?:next steps|conclusion|recommendations):\s*.*$`).ReplaceAllString(cleanText, "")
	
	// Limit length
	if len(cleanText) > 1000 {
		cleanText = cleanText[:1000] + "..."
	}
	
	return strings.TrimSpace(cleanText)
}

func (s *Service) inferPriority(description string) models.Priority {
	lowerDesc := strings.ToLower(description)
	
	if strings.Contains(lowerDesc, "critical") || strings.Contains(lowerDesc, "urgent") || strings.Contains(lowerDesc, "immediate") {
		return models.PriorityCritical
	}
	if strings.Contains(lowerDesc, "high") || strings.Contains(lowerDesc, "important") || strings.Contains(lowerDesc, "essential") {
		return models.PriorityHigh
	}
	if strings.Contains(lowerDesc, "low") || strings.Contains(lowerDesc, "optional") || strings.Contains(lowerDesc, "consider") {
		return models.PriorityLow
	}
	return models.PriorityMedium
}

func (s *Service) inferActionPriority(description string) models.Priority {
	lowerDesc := strings.ToLower(description)
	
	if strings.Contains(lowerDesc, "immediately") || strings.Contains(lowerDesc, "first") || strings.Contains(lowerDesc, "urgent") {
		return models.PriorityHigh
	}
	if strings.Contains(lowerDesc, "then") || strings.Contains(lowerDesc, "next") || strings.Contains(lowerDesc, "second") {
		return models.PriorityMedium
	}
	if strings.Contains(lowerDesc, "finally") || strings.Contains(lowerDesc, "later") || strings.Contains(lowerDesc, "consider") {
		return models.PriorityLow
	}
	return models.PriorityMedium
}

func (s *Service) inferRiskLevel(riskText string) string {
	lowerText := strings.ToLower(riskText)
	
	if strings.Contains(lowerText, "high risk") || strings.Contains(lowerText, "significant risk") {
		return "high"
	}
	if strings.Contains(lowerText, "low risk") || strings.Contains(lowerText, "minimal risk") {
		return "low"
	}
	return "medium"
}

func (s *Service) inferConfidenceScore(description string) float64 {
	lowerDesc := strings.ToLower(description)
	
	// Look for confidence indicators
	if strings.Contains(lowerDesc, "certain") || strings.Contains(lowerDesc, "proven") || strings.Contains(lowerDesc, "established") {
		return 0.9
	}
	if strings.Contains(lowerDesc, "likely") || strings.Contains(lowerDesc, "probable") || strings.Contains(lowerDesc, "recommended") {
		return 0.8
	}
	if strings.Contains(lowerDesc, "possible") || strings.Contains(lowerDesc, "consider") || strings.Contains(lowerDesc, "suggest") {
		return 0.6
	}
	if strings.Contains(lowerDesc, "uncertain") || strings.Contains(lowerDesc, "unclear") || strings.Contains(lowerDesc, "may") {
		return 0.4
	}
	
	return 0.7 // Default confidence
}

func (s *Service) parseImpactAssessment(description string) models.ImpactAssessment {
	lowerDesc := strings.ToLower(description)
	
	var overallImpact string
	if strings.Contains(lowerDesc, "significant impact") || strings.Contains(lowerDesc, "major impact") {
		overallImpact = "High"
	} else if strings.Contains(lowerDesc, "minimal impact") || strings.Contains(lowerDesc, "small impact") {
		overallImpact = "Low"
	} else {
		overallImpact = "Medium"
	}
	
	return models.ImpactAssessment{
		OverallImpact: overallImpact,
	}
}

func (s *Service) parseImplementationGuidance(description string) models.ImplementationGuidance {
	steps := []string{}
	
	// Extract steps if mentioned
	stepPattern := regexp.MustCompile(`(?i)(?:step|phase)\s*\d*[:\.]?\s*([^\n\.]+)`)
	matches := stepPattern.FindAllStringSubmatch(description, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			steps = append(steps, strings.TrimSpace(match[1]))
		}
	}
	
	if len(steps) == 0 {
		steps = []string{"Review recommendation details", "Develop implementation plan", "Execute with monitoring"}
	}
	
	return models.ImplementationGuidance{
		Steps: steps,
	}
}

func (s *Service) parseRisks(text string) []models.Risk {
	var risks []models.Risk
	
	riskPattern := regexp.MustCompile(`(?i)(?:risk|concern|challenge)[:\s]*([^\n\.]+)`)
	matches := riskPattern.FindAllStringSubmatch(text, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			risk := models.Risk{
				Description: strings.TrimSpace(match[1]),
				Probability: 0.5, // Default probability
				Impact:      "medium",
				Mitigation:  "Monitor and address as needed",
			}
			risks = append(risks, risk)
		}
	}
	
	return risks
}

func (s *Service) parseBenefits(text string) []models.Benefit {
	var benefits []models.Benefit
	
	benefitPattern := regexp.MustCompile(`(?i)(?:benefit|advantage|improvement)[:\s]*([^\n\.]+)`)
	matches := benefitPattern.FindAllStringSubmatch(text, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			benefit := models.Benefit{
				Description:     strings.TrimSpace(match[1]),
				ConfidenceLevel: 0.7,
			}
			benefits = append(benefits, benefit)
		}
	}
	
	return benefits
}

func (s *Service) parseTimeline(text string) models.Timeline {
	// Extract timeline information
	timePattern := regexp.MustCompile(`(?i)(?:timeline|duration|timeframe)[:\s]*([^\n\.]+)`)
	matches := timePattern.FindStringSubmatch(text)
	
	duration := "To be determined"
	if len(matches) > 1 {
		duration = strings.TrimSpace(matches[1])
	}
	
	return models.Timeline{
		EstimatedDuration: duration,
		Phases:           []string{"Planning", "Implementation", "Review"},
	}
}

func (s *Service) calculateConfidenceScore(recommendations []models.Recommendation, context *ContextData) float64 {
	if len(recommendations) == 0 {
		return 0.5
	}
	
	totalConfidence := 0.0
	for _, rec := range recommendations {
		totalConfidence += rec.ConfidenceScore
	}
	
	avgConfidence := totalConfidence / float64(len(recommendations))
	
	// Adjust based on context quality
	contextBonus := 0.0
	if context.TotalSources > 0 {
		contextBonus = 0.1 * float64(context.TotalSources) / 10.0 // Max 10% bonus
		if contextBonus > 0.1 {
			contextBonus = 0.1
		}
	}
	
	finalConfidence := avgConfidence + contextBonus
	if finalConfidence > 1.0 {
		finalConfidence = 1.0
	}
	
	return finalConfidence
}

func (s *Service) buildDocumentReferences(context *ContextData) []models.DocumentReference {
	var references []models.DocumentReference
	
	for _, doc := range context.Documents {
		if doc.Document != nil {
			ref := models.DocumentReference{
				DocumentID: doc.Document.ID,
				Title:      doc.Document.Name,
				Relevance:  doc.Score,
			}
			references = append(references, ref)
		}
	}
	
	return references
}