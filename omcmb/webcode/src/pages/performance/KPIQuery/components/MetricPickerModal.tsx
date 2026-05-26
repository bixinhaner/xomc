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

interface MetricPickerModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (selectedPaths: string[]) => void;
  initialSelected?: string[];
  initialDeviceType?: DeviceType;
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
}: MetricPickerModalProps) {
  const [deviceType, setDeviceType] = useState<DeviceType>(initialDeviceType);
  const [keyword, setKeyword] = useState('');
  const [keywordDraft, setKeywordDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<string[]>(initialSelected);

  // 不用 useEffect 同步 — Modal destroyOnHidden 关闭即卸载，useState 初值在下次打开时取到最新 initialSelected/initialDeviceType。

  const { data, isLoading } = useIndicatorList(deviceType, {
    keyword: keyword || undefined,
    page,
    pageSize,
  });

  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns = useMemo(
    () => [
      {
        title: '指标路径',
        dataIndex: 'enName',
        key: 'path',
        width: 280,
        ellipsis: true,
        render: (n: string) => <Text code>{n}</Text>,
      },
      {
        title: '中文名',
        dataIndex: 'cnName',
        key: 'cn',
        width: 200,
        ellipsis: true,
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
    onConfirm(selected);
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
          rowKey="enName"
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
                  {path}
                </Tag>
              ))
            )}
          </div>
        </div>
      </Space>
    </Modal>
  );
}
