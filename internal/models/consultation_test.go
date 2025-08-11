package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestConsultationSession_Validate(t *testing.T) {
	userID := primitive.NewObjectID()

	tests := []struct {
		name    string
		session ConsultationSession
		wantErr error
	}{
		{
			name: "valid consultation session",
			session: ConsultationSession{
				UserID: userID,
				Type:   ConsultationTypePolicy,
				Query:  "What are the requirements for FISMA compliance?",
			},
			wantErr: nil,
		},
		{
			name: "missing user ID",
			session: ConsultationSession{
				Type:  ConsultationTypePolicy,
				Query: "What are the requirements for FISMA compliance?",
			},
			wantErr: ErrConsultationUserIDRequired,
		},
		{
			name: "missing query",
			session: ConsultationSession{
				UserID: userID,
				Type:   ConsultationTypePolicy,
			},
			wantErr: ErrConsultationQueryRequired,
		},
		{
			name: "missing type",
			session: ConsultationSession{
				UserID: userID,
				Query:  "What are the requirements for FISMA compliance?",
			},
			wantErr: ErrConsultationTypeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if err != tt.wantErr {
				t.Errorf("ConsultationSession.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConsultationSession_IsCompleted(t *testing.T) {
	tests := []struct {
		name   string
		status SessionStatus
		want   bool
	}{
		{
			name:   "completed status",
			status: SessionStatusCompleted,
			want:   true,
		},
		{
			name:   "active status",
			status: SessionStatusActive,
			want:   false,
		},
		{
			name:   "failed status",
			status: SessionStatusFailed,
			want:   false,
		},
		{
			name:   "cancelled status",
			status: SessionStatusCancelled,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := ConsultationSession{Status: tt.status}
			if got := session.IsCompleted(); got != tt.want {
				t.Errorf("ConsultationSession.IsCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsultationSession_HasResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *ConsultationResponse
		want     bool
	}{
		{
			name: "has response",
			response: &ConsultationResponse{
				ConfidenceScore: 0.85,
			},
			want: true,
		},
		{
			name:     "no response",
			response: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := ConsultationSession{Response: tt.response}
			if got := session.HasResponse(); got != tt.want {
				t.Errorf("ConsultationSession.HasResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsultationSession_GetDuration(t *testing.T) {
	createdAt := time.Now()
	updatedAt := createdAt.Add(5 * time.Minute)

	session := ConsultationSession{
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	duration := session.GetDuration()
	expected := 5 * time.Minute

	if duration != expected {
		t.Errorf("ConsultationSession.GetDuration() = %v, want %v", duration, expected)
	}
}

func TestConsultationSession_BSONSerialization(t *testing.T) {
	userID := primitive.NewObjectID()
	docID := primitive.NewObjectID()
	now := time.Now()

	session := ConsultationSession{
		ID:     primitive.NewObjectID(),
		UserID: userID,
		Type:   ConsultationTypePolicy,
		Query:  "What are the FISMA compliance requirements?",
		Response: &ConsultationResponse{
			Recommendations: []Recommendation{
				{
					ID:          primitive.NewObjectID(),
					Title:       "Implement FISMA Controls",
					Description: "Implement required FISMA security controls",
					Priority:    PriorityHigh,
					Impact: ImpactAssessment{
						Financial:     stringPtr("$100,000 - $500,000"),
						Operational:   stringPtr("Significant process changes required"),
						Strategic:     stringPtr("Critical for compliance"),
						Compliance:    stringPtr("Required for FISMA compliance"),
						OverallImpact: "High",
					},
					Implementation: ImplementationGuidance{
						Steps:          []string{"Assess current state", "Develop implementation plan", "Execute controls"},
						Resources:      []string{"Security team", "IT infrastructure", "Training materials"},
						Prerequisites:  []string{"Management approval", "Budget allocation"},
						Considerations: []string{"Timeline constraints", "Resource availability"},
					},
					Risks: []Risk{
						{
							Description: "Implementation delays",
							Probability: 0.3,
							Impact:      "medium",
							Mitigation:  "Regular progress reviews and contingency planning",
						},
					},
					Benefits: []Benefit{
						{
							Description:     "Enhanced security posture",
							ExpectedValue:   stringPtr("Reduced security incidents by 70%"),
							TimeToRealize:   stringPtr("6-12 months"),
							ConfidenceLevel: 0.8,
						},
					},
					Timeline: Timeline{
						EstimatedDuration: "12 months",
						Phases:            []string{"Assessment", "Planning", "Implementation", "Validation"},
						Milestones:        []string{"Controls identified", "Plan approved", "Implementation complete"},
						Dependencies:      []string{"Budget approval", "Resource allocation"},
					},
					ConfidenceScore: 0.9,
				},
			},
			Analysis: Analysis{
				Summary:         "FISMA compliance requires comprehensive security controls implementation",
				KeyFindings:     []string{"Current gaps in security controls", "Need for staff training"},
				Assumptions:     []string{"Management support available", "Budget will be approved"},
				Limitations:     []string{"Limited historical data", "Regulatory changes possible"},
				MethodologyUsed: "Risk-based assessment approach",
				DataSourcesUsed: []string{"NIST guidelines", "Previous assessments"},
			},
			Sources: []DocumentReference{
				{
					DocumentID: docID,
					Title:      "FISMA Implementation Guide",
					Section:    stringPtr("Section 3.2"),
					PageNumber: intPtr(45),
					Relevance:  0.95,
				},
			},
			ConfidenceScore: 0.85,
			RiskAssessment: RiskAnalysis{
				OverallRiskLevel: "Medium",
				RiskFactors: []Risk{
					{
						Description: "Compliance deadline pressure",
						Probability: 0.6,
						Impact:      "high",
						Mitigation:  "Phased implementation approach",
					},
				},
				MitigationPlan: "Implement controls in priority order with regular reviews",
			},
			NextSteps: []ActionItem{
				{
					Description: "Conduct security assessment",
					Priority:    PriorityHigh,
					DueDate:     timePtr(now.Add(30 * 24 * time.Hour)),
					AssignedTo:  stringPtr("Security Team"),
					Status:      "pending",
				},
			},
			GeneratedAt:    now,
			ProcessingTime: 2 * time.Second,
		},
		Context: ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{docID},
			PreviousSessions: []primitive.ObjectID{},
			UserContext: map[string]interface{}{
				"department": "IT Security",
				"role":       "Security Analyst",
			},
			SystemContext: map[string]interface{}{
				"version": "1.0",
				"model":   "gpt-4",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    SessionStatusCompleted,
		Tags:      []string{"fisma", "compliance", "security"},
		Metadata: map[string]interface{}{
			"priority": "high",
			"category": "compliance",
		},
	}

	// Test BSON marshaling
	data, err := bson.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal consultation session to BSON: %v", err)
	}

	// Test BSON unmarshaling
	var unmarshaled ConsultationSession
	err = bson.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal consultation session from BSON: %v", err)
	}

	// Verify key fields
	if unmarshaled.UserID != session.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, session.UserID)
	}
	if unmarshaled.Type != session.Type {
		t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, session.Type)
	}
	if unmarshaled.Query != session.Query {
		t.Errorf("Query mismatch: got %v, want %v", unmarshaled.Query, session.Query)
	}
	if unmarshaled.Status != session.Status {
		t.Errorf("Status mismatch: got %v, want %v", unmarshaled.Status, session.Status)
	}

	// Verify response
	if unmarshaled.Response == nil {
		t.Error("Response should not be nil")
	} else {
		if len(unmarshaled.Response.Recommendations) != len(session.Response.Recommendations) {
			t.Errorf("Recommendations length mismatch: got %v, want %v",
				len(unmarshaled.Response.Recommendations), len(session.Response.Recommendations))
		}
		if unmarshaled.Response.ConfidenceScore != session.Response.ConfidenceScore {
			t.Errorf("ConfidenceScore mismatch: got %v, want %v",
				unmarshaled.Response.ConfidenceScore, session.Response.ConfidenceScore)
		}
	}

	// Verify context
	if len(unmarshaled.Context.RelatedDocuments) != len(session.Context.RelatedDocuments) {
		t.Errorf("RelatedDocuments length mismatch: got %v, want %v",
			len(unmarshaled.Context.RelatedDocuments), len(session.Context.RelatedDocuments))
	}

	// Verify tags and metadata
	if len(unmarshaled.Tags) != len(session.Tags) {
		t.Errorf("Tags length mismatch: got %v, want %v", len(unmarshaled.Tags), len(session.Tags))
	}
	if len(unmarshaled.Metadata) != len(session.Metadata) {
		t.Errorf("Metadata length mismatch: got %v, want %v", len(unmarshaled.Metadata), len(session.Metadata))
	}
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
