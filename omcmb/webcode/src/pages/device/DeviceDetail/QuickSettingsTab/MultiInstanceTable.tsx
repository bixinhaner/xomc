import { useCallback, useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Popconfirm, Space, Table, Typography, message } from 'antd';
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
      message.error('行校验失败，请修正');
      return;
    }
    if (updates.length === 0) {
      message.info('该行无变更');
      return;
    }
    try {
      await updateMutation.mutateAsync({ deviceId, parameters: updates });
      message.success(`第 ${instId} 行已下发 ${updates.length} 项变更`);
      // 清空 edits（schema 重新拉取时会同步当前值）
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.delete(instId);
        return next;
      });
      void refetch();
    } catch (err) {
      message.error(`第 ${instId} 行下发失败，输入值已保留可重试`);
      console.error('MultiInstanceTable: row save failed', err);
    }
  };

  const handleAdd = async () => {
    try {
      // objectPath 已含尾部 "."；AddObject 后端期望同样形态（参考 useAddObject 现有调用）
      await addMutation.mutateAsync({ deviceId, objectPath });
      message.success('已下发 AddObject，请在新行填值后点击 Save 完成 SetParameterValues');
      void refetch();
    } catch (err) {
      message.error('AddObject 失败');
      console.error('MultiInstanceTable: AddObject failed', err);
    }
  };

  const handleDelete = async (instId: string) => {
    Modal.confirm({
      title: '确认删除',
      content: `删除实例 ${instId} ？`,
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteMutation.mutateAsync({ deviceId, objectPath: `${objectPath}${instId}.` });
          message.success(`已下发 DeleteObject(${instId})`);
          void refetch();
        } catch (err) {
          message.error('DeleteObject 失败');
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
            保存
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
          <Button type="primary" onClick={() => void handleAdd()} disabled={!canAdd} loading={addMutation.isPending}>
            新增
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
