import { useState, useCallback, useEffect, type ChangeEvent } from 'react';
import { Input, Modal, Space, Typography } from 'antd';
import type { TextAreaProps } from 'antd/es/input/TextArea';
import { useT } from '@/hooks/useT';

const { TextArea } = Input;
const { Text } = Typography;

export interface ConfirmWithNoteModalProps {
  open: boolean;
  title: string;
  message: string;
  noteLabel?: string;
  notePlaceholder?: string;
  confirmText?: string;
  confirmType?: 'primary' | 'danger';
  loading?: boolean;
  onConfirm: (note: string) => Promise<void> | void;
  onCancel: () => void;
}

export default function ConfirmWithNoteModal({
  open,
  title,
  message,
  noteLabel,
  notePlaceholder,
  confirmText,
  confirmType = 'primary',
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmWithNoteModalProps) {
  const t = useT();
  const [note, setNote] = useState('');

  // 重置描述
  useEffect(() => {
    if (!open) {
      setNote('');
    }
  }, [open]);

  const handleConfirm = useCallback(async () => {
    await onConfirm(note);
  }, [note, onConfirm]);

  const handleCancel = useCallback(() => {
    setNote('');
    onCancel();
  }, [onCancel]);

  const textAreaProps: TextAreaProps = {
    value: note,
    onChange: (e: ChangeEvent<HTMLTextAreaElement>) => setNote(e.target.value),
    placeholder: notePlaceholder || t('alarm.notePlaceholder'),
    rows: 3,
    maxLength: 500,
    showCount: true,
  };

  return (
    <Modal
      title={title}
      open={open}
      onOk={handleConfirm}
      onCancel={handleCancel}
      okText={confirmText || t('common.confirm')}
      okType={confirmType}
      cancelText={t('common.cancel')}
      confirmLoading={loading}
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Text>{message}</Text>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {noteLabel || t('alarm.noteLabel')}
          </Text>
          <TextArea {...textAreaProps} />
        </div>
      </Space>
    </Modal>
  );
}
