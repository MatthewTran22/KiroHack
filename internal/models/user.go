package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleAdmin      UserRole = "admin"
	UserRoleAnalyst    UserRole = "analyst"
	UserRoleManager    UserRole = "manager"
	UserRoleViewer     UserRole = "viewer"
	UserRoleConsultant UserRole = "consultant"
)

// SecurityClearance represents the security clearance level of a user
type SecurityClearance string

const (
	SecurityClearancePublic       SecurityClearance = "public"
	SecurityClearanceInternal     SecurityClearance = "internal"
	SecurityClearanceConfidential SecurityClearance = "confidential"
	SecurityClearanceSecret       SecurityClearance = "secret"
	SecurityClearanceTopSecret    SecurityClearance = "top_secret"
)

// Permission represents a specific permission that can be granted to a user
type Permission struct {
	Resource string   `json:"resource" bson:"resource"` // e.g., "documents", "consultations", "users"
	Actions  []string `json:"actions" bson:"actions"`   // e.g., ["read", "write", "delete"]
}

// User represents a user in the system
type User struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email             string             `json:"email" bson:"email"`
	Name              string             `json:"name" bson:"name"`
	Department        string             `json:"department" bson:"department"`
	Role              UserRole           `json:"role" bson:"role"`
	Permissions       []Permission       `json:"permissions" bson:"permissions"`
	SecurityClearance SecurityClearance  `json:"security_clearance" bson:"security_clearance"`
	PasswordHash      string             `json:"-" bson:"password_hash"` // Hidden from JSON
	LastLogin         *time.Time         `json:"last_login,omitempty" bson:"last_login,omitempty"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
	IsActive          bool               `json:"is_active" bson:"is_active"`
	MFAEnabled        bool               `json:"mfa_enabled" bson:"mfa_enabled"`
	MFASecret         string             `json:"-" bson:"mfa_secret"` // Hidden from JSON
}

// Validate validates the user model
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrUserEmailRequired
	}
	if u.Name == "" {
		return ErrUserNameRequired
	}
	if u.Department == "" {
		return ErrUserDepartmentRequired
	}
	if u.Role == "" {
		return ErrUserRoleRequired
	}
	if u.SecurityClearance == "" {
		return ErrUserSecurityClearanceRequired
	}
	return nil
}

// HasPermission checks if the user has a specific permission for a resource and action
func (u *User) HasPermission(resource, action string) bool {
	for _, perm := range u.Permissions {
		if perm.Resource == resource {
			for _, a := range perm.Actions {
				if a == action {
					return true
				}
			}
		}
	}
	return false
}

// CanAccessClassification checks if the user can access documents with a specific classification
func (u *User) CanAccessClassification(classification string) bool {
	switch u.SecurityClearance {
	case SecurityClearanceTopSecret:
		return true // Can access all classifications
	case SecurityClearanceSecret:
		return classification != "TOP_SECRET"
	case SecurityClearanceConfidential:
		return classification != "TOP_SECRET" && classification != "SECRET"
	case SecurityClearanceInternal:
		return classification == "PUBLIC" || classification == "INTERNAL"
	case SecurityClearancePublic:
		return classification == "PUBLIC"
	default:
		return false
	}
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// UpdateLastLogin updates the user's last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLogin = &now
	u.UpdatedAt = now
}
