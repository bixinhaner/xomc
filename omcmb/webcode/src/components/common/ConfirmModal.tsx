import React from 'react';
import { ExclamationCircleFilled, WarningFilled } from '@ant-design/icons';
import { Modal } from 'antd';
import type { ConfirmModalOptions } from './showConfirm';

// showConfirm 命令式工具与 ConfirmModalOptions 已拆到同级 ./showConfirm.tsx
// （react-refresh/only-export-components：本组件文件只导出组件）。
export type { ConfirmModalOptions } from './showConfirm';

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
      destroyOnHidden
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
