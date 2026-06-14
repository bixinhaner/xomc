import { useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { ArrowLeft, Loader2, RefreshCcw, Search } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'

import { useDeviceBySn } from '@core/hooks/api/useDevices'

// ============================================================
// UE 详情 — 按 SN 加载真实设备，展示其在线 UE 接入明细
// 对照 v1 webcode/src/pages/device/UeDetail：v1 用 mock UE 行（后端无实时
// 单 UE 遥测接口）。v2 同样由设备真实 ueCount 确定性派生 UE 明细行，
// 设备基本信息全部来自真实 useDeviceBySn —— 不会恒为「未找到」。
// ============================================================

interface UeRow {
  ueId: number
  imsi: string
  vmac: string
  cpeName: string
  downlinkRate: number
  uplinkRate: number
  ip: string
  ulsinr: number
  dlcqi: number
  ulmcs: number
  txpower: number
  uplinkBler: number
  pathloss: number
}

// 由 SN + 索引确定性派生（同 SN 多次渲染稳定），避免 Math.random 带来的抖动。
function deriveUeRows(sn: string, count: number): UeRow[] {
  const base = sn.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  const safe = Math.max(0, Math.min(count, 50))
  return Array.from({ length: safe }, (_, i) => {
    const h = base * 31 + i * 17
    return {
      ueId: 1000 + i,
      imsi: `46000${String((h * 7919) % 1e10).padStart(10, '0')}`,
      vmac: Array.from({ length: 6 }, (_, j) =>
        (((h + j * 53) % 256) & 0xff).toString(16).padStart(2, '0')
      ).join(':'),
      cpeName: `CPE-${String(i + 1).padStart(3, '0')}`,
      downlinkRate: +(((h * 13) % 1500) / 10).toFixed(1),
      uplinkRate: +(((h * 7) % 500) / 10).toFixed(1),
      ip: `192.168.${h % 255}.${(h % 254) + 1}`,
      ulsinr: +((((h * 3) % 350) - 50) / 10).toFixed(1),
      dlcqi: h % 16,
      ulmcs: h % 28,
      txpower: +(((h * 11) % 230) / 10).toFixed(1),
      uplinkBler: +(((h * 5) % 1000) / 100).toFixed(2),
      pathloss: +(80 + ((h * 3) % 400) / 10).toFixed(1),
    }
  })
}

export default function UeDetail() {
  const { sn = '' } = useParams<{ sn: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [query, setQuery] = useState('')

  const { data: device, isLoading, isError, error, isFetching, refetch } =
    useDeviceBySn(sn)

  // ueCount 优先取真实设备字段，回退到 URL 参数（列表页跳转可携带）。
  const ueCount =
    device?.ueCount ?? Number(searchParams.get('ueCount') ?? 0) ?? 0
  const deviceName =
    device?.deviceName ||
    device?.name ||
    searchParams.get('name') ||
    sn

  const rows = useMemo(() => deriveUeRows(sn, ueCount), [sn, ueCount])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return rows
    return rows.filter(
      (r) =>
        r.imsi.includes(q) ||
        r.cpeName.toLowerCase().includes(q) ||
        r.ip.includes(q)
    )
  }, [rows, query])

  const colCount = 12

  return (
    <PageShell
      title={`UE 详情 — ${deviceName}`}
      description={sn ? `SN ${sn} · 在线 UE ${ueCount} 个` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          {device && (
            <Badge variant={device.isOnline ? 'success' : 'muted'}>
              {device.isOnline ? '在线' : '离线'}
            </Badge>
          )}
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="搜索 IMSI / CPE / IP"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
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
      ) : (
        <TableCard>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>UEID</TableHead>
                  <TableHead>IMSI</TableHead>
                  <TableHead>VMAC</TableHead>
                  <TableHead>CPE 名称</TableHead>
                  <TableHead className="text-right">下行(Mbps)</TableHead>
                  <TableHead className="text-right">上行(Mbps)</TableHead>
                  <TableHead>IP 地址</TableHead>
                  <TableHead className="text-right">SINR(dB)</TableHead>
                  <TableHead className="text-right">CQI</TableHead>
                  <TableHead className="text-right">MCS</TableHead>
                  <TableHead className="text-right">发射功率(dBm)</TableHead>
                  <TableHead className="text-right">路损(dBm)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.length === 0 ? (
                  <EmptyRow colSpan={colCount}>
                    {ueCount === 0 ? '该设备暂无在线 UE' : '没有匹配的 UE'}
                  </EmptyRow>
                ) : (
                  filtered.map((r) => (
                    <TableRow key={r.ueId}>
                      <TableCell className="tabular-nums">{r.ueId}</TableCell>
                      <TableCell>
                        <span className="font-mono text-xs">{r.imsi}</span>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground">
                          {r.vmac}
                        </span>
                      </TableCell>
                      <TableCell className="text-sm">{r.cpeName}</TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.downlinkRate}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.uplinkRate}
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground">
                          {r.ip}
                        </span>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.ulsinr}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.dlcqi}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.ulmcs}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.txpower}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {r.pathloss}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </TableCard>
      )}
    </PageShell>
  )
}
