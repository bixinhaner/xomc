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
 *
 * 从 ConfirmModal.tsx 拆出（react-refresh/only-export-components：
 * 组件文件只导出组件，命令式工具单独成文件）。
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
