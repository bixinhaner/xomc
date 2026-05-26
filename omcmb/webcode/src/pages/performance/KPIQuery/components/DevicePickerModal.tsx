/**
 * T-0174 设备选择 Modal。
 *
 * - 服务端分页 + SN/OUI/站点名 搜索 + 已选面板 + 批量粘贴。
 * - 用 frontend-core 的 useDeviceList（已含 search 参数）。
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
  Tooltip,
  App,
  Pagination,
  Divider,
} from 'antd';
import { SearchOutlined, CopyOutlined, ClearOutlined } from '@ant-design/icons';
import { useDeviceList } from '@core/hooks/api/useDevices';
import type { Device } from '@core/types/device';

const { Text } = Typography;
const { TextArea } = Input;

interface DevicePickerModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (selectedSns: string[]) => void;
  initialSelected?: string[];
}

export default function DevicePickerModal({
  open,
  onClose,
  onConfirm,
  initialSelected = [],
}: DevicePickerModalProps) {
  const { message } = App.useApp();
  const [search, setSearch] = useState('');
  const [searchDraft, setSearchDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<string[]>(initialSelected);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');

  // 不用 useEffect 同步 — Modal destroyOnHidden 关闭即卸载，useState 初值在下次打开时取到最新 initialSelected。

  const { data, isLoading } = useDeviceList(
    { page, pageSize, searchText: search || undefined },
    { enabled: open },
  );

  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns = useMemo(
    () => [
      { title: 'SN', dataIndex: 'sn', key: 'sn', width: 200, ellipsis: true },
      { title: '主机名', dataIndex: 'hostName', key: 'host', width: 180, ellipsis: true },
      { title: '制式', dataIndex: 'networkType', key: 'tech', width: 90 },
      { title: '运营商', dataIndex: 'carrier', key: 'carrier', width: 90 },
      {
        title: '在线状态',
        dataIndex: 'isOnline',
        key: 'online',
        width: 90,
        render: (v: boolean) =>
          v ? <Tag color="green">在线</Tag> : <Tag color="default">离线</Tag>,
      },
    ],
    [],
  );

  const handleSearch = () => {
    setSearch(searchDraft.trim());
    setPage(1);
  };

  const handleBatchPaste = () => {
    const raw = pasteText.trim();
    if (!raw) {
      message.warning('请粘贴 SN 列表');
      return;
    }
    const sns = raw
      .split(/[,;\n\r\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (sns.length === 0) {
      message.warning('未解析到 SN');
      return;
    }
    const merged = Array.from(new Set([...selected, ...sns]));
    const added = merged.length - selected.length;
    setSelected(merged);
    setPasteOpen(false);
    setPasteText('');
    message.success(`已加入 ${added} 个 SN（共选中 ${merged.length} 个）`);
  };

  const handleConfirm = () => {
    onConfirm(selected);
    onClose();
  };

  const rowSelection = {
    selectedRowKeys: selected,
    onChange: (keys: React.Key[]) => setSelected(keys as string[]),
  };

  return (
    <>
      <Modal
        title={`选择设备（已选 ${selected.length} 个）`}
        open={open}
        onCancel={onClose}
        onOk={handleConfirm}
        okText="确定"
        cancelText="取消"
        width={900}
        destroyOnHidden
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Space>
            <Input
              placeholder="按 SN / OUI / 主机名 搜索"
              prefix={<SearchOutlined />}
              value={searchDraft}
              onChange={(e) => setSearchDraft(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 320 }}
              allowClear
              onClear={() => {
                setSearchDraft('');
                setSearch('');
                setPage(1);
              }}
            />
            <Button onClick={handleSearch} type="primary">
              搜索
            </Button>
            <Button icon={<CopyOutlined />} onClick={() => setPasteOpen(true)}>
              批量粘贴
            </Button>
          </Space>

          <Table<Device>
            rowKey="sn"
            size="small"
            loading={isLoading}
            columns={columns}
            dataSource={items}
            rowSelection={{ ...rowSelection, preserveSelectedRowKeys: true }}
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
              <Text strong>已选 SN：</Text>
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
                <Text type="secondary">未选择任何设备</Text>
              ) : (
                selected.map((sn) => (
                  <Tooltip key={sn} title={sn}>
                    <Tag
                      closable
                      onClose={() => setSelected(selected.filter((s) => s !== sn))}
                      style={{ marginBottom: 4 }}
                    >
                      {sn}
                    </Tag>
                  </Tooltip>
                ))
              )}
            </div>
          </div>
        </Space>
      </Modal>

      <Modal
        title="批量粘贴 SN"
        open={pasteOpen}
        onCancel={() => setPasteOpen(false)}
        onOk={handleBatchPaste}
        okText="加入已选"
        cancelText="取消"
        width={520}
        destroyOnHidden
      >
        <Text type="secondary">支持逗号、分号、空白字符或换行分隔；自动去重</Text>
        <TextArea
          rows={8}
          value={pasteText}
          onChange={(e) => setPasteText(e.target.value)}
          placeholder={'SN-001\nSN-002, SN-003'}
          style={{ marginTop: 8 }}
        />
      </Modal>
    </>
  );
}
