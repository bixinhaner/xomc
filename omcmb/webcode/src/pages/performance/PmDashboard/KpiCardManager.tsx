/**
 * G6-Gap-13：KPI 卡片配置 UI — 添加 / 移除 / 排序。
 *
 * 写入 pm_user_dashboard_preferences.kpi_card_layout（按制式分键，G6-Gap-3 已建好）。
 *
 * Layout JSON 形态约定：
 *   {
 *     order: string[]   // KPI key 列表（按显示顺序）
 *     hidden: string[]  // 用户主动隐藏的 key
 *   }
 *
 * 可选 KPI 候选集由 BUILTIN_KPI_CARDS 给出（11 个常用 KPI）；后续可扩展为读
 * indicator 字典动态拉取，组件签名不变。
 */

import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Drawer,
  List,
  Space,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import {
  ArrowUpOutlined,
  ArrowDownOutlined,
  PlusOutlined,
  MinusOutlined,
} from '@ant-design/icons';
import {
  usePmUserPreferences,
  useUpdatePmUserPreferences,
} from '@core/hooks/api/usePmDashboard';
import type { Technology } from '@core/types/pmDashboard';

interface Props {
  open: boolean;
  technology: Technology;
  onClose: () => void;
}

interface KpiOption {
  key: string;
  label: string;
  metricPath: string;
  unit: string;
}

// 11 个常用 LTE / NR / GSM 公共 KPI（v1 内置；后续可改为从 indicator 字典拉）
const BUILTIN_KPI_CARDS: KpiOption[] = [
  { key: 'rrcSuccRate', label: 'RRC 建立成功率', metricPath: 'L.RRC.SuccRate', unit: '%' },
  { key: 'erabSuccRate', label: 'E-RAB 建立成功率', metricPath: 'L.ERAB.SuccRate', unit: '%' },
  { key: 'hoSuccRate', label: '切换成功率', metricPath: 'L.HO.SuccRate', unit: '%' },
  { key: 'dropRate', label: '掉话率', metricPath: 'L.Call.DropRate', unit: '%' },
  { key: 'voLteSuccRate', label: 'VoLTE 接通率', metricPath: 'L.VoLTE.SuccRate', unit: '%' },
  { key: 'dlThroughput', label: '下行平均吞吐量', metricPath: 'L.DL.Throughput', unit: 'Mbps' },
  { key: 'ulThroughput', label: '上行平均吞吐量', metricPath: 'L.UL.Throughput', unit: 'Mbps' },
  { key: 'cellAvail', label: '小区可用率', metricPath: 'L.Cell.Avail', unit: '%' },
  { key: 'activeUsers', label: '在线用户数', metricPath: 'L.User.Active', unit: '人' },
  { key: 'prbUsageDL', label: '下行 PRB 利用率', metricPath: 'L.PRB.DL.Usage', unit: '%' },
  { key: 'prbUsageUL', label: '上行 PRB 利用率', metricPath: 'L.PRB.UL.Usage', unit: '%' },
];

function readLayout(layout: Record<string, unknown> | undefined): { order: string[]; hidden: string[] } {
  if (!layout) return { order: [], hidden: [] };
  const order = Array.isArray(layout.order) ? (layout.order as string[]) : [];
  const hidden = Array.isArray(layout.hidden) ? (layout.hidden as string[]) : [];
  return { order, hidden };
}

export function KpiCardManager({ open, technology, onClose }: Props) {
  const { data: prefs } = usePmUserPreferences(technology);
  const updateMut = useUpdatePmUserPreferences();

  const initial = useMemo(() => readLayout(prefs?.kpiCardLayout), [prefs?.kpiCardLayout]);
  const [selected, setSelected] = useState<string[]>(initial.order);

  useEffect(() => {
    if (open) setSelected(initial.order);
  }, [open, initial.order]);

  const optionMap = useMemo(() => {
    const m = new Map<string, KpiOption>();
    BUILTIN_KPI_CARDS.forEach((o) => m.set(o.key, o));
    return m;
  }, []);

  const available = BUILTIN_KPI_CARDS.filter((o) => !selected.includes(o.key));

  const handleAdd = (key: string) => setSelected((s) => [...s, key]);
  const handleRemove = (key: string) => setSelected((s) => s.filter((k) => k !== key));
  const handleMove = (key: string, dir: -1 | 1) => {
    setSelected((s) => {
      const i = s.indexOf(key);
      if (i < 0) return s;
      const j = i + dir;
      if (j < 0 || j >= s.length) return s;
      const next = [...s];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  };

  const handleSave = async () => {
    await updateMut.mutateAsync({
      technology,
      kpiCardLayout: { order: selected, hidden: initial.hidden },
      currentDashboardId: prefs?.currentDashboardId,
      sharedFilters: prefs?.sharedFilters,
    });
    message.success('已保存 KPI 卡片配置');
    onClose();
  };

  return (
    <Drawer
      title={`KPI 卡片管理 (${technology.toUpperCase()})`}
      width={520}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" loading={updateMut.isPending} onClick={handleSave}>
            保存
          </Button>
        </Space>
      }
    >
      <Typography.Title level={5} style={{ marginTop: 0 }}>
        已选（{selected.length}）
      </Typography.Title>
      <List
        size="small"
        bordered
        dataSource={selected}
        locale={{ emptyText: '未选择，去下方候选区添加' }}
        renderItem={(key, i) => {
          const opt = optionMap.get(key);
          return (
            <List.Item
              actions={[
                <Tooltip key="up" title="上移">
                  <Button
                    size="small"
                    type="text"
                    icon={<ArrowUpOutlined />}
                    disabled={i === 0}
                    onClick={() => handleMove(key, -1)}
                  />
                </Tooltip>,
                <Tooltip key="dn" title="下移">
                  <Button
                    size="small"
                    type="text"
                    icon={<ArrowDownOutlined />}
                    disabled={i === selected.length - 1}
                    onClick={() => handleMove(key, 1)}
                  />
                </Tooltip>,
                <Tooltip key="rm" title="移除">
                  <Button
                    size="small"
                    type="text"
                    danger
                    icon={<MinusOutlined />}
                    onClick={() => handleRemove(key)}
                  />
                </Tooltip>,
              ]}
            >
              <Space>
                <Tag>{i + 1}</Tag>
                <span>{opt?.label ?? key}</span>
                {opt && <Tag color="default">{opt.metricPath}</Tag>}
                {opt?.unit && <Tag color="cyan">{opt.unit}</Tag>}
              </Space>
            </List.Item>
          );
        }}
      />

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        候选（{available.length}）
      </Typography.Title>
      <List
        size="small"
        bordered
        dataSource={available}
        locale={{ emptyText: '已全部添加' }}
        renderItem={(opt) => (
          <List.Item
            actions={[
              <Button
                key="add"
                size="small"
                type="text"
                icon={<PlusOutlined />}
                onClick={() => handleAdd(opt.key)}
              >
                添加
              </Button>,
            ]}
          >
            <Space>
              <span>{opt.label}</span>
              <Tag color="default">{opt.metricPath}</Tag>
              <Tag color="cyan">{opt.unit}</Tag>
            </Space>
          </List.Item>
        )}
      />
    </Drawer>
  );
}
