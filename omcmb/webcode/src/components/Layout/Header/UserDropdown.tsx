import { useState } from 'react';
import { Avatar, Dropdown, Space, Modal, Form, Input, App } from 'antd';
import type { MenuProps } from 'antd';
import {
  UserOutlined,
  LockOutlined,
  TranslationOutlined,
  LogoutOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';
import { useUserStore } from '@/store/userStore';
import { useAppStore } from '@/store/appStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { adminApi } from '@/services/api/adminApi';
import styles from './Header.module.css';

export default function UserDropdown() {
  const token = useThemeToken();
  const currentUser = useUserStore((s) => s.currentUser);
  const logout = useUserStore((s) => s.logout);
  const toggleLocale = useAppStore((s) => s.toggleLocale);
  const locale = useAppStore((s) => s.locale);
  const t = useT();
  const { message } = App.useApp();
  const [changePwdVisible, setChangePwdVisible] = useState(false);
  const [pwdForm] = Form.useForm();

  const displayName = currentUser?.displayName ?? currentUser?.username ?? t('user.notLoggedIn');
  const avatarText = displayName.charAt(0).toUpperCase();
  const roleName = currentUser?.role ? t(`user.role.${currentUser.role}`) : '';

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

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
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
    <>
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
        destroyOnClose
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
