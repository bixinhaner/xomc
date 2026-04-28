import { useCallback, useState } from 'react';
import { Alert, Button, Modal, Progress, Typography, Upload } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import { CheckCircleOutlined, DownloadOutlined, InboxOutlined, UploadOutlined } from '@ant-design/icons';

const { Dragger } = Upload;
const { Text } = Typography;

export interface BatchImportModalProps {
  open: boolean;
  onClose: () => void;
  onImport: (fileList: UploadFile[]) => void;
  onDownloadTemplate: () => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

/**
 * 设备分组 - 批量导入弹窗
 * 拆分自 DeviceListPanel 以保持单文件 ≤ 400 行（W2.C.2 / T-0054）。
 */
export default function BatchImportModal({
  open,
  onClose,
  onImport,
  onDownloadTemplate,
  t,
}: BatchImportModalProps) {
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importDone, setImportDone] = useState(false);

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      if (!file.name.endsWith('.csv')) {
        void Modal.error({ title: t('common.error'), content: t('device.fileFormatError') });
        return false;
      }
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        void Modal.error({ title: t('common.error'), content: t('device.fileSizeError') });
        return false;
      }
      setFileList([file]);
      setImportDone(false);
      return false;
    },
    onRemove: () => {
      setFileList([]);
      setImportDone(false);
    },
  };

  const handleImportConfirm = useCallback(async () => {
    if (fileList.length === 0) {
      void Modal.warning({ title: t('common.warning'), content: t('device.selectFileFirst') });
      return;
    }
    setImporting(true);
    setImportProgress(0);
    // 模拟导入进度
    for (let p = 0; p <= 100; p += 10) {
      await new Promise<void>((resolve) => setTimeout(resolve, 150));
      setImportProgress(p);
    }
    setImporting(false);
    setImportDone(true);
    onImport(fileList);
    // 延迟关闭弹窗，让用户看到成功提示
    await new Promise<void>((resolve) => setTimeout(resolve, 1000));
    onClose();
  }, [fileList, onImport, onClose, t]);

  const handleCancel = useCallback(() => {
    if (!importing) {
      onClose();
    }
  }, [importing, onClose]);

  return (
    <Modal
      title={t('common.batchImport')}
      open={open}
      onCancel={handleCancel}
      footer={null}
      width={520}
      maskClosable={!importing}
      closable={!importing}
    >
      <div style={{ marginBottom: 12 }}>
        <Button
          icon={<DownloadOutlined />}
          size="small"
          onClick={onDownloadTemplate}
        >
          {t('device.downloadImportTemplate')}
        </Button>
      </div>

      <Dragger {...uploadProps} style={{ marginBottom: 16 }}>
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
        </p>
        <p className="ant-upload-text">{t('common.upload')}</p>
        <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
          .csv
        </p>
      </Dragger>

      {importing && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t('common.loading')}
          </Text>
          <Progress percent={importProgress} status="active" />
        </div>
      )}

      {importDone && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircleOutlined />}
          message={t('device.importSuccess')}
          style={{ marginBottom: 16 }}
        />
      )}

      <Button
        type="primary"
        icon={<UploadOutlined />}
        loading={importing}
        onClick={() => void handleImportConfirm()}
        disabled={fileList.length === 0}
        block
      >
        {t('common.import')}
      </Button>
    </Modal>
  );
}
