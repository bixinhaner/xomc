import type { PageRequest, PageResponse } from '../../types/pagination';
import type { ManagedFile, FileType, FileStatus } from '../data/fileManagement';
import { mockFiles } from '../data/fileManagement';
import { delay, paginate, generateId } from '../utils';

let files = [...mockFiles];

export const fileService = {
  async getList(
    params: { fileType?: FileType; status?: FileStatus; keyword?: string; deviceSn?: string } & PageRequest
  ): Promise<PageResponse<ManagedFile>> {
    await delay(100, 200);
    let filtered = [...files];
    if (params.fileType) filtered = filtered.filter((f) => f.fileType === params.fileType);
    if (params.status) filtered = filtered.filter((f) => f.status === params.status);
    if (params.deviceSn) filtered = filtered.filter((f) => f.deviceSn === params.deviceSn);
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = filtered.filter(
        (f) =>
          f.fileName.toLowerCase().includes(kw) ||
          (f.description ?? '').toLowerCase().includes(kw)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getById(id: string): Promise<ManagedFile | null> {
    await delay(80, 150);
    return files.find((f) => f.id === id) ?? null;
  },

  async upload(data: Omit<ManagedFile, 'id' | 'uploadTime'>): Promise<ManagedFile> {
    await delay(500, 2000);
    const newItem: ManagedFile = {
      ...data,
      id: generateId('file'),
      uploadTime: new Date().toISOString(),
    };
    files.push(newItem);
    return newItem;
  },

  async delete(ids: string[]): Promise<void> {
    await delay(150, 300);
    files = files.filter((f) => !ids.includes(f.id));
  },

  async download(id: string): Promise<{ url: string; fileName: string }> {
    await delay(200, 500);
    const file = files.find((f) => f.id === id);
    if (!file) throw new Error(`File ${id} not found`);
    return { url: file.downloadUrl, fileName: file.fileName };
  },

  async distribute(_fileId: string, _deviceSns: string[]): Promise<{ taskId: string }> {
    await delay(300, 600);
    return { taskId: generateId('dist') };
  },

  async getStorageStats(): Promise<{ totalSize: number; usedSize: number; fileCount: number; byType: Record<string, number> }> {
    await delay(80, 150);
    const usedSize = files.reduce((sum, f) => sum + f.fileSize, 0);
    const byType: Record<string, number> = {};
    for (const f of files) {
      byType[f.fileType] = (byType[f.fileType] ?? 0) + f.fileSize;
    }
    return {
      totalSize: 1024 * 1024 * 1024 * 10,
      usedSize,
      fileCount: files.length,
      byType,
    };
  },
};
