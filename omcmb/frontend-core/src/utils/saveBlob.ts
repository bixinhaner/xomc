/**
 * saveBlob — 浏览器端触发"另存为"下载(三库 XML 下载等共用)。
 *
 * 统一封装 createObjectURL + 隐藏 <a download> 点击 + revoke 清理,
 * 调用方只关心数据与文件名(与 fileApi.download 同范式,抽出复用)。
 */
export function filenameFromContentDisposition(
  disposition: string | undefined,
  fallback: string,
): string {
  if (!disposition) return fallback;

  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(disposition);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1].trim());
    } catch {
      return utf8Match[1].trim();
    }
  }

  const filenameMatch = /filename=(?:"([^"]+)"|([^;]+))/i.exec(disposition);
  const raw = filenameMatch?.[1] ?? filenameMatch?.[2];
  if (!raw) return fallback;

  try {
    return decodeURIComponent(raw.trim());
  } catch {
    return raw.trim();
  }
}

export function saveBlob(
  data: BlobPart,
  filename: string,
  type = 'application/octet-stream',
): void {
  const url = window.URL.createObjectURL(new Blob([data], { type }));
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}
