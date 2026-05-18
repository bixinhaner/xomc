import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Form, Input, Row, Space, Tag, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useParameterSchema, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import type { ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { applyFapInstance, validateValue } from './validators';

const { Text } = Typography;

/** Save 操作的"上次提交"状态摘要(持久化反馈,不依赖一闪而过的 toast)。 */
interface LastSubmitState {
  status: 'success' | 'failed';
  count: number;
  at: Date;
  errorMsg?: string;
}

function formatTime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

interface CellParameterFormProps {
  deviceId: string;
  fapInstance: number;
  group: QuickSettingsGroup;
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
export default function CellParameterForm({ deviceId, fapInstance, group, locale }: CellParameterFormProps) {
  const [form] = Form.useForm();
  const updateMutation = useUpdateParameters();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [lastSubmit, setLastSubmit] = useState<LastSubmitState | null>(null);

  // 单实例分组：每条 standardPath 单独查 schema（少量字段，不批量优化）
  // 注：useParameterSchema 接受 pathPrefix，前缀匹配即可；这里以分组共用前缀粗查再过滤
  // 为简化，取 group 中 standardPath 的公共前缀作 pathPrefix
  const commonPrefix = useMemo(() => commonPathPrefix(group.params.map((p) => applyFapInstance(p.standardPath || '', fapInstance))), [group, fapInstance]);
  const { data: schemaResp, isLoading } = useParameterSchema(deviceId, commonPrefix);

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  // 初始化字段当前值
  useEffect(() => {
    if (!schemaResp) return;
    const initial: Record<string, string> = {};
    group.params.forEach((p) => {
      const path = applyFapInstance(p.standardPath || '', fapInstance);
      const item = schemaByPath.get(path);
      initial[p.name] = item?.currentValue ?? '';
    });
    form.setFieldsValue(initial);
  }, [schemaResp, group, fapInstance, form, schemaByPath]);

  const handleSave = async () => {
    const values = form.getFieldsValue() as Record<string, string>;
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};

    for (const p of group.params) {
      const path = applyFapInstance(p.standardPath || '', fapInstance);
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
      message.error({ content: '校验失败,请检查标红字段', duration: 6 });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: '无变更', duration: 4 });
      return;
    }

    setFieldErrors({});
    try {
      await updateMutation.mutateAsync({ deviceId, parameters: updates });
      message.success({
        content: `已下发 ${updates.length} 项变更,请在右上角铃铛查看任务结果`,
        duration: 6,
      });
      setLastSubmit({ status: 'success', count: updates.length, at: new Date() });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      // T-0144 改用 notification.error 持久化弹窗(用户需点击关闭,失败信息不丢)
      notification.error({
        message: `下发失败(${group.titleZh})`,
        description: `${updates.length} 项变更下发失败:${errMsg}。输入值已保留,可修正后重试。`,
        duration: 0, // 不自动消失
      });
      setLastSubmit({ status: 'failed', count: updates.length, at: new Date(), errorMsg: errMsg });
      console.error('CellParameterForm: update failed', err);
    }
  };

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;

  return (
    <Card
      title={title}
      size="small"
      extra={
        <Space>
          {lastSubmit && (
            <Tag
              icon={lastSubmit.status === 'success' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
              color={lastSubmit.status === 'success' ? 'success' : 'error'}
            >
              上次提交:{lastSubmit.status === 'success' ? '成功' : '失败'} {lastSubmit.count} 项 · {formatTime(lastSubmit.at)}
            </Tag>
          )}
          <Button type="primary" onClick={handleSave} loading={updateMutation.isPending}>
            {updateMutation.isPending ? '下发中...' : '保 存'}
          </Button>
        </Space>
      }
      loading={isLoading}
      style={{ marginBottom: 16 }}
    >
      <Form form={form} layout="vertical">
        <Row gutter={16}>
          {group.params.map((p) => {
            const path = applyFapInstance(p.standardPath || '', fapInstance);
            const item = schemaByPath.get(path);
            const writable = item?.writable ?? false;
            const error = fieldErrors[p.name];
            const label = (
              <Space size={4}>
                <span>{locale === 'zh-CN' ? p.titleZh : p.titleEn}</span>
                {!writable && <Text type="secondary" style={{ fontSize: 12 }}>(只读)</Text>}
              </Space>
            );
            return (
              <Col span={12} key={p.name}>
                <Form.Item
                  label={label}
                  name={p.name}
                  validateStatus={error ? 'error' : undefined}
                  help={error}
                >
                  <Input disabled={!writable} placeholder={item?.defaultValue || ''} />
                </Form.Item>
              </Col>
            );
          })}
        </Row>
      </Form>
    </Card>
  );
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
