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
  formatTime,
} from '@/components/layout/PageShell'

import { useMMLTemplates } from '@core/hooks/api/useMML'

// ============================================================
// MML 私有命令详情 — 按 id 展示单条自定义命令 + 参数 / PATH 明细
// 带 :id 参数路由，hidden（不进侧栏），由私有命令列表行点击进入。
// 数据：useMMLTemplates(scope=private, 大页) 后按 id 定位（自定义命令为低频数据）。
// ============================================================

const OP_VARIANT: Record<string, 'default' | 'warning' | 'outline'> = {
  MOD: 'warning',
  LST: 'default',
}

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
      <dd
        className={
          mono ? 'mt-0.5 break-words font-mono text-xs' : 'mt-0.5 break-words'
        }
      >
        {value}
      </dd>
    </div>
  )
}

export default function PrivateCommandDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error } = useMMLTemplates({
    templateScope: 'private',
    page: 1,
    pageSize: 500,
  })

  const command = useMemo(
    () => (data?.items ?? []).find((c) => c.id === id),
    [data, id]
  )

  const paramEntries = useMemo(
    () => (command ? Object.entries(command.parameters ?? {}) : []),
    [command]
  )
  const paths = command?.paramPaths ?? []

  return (
    <PageShell
      title={command ? command.commandName : 'MML 私有命令详情'}
      description={command ? command.commandCode : id}
      toolbar={
        <Button variant="ghost" size="sm" onClick={() => navigate('/mml/private-command')}>
          <ArrowLeft className="size-4" /> 返回私有命令
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
          <span className="text-sm">未找到私有命令 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/mml/private-command')}>
            返回私有命令
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-4">
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              命令信息
              <Badge variant={OP_VARIANT[command.operationType] ?? 'outline'}>
                {command.operationType}
              </Badge>
              <Badge variant={command.commandScope === 'public' ? 'success' : 'muted'}>
                {command.commandScope === 'public' ? '公有' : '私有'}
              </Badge>
            </h3>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
              <Field label="命令名称" value={command.commandName} />
              <Field label="命令码" value={command.commandCode} mono />
              <Field label="分组" value={command.categoryGroup || '—'} />
              <Field label="创建人" value={command.creator} />
              <Field label="创建时间" value={formatTime(command.createdAt)} />
              <Field label="更新时间" value={formatTime(command.updatedAt)} />
              <Field label="描述" value={command.description || '—'} span />
            </dl>
          </Card>

          <Card className="p-4">
            <h3 className="mb-3 text-sm font-semibold">PATH 列表（{paths.length}）</h3>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>TR-069 路径</TableHead>
                    <TableHead>下发值</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paths.length === 0 ? (
                    <EmptyRow colSpan={2}>该命令无 PATH</EmptyRow>
                  ) : (
                    paths.map((p) => {
                      const v = command.parameters?.[p]
                      return (
                        <TableRow key={p}>
                          <TableCell className="font-mono text-xs">{p}</TableCell>
                          <TableCell className="font-mono text-xs">
                            {v != null ? String(v) : '—'}
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            </TableCard>
          </Card>

          {paramEntries.length > 0 && (
            <Card className="p-4">
              <h3 className="mb-3 text-sm font-semibold">
                参数键值（{paramEntries.length}）
              </h3>
              <dl className="grid grid-cols-1 gap-x-6 gap-y-2 text-sm sm:grid-cols-2">
                {paramEntries.map(([k, v]) => (
                  <Field key={k} label={k} value={String(v)} mono />
                ))}
              </dl>
            </Card>
          )}
        </div>
      )}
    </PageShell>
  )
}
