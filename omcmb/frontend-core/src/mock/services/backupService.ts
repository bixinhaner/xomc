import type { PageRequest, PageResponse } from '../../types/pagination';
import type { BackupTask, BackupSchedule, FTPConfig } from '../data/backup';
import { mockBackupTasks, mockBackupSchedules, mockFTPConfigs } from '../data/backup';
import { delay, paginate, generateId } from '../utils';

let backupTasks = [...mockBackupTasks];
let schedules = [...mockBackupSchedules];
let ftpConfigs = [...mockFTPConfigs];

export const backupService = {
  async getTasks(
    params: { status?: string; taskType?: string } & PageRequest
  ): Promise<PageResponse<BackupTask>> {
    await delay(100, 200);
    let filtered = [...backupTasks];
    if (params.status) filtered = filtered.filter((t) => t.status === params.status);
    if (params.taskType) filtered = filtered.filter((t) => t.taskType === params.taskType);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getTaskById(id: string): Promise<BackupTask | null> {
    await delay(80, 150);
    return backupTasks.find((t) => t.id === id) ?? null;
  },

  async createTask(data: Omit<BackupTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>): Promise<BackupTask> {
    await delay(200, 400);
    const newItem: BackupTask = {
      ...data,
      id: generateId('bkp'),
      status: 'pending',
      progress: 0,
      successCount: 0,
      failCount: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    backupTasks.push(newItem);
    return newItem;
  },

  async cancelTask(id: string): Promise<void> {
    await delay(150, 300);
    const idx = backupTasks.findIndex((t) => t.id === id);
    if (idx !== -1) {
      backupTasks[idx] = {
        ...backupTasks[idx],
        status: 'cancelled',
        updatedAt: new Date().toISOString(),
      };
    }
  },

  async deleteTasks(ids: string[]): Promise<void> {
    await delay(150, 300);
    backupTasks = backupTasks.filter((t) => !ids.includes(t.id));
  },

  async getSchedules(params: PageRequest): Promise<PageResponse<BackupSchedule>> {
    await delay(80, 150);
    return paginate(schedules, params.page, params.pageSize);
  },

  async createSchedule(data: Omit<BackupSchedule, 'id' | 'createTime'>): Promise<BackupSchedule> {
    await delay(200, 400);
    const newItem: BackupSchedule = {
      ...data,
      id: generateId('sch'),
      createTime: new Date().toISOString(),
    };
    schedules.push(newItem);
    return newItem;
  },

  async updateSchedule(id: string, data: Partial<BackupSchedule>): Promise<BackupSchedule> {
    await delay(150, 300);
    const idx = schedules.findIndex((s) => s.id === id);
    if (idx === -1) throw new Error(`Schedule ${id} not found`);
    schedules[idx] = { ...schedules[idx], ...data };
    return schedules[idx];
  },

  async deleteSchedules(ids: string[]): Promise<void> {
    await delay(150, 300);
    schedules = schedules.filter((s) => !ids.includes(s.id));
  },

  async getFTPConfigs(params: PageRequest): Promise<PageResponse<FTPConfig>> {
    await delay(80, 150);
    return paginate(ftpConfigs, params.page, params.pageSize);
  },

  async createFTPConfig(data: Omit<FTPConfig, 'id' | 'createTime'>): Promise<FTPConfig> {
    await delay(200, 400);
    const newItem: FTPConfig = {
      ...data,
      id: generateId('ftp'),
      createTime: new Date().toISOString(),
    };
    ftpConfigs.push(newItem);
    return newItem;
  },

  async updateFTPConfig(id: string, data: Partial<FTPConfig>): Promise<FTPConfig> {
    await delay(150, 300);
    const idx = ftpConfigs.findIndex((f) => f.id === id);
    if (idx === -1) throw new Error(`FTP config ${id} not found`);
    ftpConfigs[idx] = { ...ftpConfigs[idx], ...data };
    return ftpConfigs[idx];
  },

  async deleteFTPConfigs(ids: string[]): Promise<void> {
    await delay(150, 300);
    ftpConfigs = ftpConfigs.filter((f) => !ids.includes(f.id));
  },

  async testFTPConnection(id: string): Promise<{ success: boolean; message: string }> {
    await delay(1000, 3000);
    void id;
    return { success: Math.random() > 0.2, message: Math.random() > 0.2 ? '连接测试成功' : 'FTP连接超时' };
  },
};
