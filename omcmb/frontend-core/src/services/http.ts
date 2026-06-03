import axios, {
  type AxiosInstance,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
  type AxiosError,
} from 'axios';
import { useUserStore } from '../store/userStore';
import { useAppStore } from '../store/appStore';
import { useMock } from './apiSwitch';

// --- Parameter name conversion (camelCase → snake_case) ---

// Specific field mappings for pagination + 已知 snake_case 后端字段。
// 仅显式登记需要转换的字段，避免误伤已经是 snake_case 的参数。
const paramKeyMap: Record<string, string> = {
  pageSize: 'page_size',
  sortField: 'sort_by',
  sortOrder: 'sort_dir',
  apiGroup: 'api_group',
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

    // 注入当前界面语言 Accept-Language（zh-CN / en-US），供后端按语言返回本地化显示名
    // （指标名 / 内置任务名）。值即 appStore.locale，无需映射。
    if (config.headers) {
      config.headers['Accept-Language'] = useAppStore.getState().locale;
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
//
// v0.6 起后端逐步迁移到统一信封 {ret:1, msg, data}（参 omcgo/docs/architecture/api-envelope.md）。
// 拦截器逻辑：
//   - 成功 2xx + body 含 `ret` 字段 → 视为信封；ret=1 拆出 data 节点；ret=0 reject 业务错误
//   - 成功 2xx + body 不含 `ret` → 兼容期：原样透传（迁移完成后删该分支）
//   - 失败 4xx/5xx → 进 onRejected；后端错误信封统一 {ret:0, msg, data:null, biz_code?}
//
// 业务侧无感切换：service 文件中 `const { data } = await http.get(...)` 拿到的就是裸业务体，
// 与之前未包装时一致。

type EnvelopeBody<T = unknown> = {
  ret?: number;
  msg?: string;
  data?: T;
  biz_code?: number;
  request_id?: string;
};

http.interceptors.response.use(
  (response: AxiosResponse) => {
    const body = response.data as EnvelopeBody | unknown;
    if (body && typeof body === 'object' && 'ret' in (body as object)) {
      const env = body as EnvelopeBody;
      if (env.ret === 1) {
        // 信封成功：把 data 节点向下游透出，使 `const { data } = await http.get(...)` 拿到业务体
        return { ...response, data: env.data };
      }
      if (env.ret === 0) {
        // 信封业务失败（罕见：HTTP 200 + ret=0 不推荐，但兼容防御）
        const err = new Error(env.msg || 'Business request failed') as Error & {
          bizCode?: number;
          requestId?: string;
        };
        err.bizCode = env.biz_code;
        err.requestId = env.request_id;
        return Promise.reject(err);
      }
    }
    // 兼容期：未包装的裸响应原样返回（待全量迁移完成后移除此分支）
    return response;
  },
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
      // /auth/login 的 401 不是 token 过期 —— 是用户名密码错。如果走下面的
      // clearAuth + window.location.href 硬跳分支，登录页 try/catch 里的
      // message.error toast 还没渲染就被页面刷新清掉，用户看到"输错密码无任
      // 何反馈"。透传给下面的 envelope-message-extract 分支让业务层自己 toast。
      const reqURL = originalRequest.url || '';
      if (reqURL.endsWith('/auth/login') || reqURL.includes('/auth/login?')) {
        // fall through to message extraction below — 不走 token refresh / 不跳转
      } else {
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
        // 用裸 axios 避免 401 拦截器循环；裸 axios 不走信封剥壳拦截器，
        // 后端 /auth/refresh 返回 {ret:1, msg, data:{access_token,...}}，必须手动剥一层。
        const { data: envelope } = await axios.post<{
          ret: number;
          msg?: string;
          data?: {
            access_token: string;
            refresh_token: string;
            expires_at: string;
            token_type?: string;
          };
        }>(
          `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/auth/refresh`,
          { refresh_token: refreshToken }
        );

        if (envelope.ret !== 1 || !envelope.data?.access_token) {
          throw new Error(envelope.msg || 'refresh token failed');
        }
        const tokenPair = envelope.data;

        const { setTokenPair } = useUserStore.getState();
        setTokenPair(tokenPair);
        onTokenRefreshed(tokenPair.access_token);

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${tokenPair.access_token}`;
        }
        return http(originalRequest);
      } catch {
        clearAuth();
        window.location.href = '/login';
        return Promise.reject(error);
      } finally {
        isRefreshing = false;
      }
      } // end else: token-refresh branch (login URL skipped)
    }

    // Extract error message from response body.
    // v0.6 信封：{ret:0, msg, data:null, biz_code?, request_id?}
    // 兼容老格式：{code, message, details, error}
    const responseData = error.response?.data as
      | {
          ret?: number;
          msg?: string;
          biz_code?: number;
          request_id?: string;
          // legacy fields (pre-v0.6, 兼容期保留)
          code?: number;
          message?: string;
          details?: string;
          error?: string;
        }
      | undefined;

    if (responseData) {
      const message =
        responseData.msg ||
        responseData.details ||
        responseData.message ||
        responseData.error;
      // 暴露业务错误码 + request_id + userMessage 到 error 对象。
      // userMessage 是业务层（LoginPage / KPIStandardReport 等）读取的友好文案
      // 字段约定，与 error.message（"Request failed with status code 401"
      // 风格）区分。此前只写 error.message 导致业务 axiosErr.userMessage 永远
      // undefined → toast fallback 到通用文案；改成同时写两个字段。
      const enrichedErr = error as AxiosError & {
        bizCode?: number;
        requestId?: string;
        userMessage?: string;
      };
      if (message) {
        error.message = message;
        enrichedErr.userMessage = message;
      }
      if (responseData.biz_code !== undefined) {
        enrichedErr.bizCode = responseData.biz_code;
      } else if (responseData.code !== undefined) {
        enrichedErr.bizCode = responseData.code;
      }
      if (responseData.request_id) {
        enrichedErr.requestId = responseData.request_id;
      }
    }

    return Promise.reject(error);
  }
);

export { http };
export default http;
