import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, Search, Trash2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  useMMLTemplates,
  useDeleteMMLTemplate,
} from '@core/hooks/api/useMML'
import type { MMLCustomCommand } from '@core/types/mml'

// ============================================================
// MML 私有命令 — 用户自定义命令列表（对齐 v1 webcode mml/PrivateCommand）
// 后端 GET /mml/templates?command_scope=private 服务端 RBAC 自动过滤。
// 行可进详情子路由 /mml/private-command/:id；支持删除。
// ============================================================

const OP_VARIANT: Record<string, 'default' | 'warning' | 'outline'> = {
  MOD: 'warning',
  LST: 'default',
}

const PAGE_SIZE = 20

export default function PrivateCommand() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [codeInput, setCodeInput] = useState('')
  const [commandCode, setCommandCode] = useState('')
  const [deleteErr, setDeleteErr] = useState<string | null>(null)

  const params = useMemo(
    () => ({
      templateScope: 'private',
      page,
      pageSize: PAGE_SIZE,
      ...(commandCode.trim() ? { commandCode: commandCode.trim() } : {}),
    }),
    [page, commandCode]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useMMLTemplates(params)
  const deleteMutation = useDeleteMMLTemplate()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const handleDelete = (c: MMLCustomCommand) => {
    if (!window.confirm(`确认删除私有命令「${c.commandName}」？此操作不可撤销。`)) return
    setDeleteErr(null)
    deleteMutation.mutate(c.id, {
      onError: (e) => setDeleteErr(e instanceof Error ? e.message : '删除失败'),
    })
  }

  const cols = ['命令名称', '命令码', '操作类型', '分组', '描述', '创建人', '更新时间', '操作']

  return (
    <PageShell
      title="MML 私有命令"
      description="用户自定义命令（按创建者 / 分组共享自动过滤）"
      isFetching={isFetching}
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索命令码"
            value={codeInput}
            onChange={(e) => setCodeInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                setCommandCode(codeInput)
                setPage(1)
              }
            }}
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setCommandCode(codeInput)
            setPage(1)
          }}
        >
          <Search /> 搜索
        </Button>
        <span className="text-sm text-muted-foreground">共 {total} 条</span>
        <div className="ml-auto">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      {deleteErr && (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
          {deleteErr}
        </div>
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
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无私有命令</EmptyRow>
            ) : (
              rows.map((c) => (
                <TableRow key={c.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() => navigate(`/mml/private-command/${c.id}`)}
                    >
                      {c.commandName}
                    </button>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{c.commandCode}</TableCell>
                  <TableCell>
                    <Badge variant={OP_VARIANT[c.operationType] ?? 'outline'}>
                      {c.operationType}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {c.categoryGroup || '—'}
                  </TableCell>
                  <TableCell className="max-w-xs text-xs text-muted-foreground">
                    <span className="line-clamp-1">{c.description || '—'}</span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{c.creator}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(c.updatedAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/mml/private-command/${c.id}`)}
                      >
                        详情
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        disabled={deleteMutation.isPending}
                        onClick={() => handleDelete(c)}
                      >
                        <Trash2 />
                      </Button>
                    </div>
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
    </PageShell>
  )
}
