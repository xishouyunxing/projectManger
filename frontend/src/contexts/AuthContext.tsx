import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  ReactNode,
} from 'react';
import api from '../services/api';

interface Department {
  id: number;
  name: string;
  description: string;
  status: string;
}

interface User {
  id: number;
  employee_id: string;
  name: string;
  department: Department | null;
  role: string;
  role_id?: number;
}

interface LinePerm {
  can_view: boolean;
  can_download: boolean;
  can_upload: boolean;
  can_manage: boolean;
}

interface UserPermissions {
  codes: string[];
  lines: Record<string, LinePerm>;
  managed_line_ids: string[];
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
  hasLinePermission: (lineId: number | string, action: string) => boolean;
  isLineManager: (lineId: number | string) => boolean;
}

const defaultPermissions: UserPermissions = {
  codes: [],
  lines: {},
  managed_line_ids: [],
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// 4小时无交互自动登出（单位：毫秒）
const AUTO_LOGOUT_TIME = 4 * 60 * 60 * 1000;

// 同步读取 localStorage，保证首次渲染时就能判断登录态
const getInitialState = () => {
  const storedToken = localStorage.getItem('token');
  const storedUser = localStorage.getItem('user');
  const storedPerms = localStorage.getItem('permissions');
  const storedLastActivity = localStorage.getItem('lastActivity');

  // 检查是否超时
  if (storedLastActivity && storedToken && storedUser) {
    const lastActivity = parseInt(storedLastActivity, 10);
    if (Date.now() - lastActivity > AUTO_LOGOUT_TIME) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('permissions');
      localStorage.removeItem('lastActivity');
      return { user: null, token: null, permissions: defaultPermissions };
    }
  }

  if (!storedUser) {
    return { token: storedToken, user: null, permissions: defaultPermissions };
  }

  try {
    let permissions = defaultPermissions;
    if (storedPerms) {
      try {
        permissions = JSON.parse(storedPerms);
      } catch {
        // ignore
      }
    }
    return { token: storedToken, user: JSON.parse(storedUser), permissions };
  } catch (error) {
    console.warn('Invalid cached user data, clearing auth state.', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    localStorage.removeItem('lastActivity');
    return { user: null, token: null, permissions: defaultPermissions };
  }
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const initialState = getInitialState();
  const [user, setUser] = useState<User | null>(initialState.user);
  const [token, setToken] = useState<string | null>(initialState.token);
  const [permissions, setPermissions] = useState<UserPermissions>(initialState.permissions);

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
    const handleActivity = () => updateLastActivity();
    events.forEach((event) => window.addEventListener(event, handleActivity));
    return () => events.forEach((event) => window.removeEventListener(event, handleActivity));
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

    const { token: newToken, user: newUser, permissions: newPerms } = response.data;

    setToken(newToken);
    setUser(newUser);
    setPermissions(newPerms || defaultPermissions);
    localStorage.setItem('token', newToken);
    localStorage.setItem('user', JSON.stringify(newUser));
    localStorage.setItem('permissions', JSON.stringify(newPerms || defaultPermissions));
    updateLastActivity();
  };

  const logout = () => {
    setUser(null);
    setToken(null);
    setPermissions(defaultPermissions);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    localStorage.removeItem('lastActivity');
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'system_admin';
  const isLineAdmin = user?.role === 'line_admin';

  const hasPermission = useCallback((code: string): boolean => {
    if (isAdmin) return true;
    return permissions.codes.includes(code);
  }, [isAdmin, permissions.codes]);

  const hasLinePermission = useCallback((lineId: number | string, action: string): boolean => {
    if (isAdmin) return true;
    const key = String(lineId);
    const lp = permissions.lines[key];
    if (!lp) return false;
    switch (action) {
      case 'view': return lp.can_view || lp.can_manage;
      case 'download': return lp.can_download || lp.can_manage;
      case 'upload': return lp.can_upload || lp.can_manage;
      case 'manage': return lp.can_manage;
      default: return false;
    }
  }, [isAdmin, permissions.lines]);

  const isLineManager = useCallback((lineId: number | string): boolean => {
    if (isAdmin) return true;
    return permissions.managed_line_ids.includes(String(lineId));
  }, [isAdmin, permissions.managed_line_ids]);

  return (
    <AuthContext.Provider value={{
      user, token, login, logout,
      isAdmin, isLineAdmin, permissions,
      hasPermission, hasLinePermission, isLineManager,
    }}>
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
