import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from 'react';
import api from '../services/api';

interface User {
  id: number;
  employee_id: string;
  name: string;
  department: {
    id: number;
    name: string;
    description: string;
    status: string;
  } | null;
  role: string;
  role_id?: number;
}

interface LinePermission {
  can_view: boolean;
  can_download: boolean;
  can_upload: boolean;
  can_manage: boolean;
}

interface UserPermissions {
  codes: string[];
  lines: Record<number, LinePermission>;
  managedLineIds: number[];
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (employeeId: string, password: string) => Promise<void>;
  logout: () => void;
  isAdmin: boolean;
  isLineAdmin: boolean;
  permissions: UserPermissions;
  hasPermission: (code: string) => boolean;
  hasLinePermission: (lineId: number, action: string) => boolean;
  isLineManager: (lineId: number) => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// 4小时无交互自动登出（单位：毫秒）
const AUTO_LOGOUT_TIME = 4 * 60 * 60 * 1000;

const emptyPermissions: UserPermissions = {
  codes: [],
  lines: {},
  managedLineIds: [],
};

const getInitialState = () => {
  const storedToken = localStorage.getItem('token');
  const storedUser = localStorage.getItem('user');
  const storedPermissions = localStorage.getItem('permissions');
  const storedLastActivity = localStorage.getItem('lastActivity');

  // 检查是否超时
  if (storedLastActivity && storedToken && storedUser) {
    const lastActivity = parseInt(storedLastActivity, 10);
    if (Date.now() - lastActivity > AUTO_LOGOUT_TIME) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('permissions');
      localStorage.removeItem('lastActivity');
      return { user: null, token: null, permissions: emptyPermissions };
    }
  }

  if (!storedUser) {
    return {
      token: storedToken,
      user: null,
      permissions: storedPermissions
        ? JSON.parse(storedPermissions)
        : emptyPermissions,
    };
  }

  try {
    return {
      token: storedToken,
      user: JSON.parse(storedUser),
      permissions: storedPermissions
        ? JSON.parse(storedPermissions)
        : emptyPermissions,
    };
  } catch (error) {
    console.warn('Invalid cached user data, clearing auth state.', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    localStorage.removeItem('lastActivity');
    return { user: null, token: null, permissions: emptyPermissions };
  }
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const initialState = getInitialState();
  const [user, setUser] = useState<User | null>(initialState.user);
  const [token, setToken] = useState<string | null>(initialState.token);
  const [permissions, setPermissions] = useState<UserPermissions>(
    initialState.permissions,
  );

  const updateLastActivity = () => {
    localStorage.setItem('lastActivity', Date.now().toString());
  };

  useEffect(() => {
    if (token) {
      const storedLastActivity = localStorage.getItem('lastActivity');
      if (!storedLastActivity) {
        updateLastActivity();
      }
    }
  }, [token]);

  useEffect(() => {
    if (!token) return;

    const events = ['mousedown', 'keydown', 'scroll', 'touchstart', 'click'];
    const handleActivity = () => {
      updateLastActivity();
    };

    events.forEach((event) => {
      window.addEventListener(event, handleActivity);
    });

    return () => {
      events.forEach((event) => {
        window.removeEventListener(event, handleActivity);
      });
    };
  }, [token]);

  useEffect(() => {
    if (!token) return;

    const checkTimeout = () => {
      const lastActivity = localStorage.getItem('lastActivity');
      if (lastActivity) {
        const elapsed = Date.now() - parseInt(lastActivity, 10);
        if (elapsed > AUTO_LOGOUT_TIME) {
          logout();
        }
      }
    };

    const intervalId = setInterval(checkTimeout, 60000);

    return () => clearInterval(intervalId);
  }, [token]);

  const login = async (employeeId: string, password: string) => {
    const response = await api.post('/login', {
      employee_id: employeeId,
      password,
    });

    const { token, user, permissions: permData } = response.data;
    setToken(token);
    setUser(user);

    const parsedPermissions: UserPermissions = {
      codes: permData?.codes || [],
      lines: permData?.lines || {},
      managedLineIds: permData?.managed_line_ids || [],
    };
    setPermissions(parsedPermissions);

    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    localStorage.setItem('permissions', JSON.stringify(parsedPermissions));
    updateLastActivity();
  };

  const logout = () => {
    setUser(null);
    setToken(null);
    setPermissions(emptyPermissions);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    localStorage.removeItem('lastActivity');
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'system_admin';
  const isLineAdmin = user?.role === 'line_admin';

  const hasPermission = (code: string): boolean => {
    if (isAdmin) return true;
    return permissions.codes.includes(code);
  };

  const hasLinePermission = (lineId: number, action: string): boolean => {
    if (isAdmin) return true;
    const linePerm = permissions.lines[lineId];
    if (!linePerm) return false;
    switch (action) {
      case 'view':
        return linePerm.can_view;
      case 'download':
        return linePerm.can_download;
      case 'upload':
        return linePerm.can_upload;
      case 'manage':
        return linePerm.can_manage;
      default:
        return false;
    }
  };

  const isLineManager = (lineId: number): boolean => {
    if (isAdmin) return true;
    return permissions.managedLineIds.includes(lineId);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        login,
        logout,
        isAdmin,
        isLineAdmin,
        permissions,
        hasPermission,
        hasLinePermission,
        isLineManager,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
