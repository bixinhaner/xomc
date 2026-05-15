export type TaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';

export interface SingleTask {
  id: string;
  neName: string;
  neSn: string;
  type: string;
  currentStep: number;
  totalSteps: number;
  status: TaskStatus;
  message: string;
  progress: number;
  createdAt: string;
  updatedAt: string;
}

export interface BatchTask {
  id: string;
  taskName: string;
  type: string;
  totalCount: number;
  successCount: number;
  failCount: number;
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
}

export interface ExportTask {
  id: string;
  taskType: string;
  fileName: string;
  fileSize?: number;
  status: TaskStatus;
  executor: string;
  startTime: string;
  endTime?: string;
  resultDetail: string;
  /** 导出结果下载链接（MinIO presigned URL，仅 success 状态有值） */
  downloadUrl?: string;
}
