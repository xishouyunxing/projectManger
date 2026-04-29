package services

import (
	"crane-system/database"
	"crane-system/models"
	"sync"
	"time"
)

const permCacheTTL = 5 * time.Minute

// CachedPermissions 缓存单个用户的完整权限数据。
type CachedPermissions struct {
	FunctionCodes  []string
	LinePermissions map[uint]LinePerm
	ManagedLineIDs  []uint
	ExpiresAt       time.Time
}

// LinePerm 表示用户对某条产线的有效权限。
type LinePerm struct {
	CanView     bool
	CanDownload bool
	CanUpload   bool
	CanManage   bool
	Source      string // "user_override", "role", "none"
}

var (
	permCache   sync.Map
	permCacheMu sync.Mutex
)

// GetUserPermissions 获取用户的完整权限数据，优先走缓存。
func GetUserPermissions(userID uint) (*CachedPermissions, error) {
	if v, ok := permCache.Load(userID); ok {
		cached := v.(*CachedPermissions)
		if time.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}

	permCacheMu.Lock()
	defer permCacheMu.Unlock()

	// 双重检查
	if v, ok := permCache.Load(userID); ok {
		cached := v.(*CachedPermissions)
		if time.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}

	cached, err := loadUserPermissions(userID)
	if err != nil {
		return nil, err
	}
	permCache.Store(userID, cached)
	return cached, nil
}

// loadUserPermissions 从数据库加载用户权限（不含缓存）。
func loadUserPermissions(userID uint) (*CachedPermissions, error) {
	var user models.User
	if err := database.DB.Select("id", "role", "role_id").First(&user, userID).Error; err != nil {
		return nil, err
	}

	result := &CachedPermissions{
		FunctionCodes:  []string{},
		LinePermissions: make(map[uint]LinePerm),
		ManagedLineIDs:  []uint{},
		ExpiresAt:       time.Now().Add(permCacheTTL),
	}

	// system_admin 不需要加载具体权限
	if user.Role == "system_admin" {
		return result, nil
	}

	// 加载功能权限
	if err := loadFunctionPermissions(userID, user.RoleID, result); err != nil {
		return nil, err
	}

	// 加载产线权限
	if err := loadLinePermissions(userID, user.RoleID, result); err != nil {
		return nil, err
	}

	// 加载产线管理员绑定
	if err := loadManagedLines(userID, result); err != nil {
		return nil, err
	}

	return result, nil
}

// loadFunctionPermissions 加载用户的功能权限（角色基础 + 用户覆盖）。
func loadFunctionPermissions(userID uint, roleID *uint, result *CachedPermissions) error {
	rolePermSet := map[uint]bool{}

	if roleID != nil {
		var rolePerms []models.RolePermission
		if err := database.DB.Where("role_id = ?", *roleID).Find(&rolePerms).Error; err != nil {
			return err
		}
		for _, rp := range rolePerms {
			rolePermSet[rp.PermissionID] = true
		}
	}

	// 应用用户覆盖
	overrides := map[uint]bool{} // permissionID -> granted
	var userOverrides []models.UserPermissionOverride
	if err := database.DB.Where("user_id = ?", userID).Find(&userOverrides).Error; err != nil {
		return err
	}
	for _, uo := range userOverrides {
		overrides[uo.PermissionID] = uo.Granted
	}

	// 加载所有权限定义
	var allPerms []models.Permission
	if err := database.DB.Find(&allPerms).Error; err != nil {
		return err
	}

	permMap := make(map[uint]models.Permission, len(allPerms))
	for _, p := range allPerms {
		permMap[p.ID] = p
	}

	// 合并：角色权限 + 用户覆盖
	enabledIDs := map[uint]bool{}
	for pid := range rolePermSet {
		enabledIDs[pid] = true
	}
	for pid, granted := range overrides {
		if granted {
			enabledIDs[pid] = true
		} else {
			delete(enabledIDs, pid)
		}
	}

	for pid := range enabledIDs {
		if p, ok := permMap[pid]; ok {
			result.FunctionCodes = append(result.FunctionCodes, p.Code)
		}
	}

	return nil
}

// loadLinePermissions 加载用户的产线权限（用户覆盖 → 角色权限）。
func loadLinePermissions(userID uint, roleID *uint, result *CachedPermissions) error {
	// 加载所有产线
	var lines []models.ProductionLine
	if err := database.DB.Select("id").Find(&lines).Error; err != nil {
		return err
	}

	// 加载角色产线权限
	roleLineMap := map[uint]models.RoleLinePermission{}
	if roleID != nil {
		var roleLines []models.RoleLinePermission
		if err := database.DB.Where("role_id = ?", *roleID).Find(&roleLines).Error; err != nil {
			return err
		}
		for _, rl := range roleLines {
			roleLineMap[rl.ProductionLineID] = rl
		}
	}

	// 加载用户产线覆盖（复用现有 user_permissions 表）
	var userPerms []models.UserPermission
	if err := database.DB.Where("user_id = ?", userID).Find(&userPerms).Error; err != nil {
		return err
	}
	userPermMap := map[uint]models.UserPermission{}
	for _, up := range userPerms {
		userPermMap[up.ProductionLineID] = up
	}

	// 合并：用户覆盖优先，否则回退到角色权限
	for _, line := range lines {
		if up, ok := userPermMap[line.ID]; ok {
			result.LinePermissions[line.ID] = LinePerm{
				CanView:     up.CanView,
				CanDownload: up.CanDownload,
				CanUpload:   up.CanUpload,
				CanManage:   up.CanManage,
				Source:      "user_override",
			}
		} else if rl, ok := roleLineMap[line.ID]; ok {
			result.LinePermissions[line.ID] = LinePerm{
				CanView:     rl.CanView,
				CanDownload: rl.CanDownload,
				CanUpload:   rl.CanUpload,
				CanManage:   rl.CanManage,
				Source:      "role",
			}
		} else {
			result.LinePermissions[line.ID] = LinePerm{
				Source: "none",
			}
		}
	}

	return nil
}

// loadManagedLines 加载用户作为产线管理员负责的产线列表。
func loadManagedLines(userID uint, result *CachedPermissions) error {
	var assignments []models.LineAdminAssignment
	if err := database.DB.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return err
	}
	for _, a := range assignments {
		result.ManagedLineIDs = append(result.ManagedLineIDs, a.ProductionLineID)
	}
	return nil
}

// InvalidateUserCache 清除指定用户的权限缓存。
func InvalidateUserCache(userID uint) {
	permCache.Delete(userID)
}

// InvalidateAllCache 清除所有用户的权限缓存（角色权限配置变更时使用）。
func InvalidateAllCache() {
	permCache.Range(func(key, value any) bool {
		permCache.Delete(key)
		return true
	})
}

// UserHasPermission 检查用户是否拥有指定功能权限。
func UserHasPermission(userID uint, code string) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	for _, c := range perms.FunctionCodes {
		if c == code {
			return true
		}
	}
	return false
}

// UserHasLinePermission 检查用户对指定产线是否有指定操作权限。
func UserHasLinePermission(userID uint, lineID uint, action string) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	lp, ok := perms.LinePermissions[lineID]
	if !ok {
		return false
	}
	switch action {
	case "view":
		return lp.CanView || lp.CanManage
	case "download":
		return lp.CanDownload || lp.CanManage
	case "upload":
		return lp.CanUpload || lp.CanManage
	case "manage":
		return lp.CanManage
	default:
		return false
	}
}

// IsLineManager 检查用户是否是指定产线的管理员。
func IsLineManager(userID uint, lineID uint) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	for _, id := range perms.ManagedLineIDs {
		if id == lineID {
			return true
		}
	}
	return false
}
