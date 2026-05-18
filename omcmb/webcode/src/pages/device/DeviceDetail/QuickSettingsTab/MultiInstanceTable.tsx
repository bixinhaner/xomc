import { useCallback, useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Popconfirm, Space, Table, Tag, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import type { ColumnType } from 'antd/es/table';
import {
  useAddObject,
  useDeleteObject,
  useParameterSchema,
  useUpdateParameters,
} from '@core/hooks/api/useDeviceParameters';
import type { ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { applyFapInstance, validateValue } from './validators';

const { Text } = Typography;

/** 行级 Save / Add / Delete 的"上次提交"状态摘要。 */
interface LastActionState {
  action: 'save' | 'add' | 'delete';
  status: 'success' | 'failed';
  detail: string;
  at: Date;
}

function formatTime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

interface MultiInstanceTableProps {
  deviceId: string;
  fapInstance: number;
  group: QuickSettingsGroup;
  locale: 'zh-CN' | 'en-US';
}

interface RowEditState {
  /** 行内字段当前编辑值（leaf → value）。空 = 未编辑（取 schema 当前值）。 */
  edits: Record<string, string>;
  /** 行内字段错误（leaf → message）。 */
  errors: Record<string, string>;
}

/**
 * 多实例分组表格（异频邻区频点列表 / 邻区列表）。
 *
 * 行为：
 *  1. objectPath 中外层 FAPService.{i} 替换后作为 pathPrefix 查 schema
 *  2. schema.objects 中 path 等于 objectPath 的项给出 currentInstances（实例号数组）
 *  3. 每个实例渲染为一行；行内字段值从 schema.parameters[path=objectPath+instance+leaf] 取
 *  4. 行级 Save：收集脏字段 → 一次 SetParameterValues
 *  5. 新增：先 AddObject 拿新实例号 → 用户填值 → 行 Save 触发 SetParameterValues（两步）
 *  6. 删除：DeleteObject
 *  7. 失败标红保留输入值，"重试"按钮原值重发
 */
export default function MultiInstanceTable({ deviceId, fapInstance, group, locale }: MultiInstanceTableProps) {
  // group.objectPath 形如 "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."
  // - 外层 FAPService.{i} → 用 fapInstance 替换
  // - 内层 Carrier.{i}. 末段是实例号占位符 — 剥离后得到父对象路径,用于查 schema.objects / AddObject / 拼接行 path 前缀
  const objectPath = useMemo(() => {
    const fapped = applyFapInstance(group.objectPath || '', fapInstance);
    return fapped.replace(/\{i\}\.$/, '');
  }, [group, fapInstance]);
  const { data: schemaResp, isLoading, refetch } = useParameterSchema(deviceId, objectPath);
  const updateMutation = useUpdateParameters();
  const addMutation = useAddObject();
  const deleteMutation = useDeleteObject();

  // 行编辑状态：以 instanceId 为 key，仅保留用户编辑过的字段（避免 effect 同步 schema 触发级联 render）
  const [rowEdits, setRowEdits] = useState<Map<string, RowEditState>>(new Map());
  // T-0144:上次操作摘要(持久化反馈)
  const [lastAction, setLastAction] = useState<LastActionState | null>(null);

  // schema.objects 给出 currentInstances；schema.parameters 给出值
  const objectEntry = useMemo(
    () => schemaResp?.objects.find((o) => o.path === objectPath),
    [schemaResp, objectPath],
  );
  const canAdd = objectEntry?.canAdd ?? false;
  const canDelete = objectEntry?.canDeleteAny ?? false;

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  // 实例号列表直接从 schema 派生（不再走 setState in effect）
  const instanceIds = useMemo(() => {
    if (!objectEntry) return [] as string[];
    return objectEntry.currentInstances.map((n) => String(n)).sort((a, b) => Number(a) - Number(b));
  }, [objectEntry]);

  const cellValue = useCallback(
    (instId: string, leaf: string): string => {
      const edit = rowEdits.get(instId);
      if (edit && leaf in edit.edits) return edit.edits[leaf];
      const path = `${objectPath}${instId}.${leaf}`;
      return schemaByPath.get(path)?.currentValue ?? '';
    },
    [rowEdits, objectPath, schemaByPath],
  );

  const setCellValue = (instId: string, leaf: string, value: string) => {
    setRowEdits((prev) => {
      const next = new Map(prev);
      const row = next.get(instId) ?? { edits: {}, errors: {} };
      next.set(instId, {
        edits: { ...row.edits, [leaf]: value },
        errors: { ...row.errors, [leaf]: '' },
      });
      return next;
    });
  };

  const collectRowUpdates = (instId: string): {
    updates: ParameterUpdateRequest[];
    errors: Record<string, string>;
  } => {
    const row = rowEdits.get(instId);
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};
    if (!row) return { updates, errors };
    for (const p of group.params) {
      const leaf = p.leaf || '';
      if (!(leaf in row.edits)) continue;
      const newVal = row.edits[leaf];
      const path = `${objectPath}${instId}.${leaf}`;
      const item = schemaByPath.get(path);
      const oldVal = item?.currentValue ?? '';
      if (newVal === oldVal) continue;

      const err = validateValue(newVal, (item?.type as never) ?? 'string', item?.constraints);
      if (err) {
        errors[leaf] = err;
        continue;
      }
      updates.push({
        parameterPath: path,
        parameterValue: newVal,
        parameterType: (item?.type as never) ?? 'string',
      });
    }
    return { updates, errors };
  };

  const handleSaveRow = async (instId: string) => {
    const { updates, errors } = collectRowUpdates(instId);
    if (Object.keys(errors).length > 0) {
      setRowEdits((prev) => {
        const next = new Map(prev);
        const row = next.get(instId) ?? { edits: {}, errors: {} };
        next.set(instId, { ...row, errors });
        return next;
      });
      message.error({ content: '行校验失败,请修正', duration: 6 });
      return;
    }
    if (updates.length === 0) {
      message.info({ content: '该行无变更', duration: 4 });
      return;
    }
    try {
      await updateMutation.mutateAsync({ deviceId, parameters: updates });
      message.success({
        content: `第 ${instId} 行已下发 ${updates.length} 项变更,请在右上角铃铛查看任务结果`,
        duration: 6,
      });
      setLastAction({ action: 'save', status: 'success', detail: `第 ${instId} 行 ${updates.length} 项`, at: new Date() });
      // 清空 edits(schema 重新拉取时会同步当前值)
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.delete(instId);
        return next;
      });
      void refetch();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `第 ${instId} 行下发失败(${group.titleZh})`,
        description: `${updates.length} 项变更失败:${errMsg}。输入值已保留,可修正后重试。`,
        duration: 0,
      });
      setLastAction({ action: 'save', status: 'failed', detail: `第 ${instId} 行:${errMsg}`, at: new Date() });
      console.error('MultiInstanceTable: row save failed', err);
    }
  };

  const handleAdd = async () => {
    try {
      // objectPath 已含尾部 "."(AddObject 后端期望同样形态,参考 useAddObject 现有调用)
      await addMutation.mutateAsync({ deviceId, objectPath });
      message.success({
        content: '已下发 AddObject,请在新行填值后点击保存完成 SetParameterValues',
        duration: 6,
      });
      setLastAction({ action: 'add', status: 'success', detail: '新增实例', at: new Date() });
      void refetch();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `AddObject 失败(${group.titleZh})`,
        description: errMsg,
        duration: 0,
      });
      setLastAction({ action: 'add', status: 'failed', detail: errMsg, at: new Date() });
      console.error('MultiInstanceTable: AddObject failed', err);
    }
  };

  const handleDelete = async (instId: string) => {
    Modal.confirm({
      title: '确认删除',
      content: `删除实例 ${instId} ?`,
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteMutation.mutateAsync({ deviceId, objectPath: `${objectPath}${instId}.` });
          message.success({
            content: `已下发 DeleteObject(${instId}),请在右上角铃铛查看任务结果`,
            duration: 6,
          });
          setLastAction({ action: 'delete', status: 'success', detail: `实例 ${instId}`, at: new Date() });
          void refetch();
        } catch (err) {
          const errMsg = err instanceof Error ? err.message : String(err);
          notification.error({
            message: `DeleteObject 失败(${group.titleZh})`,
            description: `实例 ${instId} 删除失败:${errMsg}`,
            duration: 0,
          });
          setLastAction({ action: 'delete', status: 'failed', detail: `实例 ${instId}:${errMsg}`, at: new Date() });
          console.error('MultiInstanceTable: DeleteObject failed', err);
        }
      },
    });
  };

  const columns: ColumnType<string>[] = [
    {
      title: '实例',
      dataIndex: 'instanceId',
      key: 'instanceId',
      width: 80,
      fixed: 'left',
      render: (_v: unknown, instId: string) => <Text strong>{instId}</Text>,
    },
    ...group.params.map<ColumnType<string>>((p) => ({
      title: locale === 'zh-CN' ? p.titleZh : p.titleEn,
      key: p.leaf || p.name,
      dataIndex: p.leaf || p.name,
      width: 150,
      render: (_v: unknown, instId: string) => {
        const leaf = p.leaf || '';
        const row = rowEdits.get(instId);
        const error = row?.errors[leaf];
        const path = `${objectPath}${instId}.${leaf}`;
        const item = schemaByPath.get(path);
        const writable = item?.writable ?? false;
        return (
          <Input
            value={cellValue(instId, leaf)}
            onChange={(e) => setCellValue(instId, leaf, e.target.value)}
            status={error ? 'error' : undefined}
            disabled={!writable}
            size="small"
          />
        );
      },
    })),
    {
      title: '操作',
      key: 'actions',
      width: 160,
      fixed: 'right',
      render: (_v: unknown, instId: string) => (
        <Space size="small">
          <Button size="small" type="primary" onClick={() => void handleSaveRow(instId)} loading={updateMutation.isPending}>
            {updateMutation.isPending ? '下发中...' : '保 存'}
          </Button>
          <Popconfirm title="确认删除？" onConfirm={() => void handleDelete(instId)} disabled={!canDelete}>
            <Button size="small" danger disabled={!canDelete}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;

  return (
    <Card
      title={title}
      size="small"
      extra={
        <Space>
          {lastAction && (
            <Tag
              icon={lastAction.status === 'success' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
              color={lastAction.status === 'success' ? 'success' : 'error'}
            >
              上次{lastAction.action === 'save' ? '保存' : lastAction.action === 'add' ? '新增' : '删除'}:
              {lastAction.status === 'success' ? '成功' : '失败'} · {lastAction.detail} · {formatTime(lastAction.at)}
            </Tag>
          )}
          <Button type="primary" onClick={() => void handleAdd()} disabled={!canAdd} loading={addMutation.isPending}>
            {addMutation.isPending ? '下发中...' : '新 增'}
          </Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Table<string>
        rowKey={(instId) => instId}
        dataSource={instanceIds}
        columns={columns}
        loading={isLoading}
        size="small"
        pagination={false}
        scroll={{ x: 'max-content' }}
      />
    </Card>
  );
}
