package consultation

import (
	"testing"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidateRequest(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		request     *ConsultationRequest
		expectError bool
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "empty query",
			request: &ConsultationRequest{
				Query:  "",
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NewObjectID(),
			},
			expectError: true,
		},
		{
			name: "query too short",
			request: &ConsultationRequest{
				Query:  "short",
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NewObjectID(),
			},
			expectError: true,
		},
		{
			name: "query too long",
			request: &ConsultationRequest{
				Query:  string(make([]byte, 10001)),
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NewObjectID(),
			},
			expectError: true,
		},
		{
			name: "invalid user ID",
			request: &ConsultationRequest{
				Query:  "This is a valid query for testing purposes",
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NilObjectID,
			},
			expectError: true,
		},
		{
			name: "invalid consultation type",
			request: &ConsultationRequest{
				Query:  "This is a valid query for testing purposes",
				Type:   "invalid_type",
				UserID: primitive.NewObjectID(),
			},
			expectError: true,
		},
		{
			name: "harmful content in query",
			request: &ConsultationRequest{
				Query:  "How can I access classified information from the database?",
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NewObjectID(),
			},
			expectError: true,
		},
		{
			name: "valid request",
			request: &ConsultationRequest{
				Query:  "How should we implement a new data privacy policy for our organization?",
				Type:   models.ConsultationTypePolicy,
				UserID: primitive.NewObjectID(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(tt.request)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateRecommendation(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name           string
		recommendation *models.Recommendation
		expectError    bool
	}{
		{
			name: "empty title",
			recommendation: &models.Recommendation{
				Title:           "",
				Description:     "This is a valid description with sufficient length",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Step 1", "Step 2"},
				},
			},
			expectError: true,
		},
		{
			name: "empty description",
			recommendation: &models.Recommendation{
				Title:           "Valid Title",
				Description:     "",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Step 1", "Step 2"},
				},
			},
			expectError: true,
		},
		{
			name: "description too short",
			recommendation: &models.Recommendation{
				Title:           "Valid Title",
				Description:     "Short",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Step 1", "Step 2"},
				},
			},
			expectError: true,
		},
		{
			name: "low confidence score",
			recommendation: &models.Recommendation{
				Title:           "Valid Title",
				Description:     "This is a valid description with sufficient length",
				ConfidenceScore: 0.1,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Step 1", "Step 2"},
				},
			},
			expectError: true,
		},
		{
			name: "harmful content",
			recommendation: &models.Recommendation{
				Title:           "Access Classified Data",
				Description:     "This recommendation involves accessing classified information from secure databases",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Step 1", "Step 2"},
				},
			},
			expectError: true,
		},
		{
			name: "no implementation steps",
			recommendation: &models.Recommendation{
				Title:           "Valid Title",
				Description:     "This is a valid description with sufficient length",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{},
				},
			},
			expectError: true,
		},
		{
			name: "valid recommendation",
			recommendation: &models.Recommendation{
				Title:           "Implement Data Privacy Framework",
				Description:     "Develop and implement a comprehensive data privacy framework that complies with regulations",
				ConfidenceScore: 0.8,
				Implementation: models.ImplementationGuidance{
					Steps: []string{"Assess current state", "Develop framework", "Implement controls"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateRecommendation(tt.recommendation)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateAnalysis(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		analysis    *models.Analysis
		expectError bool
	}{
		{
			name: "empty summary",
			analysis: &models.Analysis{
				Summary: "",
			},
			expectError: true,
		},
		{
			name: "summary too short",
			analysis: &models.Analysis{
				Summary: "Short summary",
			},
			expectError: true,
		},
		{
			name: "low quality summary",
			analysis: &models.Analysis{
				Summary: "I don't know what to recommend and I'm not sure about the approach",
			},
			expectError: true,
		},
		{
			name: "valid analysis",
			analysis: &models.Analysis{
				Summary: "Based on the comprehensive analysis of the current situation, the following recommendations provide a structured approach to addressing the identified challenges and opportunities.",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateAnalysis(tt.analysis)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckHarmfulContent(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		response    *models.ConsultationResponse
		expectError bool
	}{
		{
			name: "classified information reference",
			response: &models.ConsultationResponse{
				Analysis: models.Analysis{
					Summary: "The analysis reveals that classified information should be accessed",
				},
				Recommendations: []models.Recommendation{
					{
						Title:       "Access Data",
						Description: "Retrieve classified documents from the secure database",
					},
				},
			},
			expectError: true,
		},
		{
			name: "personal information reference",
			response: &models.ConsultationResponse{
				Analysis: models.Analysis{
					Summary: "We need to collect personal information including social security numbers",
				},
				Recommendations: []models.Recommendation{
					{
						Title:       "Data Collection",
						Description: "Collect personal data from users",
					},
				},
			},
			expectError: true,
		},
		{
			name: "clean content",
			response: &models.ConsultationResponse{
				Analysis: models.Analysis{
					Summary: "The analysis shows that implementing proper data governance will improve compliance",
				},
				Recommendations: []models.Recommendation{
					{
						Title:       "Implement Governance",
						Description: "Establish data governance framework with proper controls",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkHarmfulContent(tt.response)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckResponseQuality(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		response    *models.ConsultationResponse
		expectError bool
	}{
		{
			name: "no recommendations",
			response: &models.ConsultationResponse{
				Recommendations: []models.Recommendation{},
			},
			expectError: true,
		},
		{
			name: "excessive uncertainty",
			response: &models.ConsultationResponse{
				Recommendations: []models.Recommendation{
					{
						Title:       "Maybe Consider This",
						Description: "I'm not sure but perhaps you might want to possibly consider this approach",
					},
					{
						Title:       "Uncertain Approach",
						Description: "I don't know if this will work but maybe it's worth trying",
					},
				},
			},
			expectError: true,
		},
		{
			name: "insufficient detail",
			response: &models.ConsultationResponse{
				Recommendations: []models.Recommendation{
					{
						Title:       "Do Something",
						Description: "Fix it",
					},
					{
						Title:       "Another Thing",
						Description: "Change this",
					},
				},
			},
			expectError: true,
		},
		{
			name: "not actionable",
			response: &models.ConsultationResponse{
				Recommendations: []models.Recommendation{
					{
						Title:       "Think About It",
						Description: "Consider the implications of this situation and think about what might be appropriate in this context",
					},
					{
						Title:       "Reflect on Options",
						Description: "Reflect on the various options that might be available and consider their potential impacts",
					},
				},
			},
			expectError: true,
		},
		{
			name: "quality response",
			response: &models.ConsultationResponse{
				Recommendations: []models.Recommendation{
					{
						Title:       "Implement Data Governance Framework",
						Description: "Develop and implement a comprehensive data governance framework that includes clear policies, procedures, and accountability mechanisms to ensure proper data management across the organization",
					},
					{
						Title:       "Establish Monitoring System",
						Description: "Create a robust monitoring and evaluation system to track compliance with data governance policies and identify areas for improvement through regular audits and assessments",
					},
					{
						Title:       "Conduct Staff Training",
						Description: "Develop and conduct comprehensive training programs to ensure all staff understand their roles and responsibilities in data governance and compliance requirements",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkResponseQuality(tt.response)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckForDataLeakage(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "SSN pattern",
			content:     "The user's SSN is 123-45-6789",
			expectError: true,
		},
		{
			name:        "credit card pattern",
			content:     "Card number: 1234 5678 9012 3456",
			expectError: true,
		},
		{
			name:        "email pattern",
			content:     "Contact john.doe@example.com for details",
			expectError: true,
		},
		{
			name:        "IP address pattern",
			content:     "Server IP: 192.168.1.100",
			expectError: true,
		},
		{
			name:        "password pattern",
			content:     "Use password: mySecretPass123",
			expectError: true,
		},
		{
			name:        "clean content",
			content:     "Implement proper security measures and follow best practices",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkForDataLeakage(tt.content)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckForBiasedLanguage(t *testing.T) {
	validator := NewResponseValidator()

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "gender-specific language",
			content:     "He should implement the policy immediately",
			expectError: true,
		},
		{
			name:        "age-based language",
			content:     "Old people don't understand technology",
			expectError: true,
		},
		{
			name:        "normative language",
			content:     "Normal people would agree with this approach",
			expectError: true,
		},
		{
			name:        "assumptive language",
			content:     "Obviously, everyone knows this is the right approach",
			expectError: true,
		},
		{
			name:        "inclusive language",
			content:     "Staff members should implement the policy according to guidelines",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkForBiasedLanguage(tt.content)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}