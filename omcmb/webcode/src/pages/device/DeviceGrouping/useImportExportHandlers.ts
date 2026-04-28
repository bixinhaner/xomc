import { useCallback } from 'react';
import type { App as AppNS } from 'antd';
import type { UploadFile } from 'antd';

/**
 * 导出/导入/下载模板小型 handler 集合。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 */
export function useImportExportHandlers(deps: {
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, unknown>) => string;
  refetch: () => Promise<unknown>;
}) {
  const { message, t, refetch } = deps;

  const handleExport = useCallback(() => {
    // TODO: 调用 CSV 导出 API
    message.success(t('common.exportInProgress'));
  }, [message, t]);

  const handleImport = useCallback(
    async (fileList: UploadFile[]) => {
      if (fileList.length === 0) {
        void message.error(t('device.fileRequired'));
        return;
      }
      void message.success(t('device.importSuccess'));
      await refetch();
    },
    [message, t, refetch]
  );

  const handleDownloadTemplate = useCallback(() => {
    const csvContent = 'SN,名称,经度,纬度,高度,备注\n';
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
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
