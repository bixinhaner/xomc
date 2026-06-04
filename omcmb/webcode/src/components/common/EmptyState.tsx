import React, { useMemo } from 'react';
import { Button, Empty } from 'antd';
import {
  DatabaseOutlined,
  ExclamationCircleOutlined,
  LockOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

export type EmptyVariant = 'no-data' | 'no-results' | 'error' | 'no-permission';

export interface EmptyStateProps {
  variant?: EmptyVariant;
  title?: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
    type?: 'primary' | 'default' | 'dashed' | 'link';
  };
  style?: React.CSSProperties;
}

const EmptyState: React.FC<EmptyStateProps> = ({
  variant = 'no-data',
  title,
  description,
  action,
  style,
}) => {
  const t = useT();
  const token = useThemeToken();

  const VARIANT_CONFIG: Record<
    EmptyVariant,
    {
      icon: React.ReactNode;
      defaultTitle: string;
      defaultDescription: string;
      iconColor: string;
    }
  > = useMemo(
    () => ({
      'no-data': {
        icon: <DatabaseOutlined style={{ fontSize: 48 }} />,
        defaultTitle: t('empty.noData'),
        defaultDescription: t('empty.noDataDesc'),
        iconColor: token.colorTextDisabled,
      },
      'no-results': {
        icon: <SearchOutlined style={{ fontSize: 48 }} />,
        defaultTitle: t('empty.noResult'),
        defaultDescription: t('empty.noResultDesc'),
        iconColor: token.colorTextDisabled,
      },
      error: {
        icon: <ExclamationCircleOutlined style={{ fontSize: 48 }} />,
        defaultTitle: t('empty.loadFailed'),
        defaultDescription: t('empty.loadFailedDesc'),
        iconColor: '#F5222D',
      },
      'no-permission': {
        icon: <LockOutlined style={{ fontSize: 48 }} />,
        defaultTitle: t('empty.noPermission'),
        defaultDescription: t('empty.noPermissionDesc'),
        iconColor: '#FA8C16',
      },
    }),
    [t, token],
  );

  const config = VARIANT_CONFIG[variant];
  const displayTitle = title ?? config.defaultTitle;
  const displayDescription = description ?? config.defaultDescription;

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '48px 24px',
        textAlign: 'center',
        ...style,
      }}
    >
      <div style={{ color: config.iconColor, marginBottom: 16 }}>
        {config.icon}
      </div>

      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        styles={{ image: { display: 'none' } }}
        description={
          <div>
            <div style={{ fontSize: 15, fontWeight: 500, color: token.colorText, marginBottom: 6 }}>
              {displayTitle}
            </div>
            <div style={{ fontSize: 13, color: token.colorTextSecondary }}>
              {displayDescription}
            </div>
          </div>
        }
      >
        {action && (
          <Button
            type={action.type ?? 'primary'}
            onClick={action.onClick}
            style={{ marginTop: 8 }}
          >
            {action.label}
          </Button>
        )}
      </Empty>
    </div>
  );
};

export default EmptyState;
