package services

import (
	"crane-system/models"
	"net/http"
)

type LineAction string

const (
	LineActionView     LineAction = models.PermissionActionView
	LineActionDownload LineAction = models.PermissionActionDownload
	LineActionUpload   LineAction = models.PermissionActionUpload
	LineActionManage   LineAction = models.PermissionActionManage
)

type AuthDecision struct {
	Allowed    bool
	StatusCode int
	Message    string
}

func Allow() AuthDecision {
	return AuthDecision{Allowed: true}
}

func Deny(statusCode int, message string) AuthDecision {
	return AuthDecision{Allowed: false, StatusCode: statusCode, Message: message}
}

func IsSystemAdminRole(role string) bool {
	return role == "admin" || role == "system_admin"
}

func AuthorizeOwnerOrAdmin(userID uint, role string, targetUserID uint) AuthDecision {
	if IsSystemAdminRole(role) {
		return Allow()
	}
	if userID == 0 {
		return Deny(http.StatusUnauthorized, "未认证")
	}
	if userID != targetUserID {
		return Deny(http.StatusForbidden, "无权操作")
	}
	return Allow()
}

func CheckLineAction(userID uint, role string, productionLineID uint, action LineAction) AuthDecision {
	if IsSystemAdminRole(role) {
		return Allow()
	}
	if userID == 0 {
		return Deny(http.StatusUnauthorized, "未认证")
	}
	if role == "line_admin" && IsLineManager(userID, productionLineID) {
		return Allow()
	}
	if !UserHasLinePermission(userID, productionLineID, string(action)) {
		return Deny(http.StatusForbidden, "无权操作")
	}
	return Allow()
}

func ResolveAuthorizedLineIDs(userID uint, role string, action LineAction) (map[uint]struct{}, AuthDecision) {
	if IsSystemAdminRole(role) {
		return nil, Allow()
	}
	if userID == 0 {
		return nil, Deny(http.StatusUnauthorized, "未认证")
	}

	perms, err := GetUserPermissions(userID)
	if err != nil {
		return nil, Deny(http.StatusInternalServerError, "查询权限失败")
	}

	allowedLineIDs := map[uint]struct{}{}
	for lineID, lp := range perms.LinePermissions {
		if LinePermissionAllowsAction(lp.CanView, lp.CanDownload, lp.CanUpload, lp.CanManage, action) {
			allowedLineIDs[lineID] = struct{}{}
		}
	}
	if role == "line_admin" {
		for _, managedID := range perms.ManagedLineIDs {
			allowedLineIDs[managedID] = struct{}{}
		}
	}
	return allowedLineIDs, Allow()
}

func AuthorizeLineAdminScope(userID uint, role string, productionLineID uint) AuthDecision {
	if IsSystemAdminRole(role) {
		return Allow()
	}
	if userID == 0 {
		return Deny(http.StatusUnauthorized, "未认证")
	}
	if role == "line_admin" && IsLineManager(userID, productionLineID) {
		return Allow()
	}
	return Deny(http.StatusForbidden, "无权操作该产线")
}

func LinePermissionAllowsAction(canView, canDownload, canUpload, canManage bool, action LineAction) bool {
	switch action {
	case LineActionView:
		return canView || canManage
	case LineActionDownload:
		return canDownload || canManage
	case LineActionUpload:
		return canUpload || canManage
	case LineActionManage:
		return canManage
	default:
		return false
	}
}
