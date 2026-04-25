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
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (employeeId: string, password: string) => Promise<void>;
  logout: () => void;
  isAdmin: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// 4小时无交互自动登出（单位：毫秒）
const AUTO_LOGOUT_TIME = 4 * 60 * 60 * 1000;

// 获取初始状态（同步读取 localStorage）
const getInitialState = () => {
  const storedToken = localStorage.getItem('token');
  const storedUser = localStorage.getItem('user');
  const storedLastActivity = localStorage.getItem('lastActivity');

  // 检查是否超时
  if (storedLastActivity && storedToken && storedUser) {
    const lastActivity = parseInt(storedLastActivity, 10);
    if (Date.now() - lastActivity > AUTO_LOGOUT_TIME) {
      // 超时，清除存储并返回初始状态
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('lastActivity');
      return { user: null, token: null };
    }
  }

  if (!storedUser) {
    return { token: storedToken, user: null };
  }

  try {
    return { token: storedToken, user: JSON.parse(storedUser) };
  } catch (error) {
    console.warn('Invalid cached user data, clearing auth state.', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('lastActivity');
    return { user: null, token: null };
  }
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const initialState = getInitialState();
  const [user, setUser] = useState<User | null>(initialState.user);
  const [token, setToken] = useState<string | null>(initialState.token);

  // 更新最后活动时间
  const updateLastActivity = () => {
    localStorage.setItem('lastActivity', Date.now().toString());
  };

  // 初始化最后活动时间
  useEffect(() => {
    if (token) {
      const storedLastActivity = localStorage.getItem('lastActivity');
      if (!storedLastActivity) {
        updateLastActivity();
      }
    }
  }, [token]);

  // 设置活动监听器
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

  // 检查是否超时
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

    const intervalId = setInterval(checkTimeout, 60000); // 每分钟检查一次

    return () => clearInterval(intervalId);
  }, [token]);

  const login = async (employeeId: string, password: string) => {
    const response = await api.post('/login', {
      employee_id: employeeId,
      password,
    });

    const { token, user } = response.data;
    setToken(token);
    setUser(user);
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    updateLastActivity();
  };

  const logout = () => {
    setUser(null);
    setToken(null);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('lastActivity');
  };

  const isAdmin = user?.role === 'admin';

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isAdmin }}>
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
