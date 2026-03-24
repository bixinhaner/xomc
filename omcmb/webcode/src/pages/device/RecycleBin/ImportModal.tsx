import React, { useCallback, useState } from 'react';
import { Alert, Modal, Progress, Upload, message } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import { CheckCircleOutlined, InboxOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

interface ImportModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  confirmLoading?: boolean;
}

export default function ImportModal({ open, onClose, onConfirm, confirmLoading }: ImportModalProps) {
  const t = useT();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importDone, setImportDone] = useState(false);

  // 重置状态
  const resetState = useCallback(() => {
    setFileList([]);
    setImporting(false);
    setImportProgress(0);
    setImportDone(false);
  }, []);

  // 关闭弹窗
  const handleClose = useCallback(() => {
    resetState();
    onClose();
  }, [resetState, onClose]);

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      // 校验文件格式为 CSV
      const isCsv = file.name.toLowerCase().endsWith('.csv') ||
        file.type === 'text/csv' ||
        file.type === 'application/vnd.ms-excel';

      if (!isCsv) {
        void message.error(t('recycle.importFormatError'));
        return false;
      }

      // 校验文件大小（最大 10MB）
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        void message.error(t('recycle.importSizeError'));
        return false;
      }

      setFileList([file]);
      setImportDone(false);
      return false; // 阻止自动上传
    },
    onRemove: () => {
      setFileList([]);
      setImportDone(false);
    },
  };

  // 执行导入
  const handleImport = useCallback(async () => {
    if (fileList.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }

    setImporting(true);
    setImportProgress(0);

    // 模拟导入进度
    for (let p = 0; p <= 100; p += 10) {
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
      setImportProgress(p);
    }

    setImporting(false);
    setImportDone(true);
    void message.success(t('status.success'));
  }, [fileList, t]);

  // 确认完成
  const handleOk = useCallback(() => {
    if (importDone) {
      onConfirm();
      handleClose();
    } else {
      void handleImport();
    }
  }, [importDone, onConfirm, handleClose, handleImport]);

  return (
    <Modal
      title={t('device.batchImport')}
      open={open}
      onOk={handleOk}
      onCancel={handleClose}
      okText={importDone ? t('common.finish') : t('common.import')}
      cancelText={importDone ? t('common.close') : t('common.cancel')}
      confirmLoading={confirmLoading || importing}
      width={520}
      destroyOnClose
    >
      <Dragger {...uploadProps} disabled={importing || importDone}>
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
        </p>
        <p className="ant-upload-text">{t('recycle.importSelectFile')}</p>
        <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
          {t('recycle.importCsvOnly')}
        </p>
      </Dragger>

      {importing && (
        <div style={{ marginTop: 16 }}>
          <span style={{ fontSize: 13, color: '#666' }}>{t('common.loading')}</span>
          <Progress percent={importProgress} status="active" />
        </div>
      )}

      {importDone && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircleOutlined />}
          message={t('recycle.importSuccess')}
          style={{ marginTop: 16 }}
        />
      )}
    </Modal>
  );
}
