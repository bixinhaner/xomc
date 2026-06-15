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
      destroyOnHidden
    >
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        <Text>{message}</Text>
        {/* #380: showCount 计数器（.ant-input-data-count）以 bottom:-(fontSize*lineHeight)≈-22px 绝对
            定位渲染到 TextArea 下边缘之外；Antd6 Modal 默认 bodyPadding:0 + footerMarginTop≈12px，
            间隙不足以容纳外溢的计数器，会与底部『确定』按钮重叠。给 TextArea 包裹层补 paddingBottom
            为外溢计数器预留行高空间（等价于 Form.Item 的下边距兜底），消除遮挡。 */}
        <div style={{ paddingBottom: 22 }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {noteLabel || t('alarm.noteLabel')}
          </Text>
          <TextArea {...textAreaProps} />
        </div>
      </Space>
    </Modal>
  );
}
