import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Loader2,
  Power,
  RefreshCcw,
  RefreshCw,
  Search,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useDeviceBySn,
  useRebootDevice,
  useSyncDeviceParams,
} from '@core/hooks/api/useDevices'
import { useDeviceParameters } from '@core/hooks/api/useDeviceParameters'
import { useAppStore } from '@core/store/appStore'
import { useDictionary } from '@core/hooks/api/useSystem'
import { activationStatusLabelOf } from '@core/utils/activationStatus'
import { DEFAULT_ALARM_SEVERITY_LABELS_ZH, formatAlarmSeverityBadgeLabel, getAlarmSeverityBadgeVariant } from '@core/utils/alarmSeverity'
import { DEVICE_SYNC_STATUS_LABELS_ZH, formatDeviceSyncStatus } from '@core/utils/deviceSyncStatus'
import { computeCumulativeOnlineDurationSeconds } from '@core/utils/onlineDuration'
import { rfStatusLabelOf } from '@core/utils/rfStatus'
import type { Device } from '@core/types/device'

// ============================================================
// 设备详情 — 按 SN 查单设备，分 Tab 展示基本信息/状态/小区/参数
// 对照 v1 webcode/src/pages/device/DeviceDetail 的业务深度
// v1 还含 KPI 趋势图 / 参数树编辑 / 快速设置 / 告警 / License Tab，
// 本页聚焦只读详情（基本/状态/小区/参数列表），重操作未覆盖（见 notes）。
// ============================================================

type TabKey = 'basic' | 'status' | 'cell' | 'params'

function openClassicQuickSettings(sn: string) {
  window.location.assign(`/device/detail/${sn}?tab=quickSettings`)
}

function fmtDuration(seconds?: number | null) {
  if (!seconds || seconds <= 0) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}天 ${h}时 ${m}分`
  if (h > 0) return `${h}时 ${m}分`
  return `${m}分`
}

type FieldVal = string | number | null | undefined

interface Field {
  label: string
  value: FieldVal
  mono?: boolean
}

function GroupCard({ title, fields }: { title: string; fields: Field[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {fields.map((field) => (
            <div key={`${title}-${field.label}`} className="rounded-lg border bg-muted/20 px-3 py-2">
              <div className="text-xs text-muted-foreground">{field.label}</div>
              <div className={cn('mt-1 text-sm', field.mono ? 'font-mono' : '')}>
                {field.value === '' || field.value == null ? '—' : String(field.value)}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function ParamsTab({ deviceId }: { deviceId: string }) {
  const [search, setSearch] = useState('')
  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceParameters(deviceId)

  const items = data?.items ?? []
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return items
    return items.filter(
      (p) =>
        p.parameterPath.toLowerCase().includes(q) ||
        (p.parameterValue ?? '').toLowerCase().includes(q)
    )
  }, [items, search])

  const colCount = 4

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索参数路径 / 值"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <span className="text-sm text-muted-foreground">
          共 {data?.total ?? items.length} 项
        </span>
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            {isFetching ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <RefreshCcw className="size-4" />
            )}
            刷新
          </Button>
        </div>
      </div>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>参数路径</TableHead>
              <TableHead>值</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>可写</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={colCount} />
            ) : isError ? (
              <ErrorRow colSpan={colCount} error={error} />
            ) : filtered.length === 0 ? (
              <EmptyRow colSpan={colCount}>
                {search ? '没有匹配的参数' : '暂无参数'}
              </EmptyRow>
            ) : (
              filtered.map((p) => (
                <TableRow key={p.parameterPath}>
                  <TableCell>
                    <span className="font-mono text-xs">{p.parameterPath}</span>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs">
                      {p.parameterValue || '—'}
                    </span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {p.parameterType || '—'}
                  </TableCell>
                  <TableCell>
                    {p.writable ? (
                      <Badge variant="success">可写</Badge>
                    ) : (
                      <Badge variant="muted">只读</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </div>
  )
}

const TABS: { key: TabKey; label: string }[] = [
  { key: 'basic', label: '基本信息' },
  { key: 'status', label: '运行状态' },
  { key: 'cell', label: '小区信息' },
  { key: 'params', label: '参数' },
]

export default function DeviceDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()
  const appLocale = useAppStore((s) => s.locale)
  const [tab, setTab] = useState<TabKey>('basic')

  const { data: device, isLoading, isError, error, isFetching, refetch } =
    useDeviceBySn(sn)
  const { data: opStateDict } = useDictionary('op_state')
  const reboot = useRebootDevice()
  const syncParams = useSyncDeviceParams()
  const rfStatusLabel = (value: string | undefined) =>
    rfStatusLabelOf(value, { on: '射频开', off: '射频关', error: '异常' }) ?? value ?? '—'

  const basicFields = (d: Device): Field[] => [
    { label: 'SN', value: d.sn, mono: true },
    { label: '名称', value: d.deviceName || d.name },
    { label: '制式', value: d.networkType },
    { label: '产品类型', value: d.productClass },
    { label: '产品型号', value: d.deviceModel },
    { label: '厂商', value: d.vendor },
    { label: '软件版本', value: d.softwareVersion, mono: true },
    { label: '固件版本', value: d.firmwareVersion, mono: true },
    { label: 'MAC', value: d.macAddress, mono: true },
    { label: 'IP 地址', value: d.ipAddress, mono: true },
    { label: '分组', value: d.groupName },
    { label: '运营商', value: d.carrier },
    { label: '站点', value: d.site || d.installAddress },
    { label: '安装详细地址', value: d.installAddress },
    { label: '经度', value: d.longitude != null ? d.longitude.toFixed(4) : '' },
    { label: '纬度', value: d.latitude != null ? d.latitude.toFixed(4) : '' },
    { label: '创建时间', value: formatTime(d.createTime) },
  ]

  const statusFields = (d: Device): Field[] => [
    { label: '连接状态', value: d.isOnline ? '在线' : '离线' },
    { label: '生命周期', value: d.lifecycleState },
    // 「激活状态」判定走 frontend-core/utils/activationStatus——与 webcode/webcode-v3 同口径。
    {
      label: '激活状态',
      value:
        activationStatusLabelOf(d.opState, opStateDict?.sysDictionaryDetails, {
          active: '激活',
          inactive: '未激活',
        }, appLocale) || '-',
    },
    { label: '同步状态', value: formatDeviceSyncStatus(d.syncStatus, DEVICE_SYNC_STATUS_LABELS_ZH) || d.syncStatus },
    { label: 'RF 状态', value: rfStatusLabel(d.rfStatus) },
    { label: 'MME/AMF', value: d.mmeStatus || d.amfStatus },
    { label: 'UE 数', value: d.ueCount },
    { label: 'CPE 数', value: d.cpeCount },
    { label: '锁定状态', value: d.lockStatus },
    { label: '首次上线', value: formatTime(d.firstOnlineTime) },
    { label: '最近上线', value: formatTime(d.onlineTime || d.lastOnlineTime) },
    { label: '最近 Inform', value: formatTime(d.lastInformTime) },
    { label: '本次运行时长', value: fmtDuration(d.upTime) },
    {
      label: '累计在线时长',
      value: fmtDuration(computeCumulativeOnlineDurationSeconds({
        isOnline: d.isOnline,
        onlineTime: d.onlineTime || d.lastOnlineTime,
        offlineTime: d.offlineTime,
        fallbackOnlineDuration: d.onlineDuration,
        cumulativeOnlineDuration: d.cumulativeOnlineDuration,
      })),
    },
    { label: '上次离线原因', value: d.lastOfflineReason },
  ]

  const cellFields = (d: Device): Field[] => {
    const common: Field[] = [
      { label: 'PCI', value: d.pci },
      { label: 'TAC', value: d.tac },
      { label: 'Band', value: d.band },
      { label: '下行频点', value: d.dlEarfcn },
      { label: '上行频点', value: d.ulEarfcn },
      { label: '带宽', value: d.bandwidth },
      { label: '发射功率', value: d.txPower },
      { label: '小区状态', value: d.cellStatus },
      { label: 'PLMN', value: d.plmnId },
    ]
    if (d.networkType === 'gNB') {
      return [
        { label: 'gNB ID', value: d.gnbId },
        { label: 'NR Cell ID', value: d.nrCellId },
        ...common,
      ]
    }
    if (d.networkType === 'GSM') {
      return [
        { label: 'LAC', value: d.lac },
        { label: 'ARFCN', value: d.arfcn },
        { label: 'BTS 数', value: d.btsNum },
        { label: '上行频率', value: d.uplinkFrequency },
        { label: '下行频率', value: d.downlinkFrequency },
      ]
    }
    return [
      { label: 'eNB ID', value: d.enbId },
      { label: 'Cell ID', value: d.cellId },
      { label: 'ECI', value: d.eci },
      ...common,
    ]
  }

  return (
    <PageShell
      title={device ? device.deviceName || device.name || sn : '设备详情'}
      description={sn ? `SN ${sn}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/device/list')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          {sn ? (
            <Button variant="outline" size="sm" onClick={() => openClassicQuickSettings(sn)}>
              快速设置
            </Button>
          ) : null}
          {device && (
            <>
              <Badge variant={device.isOnline ? 'success' : 'muted'}>
                <span
                  className={cn(
                    'mr-1 inline-block size-1.5 rounded-full',
                    device.isOnline ? 'bg-emerald-500' : 'bg-muted-foreground/40'
                  )}
                />
                {device.isOnline ? '在线' : '离线'}
              </Badge>
              <Badge variant={getAlarmSeverityBadgeVariant(device.alarmLevel)}>
                告警：{formatAlarmSeverityBadgeLabel(device.alarmLevel, 0, DEFAULT_ALARM_SEVERITY_LABELS_ZH)}
              </Badge>
            </>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
            {device && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={syncParams.isPending}
                  onClick={() => syncParams.mutate({ deviceId: device.id })}
                >
                  {syncParams.isPending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <RefreshCw className="size-4" />
                  )}
                  同步参数
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={reboot.isPending}
                  onClick={() => reboot.mutate(device.id)}
                >
                  {reboot.isPending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Power className="size-4" />
                  )}
                  重启
                </Button>
              </>
            )}
          </div>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !device ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到设备 {sn}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/device/list')}>
            返回设备列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          {/* Tab 切换 */}
          <div className="flex items-center gap-1 border-b">
            {TABS.map((t) => (
              <button
                key={t.key}
                type="button"
                className={cn(
                  '-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors',
                  tab === t.key
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                )}
                onClick={() => setTab(t.key)}
              >
                {t.label}
              </button>
            ))}
          </div>

          {tab === 'basic' && (
            <GroupCard title="基本信息" fields={basicFields(device)} />
          )}
          {tab === 'status' && (
            <GroupCard title="运行状态" fields={statusFields(device)} />
          )}
          {tab === 'cell' && (
            <GroupCard title="小区信息" fields={cellFields(device)} />
          )}
          {tab === 'params' && <ParamsTab deviceId={device.id} />}
        </div>
      )}
    </PageShell>
  )
}
