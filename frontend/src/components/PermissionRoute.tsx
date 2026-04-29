import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

interface PermissionRouteProps {
  children: JSX.Element;
  permission?: string;
}

// 通用权限路由守卫。
// permission 为空时仅检查登录状态；否则还需检查用户是否拥有该功能权限。
// system_admin 始终通过。
const PermissionRoute = ({ children, permission }: PermissionRouteProps) => {
  const { token, isAdmin, hasPermission } = useAuth();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (permission && !isAdmin && !hasPermission(permission)) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
};

export default PermissionRoute;
