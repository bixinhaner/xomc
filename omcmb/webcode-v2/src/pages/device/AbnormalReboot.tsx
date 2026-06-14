import { useMemo, useState } from 'react'
import { BarChart3, RefreshCcw, Search, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card } from '@/components/ui/card'
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
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useRebootRecordList,
  useRebootRecordStatByDevice,
} from '@core/hooks/api/useRebootRecord'
import type {
  RebootRecord,
  RebootRecordListParams,
  RebootRecordStatParams,
  RebootType,
} from '@core/services/api/rebootRecordApi'

// ============================================================
// 启动记录 — 普通重启(event_logs) ∪ 异常重启(station_fault_logs) 合并视图
// 对照 v1 webcode/src/pages/device/AbnormalReboot：可按重启类型/设备类型筛选，
// 按设备聚合统计（总次数/异常次数）。导出(xlsx)未覆盖（v2 无 xlsx 依赖）。
// ============================================================

const PAGE_SIZE = 20

function showRaw(value: string | undefined | null): string {
  return (value ?? '').trim() || '—'
}

function formatRuntime(seconds: number): string {
  if (!seconds || seconds <= 0) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}时`)
  if (minutes > 0 && days === 0) parts.push(`${minutes}分`)
  return parts.length > 0 ? parts.join('') : '< 1分'
}

function StatPanel({ params }: { params: RebootRecordStatParams }) {
  const { data, isLoading, isError, error } = useRebootRecordStatByDevice(
    params,
    true
  )
  const rows = useMemo(
    () =>
      [...(data?.items ?? [])].sort((a, b) => b.totalCount - a.totalCount),
    [data]
  )
  const colCount = 5

  return (
    <TableCard>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>设备编码</TableHead>
            <TableHead>设备名称</TableHead>
            <TableHead className="text-right">总次数</TableHead>
            <TableHead className="text-right">异常次数</TableHead>
            <TableHead>最近时间</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <LoadingRow colSpan={colCount} />
          ) : isError ? (
            <ErrorRow colSpan={colCount} error={error} />
          ) : rows.length === 0 ? (
            <EmptyRow colSpan={colCount}>暂无统计数据</EmptyRow>
          ) : (
            rows.map((r) => (
              <TableRow key={r.deviceSn}>
                <TableCell>
                  <span className="font-mono text-xs">{r.deviceSn}</span>
                </TableCell>
                <TableCell className="text-sm">{r.deviceName || '—'}</TableCell>
                <TableCell className="text-right">
                  <Badge variant="default">{r.totalCount}</Badge>
                </TableCell>
                <TableCell className="text-right">
                  {r.abnormalCount > 0 ? (
                    <Badge variant="destructive">{r.abnormalCount}</Badge>
                  ) : (
                    <span className="text-xs text-muted-foreground">0</span>
                  )}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {formatTime(r.latestAt)}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableCard>
  )
}

export default function AbnormalReboot() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [rebootType, setRebootType] = useState<RebootType>('all')
  const [deviceType, setDeviceType] = useState<string>('all')
  const [showStat, setShowStat] = useState(false)
  const [selected, setSelected] = useState<RebootRecord | null>(null)

  const queryParams = useMemo<RebootRecordListParams>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { deviceSn: keyword.trim() } : {}),
      ...(rebootType !== 'all' ? { rebootType } : {}),
      ...(deviceType !== 'all' ? { deviceType } : {}),
    }),
    [page, keyword, rebootType, deviceType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useRebootRecordList(queryParams)

  const statParams = useMemo<RebootRecordStatParams>(() => {
    const { deviceSn, deviceType: dt, rebootType: rt, startTime, endTime } =
      queryParams
    return { deviceSn, deviceType: dt, rebootType: rt, startTime, endTime }
  }, [queryParams])

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const colCount = 8

  const hasActiveFilter =
    Boolean(keyword.trim()) || rebootType !== 'all' || deviceType !== 'all'

  return (
    <PageShell
      title="启动记录"
      description={`共 ${total} 条重启记录 · 普通重启与异常重启合并视图`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索设备编码 / SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <Select
            value={rebootType}
            onValueChange={(v) => {
              setRebootType(v as RebootType)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="重启类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="normal">正常重启</SelectItem>
              <SelectItem value="abnormal">异常重启</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={deviceType}
            onValueChange={(v) => {
              setDeviceType(v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue placeholder="设备类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部设备</SelectItem>
              <SelectItem value="eNB">eNB</SelectItem>
              <SelectItem value="gNB">gNB</SelectItem>
            </SelectContent>
          </Select>
          {hasActiveFilter && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setKeyword('')
                setRebootType('all')
                setDeviceType('all')
                setPage(1)
              }}
            >
              <X className="size-4" /> 重置
            </Button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button
              variant={showStat ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowStat((s) => !s)}
            >
              <BarChart3 className="size-4" /> {showStat ? '返回列表' : '统计'}
            </Button>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {showStat ? (
        <StatPanel params={statParams} />
      ) : (
        <>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>设备编码</TableHead>
                  <TableHead>设备名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>设备 IP</TableHead>
                  <TableHead>软件版本</TableHead>
                  <TableHead>重启类型</TableHead>
                  <TableHead>运行时长</TableHead>
                  <TableHead>重启时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <LoadingRow colSpan={colCount} />
                ) : isError ? (
                  <ErrorRow colSpan={colCount} error={error} />
                ) : rows.length === 0 ? (
                  <EmptyRow colSpan={colCount}>
                    {hasActiveFilter ? '没有匹配的记录' : '暂无重启记录'}
                  </EmptyRow>
                ) : (
                  rows.map((r) => (
                    <TableRow
                      key={`${r.source}-${r.id}`}
                      className="cursor-pointer"
                      onClick={() => setSelected(r)}
                    >
                      <TableCell>
                        <span className="font-mono text-xs">{r.deviceSn}</span>
                      </TableCell>
                      <TableCell className="text-sm">
                        {r.deviceName || '—'}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {r.deviceType || '—'}
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground">
                          {r.operateIp || '—'}
                        </span>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {r.softwareVersion || '—'}
                      </TableCell>
                      <TableCell>
                        {r.isAbnormal ? (
                          <Badge variant="destructive">异常</Badge>
                        ) : (
                          <Badge variant="default">正常</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatRuntime(r.runtimeBeforeReboot)}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(r.rebootTime)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          <Pagination
            page={page}
            totalPages={totalPages}
            pageSize={PAGE_SIZE}
            onChange={setPage}
          />
        </>
      )}

      {/* 行详情 */}
      {selected && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
          onClick={() => setSelected(null)}
        >
          <Card
            className="w-full max-w-lg p-5"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-base font-medium">重启记录详情</h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelected(null)}
              >
                <X className="size-4" />
              </Button>
            </div>
            <dl className="grid grid-cols-1 gap-x-6 sm:grid-cols-2">
              <DetailRow label="设备编码" value={selected.deviceSn} mono />
              <DetailRow label="设备名称" value={selected.deviceName || '—'} />
              <DetailRow label="设备类型" value={selected.deviceType || '—'} />
              <DetailRow
                label="设备 IP"
                value={selected.operateIp || '—'}
                mono
              />
              <DetailRow
                label="软件版本"
                value={selected.softwareVersion || '—'}
                mono
              />
              <DetailRow
                label="重启类型"
                value={selected.isAbnormal ? '异常重启' : '正常重启'}
              />
              <DetailRow
                label="重启时间"
                value={formatTime(selected.rebootTime)}
              />
              {selected.isAbnormal && (
                <DetailRow
                  label="运行时长"
                  value={formatRuntime(selected.runtimeBeforeReboot)}
                />
              )}
            </dl>
            {selected.isAbnormal && (
              <div className="mt-3 space-y-2 border-t pt-3 text-sm">
                <div>
                  <span className="text-muted-foreground">重启原因</span>
                  <p className="mt-1 break-all">{showRaw(selected.reason)}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">详细原因</span>
                  <p className="mt-1 break-all">
                    {showRaw(selected.detailReason)}
                  </p>
                </div>
              </div>
            )}
          </Card>
        </div>
      )}
    </PageShell>
  )
}

function DetailRow({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-b py-2 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className={mono ? 'font-mono text-xs' : ''}>{value}</span>
    </div>
  )
}
