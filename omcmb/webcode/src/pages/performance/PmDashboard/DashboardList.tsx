/**
 * T-0164-P6 / G6 PM 仪表盘列表页。
 *
 * 显示当前用户拥有 + 被分享的所有仪表盘；支持：
 *   - 打开（跳到 editor）
 *   - 新建（弹 modal 输入 name + technology）
 *   - 派生（fork 弹 modal 输入 new_name）
 *   - 删除（仅 owner）
 *   - 分享（弹 share dialog，G6 task 10 实现）
 */

import { useState } from 'react';
import { Button, Card, Modal, Form, Input, Select, Space, Table, Tag, message } from 'antd';
import { PlusOutlined, EditOutlined, ForkOutlined, ShareAltOutlined, DeleteOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  usePmDashboardList,
  useCreatePmDashboard,
  useDeletePmDashboard,
  useForkPmDashboard,
} from '@core/hooks/api/usePmDashboard';
import { useUserStore } from '@core/store/userStore';
import type { Dashboard, Technology } from '@core/types/pmDashboard';

const techOptions: { label: string; value: Technology }[] = [
  { label: 'LTE', value: 'lte' },
  { label: '5G NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

export default function PmDashboardList() {
  const navigate = useNavigate();
  const { data: dashboards = [], isLoading } = usePmDashboardList();
  const createMut = useCreatePmDashboard();
  const deleteMut = useDeletePmDashboard();
  const forkMut = useForkPmDashboard();
  const user = useUserStore((s) => s.currentUser);
  const currentUserId = user?.id;

  const [createOpen, setCreateOpen] = useState(false);
  const [forkOpen, setForkOpen] = useState<Dashboard | null>(null);
  const [form] = Form.useForm<{ name: string; technology: Technology; description?: string }>();
  const [forkForm] = Form.useForm<{ newName: string }>();

  const handleCreate = async () => {
    const values = await form.validateFields();
    const d = await createMut.mutateAsync(values);
    message.success('创建成功');
    setCreateOpen(false);
    form.resetFields();
    navigate(`/performance/pm-dashboard/${d.id}`);
  };

  const handleDelete = (id: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '删除后不可恢复（含全部 panel 和分享关系）',
      okType: 'danger',
      onOk: async () => {
        await deleteMut.mutateAsync(id);
        message.success('已删除');
      },
    });
  };

  const handleFork = async () => {
    if (!forkOpen) return;
    const values = await forkForm.validateFields();
    const d = await forkMut.mutateAsync({ sourceId: forkOpen.id, newName: values.newName });
    message.success('已派生');
    setForkOpen(null);
    forkForm.resetFields();
    navigate(`/performance/pm-dashboard/${d.id}`);
  };

  return (
    <Card
      title="性能查看"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新建仪表盘
        </Button>
      }
    >
      <Table<Dashboard>
        rowKey="id"
        dataSource={dashboards}
        loading={isLoading}
        columns={[
          { title: '名称', dataIndex: 'name' },
          {
            title: '制式',
            dataIndex: 'technology',
            width: 100,
            render: (t: Technology) => <Tag color={t === 'nr' ? 'geekblue' : 'green'}>{t.toUpperCase()}</Tag>,
          },
          {
            title: '类型',
            width: 110,
            render: (_, r) => {
              if (r.isBuiltin) return <Tag color="blue">系统内置</Tag>;
              if (currentUserId && r.ownerId === currentUserId) return <Tag>自有</Tag>;
              return <Tag color="orange">来自分享</Tag>;
            },
          },
          {
            title: '分享给',
            dataIndex: 'sharedWith',
            render: (s: string[]) => (s?.length ? `${s.length} 用户` : '—'),
          },
          { title: '更新时间', dataIndex: 'updatedAt' },
          {
            title: '操作',
            width: 280,
            render: (_, r) => {
              const isOwner = currentUserId === r.ownerId;
              const canModify = isOwner && !r.isBuiltin; // G6-Gap-4: 内置 dashboard readonly
              return (
                <Space>
                  <Button size="small" icon={<EditOutlined />} onClick={() => navigate(`/performance/pm-dashboard/${r.id}`)}>
                    打开
                  </Button>
                  <Button size="small" icon={<ForkOutlined />} onClick={() => setForkOpen(r)}>
                    派生
                  </Button>
                  {canModify && (
                    <Button size="small" icon={<ShareAltOutlined />} onClick={() => message.info('分享对话框 G6 task 10')}>
                      分享
                    </Button>
                  )}
                  {canModify && (
                    <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(r.id)}>
                      删除
                    </Button>
                  )}
                </Space>
              );
            },
          },
        ]}
      />

      {/* 新建仪表盘 */}
      <Modal
        title="新建仪表盘"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={createMut.isPending}
      >
        <Form form={form} layout="vertical" initialValues={{ technology: 'lte' as Technology }}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入仪表盘名称' }]}>
            <Input placeholder="如：基础 LTE 性能概览" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label="制式" name="technology" rules={[{ required: true }]}>
            <Select options={techOptions} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 派生（fork） */}
      <Modal
        title={forkOpen ? `派生：${forkOpen.name}` : '派生'}
        open={Boolean(forkOpen)}
        onCancel={() => setForkOpen(null)}
        onOk={handleFork}
        confirmLoading={forkMut.isPending}
      >
        <Form form={forkForm} layout="vertical">
          <Form.Item
            label="新名称"
            name="newName"
            rules={[{ required: true, message: '请输入派生后的新名称' }]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
