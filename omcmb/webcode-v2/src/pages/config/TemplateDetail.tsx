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

import { useConfigTemplateById } from '@core/hooks/api/useConfig'

// ============================================================
// 配置模板详情 — /config/batch-template/:id（隐藏路由）。
// 真实数据：useConfigTemplateById；展示模板元信息 + 参数清单。
// ============================================================

export default function TemplateDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching, refetch } = useConfigTemplateById(id)
  const params = data?.params ?? []
  const cols = ['参数名称', '参数编码', '类型', '默认值', '单位', '只读']

  return (
    <PageShell
      title={data ? `模板：${data.templateName}` : '配置模板详情'}
      description={data?.description || '配置模板参数清单'}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/config/batch-template')}>
            <ArrowLeft className="size-4" /> 返回列表
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => void refetch()}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
        </div>
      }
    >
      {data && (
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">模板信息</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm md:grid-cols-4">
              <Field label="模板名称" value={data.templateName} />
              <Field label="参数数量" value={String(data.params?.length ?? 0)} />
              <Field label="创建人" value={data.creator || '—'} />
              <Field label="创建时间" value={formatTime(data.createTime)} />
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
              <EmptyRow colSpan={cols.length}>该模板暂无参数</EmptyRow>
            ) : (
              params.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="text-sm">{p.paramName}</TableCell>
                  <TableCell className="font-mono text-xs">{p.paramCode}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{p.paramType}</Badge>
                  </TableCell>
                  <TableCell className="text-sm">{String(p.defaultValue ?? '—')}</TableCell>
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

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 font-medium">{value}</dd>
    </div>
  )
}
