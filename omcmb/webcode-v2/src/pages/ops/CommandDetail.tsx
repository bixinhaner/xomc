import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useOpsCommandRecords } from '@core/hooks/api/useOpsTools'
import type { OpsCommandRecord } from '@core/mock/data/opsTools'

// ============================================================
// 命令执行详情 — /ops/commands/:id
// 无单条 by-id 接口，故拉一页记录后按 id 命中（与列表同一真实数据源）
// ============================================================

function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`
  return `${ms} ms`
}

function Field({
  label,
  children,
  full,
}: {
  label: string
  children: React.ReactNode
  full?: boolean
}) {
  return (
    <div className={cn(full && 'sm:col-span-2')}>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-1">{children}</dd>
    </div>
  )
}

export default function CommandDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 一次拉较大页，命中目标 id。列表查询是真实接口（mock/real 自动切换）。
  const { data, isLoading, isError, error, isFetching } = useOpsCommandRecords({
    page: 1,
    pageSize: 500,
  })

  const record: OpsCommandRecord | undefined = useMemo(
    () => (data?.items ?? []).find((r) => r.id === id),
    [data, id]
  )

  const outputLines = useMemo(
    () => (record?.output ? record.output.split('\n') : []),
    [record]
  )

  return (
    <PageShell
      title="命令执行详情"
      description={record ? record.commandText : id ? `记录 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <Button variant="ghost" size="sm" onClick={() => navigate('/ops/commands')}>
          <ArrowLeft className="size-4" /> 返回命令记录
        </Button>
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
      ) : !record ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到命令记录 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/ops/commands')}>
            返回命令记录
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-5">
            <div className="mb-4 flex flex-wrap items-center gap-2">
              <Badge variant={record.success ? 'success' : 'destructive'}>
                {record.success ? '执行成功' : '执行失败'}
              </Badge>
              <span className="rounded bg-muted/60 px-2 py-0.5 font-mono text-xs">
                {record.commandText}
              </span>
            </div>
            <dl className="grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
              <Field label="设备名称">{record.deviceName || '—'}</Field>
              <Field label="设备 SN">
                <span className="font-mono text-xs">{record.deviceSn}</span>
              </Field>
              <Field label="操作人">{record.operator}</Field>
              <Field label="执行时间">{formatTime(record.executeTime)}</Field>
              <Field label="耗时">
                <span className="font-mono text-xs">{formatDuration(record.duration)}</span>
              </Field>
              {!record.success && record.errorMessage ? (
                <Field label="错误信息" full>
                  <span className="text-destructive">{record.errorMessage}</span>
                </Field>
              ) : null}
            </dl>
          </Card>

          <Card className="p-5">
            <div className="mb-3 text-sm font-medium">输出结果</div>
            <div className="max-h-96 overflow-y-auto rounded-md bg-zinc-950 p-3 font-mono text-xs leading-relaxed">
              {record.success ? (
                outputLines.length > 0 ? (
                  outputLines.map((line, i) => (
                    <div key={i} className="text-emerald-400">
                      {line || ' '}
                    </div>
                  ))
                ) : (
                  <span className="text-zinc-500">无输出</span>
                )
              ) : (
                <>
                  {record.errorMessage ? (
                    <div className="mb-2 text-destructive">{record.errorMessage}</div>
                  ) : null}
                  {outputLines.length > 0 ? (
                    outputLines.map((line, i) => (
                      <div key={i} className="text-zinc-300">
                        {line || ' '}
                      </div>
                    ))
                  ) : (
                    <span className="text-zinc-500">执行失败，无输出</span>
                  )}
                </>
              )}
            </div>
          </Card>
        </div>
      )}
    </PageShell>
  )
}
