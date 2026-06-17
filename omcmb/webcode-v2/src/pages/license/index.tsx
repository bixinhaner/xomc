/**
 * LicensePage (webcode-v2) — 许可证模块业务页。
 *
 * 把 v2 占位页适配到 v1（webcode/src/pages/SystemLicense）的业务深度：
 *   Tab 1 系统许可证（singleton system_license，useSystemLicense）：
 *     - Basic Info：License ID / 类型 / 到期(剩余天数) / 签发方 / 被授权方 / 签名状态(+keyId) / 签发时间 / 上传时间
 *     - 设备容量卡片（devicesSupport，合计容量）
 *     - 特性列表（feature_list 三级嵌套）
 *     - 最近历史（useSystemLicenseHistory，前 5 条）
 *   Tab 2 设备许可证文件库（device_licenses，useDeviceLicenses）：
 *     - 筛选：SN / 基站名 / 产品型号
 *     - 表格：SN / 基站名 / 产品型号 / 文件名 / 大小 / 来源 / 更新人 / 更新时间 / 下载
 *     - 分页 + loading/空/错误三态
 *
 * UI 风格：shadcn/ui + Tailwind（浅色），无 antd。数据全走 @core React Query hooks。
 * 业务层在 frontend-core，本文件只消费已有 hook / 类型，不新增契约。
 *
 * v1 未完全覆盖部分见 notes：UpdateModal（上传 license 文件）仅占位入口，
 * 实际上传走主皮肤 webcode（mutation 需 FileReader + 文件选择交互，超出本轮范围）。
 */
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Download,
  FileKey,
  History,
  Loader2,
  RefreshCcw,
  Search,
  ShieldCheck,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import { useSystemLicense, useSystemLicenseHistory } from '@core/hooks/api/useSystemLicense'
import { useDeviceLicenses } from '@core/hooks/api/useDeviceLicense'
import {
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
  type SystemLicense,
  type SystemLicenseSignatureStatus,
} from '@core/services/api/systemLicenseApi'
import {
  deviceLicenseApi,
  type DeviceLicense,
  type LicenseListParams,
} from '@core/services/api/deviceLicenseApi'

import { formatSystemTime } from '@core/utils/systemTime'
import { FeatureList } from './FeatureList'

// ───────────────────────────────────────────────────────── helpers

function formatTime(iso?: string | null) {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。
  return formatSystemTime(iso, { format: 'YYYY-MM-DD HH:mm', placeholder: '—' })
}

function formatBytes(bytes?: number | null) {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v > 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

function daysRemaining(expiryISO: string | null): number | null {
  if (!expiryISO) return null
  return Math.floor((new Date(expiryISO).getTime() - Date.now()) / 86_400_000)
}

const SIGNATURE_VARIANT: Record<SystemLicenseSignatureStatus, 'success' | 'destructive' | 'warning'> = {
  verified: 'success',
  invalid: 'destructive',
  unverified: 'warning',
}

const SIGNATURE_LABEL: Record<SystemLicenseSignatureStatus, string> = {
  verified: '已验证',
  invalid: '签名无效',
  unverified: '未验证',
}

type TabKey = 'system' | 'device'

// ───────────────────────────────────────────────────────── page

export function LicensePage() {
  const [tab, setTab] = useState<TabKey>('system')

  const system = useSystemLicense()
  const isFetching = system.isFetching

  return (
    <div className="mx-auto max-w-[1400px] px-6 py-8">
      <div className="mb-6 flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">许可证管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            系统级单一许可证（容量 / 特性 / 历史）与设备许可证文件库
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {isFetching && tab === 'system' && (
            <Loader2 className="size-3.5 animate-spin" />
          )}
        </div>
      </div>

      {/* Tab 切换 */}
      <div className="mb-4 inline-flex rounded-lg border bg-card p-1">
        <TabButton active={tab === 'system'} onClick={() => setTab('system')}>
          <ShieldCheck className="size-4" /> 系统许可证
        </TabButton>
        <TabButton active={tab === 'device'} onClick={() => setTab('device')}>
          <FileKey className="size-4" /> 设备许可证库
        </TabButton>
      </div>

      {tab === 'system' ? (
        <SystemLicenseTab query={system} />
      ) : (
        <DeviceLicenseTab />
      )}
    </div>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-md px-4 py-1.5 text-sm font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:text-foreground',
      )}
    >
      {children}
    </button>
  )
}

// ───────────────────────────────────────────────────────── Tab 1: System License

function SystemLicenseTab({
  query,
}: {
  query: ReturnType<typeof useSystemLicense>
}) {
  const { data: lic, isLoading, error, refetch } = query
  const errorCode = error ? extractLicenseErrorCode(error) : 0
  const isNotConfigured = errorCode === SystemLicenseErrorCodes.NotConfigured

  if (isLoading) {
    return (
      <div className="rounded-lg border bg-card p-10 text-center">
        <Loader2 className="mx-auto size-6 animate-spin text-muted-foreground" />
        <div className="mt-2 text-sm text-muted-foreground">加载中…</div>
      </div>
    )
  }

  // 真错误（非"表空"）—— 先于空态判断，避免真错误被误判成"未配置"
  if (error && !isNotConfigured) {
    return (
      <div className="rounded-lg border border-destructive bg-card p-6">
        <div className="text-destructive">加载失败</div>
        <div className="mt-1 text-xs text-muted-foreground">
          {(error as Error)?.message}
        </div>
        <Button
          variant="outline"
          size="sm"
          className="mt-3"
          onClick={() => refetch()}
        >
          <RefreshCcw /> 重试
        </Button>
      </div>
    )
  }

  // 空态（system_license 表无 current 行，biz_code 12113）
  if (isNotConfigured || !lic) {
    return (
      <div className="rounded-lg border bg-card p-10 text-center">
        <ShieldCheck className="mx-auto size-8 text-muted-foreground" />
        <div className="mt-3 text-lg font-medium">系统未配置许可证</div>
        <div className="mx-auto mt-2 max-w-md text-sm text-muted-foreground">
          请在主皮肤 webcode 的 /license 页面上传 OEM 签发的 license 文件。
        </div>
        <Button
          variant="outline"
          size="sm"
          className="mt-4"
          onClick={() => refetch()}
        >
          <RefreshCcw /> 重试
        </Button>
      </div>
    )
  }

  return <SystemLicenseDetail lic={lic} onRefresh={() => refetch()} />
}

function SystemLicenseDetail({
  lic,
  onRefresh,
}: {
  lic: SystemLicense
  onRefresh: () => void
}) {
  const remain = daysRemaining(lic.expiryDate)
  const expiryLabel = (() => {
    if (!lic.expiryDate) return '永久'
    if (remain !== null && remain < 0) return `${formatTime(lic.expiryDate)} · 已过期`
    return `${formatTime(lic.expiryDate)} · 剩余 ${remain ?? 0} 天`
  })()

  const devices = useMemo(
    () =>
      Object.entries(lic.devicesSupport ?? {}).sort(([a], [b]) =>
        a.localeCompare(b),
      ),
    [lic.devicesSupport],
  )
  const totalCapacity = useMemo(
    () => Object.values(lic.devicesSupport ?? {}).reduce((a, b) => a + b, 0),
    [lic.devicesSupport],
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-end">
        <Button variant="outline" size="sm" onClick={onRefresh}>
          <RefreshCcw /> 刷新
        </Button>
      </div>

      {/* Basic Info */}
      <div className="rounded-lg border bg-card p-4">
        <div className="mb-3 text-sm font-medium">基本信息</div>
        <div className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
          <InfoItem label="License ID">
            <span className="font-mono text-sm">{lic.licenseId}</span>
          </InfoItem>
          <InfoItem label="类型">
            <Badge variant="outline">{lic.licenseType}</Badge>
          </InfoItem>
          <InfoItem label="到期">{expiryLabel}</InfoItem>
          <InfoItem label="签发方">{lic.issuer ?? '—'}</InfoItem>
          <InfoItem label="被授权方">{lic.licensee ?? '—'}</InfoItem>
          <InfoItem label="签名状态">
            <span className="inline-flex items-center gap-2">
              <Badge variant={SIGNATURE_VARIANT[lic.signatureStatus]}>
                {SIGNATURE_LABEL[lic.signatureStatus]}
              </Badge>
              {lic.signatureKeyId ? (
                <span className="text-xs text-muted-foreground">
                  key: {lic.signatureKeyId.slice(0, 12)}…
                </span>
              ) : null}
            </span>
          </InfoItem>
          <InfoItem label="签发时间">{formatTime(lic.issuedAt)}</InfoItem>
          <InfoItem label="上传时间">{formatTime(lic.uploadedAt)}</InfoItem>
          <InfoItem label="记录 PK">
            <span className="font-mono text-xs text-muted-foreground">
              {lic.id}
            </span>
          </InfoItem>
        </div>
      </div>

      {/* Devices Support */}
      <div className="rounded-lg border bg-card p-4">
        <div className="mb-3 text-sm font-medium">
          设备容量
          <span className="ml-2 text-xs font-normal text-muted-foreground">
            合计 {totalCapacity}
          </span>
        </div>
        {devices.length === 0 ? (
          <div className="py-4 text-sm text-muted-foreground">暂无容量配置</div>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
            {devices.map(([type, cap]) => (
              <div
                key={type}
                className="rounded-lg border bg-background px-3 py-2.5"
              >
                <div className="truncate text-xs text-muted-foreground" title={type}>
                  {type}
                </div>
                <div className="mt-0.5 text-xl font-semibold tabular-nums text-primary">
                  {cap}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Feature List */}
      <div className="rounded-lg border bg-card p-4">
        <div className="mb-1 text-sm font-medium">特性列表</div>
        <FeatureList featureList={lic.featureList} />
      </div>

      {/* Recent History */}
      <SystemLicenseHistoryPanel />
    </div>
  )
}

function InfoItem({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm">{children}</div>
    </div>
  )
}

function SystemLicenseHistoryPanel() {
  const navigate = useNavigate()
  const { data, isLoading, error } = useSystemLicenseHistory({
    page: 1,
    pageSize: 5,
  })

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="text-sm font-medium">
          最近历史
          {data ? (
            <span className="ml-2 text-xs font-normal text-muted-foreground">
              共 {data.total} 条
            </span>
          ) : null}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/license/history')}
        >
          <History /> 查看完整历史
        </Button>
      </div>
      <div className="overflow-hidden rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>License ID</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>签名</TableHead>
              <TableHead>上传时间</TableHead>
              <TableHead>替换时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={5} className="h-24 text-center">
                  <Loader2 className="mx-auto size-5 animate-spin text-muted-foreground" />
                </TableCell>
              </TableRow>
            ) : error ? (
              <TableRow>
                <TableCell colSpan={5} className="h-24 text-center text-destructive">
                  加载失败：{error instanceof Error ? error.message : '未知错误'}
                </TableCell>
              </TableRow>
            ) : !data || data.items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                  暂无历史记录
                </TableCell>
              </TableRow>
            ) : (
              data.items.map((h) => (
                <TableRow key={h.id}>
                  <TableCell className="font-mono text-xs">{h.licenseId}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{h.licenseType}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={SIGNATURE_VARIANT[h.signatureStatus]}>
                      {SIGNATURE_LABEL[h.signatureStatus]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(h.uploadedAt)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(h.replacedAt)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

// ───────────────────────────────────────────────────────── Tab 2: Device License library

const SOURCE_LABEL: Record<string, string> = {
  manual_upload: '手动导入',
}

function DeviceLicenseTab() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [searchSn, setSearchSn] = useState('')
  const [searchEnb, setSearchEnb] = useState('')
  const [productType, setProductType] = useState('')

  const params = useMemo<LicenseListParams>(
    () => ({
      page,
      pageSize,
      ...(searchSn.trim() ? { serialNumber: searchSn.trim() } : {}),
      ...(searchEnb.trim() ? { enbName: searchEnb.trim() } : {}),
      ...(productType.trim() ? { productType: productType.trim() } : {}),
    }),
    [page, pageSize, searchSn, searchEnb, productType],
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceLicenses(params)

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const handleDownload = (lic: DeviceLicense) => {
    void deviceLicenseApi.download(lic.serialNumber)
  }

  const COL_COUNT = 9

  return (
    <div className="space-y-3">
      {/* Toolbar / filters */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-56 pl-9"
            placeholder="序列号 SN"
            value={searchSn}
            onChange={(e) => {
              setSearchSn(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Input
          className="w-48"
          placeholder="基站名称"
          value={searchEnb}
          onChange={(e) => {
            setSearchEnb(e.target.value)
            setPage(1)
          }}
        />
        <Input
          className="w-48"
          placeholder="产品型号"
          value={productType}
          onChange={(e) => {
            setProductType(e.target.value)
            setPage(1)
          }}
        />
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          {isFetching && <Loader2 className="size-3.5 animate-spin" />}
          <span>共 {total} 份</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>序列号 SN</TableHead>
              <TableHead>基站名称</TableHead>
              <TableHead>产品型号</TableHead>
              <TableHead>文件名</TableHead>
              <TableHead>大小</TableHead>
              <TableHead>来源</TableHead>
              <TableHead>更新人</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={COL_COUNT} className="h-32 text-center">
                  <Loader2 className="mx-auto size-5 animate-spin text-muted-foreground" />
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={COL_COUNT} className="h-32 text-center text-destructive">
                  加载失败：{error instanceof Error ? error.message : '未知错误'}
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={COL_COUNT} className="h-32 text-center text-muted-foreground">
                  没有匹配的设备许可证
                </TableCell>
              </TableRow>
            ) : (
              rows.map((lic) => (
                <TableRow key={lic.serialNumber}>
                  <TableCell className="font-mono text-xs">
                    {lic.serialNumber}
                  </TableCell>
                  <TableCell>{lic.enbName ?? '—'}</TableCell>
                  <TableCell>
                    {lic.productType ? (
                      <Badge variant="muted">{lic.productType}</Badge>
                    ) : (
                      '—'
                    )}
                  </TableCell>
                  <TableCell className="text-xs">{lic.fileName}</TableCell>
                  <TableCell className="tabular-nums">
                    {formatBytes(lic.fileSize)}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {SOURCE_LABEL[lic.source] ?? lic.source}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {lic.updateBy ?? '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(lic.updateTime)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDownload(lic)}
                    >
                      <Download /> 下载
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          第 {page} / {totalPages} 页 · 每页 {pageSize} 条
        </span>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            上一页
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            下一页
          </Button>
        </div>
      </div>
    </div>
  )
}
