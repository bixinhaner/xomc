/**
 * T-0174 设备选择 Modal。
 *
 * - 服务端分页 + SN/OUI/站点名 搜索 + 已选面板 + 批量粘贴。
 * - 用 frontend-core 的 useDeviceList（已含 search 参数）。
 */

import { useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
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
  // 制式联动锁定（T-0188）：传入时按制式过滤设备列表（透传 useDeviceList 的 networkType，
  // 直接用小写 'lte'/'nr'/'gsm'）；不传时列全部设备，保持原行为（向后兼容 KPIQuery）。
  technology?: 'lte' | 'nr' | 'gsm';
  maxSelected?: number;
  maxSelectedMessageId?: string;
}

export default function DevicePickerModal({
  open,
  onClose,
  onConfirm,
  initialSelected = [],
  technology,
  maxSelected,
  maxSelectedMessageId = 'perf.picker.deviceLimitExceeded',
}: DevicePickerModalProps) {
  const intl = useIntl();
  const { message } = App.useApp();
  const [search, setSearch] = useState('');
  const [searchDraft, setSearchDraft] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selected, setSelected] = useState<string[]>(initialSelected);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');

  // 已选回显同步：本组件在调用页常驻不卸载（Modal 的 destroyOnHidden 只销毁弹窗 DOM、不重挂载本组件），
  // 内部 selected 仅首挂载赋值一次，关闭后用新 initialSelected 重开（如切换编辑不同模板）时不会自动刷新、
  // 会残留上次打开的选择。故用渲染期幂等同步（不用 useEffect）：监听 open 由关变开，把 selected 重置为
  // 最新入参；open 期间用户的勾选/批量粘贴不受影响（仅在打开瞬间同步一次）。
  const [prevOpen, setPrevOpen] = useState(open);
  if (prevOpen !== open) {
    setPrevOpen(open);
    if (open) setSelected(initialSelected);
  }

  const { data, isLoading } = useDeviceList(
    { page, pageSize, searchText: search || undefined, networkType: technology },
    { enabled: open },
  );

  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'perf.picker.colSn' }), dataIndex: 'sn', key: 'sn', width: 200, ellipsis: true },
      { title: intl.formatMessage({ id: 'perf.picker.colHostName' }), dataIndex: 'hostName', key: 'host', width: 180, ellipsis: true },
      { title: intl.formatMessage({ id: 'perf.picker.colTech' }), dataIndex: 'networkType', key: 'tech', width: 90 },
      {
        title: intl.formatMessage({ id: 'perf.picker.colOnline' }),
        dataIndex: 'isOnline',
        key: 'online',
        width: 90,
        render: (v: boolean) =>
          v ? (
            <Tag color="green">{intl.formatMessage({ id: 'perf.picker.online' })}</Tag>
          ) : (
            <Tag color="default">{intl.formatMessage({ id: 'perf.picker.offline' })}</Tag>
          ),
      },
    ],
    [intl],
  );

  const handleSearch = () => {
    setSearch(searchDraft.trim());
    setPage(1);
  };

  const warnIfTooManySelected = (count: number) => {
    if (maxSelected === undefined || count <= maxSelected) return false;
    message.warning(
      intl.formatMessage(
        { id: maxSelectedMessageId },
        { max: maxSelected, count },
      ),
    );
    return true;
  };

  const handleBatchPaste = () => {
    const raw = pasteText.trim();
    if (!raw) {
      message.warning(intl.formatMessage({ id: 'perf.picker.pasteSnList' }));
      return;
    }
    const sns = raw
      .split(/[,;\n\r\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (sns.length === 0) {
      message.warning(intl.formatMessage({ id: 'perf.picker.noSnParsed' }));
      return;
    }
    const merged = Array.from(new Set([...selected, ...sns]));
    const nextSelected = maxSelected === undefined ? merged : merged.slice(0, maxSelected);
    warnIfTooManySelected(merged.length);
    const added = Math.max(nextSelected.length - selected.length, 0);
    setSelected(nextSelected);
    setPasteOpen(false);
    setPasteText('');
    message.success(
      intl.formatMessage({ id: 'perf.picker.snAdded' }, { added, total: nextSelected.length }),
    );
  };

  const handleConfirm = () => {
    if (warnIfTooManySelected(selected.length)) return;
    onConfirm(selected);
    onClose();
  };

  const rowSelection = {
    selectedRowKeys: selected,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => {
      if (warnIfTooManySelected(keys.length)) return;
      setSelected(keys as string[]);
    },
  };

  return (
    <>
      <Modal
        title={intl.formatMessage({ id: 'perf.picker.deviceTitle' }, { count: selected.length })}
        open={open}
        onCancel={onClose}
        onOk={handleConfirm}
        okText={intl.formatMessage({ id: 'common.confirm' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        width={900}
        destroyOnHidden
      >
        <Space orientation="vertical" style={{ width: '100%' }} size="middle">
          <Space>
            <Input
              placeholder={intl.formatMessage({ id: 'perf.picker.searchDevicePlaceholder' })}
              prefix={<SearchOutlined />}
              value={searchDraft}
              onChange={(e) => setSearchDraft(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 320 }}
              allowClear
              onClear={() => {
                setSearch('');
                setPage(1);
              }}
            />
            <Button onClick={handleSearch} type="primary">
              {intl.formatMessage({ id: 'common.search' })}
            </Button>
            <Button icon={<CopyOutlined />} onClick={() => setPasteOpen(true)}>
              {intl.formatMessage({ id: 'perf.picker.batchPaste' })}
            </Button>
          </Space>

          <Table<Device>
            rowKey="sn"
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
            showTotal={(t) => intl.formatMessage({ id: 'perf.picker.totalCount' }, { total: t })}
            onChange={(p, s) => {
              setPage(p);
              setPageSize(s);
            }}
          />

          <Divider style={{ margin: '4px 0' }} />

          <div>
            <Space style={{ marginBottom: 6 }}>
              <Text strong>{intl.formatMessage({ id: 'perf.picker.selectedSn' })}</Text>
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={() => setSelected([])}
                disabled={selected.length === 0}
              >
                {intl.formatMessage({ id: 'common.clear' })}
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
                <Text type="secondary">{intl.formatMessage({ id: 'perf.picker.noSelectedDevice' })}</Text>
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
        title={intl.formatMessage({ id: 'perf.picker.batchPasteTitle' })}
        open={pasteOpen}
        onCancel={() => setPasteOpen(false)}
        onOk={handleBatchPaste}
        okText={intl.formatMessage({ id: 'perf.picker.addToSelected' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        width={520}
        destroyOnHidden
      >
        <Text type="secondary">{intl.formatMessage({ id: 'perf.picker.batchPasteHint' })}</Text>
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
