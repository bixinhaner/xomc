import React, { useState } from 'react';
import { Modal, Checkbox, Typography } from 'antd';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface Props {
  open: boolean;
  taskCount: number;
  onClose: () => void;
  onConfirm: (includeSuccess: boolean) => void;
}

export default function BatchRetryDialog({ open, _taskCount, onClose, onConfirm }: Props) {
  const t = useT();
  const [includeSuccess, setIncludeSuccess] = useState(false);

  const handleConfirm = () => {
    onConfirm(includeSuccess);
    setIncludeSuccess(false);
  };

  const handleClose = () => {
    setIncludeSuccess(false);
    onClose();
  };

  return (
    <Modal
      title={t('common.confirm')}
      open={open}
      onCancel={handleClose}
      onOk={handleConfirm}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
    >
      <div style={{ marginBottom: 16 }}>
        <Text>{t('provision.retryTask')}</Text>
      </div>
      <div style={{ marginBottom: 12, color: '#999', fontSize: 12 }}>
        <Text type="secondary">{t('provision.cannotRetryHint')}</Text>
      </div>
      <Checkbox
        checked={includeSuccess}
        onChange={(e) => setIncludeSuccess(e.target.checked)}
      >
        {t('provision.includeSuccessTask')}
      </Checkbox>
    </Modal>
  );
}
