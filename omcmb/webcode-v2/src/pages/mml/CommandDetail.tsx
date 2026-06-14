import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import { useAllMMLCommands } from '@core/hooks/api/useMML'

// ============================================================
// MML 命令详情 — 按 id 展示单条命令元数据 + 绑定参数
// 带 :id 参数路由，hidden（不进侧栏），由命令树行点击进入。
// 数据：useAllMMLCommands() 拉全量命令字典后按 id 定位（命令字典为低频字典型数据）。
// ============================================================

function Field({
  label,
  value,
  span,
  mono,
}: {
  label: string
  value: string
  span?: boolean
  mono?: boolean
}) {
  return (
    <div className={span ? 'col-span-2' : undefined}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className={span || !mono ? 'mt-0.5 break-words' : 'mt-0.5 break-words font-mono text-xs'}>
        {value}
      </dd>
    </div>
  )
}

export default function CommandDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useAllMMLCommands()

  const command = useMemo(
    () => (data ?? []).find((c) => c.id === id),
    [data, id]
  )

  const refs = command?.paramRefs ?? []
  const paramCols = ['参数名', 'TR-069 路径', '类型', '可写']

  return (
    <PageShell
      title={command ? command.commandName : 'MML 命令详情'}
      description={command ? command.commandCode : id}
      toolbar={
        <Button variant="ghost" size="sm" onClick={() => navigate('/mml/commands-tree')}>
          <ArrowLeft className="size-4" /> 返回命令树
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
      ) : !command ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到命令 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/mml/commands-tree')}>
            返回命令树
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-4">
            <h3 className="mb-3 text-sm font-semibold">命令信息</h3>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
              <Field label="命令名称" value={command.commandName} />
              <Field label="命令码" value={command.commandCode} mono />
              <Field label="操作类型" value={command.operationType || '—'} />
              <Field label="分类" value={command.category || '—'} />
              <Field label="描述" value={command.description || '—'} span />
              {command.targetObject && (
                <Field label="目标对象" value={command.targetObject} span mono />
              )}
              {command.requireConfirm != null && (
                <Field
                  label="二次确认"
                  value={command.requireConfirm ? '需要' : '不需要'}
                />
              )}
            </dl>
          </Card>

          <Card className="p-4">
            <h3 className="mb-3 text-sm font-semibold">
              绑定参数（{refs.length}）
            </h3>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    {paramCols.map((c) => (
                      <TableHead key={c}>{c}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {refs.length === 0 ? (
                    <EmptyRow colSpan={paramCols.length}>该命令无绑定参数</EmptyRow>
                  ) : (
                    refs.map((p) => (
                      <TableRow key={p.id}>
                        <TableCell className="text-xs">
                          {p.paramNameZh || p.paramCode}
                        </TableCell>
                        <TableCell className="font-mono text-xs">{p.tr069Path}</TableCell>
                        <TableCell className="text-xs">{p.valueType}</TableCell>
                        <TableCell>
                          <Badge variant={p.isWritable ? 'success' : 'muted'}>
                            {p.isWritable ? '可写' : '只读'}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableCard>
          </Card>
        </div>
      )}
    </PageShell>
  )
}
