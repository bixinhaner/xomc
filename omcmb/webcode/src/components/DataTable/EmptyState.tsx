import React from 'react';
import { Button } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import styles from './DataTable.module.css';

interface EmptyStateProps {
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

const EmptyState: React.FC<EmptyStateProps> = ({
  description = '暂无数据',
  action,
}) => {
  return (
    <div className={styles.emptyState}>
      <InboxOutlined className={styles.emptyIcon} />
      <div className={styles.emptyTitle}>{description}</div>
      {action && (
        <Button type="primary" size="small" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  );
};

export default EmptyState;
