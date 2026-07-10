import { useState, useMemo, useCallback, useEffect } from 'react';
import { Button, Descriptions, Drawer, Empty, Form, Input, Modal, Space, Spin, Table, Tag, Typography, message, theme } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import SearchInput from '@/components/SearchInput';
import ScriptTaskDrawer from '../components/ScriptTaskDrawer';
import CommandSelectModal from '../Console/components/CommandSelectModal';
import ExecutionModeSelector from './ExecutionModeSelector';
import { useT } from '@/hooks/useT';

import type { MMLScript, MMLTaskExecuteMode, MMLTaskPlanItem } from '@core/types/mml';
import { parseMmlScriptPlan } from '@core/utils/mmlScriptPlanParser';
import type { CommandItem, CommandParamPath } from '../Console/types';
import {
  useMMLScripts,
  useMMLScriptById,
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
// 实际接收的核心列对齐（script_name / description / content）。
interface ScriptForm {
  scriptName: string;
  description?: string;
  content: string;
}

// 脚本任务（mml/script）：脚本库列表，读 mml_scripts。
// 支持新增 / 编辑 / 删除脚本（POST、PUT、DELETE /mml/scripts）。
// 任务执行记录（mml_tasks）由独立页面 mml/task-records 承载。
export default function ScriptTask() {
  const t = useT();
  const { token } = theme.useToken();
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
  const { data: viewingDetail, isFetching: isViewingDetailFetching } = useMMLScriptById(viewing?.id ?? '');
  const detailScript = viewingDetail ?? viewing;

  // 执行脚本：打开 ScriptTaskDrawer 预填该脚本内容，由用户选设备 + 执行方式
  // （立即=手动执行 / 定时 / 周期=自动执行）后提交。提交即 POST /mml/tasks，
  // 由后端 scheduler 调度，每次执行在 mml_tasks 落一条任务记录。
  const [execScript, setExecScript] = useState<MMLScript | null>(null);

  // ---- 新增/编辑脚本弹窗 ----------------------------------------------------
  const [editing, setEditing] = useState<MMLScript | null>(null);
  const [formVisible, setFormVisible] = useState(false);
  const [form] = Form.useForm<ScriptForm>();
  const contentValue = Form.useWatch('content', form) ?? '';
  const hasScriptContent = String(contentValue || '').trim().length > 0;
  const [scriptMode, setScriptMode] = useState<MMLTaskExecuteMode>('device_bound');
  const scriptPlanPreview = useMemo(
    () => parseMmlScriptPlan(String(contentValue || ''), { format: 'auto' }),
    [contentValue]
  );
  const detectedDeviceBound = hasScriptContent && scriptPlanPreview.executeMode === 'device_bound';
  const effectiveScriptMode: MMLTaskExecuteMode = detectedDeviceBound ? 'device_bound' : scriptMode;
  const isScriptDeviceBound = effectiveScriptMode === 'device_bound';
  const scriptContentPlaceholder = isScriptDeviceBound
    ? 'LST DEVICE_INFO;1202000091177SP0005\nMOD DEVICE_INFO:USER_LABEL=Site-A;1202000091177SP0006\nLST DEVICE_INFO;1202000091177SP0006'
    : 'LST DEVICE_INFO;\nLST TIME;';
  const [commandSelectOpen, setCommandSelectOpen] = useState(false);
  const [selectedCommand, setSelectedCommand] = useState<CommandItem | null>(null);
  const [commandParamValues, setCommandParamValues] = useState<Record<string, string>>({});

  const writableParams = useMemo(
    () => selectedCommand ? writableScriptParams(selectedCommand) : [],
    [selectedCommand]
  );
  const scriptPlanPreviewColumns = useMemo(() => [
    {
      key: 'lineNo',
      title: '#',
      width: 72,
      render: (_: unknown, item: MMLTaskPlanItem) => `${item.lineNo}/${item.order}`,
    },
    {
      key: 'deviceSn',
      title: t('mml.deviceSn'),
      dataIndex: 'deviceSn',
      ellipsis: true,
    },
    {
      key: 'command',
      title: 'MML',
      render: (_: unknown, item: MMLTaskPlanItem) => (
        <Typography.Text code>{item.command.commandCode}</Typography.Text>
      ),
    },
  ], [t]);

  // 弹窗打开后再回填表单：Modal 子节点惰性挂载，openEdit 时 Form 实例可能尚未连接，
  // 因此把回填放进 formVisible 的副作用里，确保 Form 已挂载。
  useEffect(() => {
    if (!formVisible) return;
    if (editing) {
      const editingMode = parseMmlScriptPlan(editing.content || '', { format: 'auto' }).executeMode;
      setScriptMode(editingMode);
      form.setFieldsValue({
        scriptName: editing.scriptName,
        description: editing.description,
        content: editing.content,
      });
    } else {
      setScriptMode('device_bound');
      form.resetFields();
    }
  }, [formVisible, editing, form]);

  const openCreate = useCallback(() => {
    setEditing(null);
    setSelectedCommand(null);
    setCommandParamValues({});
    setFormVisible(true);
  }, []);

  const openEdit = useCallback((script: MMLScript) => {
    setEditing(script);
    setSelectedCommand(null);
    setCommandParamValues({});
    setFormVisible(true);
  }, []);

  const closeForm = useCallback(() => {
    setFormVisible(false);
    setEditing(null);
    setSelectedCommand(null);
    setCommandParamValues({});
    form.resetFields();
  }, [form]);

  const handleCommandSelected = useCallback((command: CommandItem) => {
    const defaults: Record<string, string> = {};
    for (const param of writableScriptParams(command)) {
      const key = scriptParamKey(param);
      const defaultValue = param.defaultValue ?? (param.minValue !== undefined ? String(param.minValue) : '');
      defaults[key] = defaultValue;
    }
    setSelectedCommand(command);
    setCommandParamValues(defaults);
    setCommandSelectOpen(false);
  }, []);

  const insertSelectedCommand = useCallback(() => {
    if (!selectedCommand) return;
    const line = buildScriptLine(selectedCommand, commandParamValues);
    const current = String(contentValue || '');
    const next = current.trim()
      ? `${current.replace(/\s*$/, '')}\n${line}`
      : line;
    form.setFieldValue('content', next);
    form.validateFields(['content']).catch(() => undefined);
  }, [selectedCommand, commandParamValues, contentValue, form]);

  const handleScriptModeChange = useCallback((mode: MMLTaskExecuteMode) => {
    setScriptMode(mode);
    if (!String(form.getFieldValue('content') || '').trim()) return;
    window.setTimeout(() => {
      form.validateFields(['content']).catch(() => undefined);
    }, 0);
  }, [form]);

  const validateScriptContent = useCallback((_: unknown, value?: string) => {
    const content = String(value || '');
    if (!content.trim()) return Promise.resolve();
    const nextPlan = parseMmlScriptPlan(content, { format: 'auto' });
    const nextMode: MMLTaskExecuteMode = nextPlan.executeMode === 'device_bound' ? 'device_bound' : scriptMode;
    if (nextMode === 'device_bound') {
      if (nextPlan.planItems.length === 0) {
        return Promise.reject(new Error(t('mml.executeModeDeviceBoundRequiresSn', {
          mode: t('mml.executeModeDeviceBound'),
          fallback: t('mml.executeModeCommon'),
        })));
      }
      if (nextPlan.warnings.length > 0) {
        return Promise.reject(new Error(t('mml.executeModeAllRowsRequireSn', {
          mode: t('mml.executeModeDeviceBound'),
        })));
      }
    }
    return Promise.resolve();
  }, [scriptMode, t]);

  const handleSave = useCallback(() => {
    form
      .validateFields()
      .then((vals) => {
        const nextPlan = parseMmlScriptPlan(vals.content, { format: 'auto' });
        const nextMode: MMLTaskExecuteMode = nextPlan.executeMode === 'device_bound' ? 'device_bound' : scriptMode;
        if (nextMode === 'device_bound') {
          if (nextPlan.planItems.length === 0) {
            void message.error(t('mml.executeModeDeviceBoundRequiresSn', {
              mode: t('mml.executeModeDeviceBound'),
              fallback: t('mml.executeModeCommon'),
            }));
            return;
          }
          if (nextPlan.warnings.length > 0) {
            void message.error(t('mml.executeModeAllRowsRequireSn', {
              mode: t('mml.executeModeDeviceBound'),
            }));
            return;
          }
        }
        // BUG-13 修复：确保 onSuccess 中刷新列表 + 关闭弹窗
        const onDone = () => {
          void message.success(t('mml.scriptSaved'));
          void refetch(); // 刷新列表
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
                tags: [],
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
              tags: [],
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
  }, [form, editing, username, scriptMode, createScriptMutation, updateScriptMutation, closeForm, refetch, t]);

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

  const renderScriptMode = useCallback((content?: string | null) => {
    const parsed = parseMmlScriptPlan(String(content || ''), { format: 'auto' });
    if (parsed.executeMode === 'device_bound') {
      return (
        <Space size={4} wrap>
          <Tag color="processing">{t('mml.executeModeDeviceBound')}</Tag>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('mml.planItemCount', { count: parsed.planItems.length })}
            {' / '}
            {t('mml.planDeviceCount', { count: parsed.deviceSns.length })}
          </Typography.Text>
        </Space>
      );
    }
    return (
      <Space size={4} wrap>
        <Tag>{t('mml.executeModeCommon')}</Tag>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('mml.commandsParsed', { count: parsed.commands.length })}
        </Typography.Text>
      </Space>
    );
  }, [t]);

  const columns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 220,
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
    {
      key: 'executeMode',
      title: t('mml.executeMode'),
      width: 190,
      render: (_: unknown, record) => renderScriptMode(record.content),
    },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 160, render: (val: unknown) => formatTime(val as string) },
  ], [t, renderScriptMode, openEdit, handleDelete]);

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
        scroll={{ x: 900 }}
      />

      {/* 新增 / 编辑脚本弹窗 */}
      {/* BUG-04 修复：mutation pending 时禁止 ESC/点击遮罩关闭，防止用户误以为取消但数据已提交 */}
      <Modal
        title={editing ? t('mml.editScript') : t('mml.newScript')}
        open={formVisible}
        onOk={handleSave}
        onCancel={closeForm}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={680}
        confirmLoading={createScriptMutation.isPending || updateScriptMutation.isPending}
        mask={{ closable: !createScriptMutation.isPending && !updateScriptMutation.isPending }}
        closable={!createScriptMutation.isPending && !updateScriptMutation.isPending}
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
          <Form.Item label={t('mml.scriptCommandBuilder')}>
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Space wrap>
                <Button icon={<PlusOutlined />} onClick={() => setCommandSelectOpen(true)}>
                  {t('mml.selectInsertCommand')}
                </Button>
                <Button
                  type="primary"
                  ghost
                  disabled={!selectedCommand}
                  onClick={insertSelectedCommand}
                >
                  {t('mml.insertCommand')}
                </Button>
                {selectedCommand && (
                  <>
                    <Tag color="blue">{selectedCommand.operationType}</Tag>
                    <Typography.Text code>{selectedCommand.commandCode}</Typography.Text>
                  </>
                )}
              </Space>

              {selectedCommand && (
                <div
                  style={{
                    border: `1px solid ${token.colorBorderSecondary}`,
                    borderRadius: 4,
                    padding: 12,
                    background: token.colorFillQuaternary,
                  }}
                >
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    <div>
                      <Typography.Text strong>{selectedCommand.commandName}</Typography.Text>
                      <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
                        {selectedCommand.groupName}
                      </Typography.Text>
                    </div>
                    <Typography.Text type="secondary">
                      {t('mml.commandParamPathCount', { count: selectedCommand.paramPaths.length })}
                    </Typography.Text>

                    {writableParams.length > 0 ? (
                      <div style={{ maxHeight: 220, overflow: 'auto' }}>
                        {writableParams.map((param) => {
                          const key = scriptParamKey(param);
                          return (
                            <div key={`${key}:${param.path}`} style={{ marginBottom: 8 }}>
                              <Input
                                addonBefore={
                                  <span style={{ display: 'inline-block', minWidth: 110 }}>
                                    {key}
                                  </span>
                                }
                                value={commandParamValues[key] ?? ''}
                                placeholder={param.defaultValue || param.description || param.label}
                                onChange={(e) =>
                                  setCommandParamValues((prev) => ({
                                    ...prev,
                                    [key]: e.target.value,
                                  }))
                                }
                              />
                              <Typography.Text
                                type="secondary"
                                style={{ display: 'block', marginTop: 2, fontSize: 12 }}
                              >
                                {param.label} · {param.path}
                              </Typography.Text>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <Typography.Text type="secondary">
                        {t('mml.noWritableCommandParams')}
                      </Typography.Text>
                    )}
                  </Space>
                </div>
              )}
            </Space>
          </Form.Item>
          <Form.Item label={t('mml.executeMode')}>
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              <ExecutionModeSelector
                value={effectiveScriptMode}
                deviceBoundDetected={detectedDeviceBound}
                onChange={handleScriptModeChange}
                t={t}
              />
              {hasScriptContent ? (
                <Space wrap>
                  {isScriptDeviceBound ? (
                    <Tag color="processing">{t('mml.planItemCount', { count: scriptPlanPreview.planItems.length })}</Tag>
                  ) : (
                    <Tag>{t('mml.commandsParsed', { count: scriptPlanPreview.commands.length })}</Tag>
                  )}
                  {isScriptDeviceBound ? (
                    <Tag color="blue">{t('mml.planDeviceCount', { count: scriptPlanPreview.deviceSns.length })}</Tag>
                  ) : null}
                  {detectedDeviceBound ? (
                    <Typography.Text type="secondary">
                      {t('mml.executeModeAutoDetected', { mode: t('mml.executeModeDeviceBound') })}
                    </Typography.Text>
                  ) : null}
                </Space>
              ) : null}
            </Space>
          </Form.Item>
          <Form.Item
            label={t('mml.scriptContent')}
            name="content"
            rules={[
              { required: true, message: t('mml.scriptContent') },
              { validator: validateScriptContent },
            ]}
          >
            <Input.TextArea
              rows={10}
              placeholder={scriptContentPlaceholder}
              style={{ fontFamily: "'SFMono-Regular', Consolas, Menlo, monospace", fontSize: 12 }}
            />
          </Form.Item>
          {hasScriptContent && isScriptDeviceBound ? (
            <Form.Item label={t('mml.planPreview')}>
              <Space direction="vertical" size={10} style={{ width: '100%' }}>
              {scriptPlanPreview.warnings.length > 0 ? (
                <Typography.Text type="warning">
                  {t('mml.executeModeMixedRows', { mode: t('mml.executeModeDeviceBound') })}
                </Typography.Text>
              ) : null}
              <Table<MMLTaskPlanItem>
                size="small"
                rowKey={(item) => `${item.lineNo}-${item.deviceSn}-${item.order}`}
                columns={scriptPlanPreviewColumns}
                dataSource={scriptPlanPreview.planItems}
                pagination={
                  scriptPlanPreview.planItems.length > 5
                    ? { pageSize: 5, size: 'small', showSizeChanger: false }
                    : false
                }
              />
              </Space>
            </Form.Item>
          ) : null}
        </Form>
      </Modal>

      <Drawer
        title={detailScript?.scriptName || t('mml.scriptDetail')}
        open={Boolean(viewing)}
        onClose={() => setViewing(null)}
        width={720}
        destroyOnHidden
      >
        {detailScript && (
          <Spin spinning={isViewingDetailFetching}>
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Descriptions column={2} size="small" bordered>
                <Descriptions.Item label={t('mml.scriptName')} span={2}>
                  {detailScript.scriptName}
                </Descriptions.Item>
                <Descriptions.Item label={t('mml.description')} span={2}>
                  {detailScript.description || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('mml.creator')}>
                  {detailScript.creator || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('mml.updateTime')}>
                  {formatTime(detailScript.updateTime)}
                </Descriptions.Item>
                <Descriptions.Item label={t('mml.createTime')}>
                  {formatTime(detailScript.createTime)}
                </Descriptions.Item>
                <Descriptions.Item label={t('mml.executeMode')}>
                  {renderScriptMode(detailScript.content)}
                </Descriptions.Item>
              </Descriptions>

              <div>
                <Typography.Text strong>{t('mml.scriptContent')}</Typography.Text>
                {detailScript.content?.trim() ? (
                  <pre
                    style={{
                      marginTop: 8,
                      background: token.colorFillQuaternary,
                      border: `1px solid ${token.colorBorderSecondary}`,
                      color: token.colorText,
                      padding: 12,
                      borderRadius: 4,
                      maxHeight: '55vh',
                      overflow: 'auto',
                      fontSize: 13,
                      lineHeight: 1.7,
                      whiteSpace: 'pre-wrap',
                      fontFamily: "'SFMono-Regular', Consolas, Menlo, monospace",
                    }}
                  >
                    {detailScript.content}
                  </pre>
                ) : (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={t('mml.emptyScriptContent')}
                    style={{ marginTop: 16 }}
                  />
                )}
              </div>
            </Space>
          </Spin>
        )}
      </Drawer>

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
        scriptId={execScript?.id ?? undefined}
        onSuccess={() => void refetch()}
      />
      <CommandSelectModal
        open={commandSelectOpen}
        value={selectedCommand}
        onCancel={() => setCommandSelectOpen(false)}
        onConfirm={handleCommandSelected}
        onGotoRawParams={() => {
          setCommandSelectOpen(false);
          void message.info(t('mml.rawParamScriptTip'));
        }}
      />
    </ListPageLayout>
  );
}

function writableScriptParams(command: CommandItem): CommandParamPath[] {
  if (command.operationType !== 'MOD' && command.operationType !== 'ADD') return [];
  return command.paramPaths.filter((param) => param.writable && scriptParamKey(param));
}

function scriptParamKey(param: CommandParamPath): string {
  return (param.mmlCode || param.label || '').trim();
}

function buildScriptLine(command: CommandItem, values: Record<string, string>): string {
  const params = writableScriptParams(command)
    .map((param) => {
      const key = scriptParamKey(param);
      const value = values[key]?.trim();
      return key && value ? `${key}=${value}` : '';
    })
    .filter(Boolean);
  return `${command.commandCode}${params.length ? `:${params.join(',')}` : ''};`;
}
