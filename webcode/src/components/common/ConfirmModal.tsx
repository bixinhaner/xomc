import React from 'react';
import { ExclamationCircleFilled, WarningFilled } from '@ant-design/icons';
import { Modal } from 'antd';

export interface ConfirmModalOptions {
  title: string;
  content?: React.ReactNode;
  okText?: string;
  cancelText?: string;
  onOk: () => void | Promise<void>;
  onCancel?: () => void;
  danger?: boolean;
  width?: number;
  icon?: React.ReactNode;
}

/**
 * Programmatic confirm modal. Call as a function:
 *   showConfirm({ title: '确认删除?', onOk: handleDelete, danger: true })
 */
export function showConfirm(options: ConfirmModalOptions): void {
  const {
    title,
    content,
    okText = '确定',
    cancelText = '取消',
    onOk,
    onCancel,
    danger = false,
    width = 416,
    icon,
  } = options;

  const defaultIcon = danger ? (
    <WarningFilled style={{ color: '#F5222D' }} />
  ) : (
    <ExclamationCircleFilled style={{ color: '#FA8C16' }} />
  );

  Modal.confirm({
    title,
    content,
    icon: icon ?? defaultIcon,
    okText,
    cancelText,
    okType: danger ? 'danger' : 'primary',
    width,
    centered: true,
    onOk: async () => {
      await onOk();
    },
    onCancel,
    okButtonProps: {
      danger,
    },
  });
}

/**
 * Component wrapper for use in JSX contexts where programmatic
 * usage is not preferred.
 */
export interface ConfirmModalProps extends ConfirmModalOptions {
  visible: boolean;
  loading?: boolean;
}

const ConfirmModal: React.FC<ConfirmModalProps> = ({
  visible,
  title,
  content,
  okText = '确定',
  cancelText = '取消',
  onOk,
  onCancel,
  danger = false,
  width = 416,
  icon,
  loading = false,
}) => {
  const defaultIcon = danger ? (
    <WarningFilled style={{ color: '#F5222D', fontSize: 22 }} />
  ) : (
    <ExclamationCircleFilled style={{ color: '#FA8C16', fontSize: 22 }} />
  );

  return (
    <Modal
      open={visible}
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {icon ?? defaultIcon}
          <span>{title}</span>
        </div>
      }
      okText={okText}
      cancelText={cancelText}
      okButtonProps={{ danger, loading }}
      onOk={() => void onOk()}
      onCancel={onCancel}
      width={width}
      centered
      destroyOnClose
    >
      {content && (
        <div style={{ paddingLeft: 32, paddingTop: 4, fontSize: 14 }}>
          {content}
        </div>
      )}
    </Modal>
  );
};

export default ConfirmModal;
