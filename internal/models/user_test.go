package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr error
	}{
		{
			name: "valid user",
			user: User{
				Email:             "test@government.gov",
				Name:              "Test User",
				Department:        "IT",
				Role:              UserRoleAnalyst,
				SecurityClearance: SecurityClearanceConfidential,
			},
			wantErr: nil,
		},
		{
			name: "missing email",
			user: User{
				Name:              "Test User",
				Department:        "IT",
				Role:              UserRoleAnalyst,
				SecurityClearance: SecurityClearanceConfidential,
			},
			wantErr: ErrUserEmailRequired,
		},
		{
			name: "missing name",
			user: User{
				Email:             "test@government.gov",
				Department:        "IT",
				Role:              UserRoleAnalyst,
				SecurityClearance: SecurityClearanceConfidential,
			},
			wantErr: ErrUserNameRequired,
		},
		{
			name: "missing department",
			user: User{
				Email:             "test@government.gov",
				Name:              "Test User",
				Role:              UserRoleAnalyst,
				SecurityClearance: SecurityClearanceConfidential,
			},
			wantErr: ErrUserDepartmentRequired,
		},
		{
			name: "missing role",
			user: User{
				Email:             "test@government.gov",
				Name:              "Test User",
				Department:        "IT",
				SecurityClearance: SecurityClearanceConfidential,
			},
			wantErr: ErrUserRoleRequired,
		},
		{
			name: "missing security clearance",
			user: User{
				Email:      "test@government.gov",
				Name:       "Test User",
				Department: "IT",
				Role:       UserRoleAnalyst,
			},
			wantErr: ErrUserSecurityClearanceRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if err != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_HasPermission(t *testing.T) {
	user := User{
		Permissions: []Permission{
			{
				Resource: "documents",
				Actions:  []string{"read", "write"},
			},
			{
				Resource: "consultations",
				Actions:  []string{"read"},
			},
		},
	}

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{
			name:     "has permission - documents read",
			resource: "documents",
			action:   "read",
			want:     true,
		},
		{
			name:     "has permission - documents write",
			resource: "documents",
			action:   "write",
			want:     true,
		},
		{
			name:     "has permission - consultations read",
			resource: "consultations",
			action:   "read",
			want:     true,
		},
		{
			name:     "no permission - consultations write",
			resource: "consultations",
			action:   "write",
			want:     false,
		},
		{
			name:     "no permission - users read",
			resource: "users",
			action:   "read",
			want:     false,
		},
		{
			name:     "no permission - documents delete",
			resource: "documents",
			action:   "delete",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := user.HasPermission(tt.resource, tt.action); got != tt.want {
				t.Errorf("User.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_CanAccessClassification(t *testing.T) {
	tests := []struct {
		name           string
		clearance      SecurityClearance
		classification string
		want           bool
	}{
		{
			name:           "top secret can access all",
			clearance:      SecurityClearanceTopSecret,
			classification: "TOP_SECRET",
			want:           true,
		},
		{
			name:           "top secret can access secret",
			clearance:      SecurityClearanceTopSecret,
			classification: "SECRET",
			want:           true,
		},
		{
			name:           "secret cannot access top secret",
			clearance:      SecurityClearanceSecret,
			classification: "TOP_SECRET",
			want:           false,
		},
		{
			name:           "secret can access secret",
			clearance:      SecurityClearanceSecret,
			classification: "SECRET",
			want:           true,
		},
		{
			name:           "confidential can access confidential",
			clearance:      SecurityClearanceConfidential,
			classification: "CONFIDENTIAL",
			want:           true,
		},
		{
			name:           "confidential cannot access secret",
			clearance:      SecurityClearanceConfidential,
			classification: "SECRET",
			want:           false,
		},
		{
			name:           "internal can access public",
			clearance:      SecurityClearanceInternal,
			classification: "PUBLIC",
			want:           true,
		},
		{
			name:           "internal can access internal",
			clearance:      SecurityClearanceInternal,
			classification: "INTERNAL",
			want:           true,
		},
		{
			name:           "internal cannot access confidential",
			clearance:      SecurityClearanceInternal,
			classification: "CONFIDENTIAL",
			want:           false,
		},
		{
			name:           "public can only access public",
			clearance:      SecurityClearancePublic,
			classification: "PUBLIC",
			want:           true,
		},
		{
			name:           "public cannot access internal",
			clearance:      SecurityClearancePublic,
			classification: "INTERNAL",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{SecurityClearance: tt.clearance}
			if got := user.CanAccessClassification(tt.classification); got != tt.want {
				t.Errorf("User.CanAccessClassification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		role UserRole
		want bool
	}{
		{
			name: "admin role",
			role: UserRoleAdmin,
			want: true,
		},
		{
			name: "analyst role",
			role: UserRoleAnalyst,
			want: false,
		},
		{
			name: "manager role",
			role: UserRoleManager,
			want: false,
		},
		{
			name: "viewer role",
			role: UserRoleViewer,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Role: tt.role}
			if got := user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := User{
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	beforeUpdate := time.Now()
	user.UpdateLastLogin()
	afterUpdate := time.Now()

	if user.LastLogin == nil {
		t.Error("LastLogin should not be nil after UpdateLastLogin()")
	}

	if user.LastLogin.Before(beforeUpdate) || user.LastLogin.After(afterUpdate) {
		t.Errorf("LastLogin should be between %v and %v, got %v", beforeUpdate, afterUpdate, user.LastLogin)
	}

	if user.UpdatedAt.Before(beforeUpdate) || user.UpdatedAt.After(afterUpdate) {
		t.Errorf("UpdatedAt should be between %v and %v, got %v", beforeUpdate, afterUpdate, user.UpdatedAt)
	}
}

func TestUser_BSONSerialization(t *testing.T) {
	now := time.Now()
	user := User{
		ID:         primitive.NewObjectID(),
		Email:      "test@government.gov",
		Name:       "Test User",
		Department: "IT Department",
		Role:       UserRoleAnalyst,
		Permissions: []Permission{
			{
				Resource: "documents",
				Actions:  []string{"read", "write"},
			},
			{
				Resource: "consultations",
				Actions:  []string{"read"},
			},
		},
		SecurityClearance: SecurityClearanceConfidential,
		PasswordHash:      "hashed_password",
		LastLogin:         &now,
		CreatedAt:         now,
		UpdatedAt:         now,
		IsActive:          true,
		MFAEnabled:        true,
		MFASecret:         "mfa_secret",
	}

	// Test BSON marshaling
	data, err := bson.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user to BSON: %v", err)
	}

	// Test BSON unmarshaling
	var unmarshaled User
	err = bson.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal user from BSON: %v", err)
	}

	// Verify key fields
	if unmarshaled.Email != user.Email {
		t.Errorf("Email mismatch: got %v, want %v", unmarshaled.Email, user.Email)
	}
	if unmarshaled.Name != user.Name {
		t.Errorf("Name mismatch: got %v, want %v", unmarshaled.Name, user.Name)
	}
	if unmarshaled.Department != user.Department {
		t.Errorf("Department mismatch: got %v, want %v", unmarshaled.Department, user.Department)
	}
	if unmarshaled.Role != user.Role {
		t.Errorf("Role mismatch: got %v, want %v", unmarshaled.Role, user.Role)
	}
	if unmarshaled.SecurityClearance != user.SecurityClearance {
		t.Errorf("SecurityClearance mismatch: got %v, want %v", unmarshaled.SecurityClearance, user.SecurityClearance)
	}
	if len(unmarshaled.Permissions) != len(user.Permissions) {
		t.Errorf("Permissions length mismatch: got %v, want %v", len(unmarshaled.Permissions), len(user.Permissions))
	}
	if unmarshaled.IsActive != user.IsActive {
		t.Errorf("IsActive mismatch: got %v, want %v", unmarshaled.IsActive, user.IsActive)
	}
	if unmarshaled.MFAEnabled != user.MFAEnabled {
		t.Errorf("MFAEnabled mismatch: got %v, want %v", unmarshaled.MFAEnabled, user.MFAEnabled)
	}

	// Verify that sensitive fields are properly handled
	if unmarshaled.PasswordHash != user.PasswordHash {
		t.Errorf("PasswordHash mismatch: got %v, want %v", unmarshaled.PasswordHash, user.PasswordHash)
	}
	if unmarshaled.MFASecret != user.MFASecret {
		t.Errorf("MFASecret mismatch: got %v, want %v", unmarshaled.MFASecret, user.MFASecret)
	}
}
