/**
 * saveBlob — 浏览器端触发"另存为"下载(三库 XML 下载等共用)。
 *
 * 统一封装 createObjectURL + 隐藏 <a download> 点击 + revoke 清理,
 * 调用方只关心数据与文件名(与 fileApi.download 同范式,抽出复用)。
 */
export function saveBlob(data: BlobPart, filename: string): void {
  const url = window.URL.createObjectURL(new Blob([data]));
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}
