import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Lock, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  EmptyRow,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  useGroupTree,
  useCommandSubFields,
} from '@core/hooks/api/useMmlConsole'
import type { GroupTreeCommand, GroupTreeNode } from '@core/types/mmlConsole'

// ============================================================
// MML 字典命令详情 — 按 id 展示单条字典命令 + 子字段（sub-fields）
// 带 :id 参数路由，hidden（不进侧栏），由命令字典管理页命令叶子点击进入。
// 数据：useGroupTree 定位命令（含所属分组）+ useCommandSubFields 拉子字段元数据。
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

function findCommand(
  groups: GroupTreeNode[],
  id: string
): { command: GroupTreeCommand; group: GroupTreeNode } | null {
  for (const g of groups) {
    const c = (g.commands ?? []).find((cmd) => cmd.id === id)
    if (c) return { command: c, group: g }
  }
  return null
}

export default function CatalogCommandDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: tree, isLoading, isError, error } = useGroupTree()

  const ctx = useMemo(
    () => (tree ? findCommand(tree, id) : null),
    [tree, id]
  )
  const command = ctx?.command ?? null

  const { data: subFields, isLoading: sfLoading } = useCommandSubFields(
    command?.id
  )
  const fields = subFields ?? []

  const sfCols = ['字段', 'MML 码', 'TR-069 路径', '类型', '访问', '必填']

  return (
    <PageShell
      title={command ? command.displayName : 'MML 字典命令详情'}
      description={command ? command.commandCode : id}
      toolbar={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/mml/admin/catalog')}
        >
          <ArrowLeft className="size-4" /> 返回命令字典
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
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/mml/admin/catalog')}
          >
            返回命令字典
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-4">
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              命令信息
              <Badge variant="outline">{command.operationType}</Badge>
              {command.catalogProtected && (
                <Badge variant="muted" className="gap-1">
                  <Lock className="size-3" /> 受保护
                </Badge>
              )}
            </h3>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
              <Field label="显示名" value={command.displayName} />
              <Field label="命令码" value={command.commandCode} mono />
              <Field label="逻辑码" value={command.logicalCode} mono />
              <Field label="所属分组" value={ctx?.group.displayName ?? '—'} />
              {command.rpcMethod && (
                <Field label="RPC 方法" value={command.rpcMethod} />
              )}
              {command.targetObject && (
                <Field label="目标对象" value={command.targetObject} span mono />
              )}
              <Field label="来源" value={command.source ?? '—'} />
              <Field
                label="二次确认"
                value={command.requireConfirm ? '需要' : '不需要'}
              />
            </dl>
          </Card>

          <Card className="p-4">
            <h3 className="mb-3 text-sm font-semibold">子字段（{fields.length}）</h3>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    {sfCols.map((c) => (
                      <TableHead key={c}>{c}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sfLoading ? (
                    <LoadingRow colSpan={sfCols.length} />
                  ) : fields.length === 0 ? (
                    <EmptyRow colSpan={sfCols.length}>该命令无子字段</EmptyRow>
                  ) : (
                    fields.map((sf) => (
                      <TableRow key={sf.id}>
                        <TableCell className="text-xs">{sf.label}</TableCell>
                        <TableCell className="font-mono text-xs">{sf.mmlCode}</TableCell>
                        <TableCell className="font-mono text-xs">{sf.tr069Path}</TableCell>
                        <TableCell className="text-xs">{sf.valueType}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {sf.accessType}
                        </TableCell>
                        <TableCell>
                          <Badge variant={sf.isRequired ? 'warning' : 'muted'}>
                            {sf.isRequired ? '必填' : '可选'}
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
