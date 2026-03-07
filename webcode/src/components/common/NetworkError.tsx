import React from 'react';
import { Result, Button } from 'antd';
import { WifiOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface NetworkErrorProps {
  onRetry?: () => void;
  message?: string;
}

const NetworkError: React.FC<NetworkErrorProps> = ({ onRetry, message }) => {
  const t = useT();
  return (
    <Result
      icon={<WifiOutlined style={{ color: '#FF4D4F' }} />}
      title={t('error.networkError')}
      subTitle={message || t('error.networkErrorDesc')}
      extra={
        onRetry ? (
          <Button type="primary" onClick={onRetry}>
            {t('error.retry')}
          </Button>
        ) : null
      }
    />
  );
};

export default NetworkError;
