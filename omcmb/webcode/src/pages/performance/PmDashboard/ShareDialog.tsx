/**
 * Share dialog — 给 dashboard 分享给其它用户（仅 owner 可见）。
 *
 * 当前列出 sharedWith 已分享用户 + 输入新用户 UUID 加入 / 移除。
 * v2 计划：接 users API 提供下拉选择（避免人工输入 UUID）。
 */

import { useState } from 'react';
import { Button, Form, Input, List, Modal, Space, Tag, message } from 'antd';
import { useSharePmDashboard, useUnsharePmDashboard } from '@core/hooks/api/usePmDashboard';
import type { Dashboard } from '@core/types/pmDashboard';

interface Props {
  open: boolean;
  dashboard: Dashboard;
  onClose: () => void;
}

export function ShareDialog({ open, dashboard, onClose }: Props) {
  const shareMut = useSharePmDashboard();
  const unshareMut = useUnsharePmDashboard();
  const [form] = Form.useForm<{ userId: string }>();
  const [pending, setPending] = useState(false);

  const handleAdd = async () => {
    const { userId } = await form.validateFields();
    setPending(true);
    try {
      await shareMut.mutateAsync({ dashboardId: dashboard.id, userIds: [userId] });
      message.success('已分享');
      form.resetFields();
    } finally {
      setPending(false);
    }
  };

  const handleRemove = async (userId: string) => {
    await unshareMut.mutateAsync({ dashboardId: dashboard.id, userId });
    message.success('已撤销');
  };

  return (
    <Modal
      title={`分享：${dashboard.name}`}
      open={open}
      onCancel={onClose}
      footer={<Button onClick={onClose}>关闭</Button>}
    >
      <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item
          name="userId"
          rules={[{ required: true, pattern: /^[0-9a-f-]{36}$/i, message: '需要合法 UUID' }]}
          style={{ flex: 1 }}
        >
          <Input placeholder="输入用户 UUID（v2：改下拉选择）" />
        </Form.Item>
        <Form.Item>
          <Button type="primary" loading={pending} onClick={handleAdd}>
            分享
          </Button>
        </Form.Item>
      </Form>

      <List
        size="small"
        header={<Space>已分享给 <Tag>{dashboard.sharedWith.length}</Tag></Space>}
        bordered
        dataSource={dashboard.sharedWith}
        renderItem={(uid) => (
          <List.Item
            actions={[
              <Button key="rm" size="small" danger onClick={() => handleRemove(uid)}>
                撤销
              </Button>,
            ]}
          >
            <code>{uid}</code>
          </List.Item>
        )}
      />
    </Modal>
  );
}
