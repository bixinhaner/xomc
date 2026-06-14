import type { ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import { useBaselineConfigById } from '@core/hooks/api/useConfig'

// ============================================================
// 配置基线详情 — /config/baseline/:id（隐藏路由）。
// 真实数据：useBaselineConfigById；展示基线元信息 + 参数清单。
// ============================================================

const BASELINE_STATUS: Record<
  string,
  { variant: 'success' | 'warning' | 'muted'; label: string }
> = {
  active: { variant: 'success', label: '生效' },
  draft: { variant: 'warning', label: '草稿' },
  deprecated: { variant: 'muted', label: '废弃' },
}

export default function BaselineDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching, refetch } = useBaselineConfigById(id)
  const params = data?.params ?? []
  const cols = ['参数名称', '参数编码', '类型', '基线值', '单位', '只读']

  return (
    <PageShell
      title={data ? `基线：${data.baselineName}` : '配置基线详情'}
      description={data?.description || '配置基线参数清单'}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/config/baseline')}>
            <ArrowLeft className="size-4" /> 返回列表
          </Button>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => void refetch()}>
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      {data && (
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">基线信息</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-4">
              <Field label="设备类型" value={data.deviceType || '—'} />
              <Field label="版本" value={data.version || '—'} />
              <Field
                label="状态"
                node={
                  <Badge variant={(BASELINE_STATUS[data.status] ?? BASELINE_STATUS.draft).variant}>
                    {(BASELINE_STATUS[data.status] ?? BASELINE_STATUS.draft).label}
                  </Badge>
                }
              />
              <Field label="参数数量" value={String(data.params?.length ?? 0)} />
              <Field label="创建人" value={data.creator || '—'} />
              <Field label="更新时间" value={formatTime(data.updateTime)} />
            </dl>
          </CardContent>
        </Card>
      )}

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
            ) : params.length === 0 ? (
              <EmptyRow colSpan={cols.length}>该基线暂无参数</EmptyRow>
            ) : (
              params.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="text-sm">{p.paramName}</TableCell>
                  <TableCell className="font-mono text-xs">{p.paramCode}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{p.paramType}</Badge>
                  </TableCell>
                  <TableCell className="text-sm">{String(p.paramValue ?? p.defaultValue ?? '—')}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{p.unit || '—'}</TableCell>
                  <TableCell>
                    {p.readonly ? (
                      <Badge variant="muted">只读</Badge>
                    ) : (
                      <Badge variant="success">可写</Badge>
                    )}
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

function Field({
  label,
  value,
  node,
}: {
  label: string
  value?: string
  node?: ReactNode
}) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 font-medium">{node ?? value}</dd>
    </div>
  )
}
