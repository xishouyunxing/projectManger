package controllers

import (
	"crane-system/models"
	"crane-system/services"
)

type linePermissionBits struct {
	ProductionLineID uint
	CanView          bool
	CanDownload      bool
	CanUpload        bool
	CanManage        bool
}

func syncLinePermissionRules(subject services.PermissionSubject, bits linePermissionBits) error {
	return services.SavePermissionRuleChanges(subject, permissionRuleChangesForLineBits(bits, false))
}

func clearLinePermissionRules(subject services.PermissionSubject, lineID uint) error {
	return services.SavePermissionRuleChanges(subject, permissionRuleChangesForLineBits(linePermissionBits{
		ProductionLineID: lineID,
	}, true))
}

func permissionRuleChangesForLineBits(bits linePermissionBits, unset bool) []services.PermissionRuleChange {
	return []services.PermissionRuleChange{
		permissionRuleChangeForAction(bits.ProductionLineID, models.PermissionActionView, bits.CanView, unset),
		permissionRuleChangeForAction(bits.ProductionLineID, models.PermissionActionDownload, bits.CanDownload, unset),
		permissionRuleChangeForAction(bits.ProductionLineID, models.PermissionActionUpload, bits.CanUpload, unset),
		permissionRuleChangeForAction(bits.ProductionLineID, models.PermissionActionManage, bits.CanManage, unset),
	}
}

func permissionRuleChangeForAction(lineID uint, action string, allowed bool, unset bool) services.PermissionRuleChange {
	decision := models.PermissionDecisionDeny
	if allowed {
		decision = models.PermissionDecisionAllow
	}
	if unset {
		decision = "unset"
	}
	return services.PermissionRuleChange{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   lineID,
		Action:       action,
		Decision:     decision,
	}
}
