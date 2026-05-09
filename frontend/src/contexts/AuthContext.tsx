import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from 'react';
import api from '../services/api';

interface Department {
  id: number;
  name: string;
  description?: string;
  status?: string;
}

interface User {
  id: number;
  employee_id: string;
  name: string;
  department: Department | null;
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
  lines: Record<string, LinePermission>;
  managed_line_ids: string[];
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (employeeId: string, password: string) => Promise<void>;
  logout: () => void;
  isAdmin: boolean;
  isLineAdmin: boolean;
  isOperator: boolean;
  isProgrammer: boolean;
  isManager: boolean;
  canEdit: boolean;
  permissions: UserPermissions;
  hasPermission: (code: string) => boolean;
  hasLinePermission: (lineId: number | string, action: string) => boolean;
  isLineManager: (lineId: number | string) => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const AUTO_LOGOUT_TIME = 4 * 60 * 60 * 1000;

const emptyPermissions: UserPermissions = {
  codes: [],
  lines: {},
  managed_line_ids: [],
};

const normalizePermissions = (value: any): UserPermissions => ({
  codes: Array.isArray(value?.codes) ? value.codes : [],
  lines: value?.lines && typeof value.lines === 'object' ? value.lines : {},
  managed_line_ids: Array.isArray(value?.managed_line_ids)
    ? value.managed_line_ids.map(String)
    : Array.isArray(value?.managedLineIds)
      ? value.managedLineIds.map(String)
      : [],
});

const clearStoredAuth = () => {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  localStorage.removeItem('permissions');
  localStorage.removeItem('lastActivity');
};

const parseStoredPermissions = (storedPermissions: string | null) => {
  if (!storedPermissions) {
    return emptyPermissions;
  }
  try {
    return normalizePermissions(JSON.parse(storedPermissions));
  } catch (error) {
    console.warn('Invalid cached permission data, clearing auth state.', error);
    clearStoredAuth();
    return emptyPermissions;
  }
};

const getInitialState = () => {
  const storedUser = localStorage.getItem('user');
  const storedPermissions = localStorage.getItem('permissions');
  const storedLastActivity = localStorage.getItem('lastActivity');

  if (storedLastActivity && storedUser) {
    const lastActivity = parseInt(storedLastActivity, 10);
    if (Date.now() - lastActivity > AUTO_LOGOUT_TIME) {
      clearStoredAuth();
      return { user: null, token: null, permissions: emptyPermissions };
    }
  }

  if (!storedUser) {
    return {
      token: null,
      user: null,
      permissions: parseStoredPermissions(storedPermissions),
    };
  }

  try {
    return {
      token: 'cookie',
      user: JSON.parse(storedUser),
      permissions: parseStoredPermissions(storedPermissions),
    };
  } catch (error) {
    console.warn('Invalid cached user data, clearing auth state.', error);
    clearStoredAuth();
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

  const logout = useCallback(() => {
    void api.post('/logout').catch(() => undefined);
    setUser(null);
    setToken(null);
    setPermissions(emptyPermissions);
    clearStoredAuth();
  }, []);

  useEffect(() => {
    if (token && !localStorage.getItem('lastActivity')) {
      updateLastActivity();
    }
  }, [token]);

  useEffect(() => {
    if (!token) return;

    const events = ['mousedown', 'keydown', 'scroll', 'touchstart', 'click'];
    const handleActivity = () => updateLastActivity();

    events.forEach((event) => window.addEventListener(event, handleActivity));
    return () => {
      events.forEach((event) =>
        window.removeEventListener(event, handleActivity),
      );
    };
  }, [token]);

  useEffect(() => {
    if (!token) return;

    const checkTimeout = () => {
      const lastActivity = localStorage.getItem('lastActivity');
      if (!lastActivity) return;

      const elapsed = Date.now() - parseInt(lastActivity, 10);
      if (elapsed > AUTO_LOGOUT_TIME) {
        logout();
      }
    };

    const intervalId = window.setInterval(checkTimeout, 60000);
    return () => window.clearInterval(intervalId);
  }, [logout, token]);

  const login = async (employeeId: string, password: string) => {
    const response = await api.post('/login', {
      employee_id: employeeId,
      password,
    });

    const { user: newUser, permissions: permissionPayload } = response.data;
    const parsedPermissions = normalizePermissions(permissionPayload);

    setToken('cookie');
    setUser(newUser);
    setPermissions(parsedPermissions);
    localStorage.setItem('user', JSON.stringify(newUser));
    localStorage.setItem('permissions', JSON.stringify(parsedPermissions));
    updateLastActivity();
  };

  const isAdmin = user?.role === 'admin' || user?.role === 'system_admin';
  const isLineAdmin = user?.role === 'line_admin';
  const isOperator = user?.role === 'field_operator' || user?.role === 'operator';
  const isProgrammer = user?.role === 'offline_programmer' || user?.role === 'engineer';
  const isManager = isAdmin || isLineAdmin;
  const canEdit = isAdmin || isLineAdmin || isProgrammer;

  const hasPermission = useCallback(
    (code: string): boolean => {
      if (isAdmin) return true;
      return permissions.codes.includes(code);
    },
    [isAdmin, permissions.codes],
  );

  const hasLinePermission = useCallback(
    (lineId: number | string, action: string): boolean => {
      if (isAdmin) return true;
      const linePermission = permissions.lines[String(lineId)];
      if (!linePermission) return false;

      switch (action) {
        case 'view':
          return linePermission.can_view || linePermission.can_manage;
        case 'download':
          return linePermission.can_download || linePermission.can_manage;
        case 'upload':
          return linePermission.can_upload || linePermission.can_manage;
        case 'manage':
          return linePermission.can_manage;
        default:
          return false;
      }
    },
    [isAdmin, permissions.lines],
  );

  const isLineManager = useCallback(
    (lineId: number | string): boolean => {
      if (isAdmin) return true;
      return permissions.managed_line_ids.includes(String(lineId));
    },
    [isAdmin, permissions.managed_line_ids],
  );

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        login,
        logout,
        isAdmin,
        isLineAdmin,
        isOperator,
        isProgrammer,
        isManager,
        canEdit,
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
