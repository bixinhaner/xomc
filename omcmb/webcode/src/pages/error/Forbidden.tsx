import { Button, Result, Space } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useT } from '@/hooks/useT';
import { useUserStore } from '@core/store/userStore';

/**
 * 403 Forbidden 页面（PRD docs/prd/system/menu-dynamic-loading.md §4.3.5）。
 *
 * 行为约定：
 *   - 主操作"返回首页"→ /dashboard（默认登录跳转目标，所有角色可达）
 *   - 次操作"重新登录"→ logout + /login（角色调整后立即拉新菜单）
 *   - 副标题含"联系管理员加菜单权限"指引，引导用户行动
 *   - **故意不显示当前 path 字符串**，避免暴露未授权资源命名（CSRF / 信息收集）
 */
export default function Forbidden() {
  const t = useT();
  const navigate = useNavigate();
  const logout = useUserStore((s) => s.logout);

  const handleRelogin = () => {
    logout();
    navigate('/login', { replace: true });
  };

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
        status="403"
        title="403"
        subTitle={
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span>{t('error.403.message')}</span>
            <span style={{ color: 'var(--color-text-tertiary)' }}>
              {t('error.403.contactAdmin')}
            </span>
          </div>
        }
        extra={
          <Space>
            <Button
              type="primary"
              onClick={() => navigate('/dashboard', { replace: true })}
            >
              {t('error.backHome')}
            </Button>
            <Button onClick={handleRelogin}>{t('error.relogin')}</Button>
          </Space>
        }
      />
    </div>
  );
}
