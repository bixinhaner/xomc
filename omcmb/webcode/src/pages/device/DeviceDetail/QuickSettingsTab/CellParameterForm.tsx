import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Form, Input, Row, Select, Space, Spin, Tag, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useParameterSchema, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type CellFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { applyInstanceContext, getEffectiveEnumMeta, validateValue, type QuickSettingsInstanceContext } from './validators';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

// "上次提交"状态形状由 frontend-core/store/quickSettingsFeedbackStore (CellFeedback) 定义,
// 提升至 store 持久化,顶层 TabBar 切走再切回不丢反馈。

function formatTime(at: number): string {
  const d = new Date(at);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

/** T-0146:状态机 Tag 显示规则。 */
interface StatusTagSpec {
  color: string;
  icon: React.ReactNode;
  label: string;
}
function statusTagSpec(submit: CellFeedback, taskStatus: DeviceTaskStatus | undefined): StatusTagSpec {
  if (submit.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: '入队失败' };
  }
  // submitStatus = 'queued' 后,根据 task 真实状态分支
  switch (taskStatus) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined />, label: '基站应答成功' };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined />, label: '基站应答失败' };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined />, label: '超时' };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined />, label: '已取消' };
    case 'sent':
      return { color: 'processing', icon: <SendOutlined />, label: '已发送给基站' };
    case 'pending':
    default:
      // pending 或 task 还没拉到(刚 mutateAsync 完)
      return { color: 'processing', icon: <SyncOutlined spin />, label: '已入队,等待下发' };
  }
}

interface CellParameterFormProps {
  deviceId: string;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
}

/**
 * 单实例分组表单（如「小区参数」）。
 *
 * 行为：
 *  1. 把 group.params 的 standardPath 中 {i} 替换为当前 fapInstance
 *  2. 通过 useParameterSchema 拉每条路径的 schema（类型/约束/当前值）
 *  3. 渲染为 2 列网格 Form
 *  4. 顶部"保存"按钮收集本表单全部脏字段，一次性 SetParameterValues
 *  5. Save 部分失败时按字段标红保留输入值（继承 Antd Form 校验/状态行为）
 */
export default function CellParameterForm({ deviceId, group, instanceContext, locale }: CellParameterFormProps) {
  const [form] = Form.useForm();
  const updateMutation = useUpdateParameters();
  const queryClient = useQueryClient();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // lastSubmit 由 zustand store 托管 —— DeviceDetail 整个被卸载(切顶层 tab)也保留反馈。
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    instanceContext.fapInstance,
    instanceContext.networkType === 'nr' ? instanceContext.cellInstance : undefined,
  );
  const lastSubmit = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'cell' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraft = useQuickSettingsFeedbackStore((s) => s.clearDraft);

  // 单实例分组：每条 standardPath 单独查 schema（少量字段，不批量优化）
  // 注：useParameterSchema 接受 pathPrefix，前缀匹配即可；这里以分组共用前缀粗查再过滤
  // 为简化，取 group 中 standardPath 的公共前缀作 pathPrefix
  const commonPrefix = useMemo(
    () => commonPathPrefix(group.params.map((p) => applyInstanceContext(p.standardPath || '', instanceContext))),
    [group, instanceContext],
  );
  const { data: schemaResp, isLoading, refetch } = useParameterSchema(deviceId, commonPrefix);

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  // T-0159: 交叉镜像 — 反查表 resolved standardPath → form field name，
  // 让 onValuesChange 时能据 constraints.mirrorWith 找到对端 form field 并 setFieldValue 同步。
  const paramNameByPath = useMemo(() => {
    const map = new Map<string, string>();
    for (const p of group.params) {
      const path = applyInstanceContext(p.standardPath || '', instanceContext);
      map.set(path, p.name);
    }
    return map;
  }, [group, instanceContext]);

  // 初始化字段值 —— 优先级：store draft > 当前会话已 touched > schema 原值。
  // 未保存草稿在跨顶层 TabBar 切换后恢复；任务终态回读后再 clearDraft，统一回到设备侧值。
  useEffect(() => {
    if (!schemaResp) return;
    group.params.forEach((p) => {
      if (draft && draft[p.name] !== undefined) {
        form.setFieldValue(p.name, draft[p.name]);
        return;
      }
      // 优先级 2: 用户在当前会话已 touched
      if (form.isFieldTouched(p.name)) return;
      // 优先级 3: schema 原值
      const path = applyInstanceContext(p.standardPath || '', instanceContext);
      const item = schemaByPath.get(path);
      form.setFieldValue(p.name, item?.currentValue ?? '');
    });
  }, [schemaResp, group, instanceContext, form, schemaByPath, draft]);

  const handleSave = async () => {
    const values = form.getFieldsValue() as Record<string, string>;
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};

    for (const p of group.params) {
      const path = applyInstanceContext(p.standardPath || '', instanceContext);
      const item = schemaByPath.get(path);
      const newVal = values[p.name] ?? '';
      const oldVal = item?.currentValue ?? '';
      if (newVal === oldVal) continue;

      const err = validateValue(newVal, (item?.type as never) ?? 'string', item?.constraints);
      if (err) {
        errors[p.name] = err;
        continue;
      }
      updates.push({
        parameterPath: path,
        parameterValue: newVal,
        parameterType: (item?.type as never) ?? 'string',
      });
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      message.error({ content: '校验失败,请检查标红字段', duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: '无变更', duration: 4 });
      return;
    }

    setFieldErrors({});
    try {
      const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      message.success({
        content: `已下发 ${updates.length} 项变更,正在等待基站应答(Tag 状态会自动刷新)`,
        duration: 6,
      });
      setFeedback(fbKey, {
        kind: 'cell',
        submitStatus: 'queued',
        taskId: result.taskId,
        count: updates.length,
        at: Date.now(),
      });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `下发失败(${group.titleZh})`,
        description: `${updates.length} 项变更入队失败:${errMsg}。输入值已保留,可修正后重试。`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'cell',
        submitStatus: 'failed_to_queue',
        count: updates.length,
        at: Date.now(),
        errorMsg: errMsg,
      });
      console.error('CellParameterForm: update failed', err);
    } finally {
      // T-0157 C10: 无论成功失败都触发消息中心 invalidate ——
      // 成功路径：后端 task.created publish → 订阅器写"进行中"消息，~50ms 内拉回
      // 失败路径：CreateFailureNotifier 兜底写"入队失败"消息，~50ms 内拉回
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  };

  // T-0146:Save 后用 task_id 轮询真实 CPE 应答状态;到终态后停轮询。
  const { data: lastTask } = useDeviceTaskStatus(lastSubmit?.taskId);

  useEffect(() => {
    if (!lastTask || !['completed', 'failed', 'expired', 'cancelled'].includes(lastTask.status)) return;
    let cancelled = false;
    void (async () => {
      let refreshed;
      try {
        refreshed = await refetch();
      } catch (err) {
        if (!cancelled) {
          const errMsg = err instanceof Error ? err.message : String(err);
          notification.error({
            message: `设备侧数据回读失败(${group.titleZh})`,
            description: errMsg,
            duration: ERROR_FEEDBACK_DURATION_SECONDS,
          });
        }
        return;
      }
      if (cancelled) return;
      const nextValues: Record<string, string> = {};
      const refreshedSchemaByPath = new Map((refreshed.data?.parameters ?? []).map((item) => [item.path, item]));
      for (const p of group.params) {
        const path = applyInstanceContext(p.standardPath || '', instanceContext);
        nextValues[p.name] = refreshedSchemaByPath.get(path)?.currentValue ?? '';
      }
      form.setFieldsValue(nextValues);
      clearDraft(fbKey);
      setFieldErrors({});
    })();
    return () => {
      cancelled = true;
    };
  }, [lastTask?.id, lastTask?.status, refetch, group.params, instanceContext, form, clearDraft, fbKey, group.titleZh]);

  // T-0146:基站应答失败时弹一次 notification(只在 status 第一次变成 failed 时触发,避免重复弹)
  // notifiedFailedTaskId 同样存 store —— 切顶层 tab 再切回不会重复弹。
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastSubmit &&
      lastSubmit.notifiedFailedTaskId !== lastTask.id
    ) {
      notification.error({
        message: `基站应答失败(${group.titleZh})`,
        description: lastTask.errorMessage || '未知错误,可在通知中心查看任务详情',
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastSubmit, group.titleZh, patchFeedback, fbKey]);

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;

  return (
    <Card
      title={title}
      size="small"
      extra={
        <Space>
          {lastSubmit && (() => {
            const spec = statusTagSpec(lastSubmit, lastTask?.status);
            return (
              <Tag icon={spec.icon} color={spec.color}>
                {spec.label} · {lastSubmit.count} 项 · {formatTime(lastSubmit.at)}
              </Tag>
            );
          })()}
          <Button type="primary" onClick={handleSave} loading={updateMutation.isPending}>
            {updateMutation.isPending ? '下发中...' : '保 存'}
          </Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Spin spinning={isLoading}>
      <Form
        form={form}
        layout="vertical"
        onValuesChange={(changedValues) => {
          // 同步到 store draft，跨顶层 TabBar 切走切回可恢复
          for (const [name, value] of Object.entries(changedValues)) {
            setDraftField(fbKey, name, String(value ?? ''));
          }
          // T-0159: 交叉镜像 — 改 A 字段时把 A 的新值同步写入镜像字段 B（如 TDD 上下行带宽必须相等）。
          // antd Form.setFieldValue 不会触发 onValuesChange，故不会无限递归。
          for (const [name, value] of Object.entries(changedValues)) {
            const p = group.params.find((q) => q.name === name);
            if (!p) continue;
            const path = applyInstanceContext(p.standardPath || '', instanceContext);
            const sItem = schemaByPath.get(path);
            const mirrorPath = sItem?.constraints?.mirrorWith;
            if (!mirrorPath) continue;
            const resolvedMirror = applyInstanceContext(mirrorPath, instanceContext);
            const mirrorName = paramNameByPath.get(resolvedMirror);
            if (!mirrorName || mirrorName === name) continue;
            const current = form.getFieldValue(mirrorName);
            if (String(current ?? '') === String(value ?? '')) continue;
            form.setFieldValue(mirrorName, value);
            setDraftField(fbKey, mirrorName, String(value ?? ''));
          }
          // 实时按 schema 取值范围校验，更新 fieldErrors 让 Form.Item 即时标红
          setFieldErrors((prev) => {
            const next = { ...prev };
            for (const [name, value] of Object.entries(changedValues)) {
              const p = group.params.find((q) => q.name === name);
              if (!p) continue;
              const path = applyInstanceContext(p.standardPath || '', instanceContext);
              const sItem = schemaByPath.get(path);
              const err = validateValue(
                String(value ?? ''),
                (sItem?.type as never) ?? 'string',
                sItem?.constraints,
              );
              if (err) next[name] = err;
              else delete next[name];
              // 镜像字段同时清/重新校验（值刚被程序性写入，旧 error 应失效）
              const mirrorPath = sItem?.constraints?.mirrorWith;
              if (mirrorPath) {
                const resolvedMirror = applyInstanceContext(mirrorPath, instanceContext);
                const mirrorName = paramNameByPath.get(resolvedMirror);
                if (mirrorName && mirrorName !== name) {
                  const mItem = schemaByPath.get(resolvedMirror);
                  const mErr = validateValue(
                    String(value ?? ''),
                    (mItem?.type as never) ?? 'string',
                    mItem?.constraints,
                  );
                  if (mErr) next[mirrorName] = mErr;
                  else delete next[mirrorName];
                }
              }
            }
            return next;
          });
        }}
      >
        <Row gutter={16}>
          {group.params.map((p) => {
            const path = applyInstanceContext(p.standardPath || '', instanceContext);
            const item = schemaByPath.get(path);
            const writable = item?.writable ?? false;
            const error = fieldErrors[p.name];
            const constraintHint = formatConstraintHint(item);
            const label = (
              <Space size={4}>
                <span>{locale === 'zh-CN' ? p.titleZh : p.titleEn}</span>
                {!writable && <Text type="secondary" style={{ fontSize: 12 }}>(只读)</Text>}
                {constraintHint && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {constraintHint}
                  </Text>
                )}
              </Space>
            );
            const enumMeta = getEffectiveEnumMeta(item?.constraints, path);
            const isEnum = Boolean(enumMeta && enumMeta.values.length > 0);
            return (
              <Col span={12} key={p.name}>
                <Form.Item
                  label={label}
                  name={p.name}
                  validateStatus={error ? 'error' : undefined}
                  help={error}
                >
                  {isEnum ? (
                    <Select
                      disabled={!writable}
                      placeholder={item?.defaultValue || ''}
                      options={enumMeta!.values.map((v, idx) => ({
                        value: v,
                        label: enumMeta!.labels[idx] || v,
                      }))}
                    />
                  ) : (
                    <Input disabled={!writable} placeholder={item?.defaultValue || ''} />
                  )}
                </Form.Item>
              </Col>
            );
          })}
        </Row>
      </Form>
      </Spin>
    </Card>
  );
}

// formatConstraintHint 把 schema 取值范围渲染成 label 后的灰色提示。
// 后端 MinValue/MaxValue 是按类型复用的字段：string → 长度边界；int/unsignedInt → 值范围。
// 枚举字段不渲染提示（Select 组件已展示候选项，避免重复占位）。
function formatConstraintHint(schema?: ParameterSchemaItem): string {
  if (!schema?.constraints) return '';
  const c = schema.constraints;
  if (c.enumValues && c.enumValues.length > 0) {
    return '';
  }
  const isString = schema.type === 'string';
  // 优先用显式 maxLength/minLength；fallback 到 minValue/maxValue (按类型解释)
  const min = c.minLength ?? (isString ? c.minValue : c.minValue);
  const max = c.maxLength ?? (isString ? c.maxValue : c.maxValue);
  if (min !== undefined || max !== undefined) {
    const lo = min ?? '-∞';
    const hi = max ?? '∞';
    return isString ? `[长度 ${lo} ~ ${hi}]` : `[${lo} ~ ${hi}]`;
  }
  return '';
}

function commonPathPrefix(paths: string[]): string {
  if (paths.length === 0) return '';
  let prefix = paths[0];
  for (const p of paths.slice(1)) {
    let i = 0;
    while (i < prefix.length && i < p.length && prefix[i] === p[i]) i++;
    prefix = prefix.slice(0, i);
  }
  // 截到最后一个 '.'，使前缀对齐对象级
  const idx = prefix.lastIndexOf('.');
  return idx > 0 ? prefix.slice(0, idx + 1) : prefix;
}
