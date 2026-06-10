import { useEffect, useRef, useState } from 'react';
import { Avatar, Space, Modal, Form, Input, App } from 'antd';
import {
  UserOutlined,
  LockOutlined,
  TranslationOutlined,
  LogoutOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';
import { useUserStore } from '@core/store/userStore';
import { useAppStore } from '@core/store/appStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { adminApi } from '@core/services/api/adminApi';
import styles from './Header.module.css';

export default function UserDropdown() {
  const popupContainerRef = useRef<HTMLSpanElement>(null);
  const token = useThemeToken();
  const currentUser = useUserStore((s) => s.currentUser);
  const logout = useUserStore((s) => s.logout);
  const toggleLocale = useAppStore((s) => s.toggleLocale);
  const locale = useAppStore((s) => s.locale);
  const t = useT();
  const { message } = App.useApp();
  const [menuOpen, setMenuOpen] = useState(false);
  const [changePwdVisible, setChangePwdVisible] = useState(false);
  const [pwdForm] = Form.useForm();

  const displayName = currentUser?.displayName ?? currentUser?.username ?? t('user.notLoggedIn');
  const avatarText = displayName.charAt(0).toUpperCase();
  const roleName = currentUser?.role ?? '';

  const changePasswordMutation = useMutation({
    mutationFn: (data: { old_password: string; new_password: string }) =>
      adminApi.changePassword(data),
    onSuccess: () => {
      message.success(t('user.passwordChanged'));
      setChangePwdVisible(false);
      pwdForm.resetFields();
      // 欢迎重新登录
      setTimeout(() => logout(), 1500);
    },
    onError: () => {
      message.error(t('common.operationFailed'));
    },
  });

  useEffect(() => {
    if (!menuOpen) {
      return undefined;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!popupContainerRef.current?.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setMenuOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [menuOpen]);

  const handleMenuClick = (key: 'logout' | 'switchLang' | 'changePassword') => {
    setMenuOpen(false);
    if (key === 'logout') {
      logout();
    } else if (key === 'switchLang') {
      toggleLocale();
    } else if (key === 'changePassword') {
      setChangePwdVisible(true);
    }
  };

  const handleChangePwdOk = () => {
    pwdForm.validateFields().then((vals) => {
      changePasswordMutation.mutate({
        old_password: vals.oldPassword as string,
        new_password: vals.newPassword as string,
      });
    });
  };

  return (
    <>
      <span className={styles.userDropdownHost} ref={popupContainerRef}>
        <button
          className={styles.userTrigger}
          type="button"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          {currentUser?.avatar ? (
            <Avatar size={28} src={currentUser.avatar} />
          ) : (
            <Avatar size={28} icon={<UserOutlined />} style={{ backgroundColor: token.colorPrimary }}>
              {avatarText}
            </Avatar>
          )}
          <Space size={2} orientation="vertical" style={{ lineHeight: 1 }}>
            <span className={styles.userName}>{displayName}</span>
          </Space>
          <DownOutlined style={{ fontSize: 10 }} />
        </button>

        {menuOpen && (
          <div
            className={styles.userDropdownMenu}
            role="menu"
            style={{ zIndex: token.zIndexPopupBase + 20 }}
          >
            <div className={styles.userDropdownInfo}>
              <div className={styles.userDropdownName}>{displayName}</div>
              {roleName && <div className={styles.userDropdownMeta}>{roleName}</div>}
              {currentUser?.email && <div className={styles.userDropdownMeta}>{currentUser.email}</div>}
            </div>

            <div className={styles.userDropdownDivider} />

            <button
              className={styles.userDropdownItem}
              type="button"
              role="menuitem"
              onClick={() => handleMenuClick('changePassword')}
            >
              <LockOutlined />
              <span>{t('user.changePassword')}</span>
            </button>

            <button
              className={styles.userDropdownItem}
              type="button"
              role="menuitem"
              onClick={() => handleMenuClick('switchLang')}
            >
              <TranslationOutlined />
              <span>{locale === 'zh-CN' ? t('user.switchToEn') : t('user.switchToZh')}</span>
            </button>

            <div className={styles.userDropdownDivider} />

            <button
              className={`${styles.userDropdownItem} ${styles.userDropdownDanger}`}
              type="button"
              role="menuitem"
              onClick={() => handleMenuClick('logout')}
            >
              <LogoutOutlined />
              <span>{t('user.logout')}</span>
            </button>
          </div>
        )}
      </span>

      {/* 修改密码 Modal */}
      <Modal
        title={t('user.changePassword')}
        open={changePwdVisible}
        onOk={handleChangePwdOk}
        onCancel={() => {
          setChangePwdVisible(false);
          pwdForm.resetFields();
        }}
        confirmLoading={changePasswordMutation.isPending}
        destroyOnHidden
      >
        <Form form={pwdForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="oldPassword"
            label={t('user.oldPassword')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input.Password autoComplete="current-password" />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('user.newPassword')}
            rules={[
              { required: true, message: t('common.pleaseInput') },
              { min: 6, message: t('user.passwordMinLength6') },
            ]}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('user.confirmPassword')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('common.pleaseInput') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(t('user.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
