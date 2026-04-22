import axios, {
  type AxiosInstance,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
  type AxiosError,
} from 'axios';
import { useUserStore } from '../store/userStore';
import { useMock } from './apiSwitch';

// --- Parameter name conversion (camelCase → snake_case) ---

// Specific field mappings for pagination params
const paramKeyMap: Record<string, string> = {
  pageSize: 'page_size',
  sortField: 'sort_by',
  sortOrder: 'sort_dir',
};

const sortOrderMap: Record<string, string> = {
  ascend: 'asc',
  descend: 'desc',
};

function transformParams(
  params: Record<string, unknown> | undefined
): Record<string, unknown> | undefined {
  if (!params) return params;
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue;
    const mappedKey = paramKeyMap[key] || key;
    if (key === 'sortOrder' && typeof value === 'string') {
      result[mappedKey] = sortOrderMap[value] || value;
    } else {
      result[mappedKey] = value;
    }
  }
  return result;
}

// --- Token refresh queue ---

let isRefreshing = false;
let refreshSubscribers: Array<(token: string) => void> = [];

function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach((cb) => cb(token));
  refreshSubscribers = [];
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb);
}

// --- Axios instance ---

const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// --- Request interceptor ---

http.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Attach access token
    const { accessToken } = useUserStore.getState();
    if (accessToken && config.headers) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }

    // Convert query params from camelCase to snake_case
    if (config.params) {
      config.params = transformParams(
        config.params as Record<string, unknown>
      );
    }

    return config;
  },
  (error: AxiosError) => Promise.reject(error)
);

// --- Response interceptor ---

http.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;
    if (!originalRequest) return Promise.reject(error);

    // Network error (no response received — server unreachable)
    if (!error.response) {
      console.error('[HTTP] Network error:', error.message);
      return Promise.reject(error);
    }

    // Handle 401 — attempt token refresh
    if (error.response?.status === 401) {
      // Mock 模式下 401 意味着某端点缺 mock 适配；跳 /login 会在 mock 登录→导航→401 间死循环
      if (useMock) {
        return Promise.reject(error);
      }
      const { refreshToken, clearAuth } = useUserStore.getState();

      // No refresh token or this was already a refresh attempt → logout
      if (!refreshToken || (originalRequest as unknown as Record<string, unknown>)._isRetry) {
        clearAuth();
        window.location.href = '/login';
        return Promise.reject(error);
      }

      if (isRefreshing) {
        // Queue this request until the refresh completes
        return new Promise<AxiosResponse>((resolve) => {
          addRefreshSubscriber((newToken: string) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${newToken}`;
            }
            resolve(http(originalRequest));
          });
        });
      }

      isRefreshing = true;
      (originalRequest as unknown as Record<string, unknown>)._isRetry = true;

      try {
        // Use plain axios to avoid interceptor loop
        const { data } = await axios.post<{
          access_token: string;
          refresh_token: string;
          expires_at: string;
          token_type: string;
        }>(
          `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/auth/refresh`,
          { refresh_token: refreshToken }
        );

        const { setTokenPair } = useUserStore.getState();
        setTokenPair(data);
        onTokenRefreshed(data.access_token);

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
        }
        return http(originalRequest);
      } catch {
        clearAuth();
        window.location.href = '/login';
        return Promise.reject(error);
      } finally {
        isRefreshing = false;
      }
    }

    // Extract error message from response body
    const responseData = error.response?.data as
      | { code?: number; message?: string; details?: string; error?: string }
      | undefined;

    if (responseData) {
      const message =
        responseData.message || responseData.error || responseData.details;
      if (message) {
        error.message = message;
      }
    }

    return Promise.reject(error);
  }
);

export { http };
export default http;
