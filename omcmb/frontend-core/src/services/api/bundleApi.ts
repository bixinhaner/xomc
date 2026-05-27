/**
 * bundleApi —— 文件管理 4 Tab 批量下载共用 API（同步流式）
 *
 * 后端见 internal/bundle/。流程：
 *   1. POST /<module>/batch-download (body 是 IDs/SNs 列表) → fetch + ReadableStream
 *      拿 chunked 响应,自己 reader.read() 累加字节供 onProgress 平滑回调
 *   2. 浏览器自动下载,Content-Disposition 给文件名
 *
 * 为什么不用 axios:axios responseType:'blob' 走 XHR,在 chunked-transfer 场景下
 * 浏览器对 progress event 触发频率被严重节流(头/尾几次回调,中间长时间静默),
 * 大文件下载进度条会看起来卡死。fetch 严格每 chunk 触发一次,符合预期。
 *
 * 不需要 task table / 不需要轮询 / 不需要 presigned URL。
 */
import { useUserStore } from '../../store/userStore';

export type BundleModule =
  | 'firmware'
  | 'config_snapshot'
  | 'device_license'
  | 'mr'
  // mr_files: 按 mr_files.id 粒度打包(DeviceFilesDrawer 用户勾选若干文件下载)
  | 'mr_files';

interface BundleEndpointSpec {
  url: string;
  body: (targets: string[]) => Record<string, string[]>;
  filenamePrefix: string;
}

const ENDPOINTS: Record<BundleModule, BundleEndpointSpec> = {
  firmware: {
    url: '/firmware/batch-download',
    body: (ids) => ({ ids }),
    filenamePrefix: 'firmware',
  },
  config_snapshot: {
    url: '/backup/config-snapshots/batch-download',
    body: (sns) => ({ serial_numbers: sns }),
    filenamePrefix: 'config-snapshot',
  },
  device_license: {
    url: '/backup/device-licenses/batch-download',
    body: (sns) => ({ serial_numbers: sns }),
    filenamePrefix: 'device-license',
  },
  mr: {
    url: '/mr/files/batch-download',
    body: (sns) => ({ serial_numbers: sns }),
    filenamePrefix: 'mr-files',
  },
  mr_files: {
    url: '/mr/files/by-id/batch-download',
    body: (ids) => ({ ids }),
    filenamePrefix: 'mr-files',
  },
};

/**
 * downloadBundle —— 统一入口：POST 后端 + 触发浏览器下载。
 *
 * 参数 targets 含义因 module 不同:
 *   - firmware: firmware_versions.id 字符串列表
 *   - config_snapshot / device_license / mr: 设备 SN 列表
 *
 * 实现要点(吸取之前固件下载的教训):
 *   - 显式 type: 'application/zip' 给 Blob,防止 Chrome 推断 text/html 加 .html 后缀
 *   - 优先解析后端 Content-Disposition 拿真实文件名,降级用前端拼的 {prefix}-{ts}.zip
 *   - timeout: 0 不超时(GB 级打包可能数分钟)
 *   - onProgress 实时回调已接收字节数;后端流式 zip 不提前知道总大小,所以 total
 *     往往是 0,只能展示 "已接收 X MB" 给用户反馈下载在进行
 */
export async function downloadBundle(
  module: BundleModule,
  targets: string[],
  onProgress?: (bytesReceived: number) => void,
): Promise<void> {
  const spec = ENDPOINTS[module];
  // 用原生 fetch + ReadableStream 而不是 axios.responseType:'blob' + onDownloadProgress:
  // XHR 在 chunked-transfer + responseType:'blob' 下,浏览器对 progress event 的
  // 触发频率明显被节流(只在头/尾几次回调,中间长时间静默),导致 UI 进度卡死。
  // fetch + reader.read() 严格每个 chunk 回一次,onProgress 100% 跟随实际接收节奏。
  const { accessToken } = useUserStore.getState();
  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const response = await fetch(`${baseURL}${spec.url}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
    body: JSON.stringify(spec.body(targets)),
  });
  if (!response.ok || !response.body) {
    throw new Error(`HTTP ${response.status}: ${await response.text().catch(() => '')}`);
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    if (value) {
      chunks.push(value);
      received += value.byteLength;
      if (onProgress) onProgress(received);
    }
  }

  // 文件名:优先 Content-Disposition,否则拼默认值
  let filename = `${spec.filenamePrefix}-${formatTimestamp()}.zip`;
  const disposition = response.headers.get('content-disposition');
  if (disposition) {
    const m = /filename\*?=(?:UTF-8''|"?)([^";]+)"?/i.exec(disposition);
    if (m && m[1]) filename = decodeURIComponent(m[1].trim());
  }

  const blob = new Blob(chunks as BlobPart[], { type: 'application/zip' });
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}

function formatTimestamp(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
}
