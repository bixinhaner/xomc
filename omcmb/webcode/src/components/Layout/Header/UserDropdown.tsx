import { Avatar, Dropdown, Space } from 'antd';
import type { MenuProps } from 'antd';
import {
  UserOutlined,
  LockOutlined,
  TranslationOutlined,
  LogoutOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { useUserStore } from '@/store/userStore';
import { useAppStore } from '@/store/appStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import styles from './Header.module.css';

export default function UserDropdown() {
  const token = useThemeToken();
  const currentUser = useUserStore((s) => s.currentUser);
  const logout = useUserStore((s) => s.logout);
  const toggleLocale = useAppStore((s) => s.toggleLocale);
  const locale = useAppStore((s) => s.locale);
  const t = useT();

  const displayName = currentUser?.displayName ?? currentUser?.username ?? t('user.notLoggedIn');
  const avatarText = displayName.charAt(0).toUpperCase();
  const roleName = currentUser?.role ? t(`user.role.${currentUser.role}`) : '';

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') {
      logout();
    } else if (key === 'switchLang') {
      toggleLocale();
    } else if (key === 'changePassword') {
      // Navigate to change password page — placeholder
    }
  };

  const menuItems: MenuProps['items'] = [
    {
      key: 'userInfo',
      label: (
        <div style={{ padding: '4px 0' }}>
          <div style={{ fontWeight: 600, fontSize: 14, color: token.colorTextHeading }}>{displayName}</div>
          {roleName && (
            <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 2 }}>{roleName}</div>
          )}
          {currentUser?.email && (
            <div style={{ fontSize: 12, color: token.colorTextSecondary }}>{currentUser.email}</div>
          )}
        </div>
      ),
      disabled: true,
    },
    { type: 'divider' },
    {
      key: 'changePassword',
      icon: <LockOutlined />,
      label: t('user.changePassword'),
    },
    {
      key: 'switchLang',
      icon: <TranslationOutlined />,
      label: locale === 'zh-CN' ? t('user.switchToEn') : t('user.switchToZh'),
    },
    { type: 'divider' },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('user.logout'),
      danger: true,
    },
  ];

  return (
    <Dropdown
      menu={{ items: menuItems, onClick: handleMenuClick }}
      trigger={['click']}
      placement="bottomRight"
    >
      <div className={styles.userTrigger}>
        {currentUser?.avatar ? (
          <Avatar size={28} src={currentUser.avatar} />
        ) : (
          <Avatar size={28} icon={<UserOutlined />} style={{ backgroundColor: token.colorPrimary }}>
            {avatarText}
          </Avatar>
        )}
        <Space size={2} direction="vertical" style={{ lineHeight: 1 }}>
          <span className={styles.userName}>{displayName}</span>
        </Space>
        <DownOutlined style={{ fontSize: 10 }} />
      </div>
    </Dropdown>
  );
}
