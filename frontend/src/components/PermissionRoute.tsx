import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

interface PermissionRouteProps {
  children: JSX.Element;
  permission?: string;
}

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
