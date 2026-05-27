/**
 * useBundle —— 同步流式批量下载 hook（极简）。
 *
 * useMutation 包一层 downloadBundle,提供 isPending 给按钮 loading 用。
 * onProgress 透传到 axios onDownloadProgress,调用方拿到实时已接收字节数,
 * 通常配合 antd `message.loading` 持久消息实时更新展示。
 */
import { useMutation } from '@tanstack/react-query';
import { downloadBundle, type BundleModule } from '../../services/api/bundleApi';

export interface BatchDownloadArgs {
  module: BundleModule;
  targets: string[];
  /** 可选: axios onDownloadProgress 接收字节数回调,用于 UI 实时反馈。 */
  onProgress?: (bytesReceived: number) => void;
}

export function useBatchDownload() {
  return useMutation({
    mutationFn: ({ module, targets, onProgress }: BatchDownloadArgs) =>
      downloadBundle(module, targets, onProgress),
  });
}
