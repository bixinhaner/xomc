/**
 * T-0174 指标选择 Modal。
 *
 * - 复用 frontend-core useIndicatorList（GET /api/v1/indicators）。
 * - 关键词搜索 + 服务端分页 + 已选面板。
 * - 设备类型由调用方传入（按选定设备 technology 推断）。
 */

import { useMemo, useState } from 'react';
import {
  Modal,
  Input,
  Table,
  Space,
  Button,
  Tag,
  Typography,
  Pagination,
  Divider,
  Select,
} from 'antd';
import { SearchOutlined, ClearOutlined } from '@ant-design/icons';
import { useIndicatorList } from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';

const { Text } = Typography;

// 选中值 = 该指标在 pm_metrics 里的 metric_path：
//   KPI 落库键是 K 编号（id），counter 落库键是点分名（en_name）。
//   这样选中值直接作为查询过滤条件即可命中，无需再做映射。
function metricValueOf(r: IndicatorInfo): string {
  return r.isCounter ? r.enName ?? r.id : r.id;
}

// 选中标签的友好显示名：优先中文名，回退英文名 / 编号。
function metricLabelOf(r: IndicatorInfo): string {
  return r.cnName || r.enName || r.id;
}

interface MetricPickerModalProps {
  open: boolean;
  onClose: () => void;
  // labels: 选中值 → 友好名（KPI 友好名带 PLMN 标记）。调用方可用于摘要展示，不需要时可忽略。
  onConfirm: (selectedPaths: string[], labels: Record<string, string>) => void;
  initialSelected?: string[];
  initialDeviceType?: DeviceType;
  // 制式联动锁定（T-0188）：true 时隐藏内部「设备类型」下拉，deviceType 固定为
  // initialDeviceType 不可手动切换；不传/false 保持原下拉可切换行为（向后兼容 KPIQuery）。
  lockDeviceType?: boolean;
}

const DEVICE_TYPE_OPTIONS: { label: string; value: DeviceType }[] = [
  { label: 'eNB (LTE)', value: 'ENB' },
  { label: 'gNB (5G NR)', value: 'GNB' },
  { label: 'GSM (2G)', value: 'GSM' },
];

export default function MetricPickerModal({
  open,
  onClose,
  onConfirm,
  initialSelected = [],
  initialDeviceType = 'ENB',
  lockDeviceType = false,
}: MetricPickerModalProps) {
  // 非锁定态：用户可在弹窗内自行切换设备类型（KPIQuery 用法），用内部 state。
  const [deviceTypeState, setDeviceType] = useState<DeviceType>(initialDeviceType);
  const [keyword, setKeyword] = useState('');
  const [keywordDraft, setKeywordDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<string[]>(initialSelected);

  // 制式联动锁定（T-0188）：锁定态 deviceType 恒等于外部制式入参 initialDeviceType。
  // 本组件在调用页常驻不卸载（Modal 的 destroyOnHidden 只销毁弹窗 DOM 内容、不重挂载本组件），
  // 内部 state 仅首挂载赋值一次、不随外部制式切换更新；故锁定态直接取 prop，避免停留在首挂载制式。
  const deviceType = lockDeviceType ? initialDeviceType : deviceTypeState;

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    page,
    pageSize,
  });

  const items = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

  // 选中值 → 友好名 映射：items 变化时在渲染期幂等累积（与 PivotTable 列宽同款 render-phase sync，
  // 不用 useEffect），供「已选」面板标签显示友好名，避免露出 K 编号。
  const [labelMap, setLabelMap] = useState<Record<string, string>>({});
  const [seenItems, setSeenItems] = useState(items);
  if (seenItems !== items) {
    setSeenItems(items);
    if (items.length > 0) {
      setLabelMap((prev) => {
        const next = { ...prev };
        for (const it of items) next[metricValueOf(it)] = metricLabelOf(it);
        return next;
      });
    }
  }

  const columns = useMemo(
    () => [
      {
        title: '中文名',
        dataIndex: 'cnName',
        key: 'cn',
        width: 240,
        ellipsis: true,
        render: (n: string, r: IndicatorInfo) => n || <Text type="secondary">{r.enName}</Text>,
      },
      {
        title: '英文名 / 路径',
        dataIndex: 'enName',
        key: 'path',
        width: 260,
        ellipsis: true,
        render: (n: string) => <Text code>{n}</Text>,
      },
      {
        title: '类型',
        key: 'type',
        width: 80,
        render: (_: unknown, r: IndicatorInfo) =>
          r.isCounter ? <Tag color="blue">Counter</Tag> : <Tag color="orange">KPI</Tag>,
      },
    ],
    [],
  );

  const handleSearch = () => {
    setKeyword(keywordDraft.trim());
    setPage(1);
  };

  const handleConfirm = () => {
    const labels: Record<string, string> = {};
    selected.forEach((v) => {
      labels[v] = labelMap[v] ?? v;
    });
    onConfirm(selected, labels);
    onClose();
  };

  const rowSelection = {
    selectedRowKeys: selected,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => setSelected(keys as string[]),
  };

  return (
    <Modal
      title={`选择指标（已选 ${selected.length} 个）`}
      open={open}
      onCancel={onClose}
      onOk={handleConfirm}
      okText="确定"
      cancelText="取消"
      width={820}
      destroyOnHidden
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Space>
          {lockDeviceType ? null : (
            <>
              <Text>设备类型：</Text>
              <Select<DeviceType>
                value={deviceType}
                onChange={(v) => {
                  setDeviceType(v);
                  setPage(1);
                }}
                options={DEVICE_TYPE_OPTIONS}
                style={{ width: 140 }}
              />
            </>
          )}
          <Input
            placeholder="按指标路径 / 中文名 搜索"
            prefix={<SearchOutlined />}
            value={keywordDraft}
            onChange={(e) => setKeywordDraft(e.target.value)}
            onPressEnter={handleSearch}
            style={{ width: 280 }}
            allowClear
            onClear={() => {
              setKeywordDraft('');
              setKeyword('');
              setPage(1);
            }}
          />
          <Button onClick={handleSearch} type="primary">
            搜索
          </Button>
        </Space>

        <Table<IndicatorInfo>
          rowKey={metricValueOf}
          size="small"
          loading={isLoading}
          columns={columns}
          dataSource={items}
          rowSelection={rowSelection}
          pagination={false}
          scroll={{ y: 320 }}
        />

        <Pagination
          current={page}
          pageSize={pageSize}
          total={total}
          showSizeChanger
          showTotal={(t) => `共 ${t} 条`}
          onChange={(p, s) => {
            setPage(p);
            setPageSize(s);
          }}
        />

        <Divider style={{ margin: '4px 0' }} />

        <div>
          <Space style={{ marginBottom: 6 }}>
            <Text strong>已选指标：</Text>
            <Button
              size="small"
              icon={<ClearOutlined />}
              onClick={() => setSelected([])}
              disabled={selected.length === 0}
            >
              全部清空
            </Button>
          </Space>
          <div
            style={{
              maxHeight: 100,
              overflowY: 'auto',
              background: '#fafafa',
              padding: 8,
              borderRadius: 4,
            }}
          >
            {selected.length === 0 ? (
              <Text type="secondary">未选择任何指标</Text>
            ) : (
              selected.map((path) => (
                <Tag
                  key={path}
                  closable
                  onClose={() => setSelected(selected.filter((p) => p !== path))}
                  style={{ marginBottom: 4 }}
                >
                  {labelMap[path] ?? path}
                </Tag>
              ))
            )}
          </div>
        </div>
      </Space>
    </Modal>
  );
}
