import { useAuth } from '../contexts/AuthContext';

// 权限检查便捷 hook，透传 AuthContext 中的权限方法。
// 组件可直接 useAuth() 解构，此 hook 仅作为语义别名，便于按需引用。
export const usePermission = () => {
  const { isAdmin, isLineAdmin, hasPermission, hasLinePermission, isLineManager } = useAuth();
  return { isAdmin, isLineAdmin, hasPermission, hasLinePermission, isLineManager };
};

export default usePermission;
