import { Button, Result } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useT } from '@/hooks/useT';

export default function NotFound() {
  const t = useT();
  const navigate = useNavigate();

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        minHeight: 400,
      }}
    >
      <Result
        status="404"
        title="404"
        subTitle={t('error.404.message')}
        extra={
          <Button
            type="primary"
            onClick={() => navigate('/dashboard', { replace: true })}
          >
            {t('error.backHome')}
          </Button>
        }
      />
    </div>
  );
}
