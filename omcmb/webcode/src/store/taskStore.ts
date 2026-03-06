import { create } from 'zustand';
import type { TaskStatus, SingleTask, BatchTask, ExportTask } from '../types/task';

export type { TaskStatus, SingleTask, BatchTask, ExportTask };
export type TaskType = 'config' | 'firmware' | 'backup' | 'restore' | 'command' | 'export' | 'import';

interface TaskState {
  tasks: SingleTask[];
  batchTasks: BatchTask[];
  exportTasks: ExportTask[];
  panelExpanded: boolean;
  activeTab: 'single' | 'batch' | 'export';

  // Single tasks
  addTask: (task: SingleTask) => void;
  updateTask: (id: string, updates: Partial<SingleTask>) => void;
  removeTask: (id: string) => void;

  // Batch tasks
  addBatchTask: (task: BatchTask) => void;
  updateBatchTask: (id: string, updates: Partial<BatchTask>) => void;
  removeBatchTask: (id: string) => void;

  // Export tasks
  addExportTask: (task: ExportTask) => void;
  updateExportTask: (id: string, updates: Partial<ExportTask>) => void;
  removeExportTask: (id: string) => void;

  // Panel
  togglePanel: () => void;
  setPanelExpanded: (expanded: boolean) => void;
  setActiveTab: (tab: 'single' | 'batch' | 'export') => void;

  // Legacy aliases
  singleTasks: SingleTask[];
  addSingleTask: (task: SingleTask) => void;
  updateSingleTask: (id: string, updates: Partial<SingleTask>) => void;
  removeSingleTask: (id: string) => void;
}

export const useTaskStore = create<TaskState>()((set, get) => ({
  tasks: [],
  singleTasks: [],
  batchTasks: [],
  exportTasks: [],
  panelExpanded: false,
  activeTab: 'single',

  addTask: (task) =>
    set((state) => ({ tasks: [...state.tasks, task], singleTasks: [...state.singleTasks, task] })),

  updateTask: (id, updates) =>
    set((state) => ({
      tasks: state.tasks.map((t) => (t.id === id ? { ...t, ...updates } : t)),
      singleTasks: state.singleTasks.map((t) => (t.id === id ? { ...t, ...updates } : t)),
    })),

  removeTask: (id) =>
    set((state) => ({
      tasks: state.tasks.filter((t) => t.id !== id),
      singleTasks: state.singleTasks.filter((t) => t.id !== id),
    })),

  addSingleTask: (task) => get().addTask(task),
  updateSingleTask: (id, updates) => get().updateTask(id, updates),
  removeSingleTask: (id) => get().removeTask(id),

  addBatchTask: (task) =>
    set((state) => ({ batchTasks: [...state.batchTasks, task] })),
  updateBatchTask: (id, updates) =>
    set((state) => ({
      batchTasks: state.batchTasks.map((t) => (t.id === id ? { ...t, ...updates } : t)),
    })),
  removeBatchTask: (id) =>
    set((state) => ({ batchTasks: state.batchTasks.filter((t) => t.id !== id) })),

  addExportTask: (task) =>
    set((state) => ({ exportTasks: [...state.exportTasks, task] })),
  updateExportTask: (id, updates) =>
    set((state) => ({
      exportTasks: state.exportTasks.map((t) => (t.id === id ? { ...t, ...updates } : t)),
    })),
  removeExportTask: (id) =>
    set((state) => ({ exportTasks: state.exportTasks.filter((t) => t.id !== id) })),

  togglePanel: () => set((state) => ({ panelExpanded: !state.panelExpanded })),
  setPanelExpanded: (expanded) => set({ panelExpanded: expanded }),
  setActiveTab: (tab) => set({ activeTab: tab }),
}));
