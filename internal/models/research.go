package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ResearchStatus represents the status of a research operation
type ResearchStatus string

const (
	ResearchStatusPending    ResearchStatus = "pending"
	ResearchStatusProcessing ResearchStatus = "processing"
	ResearchStatusCompleted  ResearchStatus = "completed"
	ResearchStatusFailed     ResearchStatus = "failed"
)

// ResearchSourceType represents the type of research source
type ResearchSourceType string

const (
	ResearchSourceTypeNews       ResearchSourceType = "news"
	ResearchSourceTypeAcademic   ResearchSourceType = "academic"
	ResearchSourceTypeGovernment ResearchSourceType = "government"
	ResearchSourceTypeIndustry   ResearchSourceType = "industry"
	ResearchSourceTypeAPI        ResearchSourceType = "api"
)

// PolicySuggestionPriority represents the priority of a policy suggestion
type PolicySuggestionPriority string

const (
	PolicyPriorityLow      PolicySuggestionPriority = "low"
	PolicyPriorityMedium   PolicySuggestionPriority = "medium"
	PolicyPriorityHigh     PolicySuggestionPriority = "high"
	PolicyPriorityCritical PolicySuggestionPriority = "critical"
)

// CurrentEvent represents a current event relevant to policy analysis
type CurrentEvent struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description" bson:"description"`
	Source      string             `json:"source" bson:"source"`
	URL         string             `json:"url" bson:"url"`
	PublishedAt time.Time          `json:"published_at" bson:"published_at"`
	Relevance   float64            `json:"relevance" bson:"relevance"`
	Category    string             `json:"category" bson:"category"`
	Tags        []string           `json:"tags" bson:"tags"`
	Content     string             `json:"content" bson:"content"`
	Author      string             `json:"author" bson:"author"`
	Language    string             `json:"language" bson:"language"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// PolicyImpact represents the impact of current events on policy
type PolicyImpact struct {
	Area         string    `json:"area" bson:"area"`
	Impact       string    `json:"impact" bson:"impact"`
	Severity     string    `json:"severity" bson:"severity"` // "low", "medium", "high", "critical"
	Timeframe    string    `json:"timeframe" bson:"timeframe"`
	Stakeholders []string  `json:"stakeholders" bson:"stakeholders"`
	Mitigation   []string  `json:"mitigation" bson:"mitigation"`
	Confidence   float64   `json:"confidence" bson:"confidence"`
	Evidence     []string  `json:"evidence" bson:"evidence"`
}

// ResearchSource represents a source used in research
type ResearchSource struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Type        ResearchSourceType `json:"type" bson:"type"`
	Title       string             `json:"title" bson:"title"`
	URL         string             `json:"url" bson:"url"`
	Author      string             `json:"author" bson:"author"`
	PublishedAt time.Time          `json:"published_at" bson:"published_at"`
	Credibility float64            `json:"credibility" bson:"credibility"`
	Relevance   float64            `json:"relevance" bson:"relevance"`
	Content     string             `json:"content" bson:"content"`
	Summary     string             `json:"summary" bson:"summary"`
	Keywords    []string           `json:"keywords" bson:"keywords"`
	Language    string             `json:"language" bson:"language"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// ValidationResult represents the result of source validation
type ValidationResult struct {
	IsValid         bool      `json:"is_valid" bson:"is_valid"`
	CredibilityScore float64   `json:"credibility_score" bson:"credibility_score"`
	Issues          []string  `json:"issues" bson:"issues"`
	Recommendations []string  `json:"recommendations" bson:"recommendations"`
	ValidatedAt     time.Time `json:"validated_at" bson:"validated_at"`
	ValidatedBy     string    `json:"validated_by" bson:"validated_by"`
}

// ResearchResult represents the result of a research operation
type ResearchResult struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DocumentID      primitive.ObjectID `json:"document_id" bson:"document_id"`
	ResearchQuery   string             `json:"research_query" bson:"research_query"`
	CurrentEvents   []CurrentEvent     `json:"current_events" bson:"current_events"`
	PolicyImpacts   []PolicyImpact     `json:"policy_impacts" bson:"policy_impacts"`
	Sources         []ResearchSource   `json:"sources" bson:"sources"`
	Confidence      float64            `json:"confidence" bson:"confidence"`
	Status          ResearchStatus     `json:"status" bson:"status"`
	GeneratedAt     time.Time          `json:"generated_at" bson:"generated_at"`
	ProcessingTime  time.Duration      `json:"processing_time" bson:"processing_time"`
	ErrorMessage    *string            `json:"error_message,omitempty" bson:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata" bson:"metadata"`
}

// ImplementationPlan represents a plan for implementing a policy suggestion
type ImplementationPlan struct {
	Steps         []string  `json:"steps" bson:"steps"`
	Timeline      string    `json:"timeline" bson:"timeline"`
	Resources     []string  `json:"resources" bson:"resources"`
	Budget        *string   `json:"budget,omitempty" bson:"budget,omitempty"`
	Stakeholders  []string  `json:"stakeholders" bson:"stakeholders"`
	Dependencies  []string  `json:"dependencies" bson:"dependencies"`
	Milestones    []string  `json:"milestones" bson:"milestones"`
	RiskFactors   []string  `json:"risk_factors" bson:"risk_factors"`
	SuccessMetrics []string `json:"success_metrics" bson:"success_metrics"`
}

// PolicyRiskFactor represents a risk factor for a policy suggestion
type PolicyRiskFactor struct {
	Description string  `json:"description" bson:"description"`
	Probability float64 `json:"probability" bson:"probability"`
	Impact      string  `json:"impact" bson:"impact"`
	Mitigation  string  `json:"mitigation" bson:"mitigation"`
	Category    string  `json:"category" bson:"category"`
}

// MitigationStep represents a step to mitigate policy risks
type MitigationStep struct {
	Description string    `json:"description" bson:"description"`
	Priority    Priority  `json:"priority" bson:"priority"`
	Timeline    string    `json:"timeline" bson:"timeline"`
	Owner       string    `json:"owner" bson:"owner"`
	Resources   []string  `json:"resources" bson:"resources"`
	Dependencies []string `json:"dependencies" bson:"dependencies"`
}

// MonitoringMetric represents a metric for monitoring policy implementation
type MonitoringMetric struct {
	Name        string  `json:"name" bson:"name"`
	Description string  `json:"description" bson:"description"`
	Target      string  `json:"target" bson:"target"`
	Frequency   string  `json:"frequency" bson:"frequency"`
	Owner       string  `json:"owner" bson:"owner"`
	Threshold   *string `json:"threshold,omitempty" bson:"threshold,omitempty"`
}

// PolicyRiskAssessment represents a comprehensive risk assessment for a policy
type PolicyRiskAssessment struct {
	OverallRisk     string              `json:"overall_risk" bson:"overall_risk"`
	RiskFactors     []PolicyRiskFactor  `json:"risk_factors" bson:"risk_factors"`
	MitigationSteps []MitigationStep    `json:"mitigation_steps" bson:"mitigation_steps"`
	MonitoringPlan  []MonitoringMetric  `json:"monitoring_plan" bson:"monitoring_plan"`
	AssessedAt      time.Time           `json:"assessed_at" bson:"assessed_at"`
	AssessedBy      string              `json:"assessed_by" bson:"assessed_by"`
	Confidence      float64             `json:"confidence" bson:"confidence"`
}

// PolicySuggestion represents a policy suggestion generated from research
type PolicySuggestion struct {
	ID              primitive.ObjectID        `json:"id" bson:"_id,omitempty"`
	Title           string                    `json:"title" bson:"title"`
	Description     string                    `json:"description" bson:"description"`
	Rationale       string                    `json:"rationale" bson:"rationale"`
	CurrentContext  []CurrentEvent            `json:"current_context" bson:"current_context"`
	Implementation  ImplementationPlan        `json:"implementation" bson:"implementation"`
	RiskAssessment  PolicyRiskAssessment      `json:"risk_assessment" bson:"risk_assessment"`
	Priority        PolicySuggestionPriority  `json:"priority" bson:"priority"`
	Category        DocumentCategory          `json:"category" bson:"category"`
	Tags            []string                  `json:"tags" bson:"tags"`
	Sources         []ResearchSource          `json:"sources" bson:"sources"`
	Confidence      float64                   `json:"confidence" bson:"confidence"`
	CreatedAt       time.Time                 `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at" bson:"updated_at"`
	CreatedBy       primitive.ObjectID        `json:"created_by" bson:"created_by"`
	Status          string                    `json:"status" bson:"status"` // "draft", "review", "approved", "rejected"
	ReviewNotes     *string                   `json:"review_notes,omitempty" bson:"review_notes,omitempty"`
	ApprovedBy      *primitive.ObjectID       `json:"approved_by,omitempty" bson:"approved_by,omitempty"`
	ApprovedAt      *time.Time                `json:"approved_at,omitempty" bson:"approved_at,omitempty"`
}

// Validate validates the research result model
func (rr *ResearchResult) Validate() error {
	if rr.DocumentID.IsZero() {
		return ErrResearchDocumentIDRequired
	}
	if rr.ResearchQuery == "" {
		return ErrResearchQueryRequired
	}
	return nil
}

// IsCompleted returns true if the research is completed
func (rr *ResearchResult) IsCompleted() bool {
	return rr.Status == ResearchStatusCompleted
}

// HasSources returns true if the research has sources
func (rr *ResearchResult) HasSources() bool {
	return len(rr.Sources) > 0
}

// Validate validates the policy suggestion model
func (ps *PolicySuggestion) Validate() error {
	if ps.Title == "" {
		return ErrPolicySuggestionTitleRequired
	}
	if ps.Description == "" {
		return ErrPolicySuggestionDescriptionRequired
	}
	if ps.CreatedBy.IsZero() {
		return ErrPolicySuggestionCreatedByRequired
	}
	return nil
}

// IsApproved returns true if the policy suggestion is approved
func (ps *PolicySuggestion) IsApproved() bool {
	return ps.Status == "approved"
}

// GetAgeInDays returns the age of the policy suggestion in days
func (ps *PolicySuggestion) GetAgeInDays() int {
	return int(time.Since(ps.CreatedAt).Hours() / 24)
}

// Validate validates the current event model
func (ce *CurrentEvent) Validate() error {
	if ce.Title == "" {
		return ErrCurrentEventTitleRequired
	}
	if ce.Source == "" {
		return ErrCurrentEventSourceRequired
	}
	if ce.URL == "" {
		return ErrCurrentEventURLRequired
	}
	return nil
}

// IsRecent returns true if the event was published within the last 30 days
func (ce *CurrentEvent) IsRecent() bool {
	return time.Since(ce.PublishedAt).Hours() < 24*30
}

// Validate validates the research source model
func (rs *ResearchSource) Validate() error {
	if rs.Title == "" {
		return ErrResearchSourceTitleRequired
	}
	if rs.URL == "" {
		return ErrResearchSourceURLRequired
	}
	return nil
}

// IsCredible returns true if the source has high credibility (>= 0.7)
func (rs *ResearchSource) IsCredible() bool {
	return rs.Credibility >= 0.7
}

// IsRelevant returns true if the source has high relevance (>= 0.6)
func (rs *ResearchSource) IsRelevant() bool {
	return rs.Relevance >= 0.6
}