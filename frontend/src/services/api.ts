import axios from 'axios';

const API_BASE_URL = (import.meta as any).env.VITE_API_URL || '/api';

export const DEFAULT_TIMEOUT_MS = 10000;
export const UPLOAD_TIMEOUT_MS = 10 * 60 * 1000;

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: DEFAULT_TIMEOUT_MS,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

const normalizeApiPath = (url?: string) => {
  if (!url) {
    return '';
  }

  try {
    const parsed = new URL(url, window.location.origin);
    return parsed.pathname.replace(/^\/api(?=\/|$)/, '');
  } catch {
    return url.split('?')[0].replace(/^\/api(?=\/|$)/, '');
  }
};

export const shouldRedirectToLoginOnUnauthorized = (error: any) => {
  if (error.response?.status !== 401) {
    return false;
  }

  const requestPath = normalizeApiPath(error.config?.url);
  if (requestPath === '/login') {
    return false;
  }

  return window.location.pathname !== '/login';
};

api.interceptors.request.use(
  (config) => {
    if (config.headers['Content-Type'] === undefined) {
      config.timeout = UPLOAD_TIMEOUT_MS;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (shouldRedirectToLoginOnUnauthorized(error)) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('permissions');
      localStorage.removeItem('lastActivity');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  },
);

export const extractListData = <T = any>(payload: any): T[] => {
  if (Array.isArray(payload)) {
    return payload as T[];
  }
  if (Array.isArray(payload?.items)) {
    return payload.items as T[];
  }
  return [];
};

export const extractPagedListData = <T = any>(payload: any) => {
  const items = extractListData<T>(payload);
  return {
    items,
    total: typeof payload?.total === 'number' ? payload.total : items.length,
    page: typeof payload?.page === 'number' ? payload.page : 1,
    pageSize: typeof payload?.page_size === 'number' ? payload.page_size : items.length,
  };
};

export default api;
