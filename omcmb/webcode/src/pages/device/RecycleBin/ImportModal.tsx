import { useCallback, useState } from 'react';
import { Modal, Upload, message } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
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

  // 重置状态
  const resetState = useCallback(() => {
    setFileList([]);
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
      return false; // 阻止自动上传
    },
    onRemove: () => {
      setFileList([]);
    },
  };

  // 执行导入
  const handleImport = useCallback(async () => {
    if (fileList.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }

    // 直接提示导入成功并关闭弹窗
    void message.success(t('recycle.importSuccess'));
    onConfirm();
    handleClose();
  }, [fileList, t, onConfirm, handleClose]);

  return (
    <Modal
      title={t('device.batchImport')}
      open={open}
      onOk={handleImport}
      onCancel={handleClose}
      okText={t('common.import')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={520}
      destroyOnClose
    >
      <Dragger {...uploadProps}>
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
        </p>
        <p className="ant-upload-text">{t('recycle.importSelectFile')}</p>
        <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
          {t('recycle.importCsvOnly')}
        </p>
      </Dragger>
    </Modal>
  );
}
