import { RefreshCcw, MapPin } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
} from '@/components/layout/PageShell'

import { useSites } from '@core/hooks/api/useTopology'

export function TopologyPage() {
  const { data, isLoading, isError, error, isFetching, refetch } = useSites()
  const rows = Array.isArray(data) ? data : []

  const cols = ['站点', '地址', '设备数', '状态', '坐标']

  return (
    <PageShell
      title="拓扑视图"
      description="站点列表 · 地理视图由独立地图模块承接（待补齐）"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => refetch()}>
          <RefreshCcw /> 刷新
        </Button>
      }
    >
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无站点</EmptyRow>
            ) : (
              rows.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="font-medium">{s.name}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{s.address}</TableCell>
                  <TableCell className="tabular-nums">{s.deviceCount}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        s.status === 'normal'
                          ? 'success'
                          : s.status === 'warning'
                            ? 'warning'
                            : s.status === 'critical'
                              ? 'destructive'
                              : 'muted'
                      }
                    >
                      {s.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    <span className="inline-flex items-center gap-1">
                      <MapPin className="size-3" />
                      {s.latitude.toFixed(3)}, {s.longitude.toFixed(3)}
                    </span>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
