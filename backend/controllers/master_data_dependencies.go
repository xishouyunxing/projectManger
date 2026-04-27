package controllers

import "crane-system/database"

type masterDataDependencyCheck struct {
	Model any
	Where string
	Args  []any
	Label string
}

func findMasterDataDependency(checks []masterDataDependencyCheck) (string, error) {
	for _, check := range checks {
		var count int64
		if err := database.DB.Model(check.Model).Where(check.Where, check.Args...).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			return check.Label, nil
		}
	}
	return "", nil
}
