import axios from 'axios';

const API_BASE_URL =
  (import.meta as any).env.VITE_API_URL || '/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    // 文件上传请求（multipart/form-data）使用更长的超时时间
    if (config.headers['Content-Type'] === undefined) {
      config.timeout = 0; // 文件上传不限超时
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

// Response interceptor to handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Unauthorized - clear token and redirect to login
      localStorage.removeItem('token');
      localStorage.removeItem('user');
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
