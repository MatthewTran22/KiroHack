package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConsultationType represents the type of consultation
type ConsultationType string

const (
	ConsultationTypePolicy     ConsultationType = "policy"
	ConsultationTypeStrategy   ConsultationType = "strategy"
	ConsultationTypeOperations ConsultationType = "operations"
	ConsultationTypeTechnology ConsultationType = "technology"
	ConsultationTypeGeneral    ConsultationType = "general"
)

// SessionStatus represents the status of a consultation session
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// Priority represents the priority level of a recommendation
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// DocumentReference represents a reference to a source document
type DocumentReference struct {
	DocumentID primitive.ObjectID `json:"document_id" bson:"document_id"`
	Title      string             `json:"title" bson:"title"`
	Section    *string            `json:"section,omitempty" bson:"section,omitempty"`
	PageNumber *int               `json:"page_number,omitempty" bson:"page_number,omitempty"`
	Relevance  float64            `json:"relevance" bson:"relevance"`
}

// Risk represents a risk associated with a recommendation
type Risk struct {
	Description string  `json:"description" bson:"description"`
	Probability float64 `json:"probability" bson:"probability"` // 0.0 to 1.0
	Impact      string  `json:"impact" bson:"impact"`           // "low", "medium", "high", "critical"
	Mitigation  string  `json:"mitigation" bson:"mitigation"`
}

// Benefit represents a benefit of a recommendation
type Benefit struct {
	Description     string  `json:"description" bson:"description"`
	ExpectedValue   *string `json:"expected_value,omitempty" bson:"expected_value,omitempty"`
	TimeToRealize   *string `json:"time_to_realize,omitempty" bson:"time_to_realize,omitempty"`
	ConfidenceLevel float64 `json:"confidence_level" bson:"confidence_level"`
}

// Timeline represents the timeline for implementing a recommendation
type Timeline struct {
	EstimatedDuration string     `json:"estimated_duration" bson:"estimated_duration"`
	Phases            []string   `json:"phases" bson:"phases"`
	Milestones        []string   `json:"milestones" bson:"milestones"`
	Dependencies      []string   `json:"dependencies" bson:"dependencies"`
	StartDate         *time.Time `json:"start_date,omitempty" bson:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty" bson:"end_date,omitempty"`
}

// ImpactAssessment represents the impact assessment of a recommendation
type ImpactAssessment struct {
	Financial     *string `json:"financial,omitempty" bson:"financial,omitempty"`
	Operational   *string `json:"operational,omitempty" bson:"operational,omitempty"`
	Strategic     *string `json:"strategic,omitempty" bson:"strategic,omitempty"`
	Compliance    *string `json:"compliance,omitempty" bson:"compliance,omitempty"`
	OverallImpact string  `json:"overall_impact" bson:"overall_impact"`
}

// ImplementationGuidance provides guidance on how to implement a recommendation
type ImplementationGuidance struct {
	Steps          []string `json:"steps" bson:"steps"`
	Resources      []string `json:"resources" bson:"resources"`
	Prerequisites  []string `json:"prerequisites" bson:"prerequisites"`
	Considerations []string `json:"considerations" bson:"considerations"`
}

// ActionItem represents a specific action item from a consultation
type ActionItem struct {
	Description string     `json:"description" bson:"description"`
	Priority    Priority   `json:"priority" bson:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty" bson:"due_date,omitempty"`
	AssignedTo  *string    `json:"assigned_to,omitempty" bson:"assigned_to,omitempty"`
	Status      string     `json:"status" bson:"status"` // "pending", "in_progress", "completed"
}

// Recommendation represents a recommendation from the AI consultant
type Recommendation struct {
	ID              primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Title           string                 `json:"title" bson:"title"`
	Description     string                 `json:"description" bson:"description"`
	Priority        Priority               `json:"priority" bson:"priority"`
	Impact          ImpactAssessment       `json:"impact" bson:"impact"`
	Implementation  ImplementationGuidance `json:"implementation" bson:"implementation"`
	Risks           []Risk                 `json:"risks" bson:"risks"`
	Benefits        []Benefit              `json:"benefits" bson:"benefits"`
	Timeline        Timeline               `json:"timeline" bson:"timeline"`
	ConfidenceScore float64                `json:"confidence_score" bson:"confidence_score"`
}

// Analysis represents the analysis performed during a consultation
type Analysis struct {
	Summary         string   `json:"summary" bson:"summary"`
	KeyFindings     []string `json:"key_findings" bson:"key_findings"`
	Assumptions     []string `json:"assumptions" bson:"assumptions"`
	Limitations     []string `json:"limitations" bson:"limitations"`
	MethodologyUsed string   `json:"methodology_used" bson:"methodology_used"`
	DataSourcesUsed []string `json:"data_sources_used" bson:"data_sources_used"`
}

// RiskAnalysis represents the risk analysis for a consultation
type RiskAnalysis struct {
	OverallRiskLevel string `json:"overall_risk_level" bson:"overall_risk_level"`
	RiskFactors      []Risk `json:"risk_factors" bson:"risk_factors"`
	MitigationPlan   string `json:"mitigation_plan" bson:"mitigation_plan"`
}

// ConsultationContext provides context for a consultation session
type ConsultationContext struct {
	RelatedDocuments []primitive.ObjectID   `json:"related_documents" bson:"related_documents"`
	PreviousSessions []primitive.ObjectID   `json:"previous_sessions" bson:"previous_sessions"`
	UserContext      map[string]interface{} `json:"user_context" bson:"user_context"`
	SystemContext    map[string]interface{} `json:"system_context" bson:"system_context"`
}

// ConsultationResponse represents the response from an AI consultation
type ConsultationResponse struct {
	Recommendations []Recommendation    `json:"recommendations" bson:"recommendations"`
	Analysis        Analysis            `json:"analysis" bson:"analysis"`
	Sources         []DocumentReference `json:"sources" bson:"sources"`
	ConfidenceScore float64             `json:"confidence_score" bson:"confidence_score"`
	RiskAssessment  RiskAnalysis        `json:"risk_assessment" bson:"risk_assessment"`
	NextSteps       []ActionItem        `json:"next_steps" bson:"next_steps"`
	GeneratedAt     time.Time           `json:"generated_at" bson:"generated_at"`
	ProcessingTime  time.Duration       `json:"processing_time" bson:"processing_time"`
}

// ConversationTurn represents a single turn in a multi-turn conversation
type ConversationTurn struct {
	ID        primitive.ObjectID    `json:"id" bson:"_id,omitempty"`
	Query     string                `json:"query" bson:"query"`
	Response  *ConsultationResponse `json:"response,omitempty" bson:"response,omitempty"`
	Timestamp time.Time             `json:"timestamp" bson:"timestamp"`
	TurnIndex int                   `json:"turn_index" bson:"turn_index"`
}

// ConsultationSession represents a consultation session
type ConsultationSession struct {
	ID                primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID            primitive.ObjectID     `json:"user_id" bson:"user_id"`
	Type              ConsultationType       `json:"type" bson:"type"`
	Query             string                 `json:"query" bson:"query"`
	Response          *ConsultationResponse  `json:"response,omitempty" bson:"response,omitempty"`
	Context           ConsultationContext    `json:"context" bson:"context"`
	CreatedAt         time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" bson:"updated_at"`
	Status            SessionStatus          `json:"status" bson:"status"`
	Tags              []string               `json:"tags" bson:"tags"`
	Metadata          map[string]interface{} `json:"metadata" bson:"metadata"`
	ConversationTurns []ConversationTurn     `json:"conversation_turns,omitempty" bson:"conversation_turns,omitempty"`
	IsMultiTurn       bool                   `json:"is_multi_turn" bson:"is_multi_turn"`
}

// Validate validates the consultation session model
func (cs *ConsultationSession) Validate() error {
	if cs.UserID.IsZero() {
		return ErrConsultationUserIDRequired
	}
	if cs.Query == "" {
		return ErrConsultationQueryRequired
	}
	if cs.Type == "" {
		return ErrConsultationTypeRequired
	}
	return nil
}

// IsCompleted returns true if the consultation session is completed
func (cs *ConsultationSession) IsCompleted() bool {
	return cs.Status == SessionStatusCompleted
}

// HasResponse returns true if the consultation session has a response
func (cs *ConsultationSession) HasResponse() bool {
	return cs.Response != nil
}

// GetDuration returns the duration of the consultation session
func (cs *ConsultationSession) GetDuration() time.Duration {
	return cs.UpdatedAt.Sub(cs.CreatedAt)
}
