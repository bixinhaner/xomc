import { RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell } from '@/components/layout/PageShell'

import { useSystemLicense } from '@core/hooks/api/useSystemLicense'
import {
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
} from '@core/services/api/systemLicenseApi'

// F06 重构 Step 5：webcode-v2 老多 license 列表页 → 切到 singleton system_license。
// v2 是多皮肤候选包，UI 风格保持 lucide + tailwind，不引入 antd。
export function LicensePage() {
  const { data: lic, isLoading, error, isFetching, refetch } = useSystemLicense()
  const errorCode = error ? extractLicenseErrorCode(error) : 0
  const isNotConfigured = errorCode === SystemLicenseErrorCodes.NotConfigured

  const totalCapacity = lic ? Object.values(lic.devicesSupport ?? {}).reduce((a, b) => a + b, 0) : 0
  const daysLeft = lic?.expiryDate
    ? Math.floor((new Date(lic.expiryDate).getTime() - Date.now()) / 86_400_000)
    : null

  return (
    <PageShell
      title="License"
      description="系统级单一 license（F06 重构后）· 详情、容量、特性"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw /> 刷新
        </Button>
      }
    >
      {isLoading ? (
        <div className="rounded-lg border bg-card p-6 text-center text-muted-foreground">
          加载中…
        </div>
      ) : isNotConfigured || !lic ? (
        <div className="rounded-lg border bg-card p-8 text-center">
          <div className="text-lg font-medium">系统未配置许可证</div>
          <div className="mt-2 text-sm text-muted-foreground">
            请在主皮肤 webcode 的 /license 页面上传 OEM 签发的 license 文件。
          </div>
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive bg-card p-6">
          <div className="text-destructive">加载失败</div>
          <div className="mt-1 text-xs text-muted-foreground">{(error as Error)?.message}</div>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <InfoCard label="License ID" value={lic.licenseId} mono />
            <InfoCard label="类型" value={<Badge variant="outline">{lic.licenseType}</Badge>} />
            <InfoCard
              label="到期"
              value={
                lic.expiryDate
                  ? `${new Date(lic.expiryDate).toLocaleDateString()}${
                      daysLeft !== null && daysLeft < 0
                        ? ' · 已过期'
                        : daysLeft !== null
                          ? ` · 剩余 ${daysLeft} 天`
                          : ''
                    }`
                  : '永久'
              }
            />
            <InfoCard
              label="签名"
              value={
                <Badge variant={lic.signatureStatus === 'verified' ? 'success' : 'muted'}>
                  {lic.signatureStatus}
                </Badge>
              }
            />
          </div>

          <div className="rounded-lg border bg-card p-4">
            <div className="mb-3 text-sm font-medium">设备容量（合计 {totalCapacity}）</div>
            <div className="flex flex-wrap gap-3">
              {Object.entries(lic.devicesSupport ?? {})
                .sort(([a], [b]) => a.localeCompare(b))
                .map(([type, cap]) => (
                  <div
                    key={type}
                    className="rounded border bg-background px-3 py-2 text-sm tabular-nums"
                  >
                    <div className="text-xs text-muted-foreground">{type}</div>
                    <div className="font-medium">{cap}</div>
                  </div>
                ))}
            </div>
          </div>
        </div>
      )}
    </PageShell>
  )
}

interface InfoCardProps {
  label: string
  value: React.ReactNode
  mono?: boolean
}

function InfoCard({ label, value, mono }: InfoCardProps) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`mt-1 ${mono ? 'font-mono text-sm' : 'text-sm'}`}>{value}</div>
    </div>
  )
}
