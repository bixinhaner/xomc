import React from 'react';
import { Alert, Button, Typography } from 'antd';
import { BugOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { isChunkLoadError, reloadOnceForStaleChunk } from '@/utils/staleChunkReload';

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

export interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  onRetry?: () => void;
}

const ErrorFallback: React.FC<{
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
  onRetry: () => void;
}> = ({ error, errorInfo, onRetry }) => {
  const t = useT();

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 48,
        gap: 16,
      }}
    >
      <BugOutlined style={{ fontSize: 48, color: '#F5222D' }} />

      <Typography.Title level={5} style={{ margin: 0, color: '#F5222D' }}>
        {t('error.pageError')}
      </Typography.Title>

      <Alert
        type="error"
        message={error?.message ?? t('error.pageError')}
        description={
          process.env.NODE_ENV === 'development'
            ? errorInfo?.componentStack?.slice(0, 500)
            : t('error.tryRefresh')
        }
        style={{ maxWidth: 560, width: '100%' }}
      />

      <Button type="primary" onClick={onRetry}>
        {t('error.retry')}
      </Button>
    </div>
  );
};

class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // 懒加载分包失败（多见于发布后旧 hash 分包失效）→ 重载一次自愈，
    // 不展示报错 UI（重试按钮对 404 的分包无意义）。带防抖避免重载死循环。
    if (isChunkLoadError(error) && reloadOnceForStaleChunk()) {
      return;
    }
    this.setState({ errorInfo });
    // In production, send to error reporting service
    console.error('[ErrorBoundary] Caught error:', error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
    this.props.onRetry?.();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <ErrorFallback
          error={this.state.error}
          errorInfo={this.state.errorInfo}
          onRetry={this.handleRetry}
        />
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
