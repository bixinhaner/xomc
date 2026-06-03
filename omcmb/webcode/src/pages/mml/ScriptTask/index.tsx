import { useState, useMemo, useCallback, useEffect } from 'react';
import { Button, Form, Input, Modal, Select, Space, Tag, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import SearchInput from '@/components/SearchInput';
import ScriptTaskDrawer from '../components/ScriptTaskDrawer';
import { useT } from '@/hooks/useT';

import type { MMLScript } from '@core/types/mml';
import {
  useMMLScripts,
  useCreateMMLScript,
  useUpdateMMLScript,
  useDeleteMMLScripts,
} from '@core/hooks/api/useMML';
import { useUserStore } from '@core/store/userStore';

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

// 新增/编辑脚本弹窗的表单结构 —— 字段与后端 createScript/updateScript
// 实际接收的列对齐（script_name / description / content / tags）。
interface ScriptForm {
  scriptName: string;
  description?: string;
  tags?: string[];
  content: string;
}

// 脚本任务（mml/script）：脚本库列表，读 mml_scripts。
// 支持新增 / 编辑 / 删除脚本（POST、PUT、DELETE /mml/scripts）。
// 任务执行记录（mml_tasks）由独立页面 mml/task-records 承载。
export default function ScriptTask() {
  const t = useT();
  const username = useUserStore((s) => s.currentUser?.username) ?? '';

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [search, setSearch] = useState('');

  const { data, isLoading, refetch } = useMMLScripts({
    page,
    pageSize,
    search: search.trim() || undefined,
  });
  const createScriptMutation = useCreateMMLScript();
  const updateScriptMutation = useUpdateMMLScript();
  const deleteScriptsMutation = useDeleteMMLScripts();

  const scripts = useMemo(() => data?.items ?? [], [data]);

  const [viewing, setViewing] = useState<MMLScript | null>(null);

  // 执行脚本：打开 ScriptTaskDrawer 预填该脚本内容，由用户选设备 + 执行方式
  // （立即=手动执行 / 定时 / 周期=自动执行）后提交。提交即 POST /mml/tasks，
  // 由后端 scheduler 调度，每次执行在 mml_tasks 落一条任务记录。
  const [execScript, setExecScript] = useState<MMLScript | null>(null);

  // ---- 新增/编辑脚本弹窗 ----------------------------------------------------
  const [editing, setEditing] = useState<MMLScript | null>(null);
  const [formVisible, setFormVisible] = useState(false);
  const [form] = Form.useForm<ScriptForm>();

  // 弹窗打开后再回填表单：Modal 子节点惰性挂载，openEdit 时 Form 实例可能尚未连接，
  // 因此把回填放进 formVisible 的副作用里，确保 Form 已挂载。
  useEffect(() => {
    if (!formVisible) return;
    if (editing) {
      form.setFieldsValue({
        scriptName: editing.scriptName,
        description: editing.description,
        tags: editing.tags,
        content: editing.content,
      });
    } else {
      form.resetFields();
    }
  }, [formVisible, editing, form]);

  const openCreate = useCallback(() => {
    setEditing(null);
    setFormVisible(true);
  }, []);

  const openEdit = useCallback((script: MMLScript) => {
    setEditing(script);
    setFormVisible(true);
  }, []);

  const closeForm = useCallback(() => {
    setFormVisible(false);
    setEditing(null);
    form.resetFields();
  }, [form]);

  const handleSave = useCallback(() => {
    form
      .validateFields()
      .then((vals) => {
        const onDone = () => {
          void message.success(t('mml.scriptSaved'));
          closeForm();
        };
        const onFail = (err: unknown) =>
          void message.error(
            t('mml.scriptSaveFailed', {
              error: err instanceof Error ? err.message : 'Unknown',
            })
          );
        if (editing) {
          updateScriptMutation.mutate(
            {
              id: editing.id,
              data: {
                scriptName: vals.scriptName.trim(),
                description: vals.description ?? '',
                content: vals.content,
                tags: vals.tags ?? [],
              },
            },
            { onSuccess: onDone, onError: onFail }
          );
        } else {
          createScriptMutation.mutate(
            {
              scriptName: vals.scriptName.trim(),
              description: vals.description ?? '',
              content: vals.content,
              tags: vals.tags ?? [],
              creator: username,
              status: 'active',
              type: 'manual',
              progress: 0,
            },
            { onSuccess: onDone, onError: onFail }
          );
        }
      })
      .catch(() => undefined);
  }, [form, editing, username, createScriptMutation, updateScriptMutation, closeForm, t]);

  // 删除脚本：敏感操作，走 Modal.confirm 二次确认。
  const handleDelete = useCallback((script: MMLScript) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          deleteScriptsMutation.mutate([script.id], {
            onSuccess: () => {
              void message.success(t('common.deleteSuccess'));
              resolve();
            },
            onError: (err) => {
              void message.error(err instanceof Error ? err.message : 'Unknown');
              reject(err);
            },
          });
        }),
    });
  }, [deleteScriptsMutation, t]);

  const columns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => setViewing(record)}>{t('mml.info')}</Button>
          <Button type="link" size="small" onClick={() => setExecScript(record)}>{t('common.execute')}</Button>
          <Button type="link" size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Button type="link" size="small" danger onClick={() => handleDelete(record)}>{t('common.delete')}</Button>
        </Space>
      ),
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (v: unknown) => (v as string) || '-' },
    { key: 'deviceType', title: t('mml.deviceType'), dataIndex: 'deviceType', width: 120, render: (v: unknown) => (v as string) || '-' },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'tags', title: t('mml.tags'), dataIndex: 'tags', width: 180, render: (tags: unknown) => Array.isArray(tags) && tags.length ? (tags as string[]).map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 160, render: (val: unknown) => formatTime(val as string) },
  ], [t, openEdit, handleDelete]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('mml.newScript')}
        </Button>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <SearchInput
          placeholder={t('mml.scriptName')}
          allowClear
          style={{ width: 300 }}
          onSearch={(val) => { setSearch(val); setPage(1); }}
        />
      </div>
      <DataTable<MMLScript>
        tableId="mml-scripts"
        columns={columns}
        dataSource={scripts}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />

      {/* 新增 / 编辑脚本弹窗 */}
      <Modal
        title={editing ? t('mml.editScript') : t('mml.newScript')}
        open={formVisible}
        onOk={handleSave}
        onCancel={closeForm}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={680}
        confirmLoading={createScriptMutation.isPending || updateScriptMutation.isPending}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            label={t('mml.scriptName')}
            name="scriptName"
            rules={[{ required: true, message: t('mml.inputScriptName') }]}
          >
            <Input maxLength={128} showCount placeholder={t('mml.inputScriptName')} />
          </Form.Item>
          <Form.Item label={t('mml.description')} name="description">
            <Input maxLength={256} placeholder={t('mml.description')} />
          </Form.Item>
          <Form.Item label={t('mml.tags')} name="tags">
            <Select mode="tags" tokenSeparators={[',']} placeholder={t('mml.tags')} open={false} />
          </Form.Item>
          <Form.Item
            label={t('mml.scriptContent')}
            name="content"
            rules={[{ required: true, message: t('mml.scriptContent') }]}
          >
            <Input.TextArea
              rows={10}
              placeholder={'LST CELL;\nACT CELL:CELLID=0;'}
              style={{ fontFamily: "'SFMono-Regular', Consolas, Menlo, monospace", fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('mml.scriptDetail')} open={Boolean(viewing)} onCancel={() => setViewing(null)} footer={null} width={600}>
        {viewing && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.scriptNameLabel')}</strong>{viewing.scriptName}</p>
            <p><strong>{t('mml.description')}</strong>{viewing.description || '-'}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{viewing.creator}</p>
            <p><strong>{t('mml.updateTime')}</strong>{formatTime(viewing.updateTime)}</p>
            <div style={{ marginTop: 12 }}>
              <strong>{t('mml.scriptContent')}</strong>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, maxHeight: 300, overflow: 'auto', fontSize: 13, fontFamily: 'monospace' }}>
                {viewing.content}
              </pre>
            </div>
          </div>
        )}
      </Modal>

      {/* 执行脚本：预填脚本内容，用户补设备 + 执行方式后提交生成任务记录 */}
      <ScriptTaskDrawer
        open={Boolean(execScript)}
        onClose={() => setExecScript(null)}
        prefillContent={execScript?.content ?? ''}
        prefillTaskName={
          execScript
            ? `${execScript.scriptName}_${dayjs().format('YYYYMMDD_HHmmss')}`
            : undefined
        }
        onSuccess={() => void refetch()}
      />
    </ListPageLayout>
  );
}
