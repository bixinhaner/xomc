import { useCallback } from 'react';
import type { App as AppNS } from 'antd';
import type { BatchImportResponse, Device, EngStatus } from '@core/types/device';
import { deviceApi } from '@core/services/api/deviceApi';

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
      if (all.length === 0) {
        hide();
        void message.warning(t('common.noData'));
        return;
      }

      const csv = buildCsvForDeviceList(all, t);
      const fileName = buildExportFileName(selectedGroupName);
      triggerCsvDownload(csv, fileName);
      hide();
      void message.success(t('common.exportSuccess', { count: all.length }));
    } catch (err) {
      hide();
      const msg = err instanceof Error ? err.message : String(err);
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
    // header 字段严格对齐 BatchImportDevice（snake_case wire 名）。
    // 必填: serial_number / oui / carrier / technology
    // 可选: product_class / device_name / site_id / ip_address / longitude / latitude
    const csvComment =
      '# 必填: serial_number, oui, carrier (cmcc|ctcc|cucc), technology (lte|nr) | 可选: product_class, device_name, site_id, ip_address, longitude, latitude';
    const csvHeader =
      'serial_number,oui,carrier,technology,product_class,device_name,site_id,ip_address,longitude,latitude';
    const csvExamples = [
      'BAI-LTE-202604001,001E91,cmcc,lte,X100W,北京海淀中关村站,SITE-BJ-001,,116.310316,39.992177',
      'BAI-LTE-202604002,001E91,cmcc,lte,X100W,北京朝阳CBD站,SITE-BJ-002,,116.460886,39.914585',
      'BAI-LTE-202604003,001E91,cmcc,lte,X200W,上海浦东陆家嘴站,SITE-SH-001,,121.504602,31.238068',
    ];
    const csvContent =
      csvComment + '\n' + csvHeader + '\n' + csvExamples.join('\n') + '\n';
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

const ENG_STATUS_LABEL_KEYS: Record<EngStatus, string> = {
  commissioned: 'device.engStatus.commissioned',
  uncommissioned: 'device.engStatus.uncommissioned',
  decommissioned: 'device.engStatus.decommissioned',
};

const SOURCE_TYPE_LABEL_KEYS = {
  manual: 'device.sourceType.manual',
  rule: 'device.sourceType.rule',
} as const;

function calcOfflineDays(lastOnlineTime: string | undefined): number {
  if (!lastOnlineTime) return 0;
  const lastOnline = new Date(lastOnlineTime);
  if (Number.isNaN(lastOnline.getTime())) return 0;
  const diff = Date.now() - lastOnline.getTime();
  return Math.max(0, Math.floor(diff / (1000 * 60 * 60 * 24)));
}

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
 * 按 DeviceListPanel 列表展示字段（useDeviceColumns）拼一份 CSV。
 *
 * 列顺序与表格一致；状态/枚举字段用 i18n 文案，不下钻原始 enum 值。
 *   连接状态 / 安装状态 / 序列号 / 设备名称 / MAC地址 / 设备分组 /
 *   归属来源 / 经度 / 纬度 / 高度 / 离线天数 / 备注
 *
 * 操作列（Edit 按钮）按惯例不导出。
 */
function buildCsvForDeviceList(
  devices: Device[],
  t: (id: string, values?: Record<string, string | number>) => string,
): string {
  const headers = [
    t('device.connStatus'),
    t('device.installStatus'),
    t('device.serialNumber'),
    t('device.stationName'),
    t('device.macAddress'),
    t('device.groupName'),
    t('device.sourceType'),
    t('device.longitude'),
    t('device.latitude'),
    t('device.height'),
    t('device.offlineDays'),
    t('device.remark'),
  ];

  const rows = devices.map((d) => {
    const connText =
      d.connStatus === 'online' ? t('status.online') : t('status.offline');
    const engKey = ENG_STATUS_LABEL_KEYS[d.engStatus as EngStatus];
    const engText = engKey ? t(engKey) : String(d.engStatus ?? '');
    const src = (d.sourceType ?? 'manual') as keyof typeof SOURCE_TYPE_LABEL_KEYS;
    const sourceText = SOURCE_TYPE_LABEL_KEYS[src]
      ? t(SOURCE_TYPE_LABEL_KEYS[src])
      : String(d.sourceType ?? '');
    const offlineDays = d.connStatus === 'online' ? '-' : calcOfflineDays(d.lastOnlineTime);

    return [
      connText,
      engText,
      d.sn ?? '',
      d.name ?? '',
      d.macAddress ?? '',
      d.groupName ?? '',
      sourceText,
      d.longitude ?? '',
      d.latitude ?? '',
      d.gpsHeight ?? '',
      offlineDays,
      d.remark ?? '',
    ]
      .map(csvField)
      .join(',');
  });

  // UTF-8 BOM 前缀让 Excel 正确识别中文。
  return '﻿' + headers.map(csvField).join(',') + '\n' + rows.join('\n') + '\n';
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
