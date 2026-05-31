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
 * 2026-05-29 二次调整(本次):
 *   a. "重载 XML" / "刷新缓存" 只在一级页面保留,详情页面不显示
 *   b. 列表头改 i18n,中英文随站点语言切换
 *   c. 取消"搜索网元类型"输入框 — 数据极少(LTE/GSM/NR 等),无搜索必要
 */
import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Input,
  Select,
  Popconfirm,
  message,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import {
  useAlarmDefinitionList,
  useAlarmNeTypeStats,
  useDeleteAlarmDefinition,
  useAlarmDefinitionImportDirectory,
  useAlarmDefinitionReloadDirectory,
  useAlarmDefinitionCacheRefresh,
} from '@core/hooks/api/useAlarmDefinitions';
import type {
  AlarmDefinition,
  AlarmDefinitionFilter,
  AlarmNeTypeStat,
} from '@core/types/alarmDefinition';
import AlarmDefinitionDrawer from './AlarmDefinitionDrawer';
import { useT } from '@/hooks/useT';
// 2026-05-29:"未识别频次"入口暂时隐藏(后端聚合 / 统计逻辑未完工,详见
// backlog T-0181)。组件文件 UnknownStatsModal.tsx 保留备用,功能就绪后:
//   1) 取消下方 import 注释  2) 恢复一级 toolbar 的 <Tooltip>+<Button>
//   3) 恢复 <UnknownStatsModal /> 渲染  4) 恢复 statsOpen state
// import UnknownStatsModal from './UnknownStatsModal';

const { Text } = Typography;

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
  const inDetail = Boolean(selectedNeType);

  const setSelectedNeType = (next: string | undefined) => {
    const params = new URLSearchParams(searchParams);
    if (next) {
      params.set('neType', next);
    } else {
      params.delete('neType');
    }
    setSearchParams(params, { replace: false });
  };

  // ── 一级(NeTypes 聚合) ─────────────────────────────────────────
  // 2026-05-29 调整:LTE/GSM/NR 三五行数据,搜索框无意义,已删除。
  const { data: neTypesData, isLoading: isNeTypesLoading } = useAlarmNeTypeStats();
  const neTypesItems = useMemo<AlarmNeTypeStat[]>(
    () => neTypesData?.items || [],
    [neTypesData],
  );

  // ── 二级(AlarmDefinition 详情) ─────────────────────────────────
  const [detailFilter, setDetailFilter] = useState<AlarmDefinitionFilter>({
    page: 1,
    pageSize: 20,
  });
  const detailQueryFilter = useMemo<AlarmDefinitionFilter>(
    () => (selectedNeType ? { ...detailFilter, neType: selectedNeType } : detailFilter),
    [detailFilter, selectedNeType]
  );
  const { data: detailData, isLoading: isDetailLoading } = useAlarmDefinitionList(
    inDetail ? detailQueryFilter : { page: 1, pageSize: 1 }
  );
  const detailItems = useMemo(() => detailData?.items || [], [detailData]);

  // ── 公共 ───────────────────────────────────────────────────────
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlarmDefinition | null>(null);
  // statsOpen 暂时移除 — "未识别频次"功能未完工(见 backlog T-0181)

  const delMut = useDeleteAlarmDefinition();
  // 2026-05-29:与 parammodel 对齐拆 import / reload 两个 mutation,UI 拆两按钮:
  //   - 导入 XML(import):加法 UPSERT,不删孤儿(UI 中手工添加的告警保留)
  //   - 重载 XML(reload):destructive 全量重载,删 DB 中无 XML 对应的孤儿
  const importMut = useAlarmDefinitionImportDirectory();
  const reloadMut = useAlarmDefinitionReloadDirectory();
  const cacheMut = useAlarmDefinitionCacheRefresh();

  // 2026-05-29 用户决策:过滤下拉的源不再走 /alarm-severity-levels(后端原始数组
  // + 无 json tag + 缺 cn/en 拆分,前端 sevData 永远空 → 下拉为空);改为从当前
  // 页 detailItems 的 severityCode/severityName 排重派生 — 与表格"严重级别"列
  // 100% 一致,不出现"有选项但表里没数据"的悖论。
  // 局限:服务端分页时其它页存在但本页缺失的 severity 不会进下拉。考虑到
  // 单 ne_type 的 severity 多为 4 个低基数(Critical/Major/Minor/Warning),
  // pageSize=20 的首页通常已全覆盖。
  const severityOptions = useMemo(() => {
    const byCode = new Map<number, string>();
    detailItems.forEach((row) => {
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
  }, [detailItems]);

  // ── 一级表列 ────────────────────────────────────────────────────
  // 2026-05-29:严重级别列头改 i18n;en 仍是 Critical/Major/Minor/Warning
  // (ITU-T X.733 标准术语),zh 翻成 严重/主要/次要/警告。
  const neTypesColumns = [
    {
      title: t('alarmLibrary.col.neType'),
      dataIndex: 'neType',
      width: 160,
      render: (v: string, row: AlarmNeTypeStat) => (
        <Button
          type="link"
          size="small"
          onClick={() => setSelectedNeType(row.neType)}
          style={{ padding: 0, fontWeight: 600 }}
        >
          {v}
        </Button>
      ),
    },
    {
      title: t('alarmLibrary.col.xmlSource'),
      dataIndex: 'loadedFrom',
      width: 200,
      render: (v: string) =>
        v ? <Tag>{v}</Tag> : <Tag color="warning">{t('alarmLibrary.cell.unfilled')}</Tag>,
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
  ];

  // ── 二级表列 ────────────────────────────────────────────────────
  const detailColumns = [
    {
      title: 'identifier',
      dataIndex: 'identifier',
      width: 140,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    { title: t('alarmLibrary.col.cnName'), dataIndex: 'cnName', width: 200 },
    { title: t('alarmLibrary.col.enName'), dataIndex: 'enName', width: 220, ellipsis: true },
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
      width: 130,
      render: (_: unknown, row: AlarmDefinition) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setEditing(row);
              setDrawerOpen(true);
            }}
          />
          <Popconfirm
            title={`确认删除告警定义「${row.identifier}」?`}
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
                返回
              </Button>
              <Text strong>{selectedNeType} 的告警定义</Text>
            </Space>
          ) : (
            // 列表态左侧空 — 占位让 space-between 把右侧按钮推到最右
            <span />
          )}
          <Space wrap>
            {inDetail ? (
              // 详情态右侧:搜索 + 严重级别筛选 + 新增。重载 XML / 刷新缓存 都是全局动作,留在一级。
              // 2026-05-29 用户决策:搜索框置于"严重级别"前(主操作前置,与 kpi-library
              // 详情态 toolbar 范式一致)。
              <>
                <Input.Search
                  placeholder={t('product.alarm.searchPh')}
                  allowClear
                  onSearch={(v) =>
                    setDetailFilter((f) => ({ ...f, keyword: v || undefined, page: 1 }))
                  }
                  style={{ width: 220 }}
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
                  新增定义
                </Button>
              </>
            ) : (
              <>
                {/* 2026-05-29 与 parammodel 对齐拆两按钮:导入 = 加法 UPSERT;重载 = destructive 删孤儿 */}
                <Popconfirm
                  title={t('product.kpi.importTitle')}
                  description={
                    <div style={{ maxWidth: 320 }}>
                      {t('product.alarm.importDesc')}
                      <br />· 新增告警 → 插入
                      <br />· 已存在告警 → 更新(UI 中的手工编辑会被 XML 覆盖)
                      
                    </div>
                  }
                  okText={t('product.kpi.importOk')}
                  cancelText={t('common.cancel')}
                  placement="bottomRight"
                  onConfirm={() => {
                    importMut
                      .mutateAsync()
                      .then((r) => message.success(`已导入:${r.reloaded}`))
                      .catch((error) => message.error((error as Error).message));
                  }}
                >
                  <Button icon={<CloudUploadOutlined />} loading={importMut.isPending}>
                    导入 XML
                  </Button>
                </Popconfirm>
                <Popconfirm
                  title={t('product.paramModel.reloadTitle')}
                  description={
                    <div style={{ maxWidth: 360 }}>
                      {t('product.alarm.reloadDesc')}
                      <br />· 当前 XML 中的告警 → UPSERT (覆盖 UI 编辑)
                      
                      <br />操作不可撤销!
                    </div>
                  }
                  okText={t('product.products.reloadOk')}
                  cancelText={t('common.cancel')}
                  okButtonProps={{ danger: true }}
                  placement="bottomRight"
                  onConfirm={() => {
                    reloadMut
                      .mutateAsync()
                      .then((r) =>
                        message.success(
                          `已重载:${r.reloaded}${r.orphans_deleted > 0 ? ` (清理 ${r.orphans_deleted} 个孤儿告警)` : ''}`,
                        ),
                      )
                      .catch((error) => message.error((error as Error).message));
                  }}
                >
                  <Button icon={<CloudDownloadOutlined />} loading={reloadMut.isPending} danger>
                    重载 XML
                  </Button>
                </Popconfirm>
                <Button
                  icon={<ReloadOutlined />}
                  loading={cacheMut.isPending}
                  onClick={() =>
                    cacheMut
                      .mutateAsync()
                      .then(() => message.success(t('common.cacheRefreshed')))
                      .catch((e) => message.error((e as Error).message))
                  }
                >
                  刷新缓存
                </Button>
              </>
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
            columns={detailColumns}
            dataSource={detailItems}
            size="small"
            pagination={{
              current: detailData?.page || 1,
              pageSize: detailData?.pageSize || 20,
              total: detailData?.total || 0,
              showSizeChanger: true,
              onChange: (page, pageSize) =>
                setDetailFilter((f) => ({ ...f, page, pageSize })),
            }}
          />
        ) : (
          <Table<AlarmNeTypeStat>
            rowKey={(r) => `${r.neType}__${r.loadedFrom}`}
            loading={isNeTypesLoading}
            columns={neTypesColumns}
            dataSource={neTypesItems}
            size="small"
            pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 个网元类型` }}
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
      {/* <UnknownStatsModal open={statsOpen} onClose={() => setStatsOpen(false)} /> */}
    </div>
  );
}
