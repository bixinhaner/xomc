/**
 * AlarmLibraryPage — 告警库主页面(T-0179 drill-down 重设计)。
 *
 * 2026-05-29 用户决策:
 *   1. 与 product/param-model 页面 UI 对齐(drill-down 主从)
 *   2. 一级页面:NeTypesTable — 按 (ne_type, loaded_from) 聚合,每行点击下钻
 *   3. 二级页面:AlarmDefinitionTable — 按 ne_type 过滤,toolbar 含返回 + 新增(预填+锁定 ne_type)
 *   4. URL 同步 ?neType=ENB(刷新不回列表;返回按钮显式回一级)
 *   5. 8 个 icon 对齐 param-model 风格(返回 / 上传 / 重载 / 刷新 / 删除 / 详情 / 新增 / 警告)
 *
 * 2026-06-03 三页统一(本次):
 *   a. 合并「导入 XML / 重载 XML / 刷新缓存」为单个「导入 XML」按钮 —— 打开文件上传弹窗
 *      AlarmUploadXmlModal(端点 POST /alarm-definitions/upload-xml,后端上传内部已自动
 *      destructive 重载 + 刷新缓存);旧的 import-directory / cache/refresh 端点已下线。
 *   b. 一级 ne-types 表删除"来源(builtin/custom)"列,保留"加载源(loaded_from)"列。
 *   c. 所有文件均可删除,去掉 deletable 置灰守门。
 */
import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  Popconfirm,
  Tooltip,
  message,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  useAlarmDefinitionList,
  useAlarmNeTypeStats,
  useDeleteAlarmDefinition,
  useAlarmDeleteFile,
  useAlarmDownloadXml,
} from '@core/hooks/api/useAlarmDefinitions';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import type {
  AlarmDefinition,
  AlarmDefinitionFilter,
  AlarmNeTypeStat,
} from '@core/types/alarmDefinition';
import {
  filterDeviceScopedItemsByLicense,
  isDeviceStandardValueVisibleByLicense,
} from '@core/utils/licenseFeatures';
import AlarmDefinitionDrawer from './AlarmDefinitionDrawer';
import AlarmUploadXmlModal from './AlarmUploadXmlModal';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';
import {
  PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
  PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
} from '../pagination';
// 2026-05-29:"未识别频次"入口暂时隐藏(后端聚合 / 统计逻辑未完工,详见
// backlog T-0181)。组件文件 UnknownStatsModal.tsx 保留备用,功能就绪后:
//   1) 取消下方 import 注释  2) 恢复一级 toolbar 的 <Tooltip>+<Button>
//   3) 恢复 <UnknownStatsModal /> 渲染  4) 恢复 statsOpen state
// import UnknownStatsModal from './UnknownStatsModal';

const { Text } = Typography;
const EMPTY_LOADED_FROM = '__empty__';

const SEVERITY_TAG_COLORS: Record<number, string> = {
  1: 'red',
  31001: 'red',
  2: 'orange',
  31002: 'orange',
  3: 'gold',
  31003: 'gold',
  4: 'blue',
  31004: 'blue',
};

const FILTER_LABEL_STYLE = {
  color: 'rgba(0, 0, 0, 0.65)',
  fontSize: 12,
  whiteSpace: 'nowrap' as const,
};


export default function AlarmLibraryPage() {
  const t = useT();

  const [searchParams, setSearchParams] = useSearchParams();
  const selectedNeType = searchParams.get('neType') || undefined;
  const selectedLoadedFrom = (() => {
    const value = searchParams.get('loadedFrom');
    if (value === null) {
      return undefined;
    }
    return value === EMPTY_LOADED_FROM ? '' : value;
  })();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();
  const selectedNeTypeVisible = !selectedNeType
    || isDeviceStandardValueVisibleByLicense(selectedNeType, systemLicense, systemLicenseLoading);
  const inDetail = Boolean(selectedNeType && selectedNeTypeVisible);

  const setSelectedNeType = (next: string | undefined, loadedFrom?: string) => {
    const params = new URLSearchParams(searchParams);
    if (next) {
      params.set('neType', next);
      params.set('loadedFrom', loadedFrom === '' ? EMPTY_LOADED_FROM : (loadedFrom ?? ''));
    } else {
      params.delete('neType');
      params.delete('loadedFrom');
    }
    // #268: 进入下钻 / 返回列表都清空上一次的二级筛选(搜索词/严重级别/页码)。
    // 一级↔二级是同组件内切换,state 不随"返回"销毁,须在此显式重置。
    setDetailFilter({ page: 1, pageSize: PRODUCT_TABLE_DEFAULT_PAGE_SIZE });
    setSearchParams(params, { replace: false });
  };

  // ── 一级(NeTypes 聚合) ─────────────────────────────────────────
  // 2026-06-02 用户决策:一级列表恢复"名称"模糊搜索。后端 listNeTypes 不带过滤
  // 参数且数据量小(LTE/GSM/NR 等数行),故在前端按 neType 客户端过滤。
  const [neKeyword, setNeKeyword] = useState('');
  const [nePage, setNePage] = useState(1);
  const [nePageSize, setNePageSize] = useState(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);
  const { data: neTypesData, isLoading: isNeTypesLoading } = useAlarmNeTypeStats();
  const neTypesItems = useMemo<AlarmNeTypeStat[]>(() => {
    const all = filterDeviceScopedItemsByLicense(neTypesData?.items || [], systemLicense, systemLicenseLoading);
    const kw = neKeyword.trim().toLowerCase();
    if (!kw) return all;
    return all.filter((row) => row.neType.toLowerCase().includes(kw));
  }, [neTypesData, neKeyword, systemLicense, systemLicenseLoading]);

  // ── 二级(AlarmDefinition 详情) ─────────────────────────────────
  const [detailFilter, setDetailFilter] = useState<AlarmDefinitionFilter>({
    page: 1,
    pageSize: PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
  });
  useEffect(() => {
    if (systemLicenseLoading || !selectedNeType) return;
    if (isDeviceStandardValueVisibleByLicense(selectedNeType, systemLicense, false)) return;
    const params = new URLSearchParams(searchParams);
    params.delete('neType');
    params.delete('loadedFrom');
    setDetailFilter({ page: 1, pageSize: PRODUCT_TABLE_DEFAULT_PAGE_SIZE });
    setSearchParams(params, { replace: true });
  }, [searchParams, selectedNeType, setSearchParams, systemLicense, systemLicenseLoading]);
  const detailQueryFilter = useMemo<AlarmDefinitionFilter>(
    () => (
      selectedNeType
        ? { ...detailFilter, neType: selectedNeType, loadedFrom: selectedLoadedFrom }
        : detailFilter
    ),
    [detailFilter, selectedLoadedFrom, selectedNeType]
  );
  const { data: detailData, isLoading: isDetailLoading } = useAlarmDefinitionList(
    inDetail ? detailQueryFilter : { page: 1, pageSize: 1 }
  );
  const detailItems = useMemo(() => detailData?.items || [], [detailData]);

  // 严重级别下拉选项的数据源:独立查询,只按 neType 取全量,**不带 severityCode/keyword 过滤**。
  // 否则选项从被过滤后的 detailItems 派生时,选中某级别会让列表收缩到该级别,下拉随之只剩
  // 当前一项,无法直接切换其它级别(必须先清空)。单 ne_type severity 低基数,pageSize 取大值即可全覆盖。
  const severitySourceFilter = useMemo<AlarmDefinitionFilter>(
    () => ({ neType: selectedNeType, loadedFrom: selectedLoadedFrom, page: 1, pageSize: 200 }),
    [selectedLoadedFrom, selectedNeType]
  );
  const { data: severitySourceData } = useAlarmDefinitionList(
    inDetail ? severitySourceFilter : { page: 1, pageSize: 1 }
  );

  // ── 公共 ───────────────────────────────────────────────────────
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlarmDefinition | null>(null);
  // 2026-06-03:「导入 XML」改为打开文件上传弹窗(替代旧 import-directory 调用)。
  const [uploadOpen, setUploadOpen] = useState(false);
  // statsOpen 暂时移除 — "未识别频次"功能未完工(见 backlog T-0181)

  const delMut = useDeleteAlarmDefinition();
  const deleteFileMut = useAlarmDeleteFile();
  const downloadXmlMut = useAlarmDownloadXml();

  // 2026-06-03:严重级别下拉从 severitySourceData(不带 severityCode 过滤的独立查询)排重派生,
  // 选项稳定、不随选中收缩;与表格"严重级别"列口径仍一致(同一 alarm_definitions 数据源)。
  const severityOptions = useMemo(() => {
    const byCode = new Map<number, string>();
    (severitySourceData?.items || []).forEach((row) => {
      if (!byCode.has(row.severityCode)) {
        byCode.set(row.severityCode, row.severityName ?? '');
      }
    });
    return Array.from(byCode.entries())
      .map(([code, label]) => ({
        label: label ? `${code} - ${label}` : String(code),
        value: code,
      }))
      .sort((left, right) => left.value - right.value);
  }, [severitySourceData]);

  // ── 一级表列 ────────────────────────────────────────────────────
  // 2026-05-29:严重级别列头改 i18n;en 仍是 Critical/Major/Minor/Warning
  // (ITU-T X.733 标准术语),zh 翻成 严重/主要/次要/警告。
  const neTypesColumns = [
    {
      title: t('common.name'),
      dataIndex: 'neType',
      width: 160,
      render: (v: string, row: AlarmNeTypeStat) => (
        <Button
          type="link"
          size="small"
          onClick={() => setSelectedNeType(row.neType, row.loadedFrom)}
          style={{ padding: 0, fontWeight: 600 }}
        >
          {v}
        </Button>
      ),
    },
    {
      // 2026-06-03 用户决策:去掉"来源(builtin/custom)"列,保留"加载源(loaded_from)"列。
      // 加载源:显示完整路径(参考 product/kpi-library)
      title: t('common.loadedFrom'),
      dataIndex: 'loadedFrom',
      width: 420,
      ellipsis: true,
      render: (v: string) =>
        v ? <Tooltip title={v}><code>{v}</code></Tooltip> : <Tag color="processing">{t('alarmLibrary.cell.manualAdded')}</Tag>,
    },
    { title: t('alarmLibrary.col.totalCount'), dataIndex: 'total', width: 100 },
    {
      title: t('alarmLibrary.col.critical'),
      dataIndex: 'criticalCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="red">{v}</Tag> : <span>—</span>),
    },
    {
      title: t('alarmLibrary.col.major'),
      dataIndex: 'majorCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="orange">{v}</Tag> : <span>—</span>),
    },
    {
      title: t('alarmLibrary.col.minor'),
      dataIndex: 'minorCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="gold">{v}</Tag> : <span>—</span>),
    },
    {
      title: t('alarmLibrary.col.warning'),
      dataIndex: 'warningCnt',
      width: 100,
      render: (v: number) => (v > 0 ? <Tag color="blue">{v}</Tag> : <span>—</span>),
    },
    {
      // 2026-06-04 用户决策:内置(builtin)不可删 → 仅 custom 可删,内置置灰 + Tooltip。
      // 2026-06-05:加下载 XML 图标(builtin / custom 均可)。
      // #268:手工新增行(无加载源)同样可下载 —— 后端按 ne_type 从 DB 动态生成 XML。
      title: t('alarmLibrary.col.actions'),
      width: 110,
      render: (_: unknown, row: AlarmNeTypeStat) => (
        <Space>
          <Tooltip title={t('product.upload.downloadXml')}>
            <Button
              size="small"
              icon={<DownloadOutlined />}
              onClick={() =>
                downloadXmlMut
                  .mutateAsync(
                    row.loadedFrom
                      ? { loadedFrom: row.loadedFrom }
                      : { neType: row.neType },
                  )
                  .catch((e) => message.error((e as Error).message))
              }
            />
          </Tooltip>
          {row.deletable ? (
          <Popconfirm
            title={t('product.alarm.xml.delTitle')}
            description={
              <div style={{ maxWidth: 320 }}>
                {t('product.alarm.xml.deleteBullet1Pre')}<code>.deleted.&lt;ts&gt;</code>{t('product.alarm.xml.deleteBullet1Post')}
                <br />{t('product.alarm.xml.deleteBullet2', { count: row.total })}
              </div>
            }
            okText={t('common.delete')}
            okButtonProps={{ danger: true }}
            onConfirm={() =>
              deleteFileMut
                .mutateAsync(row.loadedFrom)
                .then((r) =>
                  message.success(
                    r.backup
                      ? t('product.alarm.xml.deleteSuccessWithBackup', { backup: r.backup })
                      : t('common.deleted'),
                  ),
                )
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
          ) : (
            <Tooltip title={t('common.builtinNoDelete')}>
              <Button size="small" danger icon={<DeleteOutlined />} disabled />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // ── 二级表列 ────────────────────────────────────────────────────
  const detailColumns = [
    {
      title: t('product.alarm.def.identifier'),
      dataIndex: 'identifier',
      width: 140,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    { title: t('alarmLibrary.col.cnName'), dataIndex: 'cnName', width: 200 },
    { title: t('alarmLibrary.col.enName'), dataIndex: 'enName', width: 220, ellipsis: true },
    {
      title: t('table.description'),
      dataIndex: 'description',
      width: 220,
      ellipsis: true,
      render: (v: string) => v || <span style={{ color: '#bfbfbf' }}>-</span>,
    },
    {
      title: t('alarmLibrary.col.severityLevel'),
      dataIndex: 'severityCode',
      width: 110,
      render: (v: number, row: AlarmDefinition) => (
        <Tag color={SEVERITY_TAG_COLORS[v] || 'default'}>
          {v} - {row.severityName}
        </Tag>
      ),
    },
    { title: t('alarmLibrary.col.eventType'), dataIndex: 'eventType', width: 130 },
    {
      title: t('alarmLibrary.col.uiVisible'),
      dataIndex: 'isShow',
      width: 80,
      render: (v: boolean) =>
        v ? <Tag color="success">{t('alarmLibrary.cell.yes')}</Tag> : <Tag>{t('alarmLibrary.cell.no')}</Tag>,
    },
    {
      title: t('alarmLibrary.col.actions'),
      width: 180,
      render: (_: unknown, row: AlarmDefinition) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(row);
              setDrawerOpen(true);
            }}
          >
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('product.alarm.confirmDeleteDef', { id: row.identifier })}
            onConfirm={() =>
              delMut
                .mutateAsync(row.identifier)
                .then(() => message.success(t('common.deleted')))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:
           · 列表态:重载 XML + 刷新缓存(全局操作放一级);"未识别频次"功能未完工,已隐藏
           · 详情态:左 [返回 + 当前 ne_type] / 右 [严重级别 + 搜索 + 新增](筛选靠右,与列表态视觉一致) */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          {inDetail ? (
            <Space wrap>
              <Button
                icon={<ArrowLeftOutlined />}
                onClick={() => setSelectedNeType(undefined)}
              >
                {t('common.back')}
              </Button>
              <Text strong>{t('product.alarm.defsOf', { neType: selectedNeType ?? '' })}</Text>
            </Space>
          ) : (
            // 列表态左侧:按"名称"(ne_type)模糊搜索;客户端过滤(数据量小)
            <SearchInput
              placeholder={t('product.alarm.neSearchPh')}
              allowClear
              onSearch={(v) => {
                setNeKeyword(v.trim());
                setNePage(1);
              }}
              style={{ width: 320 }}
              enterButton
            />
          )}
          <Space wrap>
            {inDetail ? (
              // 详情态右侧:搜索 + 严重级别筛选 + 新增。重载 XML / 刷新缓存 都是全局动作,留在一级。
              // 2026-05-29 用户决策:搜索框置于"严重级别"前(主操作前置,与 kpi-library
              // 详情态 toolbar 范式一致)。
              <>
                <SearchInput
                  placeholder={t('product.alarm.searchPh')}
                  allowClear
                  onSearch={(v) =>
                    setDetailFilter((f) => ({ ...f, keyword: v || undefined, page: 1 }))
                  }
                  style={{ width: 320 }}
                  enterButton
                />
                <Space size={4}>
                  <span style={FILTER_LABEL_STYLE}>{t('product.alarm.severityLabel')}</span>
                  <Select
                    placeholder={t('product.alarm.allFilter')}
                    allowClear
                    options={severityOptions}
                    value={detailFilter.severityCode}
                    onChange={(value) =>
                      setDetailFilter((current) => ({ ...current, severityCode: value, page: 1 }))
                    }
                    style={{ width: 180 }}
                  />
                </Space>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setEditing(null);
                    setDrawerOpen(true);
                  }}
                >
                  {t('product.alarm.btnNewDef')}
                </Button>
              </>
            ) : (
              // 2026-06-03 用户决策:合并为单个「导入 XML」按钮 → 打开文件上传弹窗。
              // 后端上传端点内部已自动 destructive 重载 + 刷新缓存,前端无需再单独调。
              <Button icon={<CloudUploadOutlined />} onClick={() => setUploadOpen(true)}>
                {t('common.importXml')}
              </Button>
            )}
          </Space>
        </Space>
      </Card>

      {/* drill-down 主体:列表态 ↔ 详情态 二选一 */}
      <Card size="small">
        {inDetail ? (
          <Table<AlarmDefinition>
            rowKey="id"
            loading={isDetailLoading}
            columns={[makeSeqColumn<AlarmDefinition>({ title: t('table.rowNumber'), current: detailData?.page || 1, pageSize: detailData?.pageSize || PRODUCT_TABLE_DEFAULT_PAGE_SIZE }), ...detailColumns]}
            dataSource={detailItems}
            size="small"
            pagination={{
              current: detailData?.page || 1,
              pageSize: detailData?.pageSize || PRODUCT_TABLE_DEFAULT_PAGE_SIZE,
              total: detailData?.total || 0,
              showSizeChanger: true,
              pageSizeOptions: PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
              showTotal: (n) => t('common.totalCount', { count: n }),
              onChange: (page, pageSize) =>
                setDetailFilter((f) => ({ ...f, page, pageSize })),
            }}
          />
        ) : (
          <Table<AlarmNeTypeStat>
            rowKey={(r) => `${r.neType}__${r.loadedFrom}`}
            loading={isNeTypesLoading}
            columns={[makeSeqColumn<AlarmNeTypeStat>({ title: t('table.rowNumber'), dataSource: neTypesItems }), ...neTypesColumns]}
            dataSource={neTypesItems}
            size="small"
            pagination={{
              current: nePage,
              pageSize: nePageSize,
              total: neTypesItems.length,
              showSizeChanger: true,
              pageSizeOptions: PRODUCT_TABLE_PAGE_SIZE_OPTIONS,
              showTotal: (n) => t('product.alarm.totalNeTypes', { count: n }),
              onChange: (p, ps) => {
                setNePage(p);
                if (ps !== nePageSize) setNePageSize(ps);
              },
            }}
          />
        )}
      </Card>

      <AlarmDefinitionDrawer
        open={drawerOpen}
        definition={editing}
        defaultNeType={!editing ? selectedNeType : undefined}
        lockNeType={!editing && Boolean(selectedNeType)}
        onClose={() => {
          setDrawerOpen(false);
          setEditing(null);
        }}
      />
      <AlarmUploadXmlModal open={uploadOpen} onClose={() => setUploadOpen(false)} />
      {/* <UnknownStatsModal open={statsOpen} onClose={() => setStatsOpen(false)} /> */}
    </div>
  );
}
