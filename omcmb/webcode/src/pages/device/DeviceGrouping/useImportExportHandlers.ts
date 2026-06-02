import { useCallback } from 'react';
import type { App as AppNS } from 'antd';
import type { BatchImportResponse, Device } from '@core/types/device';
import { deviceApi } from '@core/services/api/deviceApi';
import { EXPORT_COLUMNS, IMPORT_COLUMNS } from './deviceCsvSchema';

/**
 * 导出/导入/下载模板小型 handler 集合。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 *
 * - 导入：BatchImportModal 内部完成 CSV 解析、前端校验和后端 POST，
 *         handleImport 仅负责消化 BatchImportResponse 与刷列表。
 * - 导出：拉当前分组下全部设备（分页累加），按列表展示的字段写 CSV 下载。
 */
export function useImportExportHandlers(deps: {
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
  /** 当前选中的设备分组 ID；undefined 表示「全部」根节点。 */
  selectedGroupId: string | null;
  /** 当前选中分组名（拼文件名用），undefined 时用「all」。 */
  selectedGroupName?: string;
}) {
  const { message, t, refetch, selectedGroupId, selectedGroupName } = deps;

  const handleExport = useCallback(async () => {
    // 导出当前分组的所有设备，列字段与 DeviceListPanel 表格一致（useDeviceColumns）。
    // 分页拉取避免单次 list 接口结果超大；后端默认 pageSize 上限 200，循环到 total。
    //
    // [device-export] 前缀的 console.info：用户在 DevTools 直接 grep 这一串就能
    // 确认当前部署用的是带本修复的代码（旧 stub 完全没这种日志）。
    console.info('[device-export] start', { groupId: selectedGroupId, group: selectedGroupName });
    const hide = message.loading(t('common.exportInProgress'), 0);
    try {
      const PAGE_SIZE = 500;
      let page = 1;
      const all: Device[] = [];
      let total = 0;
      // 第一页：取 total。
      // groupId 为 null 时不传，等价于「全部」。
      const params = {
        page,
        pageSize: PAGE_SIZE,
        ...(selectedGroupId ? { groupId: selectedGroupId } : {}),
      } as Parameters<typeof deviceApi.getList>[0];
      const first = await deviceApi.getList(params);
      total = first.total ?? first.items.length;
      all.push(...first.items);
      while (all.length < total) {
        page += 1;
        const next = await deviceApi.getList({ ...params, page });
        if (next.items.length === 0) break; // 后端容错：意外提前没数据
        all.push(...next.items);
      }
      console.info('[device-export] fetched', all.length, 'devices, total=', total);
      if (all.length === 0) {
        hide();
        void message.warning(t('device.export.emptyGroup'));
        return;
      }

      const csv = buildCsvForDeviceList(all, t);
      const fileName = buildExportFileName(selectedGroupName);
      triggerCsvDownload(csv, fileName);
      hide();
      void message.success(t('common.exportSuccess', { count: all.length }));
      console.info('[device-export] downloaded', fileName, `(${csv.length} bytes)`);
    } catch (err) {
      hide();
      const msg = err instanceof Error ? err.message : String(err);
      console.error('[device-export] failed', err);
      void message.error(t('common.exportFailed', { reason: msg }));
    }
  }, [message, t, selectedGroupId, selectedGroupName]);

  const handleImport = useCallback(
    async (result: BatchImportResponse) => {
      if (result.failed > 0 && result.succeeded === 0) {
        void message.error(t('device.batchImport.allFailed', { failed: result.failed }));
      } else if (result.failed > 0) {
        void message.warning(
          t('device.batchImport.partial', {
            succeeded: result.succeeded,
            failed: result.failed,
          })
        );
      } else {
        void message.success(
          t('device.batchImport.allSucceeded', { succeeded: result.succeeded })
        );
      }
      await refetch();
    },
    [message, t, refetch]
  );

  const handleDownloadTemplate = useCallback(() => {
    // 模板表头与导出共享中文表头命名（IMPORT_COLUMNS / EXPORT_COLUMNS 共用
    // deviceCsvSchema）；解析器接受中文 + snake_case 两种表头（向后兼容）。
    const required = IMPORT_COLUMNS.filter((c) => c.required).map((c) => c.zh).join(', ');
    const optional = IMPORT_COLUMNS.filter((c) => !c.required).map((c) => c.zh).join(', ');
    // 导入按 SN 匹配已注册设备，更新名称/备注并归入当前分组（不新建设备）。
    const csvComment = `# 必填: ${required} | 可选: ${optional} | 说明: 按 SN 匹配已注册设备，更新名称/备注并归入当前分组；SN 不存在的行会失败`;
    const csvHeader = IMPORT_COLUMNS.map((c) => csvField(c.zh)).join(',');
    const csvExamples = [0, 1, 2].map((rowIdx) =>
      IMPORT_COLUMNS.map((c) => csvField(c.example[rowIdx] ?? '')).join(','),
    );
    const csvContent = csvComment + '\n' + csvHeader + '\n' + csvExamples.join('\n') + '\n';
    // UTF-8 BOM (U+FEFF) prefix 让 Excel 正确识别中文编码。
    const blob = new Blob(['﻿' + csvContent], {
      type: 'text/csv;charset=utf-8;',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'device_import_template.csv';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    void message.success(t('common.download'));
  }, [message, t]);

  return { handleExport, handleImport, handleDownloadTemplate };
}

// ─── CSV 导出工具 ───────────────────────────────────────────────────────────

/** 把一个 CSV 字段值转字符串并按需要加引号转义（含逗号 / 换行 / 引号）。 */
function csvField(value: unknown): string {
  if (value === null || value === undefined) return '';
  const s = typeof value === 'string' ? value : String(value);
  // 包含逗号 / 双引号 / 换行符 → 用双引号包起来，并把双引号 → 双双引号。
  if (s.includes(',') || s.includes('"') || s.includes('\n') || s.includes('\r')) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

/**
 * 按 deviceCsvSchema.EXPORT_COLUMNS 顺序拼 CSV。
 *
 * 列覆盖：
 *   - 设备列表 useDeviceColumns 的全部展示列（连接状态 / 安装状态 / 基站编码 /
 *     基站名称 / MAC地址 / 设备分组 / 归属来源 / 经度 / 纬度 / 高度 / 离线天数 / 备注）
 *   - 加上 OUI / 运营商 / 制式 3 列，让用户直接拿这份 CSV 编辑后回灌也能通过
 *     导入校验（roundtrip 友好）
 *
 * 状态/枚举字段走 i18n 文案。共享字段（基站编码 / OUI / 运营商 / 制式 / ...）
 * 与 IMPORT_COLUMNS 同名同序，让用户体验一致。
 */
function buildCsvForDeviceList(
  devices: Device[],
  t: (id: string, values?: Record<string, string | number>) => string,
): string {
  const headers = EXPORT_COLUMNS.map((c) => csvField(c.zh)).join(',');
  const rows = devices.map((d) =>
    EXPORT_COLUMNS.map((c) => csvField(c.getValue(d, { t }))).join(','),
  );
  // UTF-8 BOM 前缀让 Excel 正确识别中文。
  return '﻿' + headers + '\n' + rows.join('\n') + '\n';
}

function buildExportFileName(selectedGroupName?: string): string {
  const safe = (selectedGroupName ?? 'all').replace(/[\\/:*?"<>|]+/g, '_').slice(0, 40);
  const now = new Date();
  const ts = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}_${String(now.getHours()).padStart(2, '0')}${String(now.getMinutes()).padStart(2, '0')}${String(now.getSeconds()).padStart(2, '0')}`;
  return `devices_${safe}_${ts}.csv`;
}

function triggerCsvDownload(csv: string, fileName: string): void {
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
