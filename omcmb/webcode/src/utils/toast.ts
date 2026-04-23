import { message } from 'antd';

/**
 * 统一的消息提示入口。封装 antd 的 message API，便于：
 *   1. 统一错误对象 → 文本的解析规则（Error / string / unknown）
 *   2. 后续替换底层实现（例如改用 App.useApp 的 scoped messageApi、或切到 notification）
 *   3. 为 useMutation 提供便捷的 Promise 包装
 *
 * 调用风格：
 *   toast.success(t('xx.created'));
 *   toast.error(err, t('xx.createFailed'));
 *   withToast(mutation.mutateAsync(payload), { success: ..., errorPrefix: ... });
 */
export const toast = {
  success(content: string): void {
    void message.success(content);
  },
  error(err: unknown, prefix?: string): void {
    const raw =
      err instanceof Error
        ? err.message
        : typeof err === 'string'
          ? err
          : 'Unknown error';
    void message.error(prefix ? `${prefix}: ${raw}` : raw);
  },
  warning(content: string): void {
    void message.warning(content);
  },
  info(content: string): void {
    void message.info(content);
  },
};

/**
 * 把任意 Promise 包装成"成功 toast + 失败 toast + 原始 Promise"。
 * useMutation.mutateAsync(...) / 普通 async fetch 都可以用。
 * 失败会继续抛出，便于上层 .catch 做回滚等二次处理。
 */
export async function withToast<T>(
  promise: Promise<T>,
  opts: { success: string; errorPrefix?: string }
): Promise<T> {
  try {
    const result = await promise;
    toast.success(opts.success);
    return result;
  } catch (err) {
    toast.error(err, opts.errorPrefix);
    throw err;
  }
}
