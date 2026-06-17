import { useMemo, useState } from 'react';
import { Badge, Button, Checkbox, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import { useDictionaryBatch } from '@core/hooks/api/useSystem';
import type { DeviceItem, DeviceStatus } from '../types';
import { DEVICE_MODAL_PAGE_SIZE, MAX_SELECT_ALL } from '../constants';
import { mapDeviceToItem } from '../adapters';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

// §需求 2：内存级记忆上次选择的产品 + 产品类型（弹框重开沿用，免去重复选择）。
// 仅模块变量，不落 localStorage —— 浏览器刷新/页面重新打开即清空。
let lastProductId: string | undefined;
let lastProductClass: string | undefined;

const STATUS_TAG: Record<DeviceStatus, { color: string; textKey: string }> = {
  online: { color: 'success', textKey: 'mml.consoleV2.deviceSelect.status.online' },
  offline: { color: 'default', textKey: 'mml.consoleV2.deviceSelect.status.offline' },
  alarm: { color: 'warning', textKey: 'mml.consoleV2.deviceSelect.status.alarm' },
};

interface DeviceSelectModalProps {
  open: boolean;
  /** 当前已选 SN（打开时回填） */
  value: string[];
  onCancel: () => void;
  /** 确认：返回所选 SN + 所选产品 ID（设备列表强制同一产品，productId 用于不支持 path 过滤）。 */
  onConfirm: (sns: string[], productId: string) => void;
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
  const t = useT();
  // 草稿态：编辑但未应用（输入 SN / 选产品 / 选产品类型都不实时触发查询，§需求 1）。
  const [snInput, setSnInput] = useState('');
  const [productFilter, setProductFilter] = useState<string | undefined>();
  const [classFilter, setClassFilter] = useState<string | undefined>();
  // 已应用态：实际驱动查询，仅在点「搜索」时由草稿同步而来。
  const [snKeyword, setSnKeyword] = useState('');
  const [productApplied, setProductApplied] = useState<string | undefined>();
  const [classApplied, setClassApplied] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEVICE_MODAL_PAGE_SIZE);
  const [selected, setSelected] = useState<string[]>([]);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [wasOpen, setWasOpen] = useState(false);
  // 批量输入命中的 SN 列表（激活后表格只显示这些 SN 且在线，忽略其它筛选条件）。
  const [snListFilter, setSnListFilter] = useState<string[]>([]);

  // 打开时恢复上次选择（不自动全选）。渲染阶段调整 state，避开 set-state-in-effect。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setSelected(value);
      setSnInput('');
      // §需求 2：沿用上次选择的产品 + 产品类型（无缓存时 product 由下方默认首个逻辑兜底）。
      setProductFilter(lastProductId);
      setProductApplied(lastProductId);
      setClassFilter(lastProductClass);
      setClassApplied(lastProductClass);
      setSnKeyword('');
      setSnListFilter([]);
      setPage(1);
      setPageSize(DEVICE_MODAL_PAGE_SIZE);
    }
  }

  // 产品装配件下拉（全量 /products，value = 产品 UUID）。
  const { data: productResp } = useProductList();
  const productOptions = useMemo(
    () => (productResp?.items ?? []).map((p) => ({ label: p.name, value: p.id })),
    [productResp],
  );

  // 产品下拉必选 + 默认第一个：产品列表就绪后若未选，渲染阶段同步默认第一个产品（设备列表强制
  // 同一产品）。沿用本组件 set-state-in-render 的派生态写法（见上方 open!==wasOpen），条件收敛不抖动。
  if (open && productApplied === undefined && productOptions.length > 0) {
    setProductFilter(productOptions[0].value);
    setProductApplied(productOptions[0].value);
  }

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

  // 设备列表始终只显示在线设备 + 强制同一产品（productId 必带）：批量输入也限定在所选产品内。
  const filterParams = useMemo(
    () =>
      snListFilter.length > 0
        ? {
            isOnline: true as const,
            snList: snListFilter,
            ...(productApplied ? { productId: productApplied } : {}),
          }
        : {
            isOnline: true as const,
            ...(snKeyword ? { searchText: snKeyword } : {}),
            ...(productApplied ? { productId: productApplied } : {}),
            ...(classApplied ? { productClass: classApplied } : {}),
          },
    [snListFilter, snKeyword, productApplied, classApplied],
  );

  // 分页表格数据（服务端分页）。
  const { data: pageResp, isFetching } = useDeviceList(
    { page, pageSize, ...filterParams },
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

  // 点「搜索」才把草稿筛选条件应用到查询（输入/选择不实时触发，§需求 1）。
  const doSearch = (): void => {
    setSnKeyword(snInput.trim());
    setProductApplied(productFilter);
    setClassApplied(classFilter);
    setSnListFilter([]); // 普通搜索退出批量输入模式
    setPage(1);
  };

  // 批量输入：清空已选与 SN/类型筛选，但**保留所选产品**（设备列表强制同一产品，批量也限定在产品内）。
  const handlePasteConfirm = (): void => {
    const sns = pasteText
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    const uniqueInput = Array.from(new Set(sns));
    setSnListFilter(uniqueInput);
    setSelected([]); // 清空选中的数据
    setSnInput('');
    setClassFilter(undefined);
    setSnKeyword('');
    setClassApplied(undefined);
    setPage(1);
    setPasteOpen(false);
    setPasteText('');
  };

  // 列顺序：SN → 状态 → 产品 → 产品类型（状态前置、去掉设备分组，§需求 3）。
  const columns: ColumnsType<DeviceItem> = [
    { title: 'SN', dataIndex: 'sn', key: 'sn', width: 220, ellipsis: true },
    {
      title: t('mml.consoleV2.deviceSelect.col.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: DeviceStatus) => <Tag color={STATUS_TAG[s].color}>{t(STATUS_TAG[s].textKey)}</Tag>,
    },
    { title: t('mml.consoleV2.deviceSelect.col.product'), dataIndex: 'productName', key: 'productName', width: 180, ellipsis: true },
    { title: t('mml.consoleV2.deviceSelect.col.productClass'), dataIndex: 'productClass', key: 'productClass', width: 120 },
  ];

  return (
    <>
      <Modal
        title={t('mml.consoleV2.deviceSelect.title')}
        open={open}
        width={860}
        onCancel={onCancel}
        onOk={() => {
          // §需求 2：记住本次选择的产品 + 产品类型，下次打开弹框沿用。
          lastProductId = productApplied;
          lastProductClass = classApplied;
          onConfirm(selected, productApplied ?? '');
        }}
        okText={t('mml.consoleV2.deviceSelect.okText', { count: selected.length })}
        cancelText={t('common.cancel')}
        okButtonProps={{ disabled: selected.length === 0 }}
        destroyOnHidden
      >
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          {/* 输入 SN / 选产品 / 选产品类型均为草稿，点蓝色「搜索」按钮才触发查询（§需求 1）。 */}
          <Space wrap>
            <Input
              allowClear
              placeholder={t('mml.consoleV2.deviceSelect.snPlaceholder')}
              style={{ width: 200 }}
              value={snInput}
              onChange={(e) => setSnInput(e.target.value)}
              onPressEnter={doSearch}
            />
            <Select
              showSearch
              optionFilterProp="label"
              placeholder={t('mml.consoleV2.deviceSelect.productPlaceholder')}
              style={{ width: 200 }}
              options={productOptions}
              value={productFilter}
              onChange={(v) => setProductFilter(v)}
            />
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder={t('mml.consoleV2.deviceSelect.classPlaceholder')}
              style={{ width: 200 }}
              options={classOptions}
              value={classFilter}
              onChange={(v) => setClassFilter(v)}
            />
            <Button type="primary" icon={<SearchOutlined />} onClick={doSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={() => setPasteOpen(true)}>{t('mml.consoleV2.deviceSelect.batchInput')}</Button>
          </Space>

          <Space wrap>
            <Badge status="processing" text={<Text>{t('mml.consoleV2.deviceSelect.selectedCount', { count: selected.length })}</Text>} />
            {snListFilter.length > 0 && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('mml.consoleV2.deviceSelect.batchSummary', { count: snListFilter.length })}
                <Button type="link" size="small" onClick={() => setSnListFilter([])}>
                  {t('mml.consoleV2.deviceSelect.clear')}
                </Button>
              </Text>
            )}
            {overLimit && (
              <Text type="warning" style={{ fontSize: 12 }}>
                {t('mml.consoleV2.deviceSelect.overLimit', { total, max: MAX_SELECT_ALL })}
              </Text>
            )}
            {selected.length > 0 && (
              <Button type="link" size="small" onClick={() => setSelected([])}>
                {t('mml.consoleV2.deviceSelect.clearSelected')}
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
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOptions: ['10', '20', '50'],
              // 切页或改每页条数都驱动服务端重新查询（§需求 2）。
              onChange: (p, ps) => {
                setPage(p);
                setPageSize(ps);
              },
              showTotal: (count) => t('mml.consoleV2.deviceSelect.totalDevices', { count }),
              size: 'small',
            }}
            scroll={{ y: 320 }}
          />
        </Space>
      </Modal>

      <Modal
        title={t('mml.consoleV2.deviceSelect.batchInput')}
        open={pasteOpen}
        onCancel={() => setPasteOpen(false)}
        onOk={handlePasteConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        <Input.TextArea
          rows={8}
          placeholder={t('mml.consoleV2.deviceSelect.pastePlaceholder')}
          value={pasteText}
          onChange={(e) => setPasteText(e.target.value)}
        />
      </Modal>
    </>
  );
}
