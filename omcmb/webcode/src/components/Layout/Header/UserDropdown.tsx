import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Avatar, Space, Modal, Form, Input, App, Alert, Button } from 'antd';
import {
  UserOutlined,
  LockOutlined,
  LogoutOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';
import { useUserStore } from '@core/store/userStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { adminApi } from '@core/services/api/adminApi';
import styles from './Header.module.css';

export default function UserDropdown() {
  const navigate = useNavigate();
  const popupContainerRef = useRef<HTMLSpanElement>(null);
  const token = useThemeToken();
  const currentUser = useUserStore((s) => s.currentUser);
  const logout = useUserStore((s) => s.logout);
  // Issue #649：后端 must_change_password=true 时，login 页会 setMustChangePassword(true)；
  // 进入首页后本组件唤醒自动弹改密 Modal、且以不可关闭模式呈现。
  const mustChangePassword = useUserStore((s) => s.mustChangePassword);
  const setMustChangePassword = useUserStore((s) => s.setMustChangePassword);
  const t = useT();
  const { message } = App.useApp();
  const [menuOpen, setMenuOpen] = useState(false);
  const [changePwdVisible, setChangePwdVisible] = useState(false);
  const [pwdForm] = Form.useForm();

  // Issue #649：进入首页后若 store 标记还是 true，强制打开改密 Modal。
  useEffect(() => {
    if (mustChangePassword) {
      setChangePwdVisible(true);
    }
  }, [mustChangePassword]);

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
      // 改密成功后，后端已撤销服务端旧 token；前端退出用于清理本地认证状态。
      setMustChangePassword(false);
      // 欢迎重新登录
      setTimeout(() => logout(), 1500);
    },
    onError: (err: Error & { userMessage?: string }) => {
      // Issue #695：显示后端返回的具体错误信息，便于用户了解失败原因
      // http 拦截器会把后端响应的 msg 字段提取到 err.userMessage
      const errorMsg = err.userMessage || err.message || t('common.operationFailed');
      message.error(errorMsg);
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

  const handleManualLogout = () => {
    navigate('/login', { flushSync: true, replace: true, state: null });
    logout();
  };

  const handleMenuClick = (key: 'logout' | 'changePassword') => {
    setMenuOpen(false);
    if (key === 'logout') {
      handleManualLogout();
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

      {/* 修改密码 Modal
          Issue #649：mustChangePassword=true 时走阐塞模式：不可关闭 / 不可点遮罩 /
          不可 ESC / 只有“确认修改”一个按钮，另外提供“退出登录”辅助按钮。 */}
      <Modal
        title={t('user.changePassword')}
        open={changePwdVisible}
        onOk={handleChangePwdOk}
        onCancel={() => {
          if (mustChangePassword) return; // 阐塞模式下不响应关闭
          setChangePwdVisible(false);
          pwdForm.resetFields();
        }}
        confirmLoading={changePasswordMutation.isPending}
        destroyOnHidden
        closable={!mustChangePassword}
        mask={{ closable: !mustChangePassword }}
        keyboard={!mustChangePassword}
        okText={mustChangePassword ? t('login.mustChangePassword.confirm') : undefined}
        cancelButtonProps={mustChangePassword ? { style: { display: 'none' } } : undefined}
        footer={mustChangePassword ? (_close, { OkBtn }) => (
          <Space>
            <Button danger onClick={() => { setMustChangePassword(false); handleManualLogout(); }}>
              {t('user.logout')}
            </Button>
            <OkBtn />
          </Space>
        ) : undefined}
      >
        {mustChangePassword && (
          <Alert
            type="warning"
            showIcon
            message={t('login.mustChangePassword.title')}
            description={t('login.mustChangePassword.content')}
            style={{ marginBottom: 16 }}
          />
        )}
        <div style={{ marginBottom: 16, color: token.colorTextSecondary }}>
          {t('user.currentAccount', { username: currentUser?.username ?? '-' })}
        </div>
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
