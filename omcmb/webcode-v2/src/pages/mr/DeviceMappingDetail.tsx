import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useMRMappings, useToggleMRMapping } from '@core/hooks/api/useMR'
import type { MRDeviceMapping } from '@core/mock/data/mr'

// ============================================================
// 测量报告 → 设备小区映射详情（带参 /mr/device-mapping/:id）
// 真实数据：以 useMRMappings 拉一页（pageSize 大）后按 id 命中目标映射。
// 找到后展示设备/小区/采样配置 + 采集统计，并可就地启停。
// ============================================================

type FieldVal = string | number | null | undefined

interface Field {
  label: string
  value: FieldVal
  mono?: boolean
}

function InfoGrid({ fields }: { fields: Field[] }) {
  return (
    <div className="grid grid-cols-1 gap-x-8 gap-y-0 sm:grid-cols-2 lg:grid-cols-3">
      {fields.map((f) => (
        <div
          key={f.label}
          className="flex items-center justify-between gap-4 border-b py-2.5 text-sm"
        >
          <span className="shrink-0 text-muted-foreground">{f.label}</span>
          <span
            className={cn('truncate text-right', f.mono && 'font-mono text-xs')}
            title={f.value != null ? String(f.value) : undefined}
          >
            {f.value === '' || f.value == null ? '—' : f.value}
          </span>
        </div>
      ))}
    </div>
  )
}

export default function DeviceMappingDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 映射数无 by-id 接口；拉较大一页后客户端命中（数据量级可控）。
  const { data, isLoading, isError, error, isFetching } = useMRMappings({
    page: 1,
    pageSize: 500,
  })
  const toggleMutation = useToggleMRMapping()

  const mapping: MRDeviceMapping | undefined = useMemo(
    () => (data?.items ?? []).find((m) => m.id === id),
    [data, id],
  )

  const fields: Field[] = mapping
    ? [
        { label: '映射 ID', value: mapping.id, mono: true },
        { label: '设备 SN', value: mapping.deviceSn, mono: true },
        { label: '设备名称', value: mapping.deviceName },
        { label: '小区 ID', value: mapping.cellId, mono: true },
        { label: '小区名称', value: mapping.cellName },
        { label: '采样间隔', value: `${mapping.samplingInterval} 分钟` },
        { label: '采集记录数', value: mapping.totalRecords.toLocaleString() },
        { label: '最后采集时间', value: formatTime(mapping.lastCollectTime) },
      ]
    : []

  return (
    <PageShell
      title="设备小区映射详情"
      description={id ? `映射 ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/mr/device-mapping')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          {mapping && (
            <>
              {mapping.enabled ? (
                <Badge variant="success">已启用</Badge>
              ) : (
                <Badge variant="muted">已禁用</Badge>
              )}
              <Button
                variant="outline"
                size="sm"
                className="ml-auto"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate({ id: mapping.id, enabled: !mapping.enabled })}
              >
                {mapping.enabled ? '禁用映射' : '启用映射'}
              </Button>
            </>
          )}
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
      ) : !mapping ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到映射 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/mr/device-mapping')}>
            返回映射列表
          </Button>
        </Card>
      ) : (
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">映射详情</CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-2">
            <InfoGrid fields={fields} />
          </CardContent>
        </Card>
      )}
    </PageShell>
  )
}
