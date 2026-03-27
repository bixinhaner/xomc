/**
 * ImportPanel - 通用导入面板组件
 *
 * 用于Excel/CSV文件导入，支持：
 * - 文件上传（拖拽或点击）
 * - 下载模板
 * - 文件格式校验
 * - 导入结果展示
 *
 * @example
 * <ImportPanel
 *   ref={importRef}
 *   accept=".xlsx,.xls"
 *   templateUrl="/api/v1/users/import/template"
 *   importUrl="/api/v1/users/import"
 *   onSuccess={() => { refresh(); close(); }}
 * />
 *
 * // 父组件调用导入
 * importRef.current?.handleImport()
 */
import { useState, useCallback, useImperativeHandle, forwardRef } from 'react';
import {
  Button,
  Upload,
  App,
  Alert,
} from 'antd';
import {
  DownloadOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';

export interface ImportPanelRef {
  /** 执行导入操作 */
  handleImport: () => Promise<void>;
  /** 是否有选中的文件可导入 */
  canImport: boolean;
  /** 是否正在导入中 */
  loading: boolean;
}

export interface ImportPanelProps {
  /** 接受的文件格式，如 ".xlsx,.xls" */
  accept?: string;
  /** 最大文件大小（MB），默认 10 */
  maxSizeMB?: number;
  /** 下载模板URL */
  templateUrl?: string;
  /** 导入URL */
  importUrl?: string;
  /** 导入成功回调 */
  onSuccess?: (result: any) => void;
  /** 导入失败回调 */
  onError?: (error: Error) => void;
  /** 自定义导入处理函数（优先于importUrl） */
  onImport?: (file: File) => Promise<any>;
  /** 自定义下载模板函数（优先于templateUrl） */
  onDownloadTemplate?: () => void;
  /** 额外的提示信息 */
  tips?: string[];
  /** 是否显示默认提示 */
  showDefaultTips?: boolean;
}

const ImportPanel = forwardRef<ImportPanelRef, ImportPanelProps>(function ImportPanel({
  accept = '.xlsx,.xls',
  maxSizeMB = 10,
  templateUrl,
  importUrl,
  onSuccess,
  onError,
  onImport,
  onDownloadTemplate,
  tips,
  showDefaultTips = true,
}, ref) {
  const t = useT();
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<File[]>([]);
  const [loading, setLoading] = useState(false);

  const defaultTips = showDefaultTips ? [
    `${t('import.supportFormat')}: ${accept}`,
    `${t('import.maxFileSize')}: ${maxSizeMB}MB`,
  ] : [];

  const allTips = [...defaultTips, ...(tips || [])];

  const handleBeforeUpload = useCallback((file: File) => {
    // 校验文件格式
    const ext = file.name.substring(file.name.lastIndexOf('.')).toLowerCase();
    const acceptExts = accept.split(',').map(e => e.trim().toLowerCase());
    if (!acceptExts.includes(ext)) {
      message.error(t('import.formatError'));
      return false;
    }

    // 校验文件大小
    const sizeMB = file.size / 1024 / 1024;
    if (sizeMB > maxSizeMB) {
      message.error(t('import.sizeError', { max: maxSizeMB }));
      return false;
    }

    setFileList([file]);
    return false;
  }, [accept, maxSizeMB, message, t]);

  const handleRemove = useCallback(() => {
    setFileList([]);
  }, []);

  const handleDownloadTemplate = useCallback(async () => {
    if (onDownloadTemplate) {
      onDownloadTemplate();
      return;
    }

    if (!templateUrl) {
      message.warning(t('import.noTemplate'));
      return;
    }

    try {
      const link = document.createElement('a');
      link.href = templateUrl;
      link.download = '';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      message.success(t('import.downloadSuccess'));
    } catch (error) {
      message.error(t('import.downloadFailed'));
    }
  }, [onDownloadTemplate, templateUrl, message, t]);

  const handleImport = useCallback(async () => {
    if (fileList.length === 0) {
      message.warning(t('import.selectFileFirst'));
      return;
    }

    const file = fileList[0];
    setLoading(true);

    try {
      let result;

      if (onImport) {
        result = await onImport(file);
      } else if (importUrl) {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch(importUrl, {
          method: 'POST',
          body: formData,
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        result = await response.json();
      } else {
        throw new Error('No import handler provided');
      }

      message.success(t('import.success'));
      setFileList([]);
      onSuccess?.(result);
    } catch (error) {
      const err = error as Error;
      message.error(t('import.failed') + ': ' + err.message);
      onError?.(err);
    } finally {
      setLoading(false);
    }
  }, [fileList, onImport, importUrl, onSuccess, onError, message, t]);

  // 暴露方法给父组件
  useImperativeHandle(ref, () => ({
    handleImport,
    canImport: fileList.length > 0,
    loading,
  }), [handleImport, fileList.length, loading]);

  return (
    <div className="import-panel">
      <Upload.Dragger
        accept={accept}
        maxCount={1}
        fileList={fileList.map((f, i) => ({ uid: `${i}`, name: f.name, status: 'done' as const }))}
        beforeUpload={handleBeforeUpload}
        onRemove={handleRemove}
        style={{ marginBottom: 16 }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">{t('import.clickOrDrag')}</p>
        <p className="ant-upload-hint">{t('import.dragHint')}</p>
      </Upload.Dragger>

      {allTips.length > 0 && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message={
            <ul style={{ margin: 0, paddingLeft: 16 }}>
              {allTips.map((tip, index) => (
                <li key={index}>{tip}</li>
              ))}
            </ul>
          }
        />
      )}

      {templateUrl || onDownloadTemplate ? (
        <Button
          type="link"
          icon={<DownloadOutlined />}
          onClick={handleDownloadTemplate}
          style={{ padding: 0 }}
        >
          {t('import.downloadTemplate')}
        </Button>
      ) : null}
    </div>
  );
});

export default ImportPanel;
