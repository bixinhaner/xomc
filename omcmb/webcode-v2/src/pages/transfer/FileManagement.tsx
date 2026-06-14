import { useNavigate } from 'react-router-dom'
import { ChevronRight, Package, FileStack, KeySquare } from 'lucide-react'

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
  formatBytes,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useSoftwareVersions } from '@core/hooks/api/useSoftware'
import { useConfigSnapshots } from '@core/hooks/api/useConfigSnapshot'
import { useDeviceLicenses } from '@core/hooks/api/useDeviceLicense'

// ============================================================
// 文件管理 — 对照 v1 transfer/file-management (多 Tab 文件库聚合)
// v2 聚合三大可下发文件库:固件版本 / 配置快照 / 设备 License
// 展示各库实时计数 + 最近文件,并提供进入完整模块页的入口
// ============================================================

const PAGE = { page: 1, pageSize: 5 } as const

export default function FileManagement() {
  const navigate = useNavigate()

  const firmwareQuery = useSoftwareVersions(PAGE)
  const snapshotQuery = useConfigSnapshots(PAGE)
  const licenseQuery = useDeviceLicenses(PAGE)

  const firmwareTotal = firmwareQuery.data?.total ?? 0
  const snapshotTotal = snapshotQuery.data?.total ?? 0
  const licenseTotal = licenseQuery.data?.total ?? 0

  const isFetching =
    firmwareQuery.isFetching || snapshotQuery.isFetching || licenseQuery.isFetching

  return (
    <PageShell
      title="文件管理"
      description="统一文件传输可下发文件库:固件版本 / 配置快照 / 设备 License"
      isFetching={isFetching}
    >
      {/* 库概览卡片 */}
      <div className="mb-5 grid grid-cols-1 gap-3 md:grid-cols-3">
        <LibCard
          icon={<Package className="size-5" />}
          label="固件版本库"
          total={firmwareTotal}
          unit="个版本"
          onClick={() => navigate('/software')}
        />
        <LibCard
          icon={<FileStack className="size-5" />}
          label="配置快照库"
          total={snapshotTotal}
          unit="份快照"
          onClick={() => navigate('/backup')}
        />
        <LibCard
          icon={<KeySquare className="size-5" />}
          label="设备 License 库"
          total={licenseTotal}
          unit="份证书"
          onClick={() => navigate('/license')}
        />
      </div>

      <div className="flex flex-col gap-5">
        {/* 固件版本 */}
        <Section title="最近固件版本" onMore={() => navigate('/software')}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>版本号</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>设备型号</TableHead>
                <TableHead>文件大小</TableHead>
                <TableHead>发布时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {firmwareQuery.isLoading ? (
                <LoadingRow colSpan={5} />
              ) : firmwareQuery.isError ? (
                <ErrorRow colSpan={5} error={firmwareQuery.error} />
              ) : (firmwareQuery.data?.items ?? []).length === 0 ? (
                <EmptyRow colSpan={5}>暂无固件版本</EmptyRow>
              ) : (
                (firmwareQuery.data?.items ?? []).map((v) => (
                  <TableRow key={v.id}>
                    <TableCell>
                      <span className="font-mono text-xs">{v.versionCode}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm">{v.versionName || '—'}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">{v.deviceType || '—'}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs tabular-nums text-muted-foreground">
                        {formatBytes(v.fileSize)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">
                        {formatTime(v.releaseDate)}
                      </span>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </Section>

        {/* 配置快照 */}
        <Section title="最近配置快照" onMore={() => navigate('/backup')}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>设备 SN</TableHead>
                <TableHead>基站名</TableHead>
                <TableHead>文件名</TableHead>
                <TableHead>文件大小</TableHead>
                <TableHead>更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {snapshotQuery.isLoading ? (
                <LoadingRow colSpan={5} />
              ) : snapshotQuery.isError ? (
                <ErrorRow colSpan={5} error={snapshotQuery.error} />
              ) : (snapshotQuery.data?.items ?? []).length === 0 ? (
                <EmptyRow colSpan={5}>暂无配置快照</EmptyRow>
              ) : (
                (snapshotQuery.data?.items ?? []).map((s) => (
                  <TableRow key={`${s.serialNumber}-${s.fileName}`}>
                    <TableCell>
                      <span className="font-mono text-xs">{s.serialNumber}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">{s.enbName || '—'}</span>
                    </TableCell>
                    <TableCell>
                      <span className="max-w-[220px] truncate text-xs" title={s.fileName}>
                        {s.fileName}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs tabular-nums text-muted-foreground">
                        {formatBytes(s.fileSize)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">
                        {formatTime(s.updateTime)}
                      </span>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </Section>

        {/* 设备 License */}
        <Section title="最近设备 License" onMore={() => navigate('/license')}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>设备 SN</TableHead>
                <TableHead>基站名</TableHead>
                <TableHead>文件名</TableHead>
                <TableHead>文件大小</TableHead>
                <TableHead>更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {licenseQuery.isLoading ? (
                <LoadingRow colSpan={5} />
              ) : licenseQuery.isError ? (
                <ErrorRow colSpan={5} error={licenseQuery.error} />
              ) : (licenseQuery.data?.items ?? []).length === 0 ? (
                <EmptyRow colSpan={5}>暂无设备 License</EmptyRow>
              ) : (
                (licenseQuery.data?.items ?? []).map((l) => (
                  <TableRow key={`${l.serialNumber}-${l.fileName}`}>
                    <TableCell>
                      <span className="font-mono text-xs">{l.serialNumber}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">{l.enbName || '—'}</span>
                    </TableCell>
                    <TableCell>
                      <span className="max-w-[220px] truncate text-xs" title={l.fileName}>
                        {l.fileName}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs tabular-nums text-muted-foreground">
                        {formatBytes(l.fileSize)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">
                        {formatTime(l.updateTime)}
                      </span>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </Section>
      </div>
    </PageShell>
  )
}

function LibCard({
  icon,
  label,
  total,
  unit,
  onClick,
}: {
  icon: React.ReactNode
  label: string
  total: number
  unit: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'group flex items-center gap-4 rounded-lg border bg-card px-4 py-3 text-left transition-colors',
        'hover:border-primary/40 hover:bg-primary/5'
      )}
    >
      <div className="flex size-10 items-center justify-center rounded-md bg-primary/10 text-primary">
        {icon}
      </div>
      <div className="flex-1">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="text-2xl font-semibold tabular-nums">{total}</div>
      </div>
      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        {unit}
        <ChevronRight className="size-4 transition-transform group-hover:translate-x-0.5" />
      </div>
    </button>
  )
}

function Section({
  title,
  onMore,
  children,
}: {
  title: string
  onMore: () => void
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between p-4 pb-2">
        <CardTitle className="text-base font-medium">{title}</CardTitle>
        <button
          type="button"
          className="flex items-center gap-0.5 text-xs text-primary hover:underline"
          onClick={onMore}
        >
          查看全部 <ChevronRight className="size-3.5" />
        </button>
      </CardHeader>
      <CardContent className="p-0">{children}</CardContent>
    </Card>
  )
}
