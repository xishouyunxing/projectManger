import axios from 'axios';

const API_BASE_URL =
  (import.meta as any).env.VITE_API_URL || '/api';

// 全局 API 实例：业务代码统一通过这里发请求，便于集中处理 token、超时和 401 跳转。
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 将 axios config 中可能出现的绝对/相对 URL 统一成后端 API 路径。
// 目前主要用于区分登录接口 401 和其他接口 401。
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

// 请求拦截器：自动附加 JWT。
// 文件上传由浏览器自行设置 multipart boundary，因此 Content-Type 未显式设置时不限制超时。
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

// 响应拦截器：非登录接口收到 401 时清空本地登录态并回到登录页。
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (shouldRedirectToLoginOnUnauthorized(error)) {
      // Unauthorized - clear token and redirect to login
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('lastActivity');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  },
);

// 兼容历史数组响应和分页响应，列表页统一用这个函数读取 items。
export const extractListData = <T = any>(payload: any): T[] => {
  if (Array.isArray(payload)) {
    return payload as T[];
  }
  if (Array.isArray(payload?.items)) {
    return payload.items as T[];
  }
  return [];
};

// 兼容历史数组响应和分页响应，列表页统一用这个函数读取分页元信息。
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
