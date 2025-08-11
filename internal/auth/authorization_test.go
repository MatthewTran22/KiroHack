package auth

import (
	"context"
	"testing"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func createTestUser(role models.UserRole, securityClearance models.SecurityClearance) *models.User {
	return &models.User{
		ID:                primitive.NewObjectID(),
		Email:             "test@example.com",
		Name:              "Test User",
		Department:        "Test Department",
		Role:              role,
		SecurityClearance: securityClearance,
		IsActive:          true,
		Permissions:       []models.Permission{},
	}
}

func TestAuthorizationService_Authorize(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	tests := []struct {
		name       string
		user       *models.User
		resource   Resource
		action     Action
		authorized bool
	}{
		{
			name:       "admin can read documents",
			user:       createTestUser(models.UserRoleAdmin, models.SecurityClearanceTopSecret),
			resource:   ResourceDocuments,
			action:     ActionRead,
			authorized: true,
		},
		{
			name:       "admin can delete users",
			user:       createTestUser(models.UserRoleAdmin, models.SecurityClearanceTopSecret),
			resource:   ResourceUsers,
			action:     ActionDelete,
			authorized: true,
		},
		{
			name:       "analyst can read documents",
			user:       createTestUser(models.UserRoleAnalyst, models.SecurityClearanceSecret),
			resource:   ResourceDocuments,
			action:     ActionRead,
			authorized: true,
		},
		{
			name:       "analyst can write documents",
			user:       createTestUser(models.UserRoleAnalyst, models.SecurityClearanceSecret),
			resource:   ResourceDocuments,
			action:     ActionWrite,
			authorized: true,
		},
		{
			name:       "analyst cannot delete users",
			user:       createTestUser(models.UserRoleAnalyst, models.SecurityClearanceSecret),
			resource:   ResourceUsers,
			action:     ActionDelete,
			authorized: false,
		},
		{
			name:       "viewer can read documents",
			user:       createTestUser(models.UserRoleViewer, models.SecurityClearancePublic),
			resource:   ResourceDocuments,
			action:     ActionRead,
			authorized: true,
		},
		{
			name:       "viewer cannot write documents",
			user:       createTestUser(models.UserRoleViewer, models.SecurityClearancePublic),
			resource:   ResourceDocuments,
			action:     ActionWrite,
			authorized: false,
		},
		{
			name:       "manager can delete documents",
			user:       createTestUser(models.UserRoleManager, models.SecurityClearanceConfidential),
			resource:   ResourceDocuments,
			action:     ActionDelete,
			authorized: true,
		},
		{
			name:       "consultant can write consultations",
			user:       createTestUser(models.UserRoleConsultant, models.SecurityClearanceInternal),
			resource:   ResourceConsultations,
			action:     ActionWrite,
			authorized: true,
		},
		{
			name:       "consultant cannot delete consultations",
			user:       createTestUser(models.UserRoleConsultant, models.SecurityClearanceInternal),
			resource:   ResourceConsultations,
			action:     ActionDelete,
			authorized: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authorized, err := authzService.Authorize(ctx, test.user, test.resource, test.action)
			if err != nil {
				t.Fatalf("Authorization check failed: %v", err)
			}

			if authorized != test.authorized {
				t.Errorf("Expected authorization %v, got %v", test.authorized, authorized)
			}
		})
	}
}

func TestAuthorizationService_AuthorizeWithSecurityClearance(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	tests := []struct {
		name              string
		user              *models.User
		resource          Resource
		action            Action
		requiredClearance string
		authorized        bool
		expectError       bool
	}{
		{
			name:              "top secret user can access secret document",
			user:              createTestUser(models.UserRoleAnalyst, models.SecurityClearanceTopSecret),
			resource:          ResourceDocuments,
			action:            ActionRead,
			requiredClearance: "SECRET",
			authorized:        true,
			expectError:       false,
		},
		{
			name:              "secret user cannot access top secret document",
			user:              createTestUser(models.UserRoleAnalyst, models.SecurityClearanceSecret),
			resource:          ResourceDocuments,
			action:            ActionRead,
			requiredClearance: "TOP_SECRET",
			authorized:        false,
			expectError:       true,
		},
		{
			name:              "public user can access public document",
			user:              createTestUser(models.UserRoleViewer, models.SecurityClearancePublic),
			resource:          ResourceDocuments,
			action:            ActionRead,
			requiredClearance: "PUBLIC",
			authorized:        true,
			expectError:       false,
		},
		{
			name:              "internal user cannot access confidential document",
			user:              createTestUser(models.UserRoleAnalyst, models.SecurityClearanceInternal),
			resource:          ResourceDocuments,
			action:            ActionRead,
			requiredClearance: "CONFIDENTIAL",
			authorized:        false,
			expectError:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authorized, err := authzService.AuthorizeWithSecurityClearance(
				ctx, test.user, test.resource, test.action, test.requiredClearance,
			)

			if test.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !test.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if authorized != test.authorized {
				t.Errorf("Expected authorization %v, got %v", test.authorized, authorized)
			}
		})
	}
}

func TestAuthorizationService_GetUserPermissions(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	// Test admin user
	adminUser := createTestUser(models.UserRoleAdmin, models.SecurityClearanceTopSecret)
	permissions, err := authzService.GetUserPermissions(ctx, adminUser)
	if err != nil {
		t.Fatalf("Failed to get admin permissions: %v", err)
	}

	// Admin should have permissions for all resources
	expectedResources := []Resource{
		ResourceDocuments, ResourceConsultations, ResourceUsers,
		ResourceKnowledge, ResourceAudit, ResourceSystem,
	}

	if len(permissions) != len(expectedResources) {
		t.Errorf("Expected %d permissions, got %d", len(expectedResources), len(permissions))
	}

	// Test viewer user
	viewerUser := createTestUser(models.UserRoleViewer, models.SecurityClearancePublic)
	permissions, err = authzService.GetUserPermissions(ctx, viewerUser)
	if err != nil {
		t.Fatalf("Failed to get viewer permissions: %v", err)
	}

	// Viewer should have limited permissions
	if len(permissions) == 0 {
		t.Error("Viewer should have some permissions")
	}

	// Check that viewer only has read permissions
	for _, perm := range permissions {
		for _, action := range perm.Actions {
			if action != ActionRead {
				t.Errorf("Viewer should only have read permissions, found %s", action)
			}
		}
	}
}

func TestAuthorizationService_UserSpecificPermissions(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	// Create a viewer user with additional permissions
	user := createTestUser(models.UserRoleViewer, models.SecurityClearancePublic)
	user.Permissions = []models.Permission{
		{
			Resource: "documents",
			Actions:  []string{"write"}, // Additional write permission
		},
	}

	// Test that user has both role-based and user-specific permissions
	authorized, err := authzService.Authorize(ctx, user, ResourceDocuments, ActionRead)
	if err != nil {
		t.Fatalf("Authorization check failed: %v", err)
	}
	if !authorized {
		t.Error("User should have read permission from role")
	}

	authorized, err = authzService.Authorize(ctx, user, ResourceDocuments, ActionWrite)
	if err != nil {
		t.Fatalf("Authorization check failed: %v", err)
	}
	if !authorized {
		t.Error("User should have write permission from user-specific permissions")
	}
}

func TestAuthorizationService_InactiveUser(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	user := createTestUser(models.UserRoleAdmin, models.SecurityClearanceTopSecret)
	user.IsActive = false

	authorized, err := authzService.Authorize(ctx, user, ResourceDocuments, ActionRead)
	if err == nil {
		t.Error("Expected error for inactive user")
	}
	if authorized {
		t.Error("Inactive user should not be authorized")
	}
}

func TestAuthorizationService_NilUser(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	authorized, err := authzService.Authorize(ctx, nil, ResourceDocuments, ActionRead)
	if err == nil {
		t.Error("Expected error for nil user")
	}
	if authorized {
		t.Error("Nil user should not be authorized")
	}
}

func TestAuthorizationService_CanUserAccessDocument(t *testing.T) {
	authzService := NewAuthorizationService()
	ctx := context.Background()

	tests := []struct {
		name           string
		user           *models.User
		classification string
		action         Action
		canAccess      bool
		expectError    bool
	}{
		{
			name:           "top secret user can access public document",
			user:           createTestUser(models.UserRoleAnalyst, models.SecurityClearanceTopSecret),
			classification: "PUBLIC",
			action:         ActionRead,
			canAccess:      true,
			expectError:    false,
		},
		{
			name:           "public user cannot access secret document",
			user:           createTestUser(models.UserRoleViewer, models.SecurityClearancePublic),
			classification: "SECRET",
			action:         ActionRead,
			canAccess:      false,
			expectError:    true,
		},
		{
			name:           "secret user can access confidential document",
			user:           createTestUser(models.UserRoleAnalyst, models.SecurityClearanceSecret),
			classification: "CONFIDENTIAL",
			action:         ActionRead,
			canAccess:      true,
			expectError:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			canAccess, err := authzService.CanUserAccessDocument(
				ctx, test.user, test.classification, test.action,
			)

			if test.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !test.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if canAccess != test.canAccess {
				t.Errorf("Expected access %v, got %v", test.canAccess, canAccess)
			}
		})
	}
}

func TestAuthorizationService_AddRemoveRolePermission(t *testing.T) {
	authzService := NewAuthorizationService()

	// Add a new permission to viewer role
	authzService.AddRolePermission(models.UserRoleViewer, ResourceUsers, []Action{ActionRead})

	// Check that the permission was added
	permissions := authzService.GetRolePermissions(models.UserRoleViewer)
	found := false
	for _, perm := range permissions {
		if perm.Resource == ResourceUsers {
			for _, action := range perm.Actions {
				if action == ActionRead {
					found = true
					break
				}
			}
		}
	}

	if !found {
		t.Error("Permission was not added to role")
	}

	// Remove the permission
	authzService.RemoveRolePermission(models.UserRoleViewer, ResourceUsers, ActionRead)

	// Check that the permission was removed
	permissions = authzService.GetRolePermissions(models.UserRoleViewer)
	found = false
	for _, perm := range permissions {
		if perm.Resource == ResourceUsers {
			for _, action := range perm.Actions {
				if action == ActionRead {
					found = true
					break
				}
			}
		}
	}

	if found {
		t.Error("Permission was not removed from role")
	}
}
