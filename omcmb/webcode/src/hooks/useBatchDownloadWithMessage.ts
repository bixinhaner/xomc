/**
 * useBatchDownloadWithMessage —— 包一层 useBatchDownload,把 antd message
 * (loading / success / error) 的胶水代码集中,4 个文件管理 Tab 直接用 trigger()
 * 不必各自处理 progress 显示。
 *
 * 流程：
 *   1. trigger({module, targets}) 触发 POST,立刻显示 message.loading
 *   2. axios onDownloadProgress 持续把字节数 push 进 message content
 *   3. 完成 → message.success;失败 → message.error
 *
 * isPending 给按钮 loading / disabled 用,跟原 mutation 一致。
 */
import { message } from 'antd';
import { useBatchDownload, type BatchDownloadArgs } from '@core/hooks/api/useBundle';
import { useT } from '@/hooks/useT';

const MSG_KEY = 'bundle-download';

function fmtBytes(n: number): string {
  if (!n || n < 0) return '0 B';
  if (n >= 1024 * 1024 * 1024) return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB`;
  if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${n} B`;
}

export function useBatchDownloadWithMessage() {
  const t = useT();
  const inner = useBatchDownload();

  const trigger = (args: Omit<BatchDownloadArgs, 'onProgress'>) => {
    let lastBytes = 0;
    void message.loading({
      content: t('bundle.downloading', { size: '0 B' }),
      key: MSG_KEY,
      duration: 0,
    });
    inner.mutate(
      {
        ...args,
        onProgress: (bytes) => {
          lastBytes = bytes;
          void message.loading({
            content: t('bundle.downloading', { size: fmtBytes(bytes) }),
            key: MSG_KEY,
            duration: 0,
          });
        },
      },
      {
        onSuccess: () => {
          void message.success({
            content: t('bundle.downloaded', { size: fmtBytes(lastBytes) }),
            key: MSG_KEY,
            duration: 3,
          });
        },
        onError: (e) => {
          void message.error({
            content: t('bundle.failedToast', { msg: String(e) }),
            key: MSG_KEY,
            duration: 5,
          });
        },
      },
    );
  };

  return {
    trigger,
    isPending: inner.isPending,
  };
}
