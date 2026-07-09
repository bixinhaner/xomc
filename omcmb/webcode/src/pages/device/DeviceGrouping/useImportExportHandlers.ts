import { useCallback } from 'react';
import type { App as AppNS } from 'antd';
import type { BatchImportResponse, BatchPreRegisterResponse, Device } from '@core/types/device';
import { deviceApi } from '@core/services/api/deviceApi';
import { EXPORT_COLUMNS, IMPORT_COLUMNS } from './deviceCsvSchema';
import { fetchAllPaged } from '../exportPaging';

type DeviceListParams = Parameters<typeof deviceApi.getList>[0];
type ExportFilterParams = Partial<Omit<DeviceListParams, 'page' | 'pageSize'>>;

export type DeviceGroupExportMode = 'selected' | 'filtered' | 'groupAll';

export interface DeviceGroupExportRequest {
  mode: DeviceGroupExportMode;
  devices?: Device[];
  params?: ExportFilterParams;
}

/**
 * 导出/导入/下载模板小型 handler 集合。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 *
 * - 导入：BatchImportModal 内部完成 CSV 解析、前端校验和后端 POST，
 *         handleImport 仅负责消化 BatchImportResponse 与刷列表。
 * - 导出：按显式 scope 导出已勾选 / 当前搜索结果 / 当前分组全部设备。
 */
export function useImportExportHandlers(deps: {
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
  /** 刷新左侧设备分组树（导入会改变各分组的设备数，需同步刷新树上的「合计」）。 */
  refetchGroups: () => Promise<unknown>;
  /** 当前选中分组名（拼文件名用），undefined 时用「all」。 */
  selectedGroupName?: string;
}) {
  const { message, t, refetch, refetchGroups, selectedGroupName } = deps;

  // selected 模式直接导出已勾选对象；filtered/groupAll 模式按调用方给出的 params 分页拉全量。
  // 列字段与 DeviceListPanel 表格一致。
  const handleExport = useCallback(async (request: DeviceGroupExportRequest) => {
    const msgKey = 'device-grouping-export';
    message.open({ key: msgKey, type: 'loading', content: t('common.exportInProgress'), duration: 0 });
    try {
      let all: Device[] = [];
      let capped = false;
      if (request.mode === 'selected') {
        // 导出选中设备：直接用传入的设备对象，无需再请求后端。
        all = request.devices ?? [];
      } else {
        // 导出筛选结果 / 当前分组全部：分页拉取，调用方负责传入是否包含 searchText。
        let lastProgressAt = 0;
        const result = await fetchAllPaged<Device>(
          async (page, pageSize) => {
            const resp = await deviceApi.getList({
              ...(request.params ?? {}),
              page,
              pageSize,
            } as DeviceListParams);
            return { items: resp.items, total: resp.total ?? resp.items.length };
          },
          {
            onProgress: (loaded, total) => {
              const now = Date.now();
              if (loaded < total && now - lastProgressAt < 300) return;
              lastProgressAt = now;
              message.open({
                key: msgKey,
                type: 'loading',
                content: t('common.exportProgress', { loaded, total }),
                duration: 0,
              });
            },
          },
        );
        all = result.items;
        capped = result.capped;
      }
      if (all.length === 0) {
        message.open({ key: msgKey, type: 'warning', content: t('device.export.emptyGroup') });
        return;
      }

      const csv = buildCsvForDeviceList(all, t);
      const fileName = buildExportFileName(selectedGroupName);
      triggerCsvDownload(csv, fileName);
      message.open({
        key: msgKey,
        type: capped ? 'warning' : 'success',
        content: capped
          ? t('common.exportCapped', { count: all.length })
          : t('common.exportSuccess', { count: all.length }),
      });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      message.open({ key: msgKey, type: 'error', content: t('common.exportFailed', { reason: msg }) });
    }
  }, [message, t, selectedGroupName]);

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
      // 设备列表 + 分组树同步刷新（导入把设备划入当前分组，树上「合计」会变）。
      await Promise.all([refetch(), refetchGroups()]);
    },
    [message, t, refetch, refetchGroups]
  );

  const handleDownloadTemplate = useCallback(() => {
    // 模板表头与导出共享 i18nKey，随当前语言渲染列名；
    // 解析器仍接受中文 + snake_case 两种表头（向后兼容）。
    const required = IMPORT_COLUMNS.filter((c) => c.required).map((c) => t(c.i18nKey)).join(', ');
    const optional = IMPORT_COLUMNS.filter((c) => !c.required).map((c) => t(c.i18nKey)).join(', ');
    const csvComment = t('device.csv.template.comment', { required, optional });
    const csvHeader = IMPORT_COLUMNS.map((c) => csvField(t(c.i18nKey))).join(',');
    const csvExamples = [0, 1, 2].map((rowIdx) =>
      IMPORT_COLUMNS.map((c) => {
        // 优先用 exampleI18nKeys 查 i18n（英文等多语言展示对应语言示例），无 key 则降级用 example。
        const key = c.exampleI18nKeys?.[rowIdx];
        return csvField(key ? t(key) : (c.example[rowIdx] ?? ''));
      }).join(','),
    );
    const csvContent = csvComment + '\n' + csvHeader + '\n' + csvExamples.join('\n') + '\n';
    // UTF-8 BOM (U+FEFF) prefix 让 Excel 正确识别编码。
    const blob = new Blob(['\uFEFF' + csvContent], {
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

  const handlePreRegister = useCallback(
    async (result: BatchPreRegisterResponse) => {
      if (result.failed > 0 && result.created + result.updated === 0) {
        void message.error(t('device.batchPreRegister.allFailed', { failed: result.failed }));
      } else if (result.failed > 0) {
        void message.warning(
          t('device.batchPreRegister.partial', {
            created: result.created,
            updated: result.updated,
            failed: result.failed,
          })
        );
      } else {
        void message.success(
          t('device.batchPreRegister.allSucceeded', { created: result.created, updated: result.updated })
        );
      }
      await Promise.all([refetch(), refetchGroups()]);
    },
    [message, t, refetch, refetchGroups]
  );

  const handleDownloadPreRegisterTemplate = useCallback(() => {
    const comment = t('device.csv.template.preregister.comment');
    const header = [
      t('device.csv.column.sn'),
      t('device.csv.column.deviceName'),
      t('device.csv.column.remark'),
      t('device.csv.column.carrier'),
      t('device.csv.column.technology'),
    ].map(csvField).join(',');
    const examples = [
      ['120288069823C4B0060', '北京海淀中关村站', '一期', '', ''],
      ['1202000690241FB0010', '北京朝阳CBD站', '二期', 'cmcc', 'lte'],
      ['UNKNOWN-OUI-00001', '上海浦东陆家嘴站', '', 'ctcc', 'nr'],
    ].map(row => row.map(csvField).join(',')).join('\n');
    const content = comment + '\n' + header + '\n' + examples + '\n';
    const blob = new Blob(['\uFEFF' + content], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'device_preregister_template.csv';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    void message.success(t('common.download'));
  }, [message, t]);

  return { handleExport, handleImport, handleDownloadTemplate, handlePreRegister, handleDownloadPreRegisterTemplate };
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
  const headers = EXPORT_COLUMNS.map((c) => csvField(t(c.i18nKey))).join(',');
  const rows = devices.map((d) =>
    EXPORT_COLUMNS.map((c) => csvField(c.getValue(d, { t }))).join(','),
  );
  // UTF-8 BOM 前缀让 Excel 正确识别编码。
  return '\uFEFF' + headers + '\n' + rows.join('\n') + '\n';
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
