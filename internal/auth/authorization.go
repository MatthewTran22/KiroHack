package auth

import (
	"context"
	"fmt"
	"strings"

	"ai-government-consultant/internal/models"
)

// Action represents an action that can be performed on a resource
type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionAdmin  Action = "admin"
)

// Resource represents a resource in the system
type Resource string

const (
	ResourceDocuments     Resource = "documents"
	ResourceConsultations Resource = "consultations"
	ResourceUsers         Resource = "users"
	ResourceKnowledge     Resource = "knowledge"
	ResourceAudit         Resource = "audit"
	ResourceSystem        Resource = "system"
)

// AccessLevel represents the level of access to a resource
type AccessLevel struct {
	Resource              Resource `json:"resource"`
	Actions               []Action `json:"actions"`
	SecurityClearanceReq  string   `json:"security_clearance_required,omitempty"`
	DepartmentRestriction string   `json:"department_restriction,omitempty"`
}

// AuthorizationService handles role-based access control
type AuthorizationService struct {
	rolePermissions map[models.UserRole][]AccessLevel
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService() *AuthorizationService {
	service := &AuthorizationService{
		rolePermissions: make(map[models.UserRole][]AccessLevel),
	}

	// Initialize default role permissions
	service.initializeDefaultPermissions()

	return service
}

// initializeDefaultPermissions sets up default permissions for each role
func (a *AuthorizationService) initializeDefaultPermissions() {
	// Admin role - full access to everything
	a.rolePermissions[models.UserRoleAdmin] = []AccessLevel{
		{Resource: ResourceDocuments, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
		{Resource: ResourceConsultations, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
		{Resource: ResourceUsers, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
		{Resource: ResourceKnowledge, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
		{Resource: ResourceAudit, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
		{Resource: ResourceSystem, Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin}},
	}

	// Analyst role - can read/write documents and consultations, read knowledge
	a.rolePermissions[models.UserRoleAnalyst] = []AccessLevel{
		{Resource: ResourceDocuments, Actions: []Action{ActionRead, ActionWrite}},
		{Resource: ResourceConsultations, Actions: []Action{ActionRead, ActionWrite}},
		{Resource: ResourceKnowledge, Actions: []Action{ActionRead, ActionWrite}},
		{Resource: ResourceAudit, Actions: []Action{ActionRead}},
	}

	// Manager role - can read/write most resources, limited admin on some
	a.rolePermissions[models.UserRoleManager] = []AccessLevel{
		{Resource: ResourceDocuments, Actions: []Action{ActionRead, ActionWrite, ActionDelete}},
		{Resource: ResourceConsultations, Actions: []Action{ActionRead, ActionWrite, ActionDelete}},
		{Resource: ResourceUsers, Actions: []Action{ActionRead}},
		{Resource: ResourceKnowledge, Actions: []Action{ActionRead, ActionWrite, ActionDelete}},
		{Resource: ResourceAudit, Actions: []Action{ActionRead, ActionWrite}},
	}

	// Consultant role - can read/write consultations and knowledge, read documents
	a.rolePermissions[models.UserRoleConsultant] = []AccessLevel{
		{Resource: ResourceDocuments, Actions: []Action{ActionRead}},
		{Resource: ResourceConsultations, Actions: []Action{ActionRead, ActionWrite}},
		{Resource: ResourceKnowledge, Actions: []Action{ActionRead, ActionWrite}},
		{Resource: ResourceAudit, Actions: []Action{ActionRead}},
	}

	// Viewer role - read-only access to most resources
	a.rolePermissions[models.UserRoleViewer] = []AccessLevel{
		{Resource: ResourceDocuments, Actions: []Action{ActionRead}},
		{Resource: ResourceConsultations, Actions: []Action{ActionRead}},
		{Resource: ResourceKnowledge, Actions: []Action{ActionRead}},
	}
}

// Authorize checks if a user has permission to perform an action on a resource
func (a *AuthorizationService) Authorize(ctx context.Context, user *models.User, resource Resource, action Action) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("user is nil")
	}

	if !user.IsActive {
		return false, models.ErrUserInactive
	}

	// Check role-based permissions
	rolePermissions, exists := a.rolePermissions[user.Role]
	if !exists {
		return false, fmt.Errorf("unknown user role: %s", user.Role)
	}

	// Check if the role has permission for this resource and action
	hasRolePermission := false
	for _, accessLevel := range rolePermissions {
		if accessLevel.Resource == resource {
			for _, allowedAction := range accessLevel.Actions {
				if allowedAction == action {
					hasRolePermission = true
					break
				}
			}
			break
		}
	}

	// Check user-specific permissions (overrides or additional permissions)
	hasUserPermission := user.HasPermission(string(resource), string(action))

	// User must have either role permission or explicit user permission
	if !hasRolePermission && !hasUserPermission {
		return false, nil
	}

	return true, nil
}

// AuthorizeWithSecurityClearance checks authorization with security clearance requirements
func (a *AuthorizationService) AuthorizeWithSecurityClearance(ctx context.Context, user *models.User, resource Resource, action Action, requiredClearance string) (bool, error) {
	// First check basic authorization
	authorized, err := a.Authorize(ctx, user, resource, action)
	if err != nil || !authorized {
		return authorized, err
	}

	// Check security clearance if required
	if requiredClearance != "" {
		if !user.CanAccessClassification(requiredClearance) {
			return false, models.ErrInvalidSecurityClearance
		}
	}

	return true, nil
}

// GetUserPermissions returns all permissions for a user
func (a *AuthorizationService) GetUserPermissions(ctx context.Context, user *models.User) ([]AccessLevel, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	// Get role-based permissions
	rolePermissions, exists := a.rolePermissions[user.Role]
	if !exists {
		return nil, fmt.Errorf("unknown user role: %s", user.Role)
	}

	// Start with role permissions
	permissions := make([]AccessLevel, len(rolePermissions))
	copy(permissions, rolePermissions)

	// Add user-specific permissions
	for _, userPerm := range user.Permissions {
		resource := Resource(userPerm.Resource)

		// Find existing permission for this resource
		found := false
		for i, perm := range permissions {
			if perm.Resource == resource {
				// Merge actions
				actionMap := make(map[Action]bool)
				for _, action := range perm.Actions {
					actionMap[action] = true
				}
				for _, action := range userPerm.Actions {
					actionMap[Action(action)] = true
				}

				// Convert back to slice
				var mergedActions []Action
				for action := range actionMap {
					mergedActions = append(mergedActions, action)
				}
				permissions[i].Actions = mergedActions
				found = true
				break
			}
		}

		// If resource not found in role permissions, add it
		if !found {
			var actions []Action
			for _, action := range userPerm.Actions {
				actions = append(actions, Action(action))
			}
			permissions = append(permissions, AccessLevel{
				Resource: resource,
				Actions:  actions,
			})
		}
	}

	return permissions, nil
}

// CheckResourceAccess checks if a user can access a specific resource instance
func (a *AuthorizationService) CheckResourceAccess(ctx context.Context, user *models.User, resourceID string, action Action) (*AccessLevel, error) {
	// This is a simplified implementation
	// In a real system, you might need to check resource-specific permissions
	// For example, checking if a user can access a specific document based on its classification

	// For now, we'll determine the resource type from the ID format or context
	// This is a placeholder implementation
	resource := a.determineResourceType(resourceID)

	authorized, err := a.Authorize(ctx, user, resource, action)
	if err != nil {
		return nil, err
	}

	if !authorized {
		return nil, models.ErrInsufficientPermissions
	}

	return &AccessLevel{
		Resource: resource,
		Actions:  []Action{action},
	}, nil
}

// determineResourceType determines the resource type from a resource ID
func (a *AuthorizationService) determineResourceType(resourceID string) Resource {
	// This is a simplified implementation
	// In a real system, you might have a more sophisticated way to determine resource types

	if strings.HasPrefix(resourceID, "doc_") {
		return ResourceDocuments
	} else if strings.HasPrefix(resourceID, "consult_") {
		return ResourceConsultations
	} else if strings.HasPrefix(resourceID, "user_") {
		return ResourceUsers
	} else if strings.HasPrefix(resourceID, "knowledge_") {
		return ResourceKnowledge
	}

	// Default to documents if we can't determine
	return ResourceDocuments
}

// CanUserAccessDocument checks if a user can access a document based on its classification
func (a *AuthorizationService) CanUserAccessDocument(ctx context.Context, user *models.User, documentClassification string, action Action) (bool, error) {
	// Check basic document permission
	authorized, err := a.Authorize(ctx, user, ResourceDocuments, action)
	if err != nil || !authorized {
		return authorized, err
	}

	// Check security clearance
	if !user.CanAccessClassification(documentClassification) {
		return false, models.ErrInvalidSecurityClearance
	}

	return true, nil
}

// AddRolePermission adds a permission to a role
func (a *AuthorizationService) AddRolePermission(role models.UserRole, resource Resource, actions []Action) {
	if a.rolePermissions[role] == nil {
		a.rolePermissions[role] = []AccessLevel{}
	}

	// Check if permission for this resource already exists
	for i, perm := range a.rolePermissions[role] {
		if perm.Resource == resource {
			// Merge actions
			actionMap := make(map[Action]bool)
			for _, action := range perm.Actions {
				actionMap[action] = true
			}
			for _, action := range actions {
				actionMap[action] = true
			}

			// Convert back to slice
			var mergedActions []Action
			for action := range actionMap {
				mergedActions = append(mergedActions, action)
			}
			a.rolePermissions[role][i].Actions = mergedActions
			return
		}
	}

	// Add new permission
	a.rolePermissions[role] = append(a.rolePermissions[role], AccessLevel{
		Resource: resource,
		Actions:  actions,
	})
}

// RemoveRolePermission removes a permission from a role
func (a *AuthorizationService) RemoveRolePermission(role models.UserRole, resource Resource, action Action) {
	permissions := a.rolePermissions[role]
	for i, perm := range permissions {
		if perm.Resource == resource {
			// Remove the specific action
			var newActions []Action
			for _, a := range perm.Actions {
				if a != action {
					newActions = append(newActions, a)
				}
			}

			if len(newActions) == 0 {
				// Remove the entire permission if no actions left
				a.rolePermissions[role] = append(permissions[:i], permissions[i+1:]...)
			} else {
				a.rolePermissions[role][i].Actions = newActions
			}
			break
		}
	}
}

// GetRolePermissions returns all permissions for a role
func (a *AuthorizationService) GetRolePermissions(role models.UserRole) []AccessLevel {
	permissions, exists := a.rolePermissions[role]
	if !exists {
		return []AccessLevel{}
	}

	// Return a copy to prevent external modification
	result := make([]AccessLevel, len(permissions))
	copy(result, permissions)
	return result
}
