import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

const AdminRoute = ({
  children,
  allowManager = false,
}: {
  children: JSX.Element;
  allowManager?: boolean;
}) => {
  const { token, isAdmin, isManager } = useAuth();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (allowManager ? !isManager : !isAdmin) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
};

export default AdminRoute;
