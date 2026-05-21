import { useCallback } from 'react';
import type { App as AppNS } from 'antd';
import type { BatchImportResponse } from '@core/types/device';

/**
 * 导出/导入/下载模板小型 handler 集合。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 *
 * 本版本（T-0202）已切换为真实导入：BatchImportModal 内部完成 CSV 解析、
 * 前端校验和后端 POST，handleImport 仅负责消化 BatchImportResponse 与刷列表。
 */
export function useImportExportHandlers(deps: {
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
}) {
  const { message, t, refetch } = deps;

  const handleExport = useCallback(() => {
    // TODO: 调用 CSV 导出 API
    message.success(t('common.exportInProgress'));
  }, [message, t]);

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
