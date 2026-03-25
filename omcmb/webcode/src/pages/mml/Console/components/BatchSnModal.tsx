import { useState, useCallback } from 'react';
import { Button, Input, Modal, Typography } from 'antd';
import { useT } from '@/hooks/useT';

const { TextArea } = Input;

interface BatchSnModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (sns: string[]) => void;
  existingSns: Set<string>;
}

export default function BatchSnModal({
  open,
  onClose,
  onConfirm,
  existingSns,
}: BatchSnModalProps) {
  const t = useT();
  const [inputValue, setInputValue] = useState('');

  // 确认添加
  const handleConfirm = useCallback(() => {
    const sns = inputValue
      .split(/[\n,;]+/)
      .map((s) => s.trim())
      .filter(Boolean);

    if (sns.length === 0) {
      return;
    }

    onConfirm(sns);
    setInputValue('');
    onClose();
  }, [inputValue, onConfirm, onClose]);

  // 关闭弹窗
  const handleClose = useCallback(() => {
    setInputValue('');
    onClose();
  }, [onClose]);

  return (
    <Modal
      title="批量输入设备SN"
      open={open}
      onOk={handleConfirm}
      onCancel={handleClose}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      width={480}
      destroyOnClose
    >
      <div style={{ marginBottom: 12 }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          每行一个SN，或用逗号、分号分隔
        </Typography.Text>
      </div>
      <TextArea
        rows={8}
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        placeholder={'ENB00001\nENB00002\nENB00003'}
        style={{ fontFamily: 'monospace', fontSize: 12 }}
      />
      <div style={{ marginTop: 8 }}>
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          已选设备: {existingSns.size} 台
        </Typography.Text>
      </div>
    </Modal>
  );
}
