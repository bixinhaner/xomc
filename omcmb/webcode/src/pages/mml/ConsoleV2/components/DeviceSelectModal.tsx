import { useMemo, useState } from 'react';
import { Badge, Button, Checkbox, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useDeviceList, useVerifyOnlineSns } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import { useDictionaryBatch } from '@core/hooks/api/useSystem';
import type { DeviceItem, DeviceStatus } from '../types';
import { DEVICE_MODAL_PAGE_SIZE, MAX_SELECT_ALL } from '../constants';
import { mapDeviceToItem } from '../adapters';

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
 * 设备选择弹框（设计 §3.3.1 + §3.10.1 + §3.12）—— SN 搜索 + 产品 + 产品类型筛选 + 表格多选 +
 * 批量输入。**表头全选 = 选中筛选命中的全部数据（跨页，上限 MAX_SELECT_ALL=200）**。
 * 数据来源：`useDeviceList`（服务端分页/筛选）；表格走分页查询，表头全选额外用一条
 * pageSize=200 的查询取回筛选命中的 SN 集合。
 */
export default function DeviceSelectModal({
  open,
  value,
  onCancel,
  onConfirm,
}: DeviceSelectModalProps) {
  const [snInput, setSnInput] = useState('');
  const [snKeyword, setSnKeyword] = useState('');
  const [productFilter, setProductFilter] = useState<string | undefined>();
  const [classFilter, setClassFilter] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string[]>([]);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [wasOpen, setWasOpen] = useState(false);

  // 打开时恢复上次选择（不自动全选）。渲染阶段调整 state，避开 set-state-in-effect。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setSelected(value);
      setSnInput('');
      setSnKeyword('');
      setProductFilter(undefined);
      setClassFilter(undefined);
      setPage(1);
    }
  }

  // 产品装配件下拉（全量 /products，value = 产品 UUID）。
  const { data: productResp } = useProductList();
  const productOptions = useMemo(
    () => (productResp?.items ?? []).map((p) => ({ label: p.name, value: p.id })),
    [productResp],
  );

  // 产品类型字典（下拉选项）。
  const { data: dicts } = useDictionaryBatch(['product_class']);
  const classOptions = useMemo(
    () =>
      (dicts?.['product_class']?.sysDictionaryDetails ?? []).map((d) => ({
        label: d.label,
        value: d.value,
      })),
    [dicts],
  );

  // 设备列表始终只显示在线设备：is_online=true 固定下发，离线设备由后端过滤（§需求 1）。
  const filterParams = useMemo(
    () => ({
      isOnline: true as const,
      ...(snKeyword ? { searchText: snKeyword } : {}),
      ...(productFilter ? { productId: productFilter } : {}),
      ...(classFilter ? { productClass: classFilter } : {}),
    }),
    [snKeyword, productFilter, classFilter],
  );

  // 分页表格数据（服务端分页）。
  const { data: pageResp, isFetching } = useDeviceList(
    { page, pageSize: DEVICE_MODAL_PAGE_SIZE, ...filterParams },
    { enabled: open },
  );
  const rows = useMemo(() => (pageResp?.items ?? []).map(mapDeviceToItem), [pageResp]);
  const total = pageResp?.total ?? 0;

  // 表头全选目标 = 筛选命中的前 200 台 SN（跨页）。单独一条 pageSize=200 查询。
  const { data: allResp } = useDeviceList(
    { page: 1, pageSize: MAX_SELECT_ALL, ...filterParams },
    { enabled: open },
  );
  const cappedFilteredSns = useMemo(
    () => (allResp?.items ?? []).slice(0, MAX_SELECT_ALL).map((d) => d.sn),
    [allResp],
  );
  const overLimit = total > MAX_SELECT_ALL;
  const allFilteredSelected =
    cappedFilteredSns.length > 0 && cappedFilteredSns.every((sn) => selected.includes(sn));
  const someSelected = selected.length > 0 && !allFilteredSelected;

  // 批量输入：逐 SN 校验在线状态（不受当前筛选条件影响），离线/不存在的 SN 一律剔除（§需求 2）。
  const verifyOnline = useVerifyOnlineSns();

  const handlePasteConfirm = async (): Promise<void> => {
    const sns = pasteText
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    const uniqueInput = Array.from(new Set(sns));
    if (uniqueInput.length === 0) {
      setPasteOpen(false);
      setPasteText('');
      return;
    }

    let onlineSns: string[];
    try {
      onlineSns = await verifyOnline.mutateAsync(uniqueInput);
    } catch {
      Modal.error({ title: '批量输入失败', content: '校验设备在线状态失败，请重试。' });
      return;
    }

    const merged = new Set(selected);
    onlineSns.forEach((sn) => merged.add(sn));
    const next = Array.from(merged).slice(0, MAX_SELECT_ALL);
    const droppedCount = uniqueInput.length - onlineSns.length;
    setSelected(next);
    setPasteOpen(false);
    setPasteText('');
    Modal.info({
      title: '批量输入结果',
      content: `识别 ${uniqueInput.length} 个 SN，其中在线 ${onlineSns.length} 个已加入已选列表${
        droppedCount > 0 ? `，离线/不存在 ${droppedCount} 个已过滤` : ''
      }（上限 ${MAX_SELECT_ALL} 台，当前共 ${next.length} 台）。`,
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
            <Input.Search
              allowClear
              placeholder="设备编码：输入 SN 搜索"
              style={{ width: 240 }}
              value={snInput}
              onChange={(e) => setSnInput(e.target.value)}
              onSearch={(v) => {
                setSnKeyword(v.trim());
                setPage(1);
              }}
            />
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
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
              showSearch
              optionFilterProp="label"
              placeholder="产品类型：全部"
              style={{ width: 200 }}
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
                筛选命中 {total} 台，已超单次执行上限 {MAX_SELECT_ALL} 台，全选仅选中前 {MAX_SELECT_ALL} 台
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
            loading={isFetching}
            columns={columns}
            dataSource={rows}
            rowSelection={{
              selectedRowKeys: selected,
              onChange: (keys) => setSelected(keys as string[]),
              preserveSelectedRowKeys: true,
              // 表头全选接管为「筛选命中的全部数据（跨页，上限 200）」，而非仅当前页。
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
              total,
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
        confirmLoading={verifyOnline.isPending}
        okText="识别并加入"
        cancelText="取消"
        destroyOnHidden
      >
        <Input.TextArea
          rows={8}
          placeholder="每行一个 SN，或用空格 / 逗号 / 分号分隔（仅保留在线设备）"
          value={pasteText}
          onChange={(e) => setPasteText(e.target.value)}
        />
      </Modal>
    </>
  );
}
