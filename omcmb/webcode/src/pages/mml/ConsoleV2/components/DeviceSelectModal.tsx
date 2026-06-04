import { useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Checkbox,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { DeviceItem, DeviceStatus } from '../types';
import { MOCK_DEVICES } from '../mock';
import { DEVICE_MODAL_PAGE_SIZE, MAX_SELECT_ALL } from '../constants';

const { Text } = Typography;

const STATUS_TAG: Record<DeviceStatus, { color: string; text: string }> = {
  online: { color: 'success', text: '在线' },
  offline: { color: 'default', text: '离线' },
  alarm: { color: 'warning', text: '告警' },
};

interface DeviceSelectModalProps {
  open: boolean;
  /** 当前已选 SN（打开时回填） */
  value: string[];
  onCancel: () => void;
  onConfirm: (sns: string[]) => void;
}

/**
 * 设备选择弹框（设计 §3.3.1 + §3.10.1，2026-06-04 修订）—— SN/产品/产品类型 三维 AND 筛选
 * + 表格多选 + 批量输入。**表头全选 = 选中筛选命中的全部数据（跨页，不限当前页）**，上限
 * MAX_SELECT_ALL=200。当前为 mock：数据来自 MOCK_DEVICES，前端内存过滤/分页（真实接入时
 * 改服务端过滤 + GET /devices 取回筛选 SN）。
 */
export default function DeviceSelectModal({
  open,
  value,
  onCancel,
  onConfirm,
}: DeviceSelectModalProps) {
  const [snKeyword, setSnKeyword] = useState('');
  const [productFilter, setProductFilter] = useState<string | undefined>();
  const [classFilter, setClassFilter] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string[]>([]);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [wasOpen, setWasOpen] = useState(false);

  // 打开时恢复上次选择（不自动全选）。渲染阶段调整 state,避开 set-state-in-effect。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setSelected(value);
      setSnKeyword('');
      setProductFilter(undefined);
      setClassFilter(undefined);
      setPage(1);
    }
  }

  const filtered = useMemo(() => {
    const kw = snKeyword.trim().toLowerCase();
    return MOCK_DEVICES.filter((d) => {
      if (kw && !d.sn.toLowerCase().includes(kw)) return false;
      if (productFilter && d.productName !== productFilter) return false;
      if (classFilter && d.productClass !== classFilter) return false;
      return true;
    });
  }, [snKeyword, productFilter, classFilter]);

  const overLimit = filtered.length > MAX_SELECT_ALL;
  // 表头全选目标 = 筛选命中的前 200 台 SN（跨页）。
  const cappedFilteredSns = useMemo(
    () => filtered.slice(0, MAX_SELECT_ALL).map((d) => d.sn),
    [filtered],
  );
  const allFilteredSelected =
    cappedFilteredSns.length > 0 && cappedFilteredSns.every((sn) => selected.includes(sn));
  const someSelected = selected.length > 0 && !allFilteredSelected;

  const productOptions = useMemo(
    () => Array.from(new Set(MOCK_DEVICES.map((d) => d.productName))).map((p) => ({ label: p, value: p })),
    [],
  );
  const classOptions = useMemo(
    () => Array.from(new Set(MOCK_DEVICES.map((d) => d.productClass))).map((c) => ({ label: c, value: c })),
    [],
  );

  const handlePasteConfirm = () => {
    const sns = pasteText
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    const valid = new Set(MOCK_DEVICES.map((d) => d.sn));
    const merged = new Set(selected);
    let matched = 0;
    sns.forEach((sn) => {
      if (valid.has(sn)) {
        merged.add(sn);
        matched += 1;
      }
    });
    setSelected(Array.from(merged).slice(0, MAX_SELECT_ALL));
    setPasteOpen(false);
    setPasteText('');
    Modal.info({
      title: '批量输入结果',
      content: `识别 ${sns.length} 个 SN，匹配设备 ${matched} 台${
        sns.length - matched > 0 ? `，忽略未知 ${sns.length - matched} 个` : ''
      }。`,
    });
  };

  const columns: ColumnsType<DeviceItem> = [
    { title: '设备编码(SN)', dataIndex: 'sn', key: 'sn', width: 200, ellipsis: true },
    { title: '产品', dataIndex: 'productName', key: 'productName', width: 160, ellipsis: true },
    { title: '产品类型', dataIndex: 'productClass', key: 'productClass', width: 100 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: DeviceStatus) => <Tag color={STATUS_TAG[s].color}>{STATUS_TAG[s].text}</Tag>,
    },
    { title: '设备分组', dataIndex: 'groupName', key: 'groupName', width: 120, ellipsis: true },
  ];

  return (
    <>
      <Modal
        title="选择设备"
        open={open}
        width={860}
        onCancel={onCancel}
        onOk={() => onConfirm(selected)}
        okText={`确定（已选 ${selected.length} 台）`}
        cancelText="取消"
        okButtonProps={{ disabled: selected.length === 0 }}
        destroyOnHidden
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Space wrap>
            <Input
              allowClear
              placeholder="设备编码：输入 SN 模糊搜索"
              style={{ width: 240 }}
              value={snKeyword}
              onChange={(e) => {
                setSnKeyword(e.target.value);
                setPage(1);
              }}
            />
            <Select
              allowClear
              placeholder="产品：全部"
              style={{ width: 200 }}
              options={productOptions}
              value={productFilter}
              onChange={(v) => {
                setProductFilter(v);
                setPage(1);
              }}
            />
            <Select
              allowClear
              placeholder="产品类型：全部"
              style={{ width: 150 }}
              options={classOptions}
              value={classFilter}
              onChange={(v) => {
                setClassFilter(v);
                setPage(1);
              }}
            />
            <Button onClick={() => setPasteOpen(true)}>批量输入</Button>
          </Space>

          <Space wrap>
            <Badge status="processing" text={<Text>已选 {selected.length} 台</Text>} />
            {overLimit && (
              <Text type="warning" style={{ fontSize: 12 }}>
                筛选命中 {filtered.length} 台，已超单次执行上限 {MAX_SELECT_ALL} 台，全选仅选中前 {MAX_SELECT_ALL} 台
              </Text>
            )}
            {selected.length > 0 && (
              <Button type="link" size="small" onClick={() => setSelected([])}>
                清空已选
              </Button>
            )}
          </Space>

          <Table<DeviceItem>
            rowKey="sn"
            size="small"
            columns={columns}
            dataSource={filtered}
            rowSelection={{
              selectedRowKeys: selected,
              onChange: (keys) => setSelected(keys as string[]),
              preserveSelectedRowKeys: true,
              // 表头全选接管为「筛选命中的全部数据（跨页,上限 200）」,而非仅当前页。
              columnTitle: (
                <Checkbox
                  checked={allFilteredSelected}
                  indeterminate={someSelected}
                  onChange={(e) => setSelected(e.target.checked ? cappedFilteredSns : [])}
                />
              ),
            }}
            pagination={{
              current: page,
              pageSize: DEVICE_MODAL_PAGE_SIZE,
              total: filtered.length,
              onChange: setPage,
              showTotal: (t) => `共 ${t} 台`,
              size: 'small',
            }}
            scroll={{ y: 320 }}
          />
        </Space>
      </Modal>

      <Modal
        title="批量输入"
        open={pasteOpen}
        onCancel={() => setPasteOpen(false)}
        onOk={handlePasteConfirm}
        okText="识别并加入"
        cancelText="取消"
        destroyOnHidden
      >
        <Input.TextArea
          rows={8}
          placeholder="每行一个 SN，或用空格 / 逗号 / 分号分隔"
          value={pasteText}
          onChange={(e) => setPasteText(e.target.value)}
        />
      </Modal>
    </>
  );
}
