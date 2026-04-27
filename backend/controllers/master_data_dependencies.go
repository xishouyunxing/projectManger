package controllers

import "crane-system/database"

type masterDataDependencyCheck struct {
	Model any
	Where string
	Args  []any
	Label string
}

// findMasterDataDependency 用于删除主数据前的依赖检查。
// 返回第一个命中的业务依赖标签，调用方据此给出可读的阻止删除原因。
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
