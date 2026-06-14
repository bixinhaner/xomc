import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  RefreshCcw,
  Search,
  Terminal,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import {
  EmptyRow,
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
import { cn } from '@/lib/utils'

import { useDeviceList } from '@core/hooks/api/useDevices'
import {
  useGroupTree,
  useCommandSubFields,
  useExecuteStatements,
} from '@core/hooks/api/useMmlConsole'
import type {
  GroupTreeNode,
  GroupTreeCommand,
  Statement,
} from '@core/types/mmlConsole'
import type { Device } from '@core/types/device'
import type { MMLTask } from '@core/types/mml'

// ============================================================
// MML 命令树控制台 — 三栏：设备选择 / 命令树 / 操作终端
// 对齐 v1 webcode mml/Console（StepBar + DeviceTree + CommandTree + RightPanel）。
// 数据全走真实 hooks：
//   - useDeviceList     设备列表（多选下发目标）
//   - useGroupTree      命令分组树（分组 → 命令叶子）
//   - useCommandSubFields  选中命令的子字段（LST 勾选项）
//   - useExecuteStatements 执行（构造 Statement[] → POST /mml/execute-statements）
// 仅覆盖 LST（查询）执行的端到端路径；MOD/ADD/RMV 复杂表单不在本皮肤范围（见 notes）。
// ============================================================

const DEVICE_PAGE_SIZE = 50

function DeviceColumn({
  selected,
  onToggle,
  onClear,
}: {
  selected: Map<string, Device>
  onToggle: (d: Device) => void
  onClear: () => void
}) {
  const [searchInput, setSearchInput] = useState('')
  const [searchText, setSearchText] = useState('')

  const params = useMemo(
    () => ({
      page: 1,
      pageSize: DEVICE_PAGE_SIZE,
      ...(searchText.trim() ? { searchText: searchText.trim() } : {}),
    }),
    [searchText]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useDeviceList(params)
  const devices = data?.items ?? []

  return (
    <Card className="flex min-h-[28rem] flex-col p-3">
      <div className="mb-2 flex items-center justify-between">
        <h3 className="text-sm font-semibold">1 · 选择设备</h3>
        <Badge variant={selected.size > 0 ? 'success' : 'muted'}>
          已选 {selected.size}
        </Badge>
      </div>

      <div className="mb-2 flex items-center gap-1.5">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="h-8 pl-8 text-xs"
            placeholder="搜索 SN / 名称 / IP"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') setSearchText(searchInput)
            }}
          />
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          {isFetching ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <RefreshCcw className="size-3.5" />
          )}
        </Button>
      </div>

      {selected.size > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {Array.from(selected.values()).map((d) => (
            <Badge key={d.id} variant="secondary" className="gap-1">
              <span className="font-mono">{d.sn}</span>
              <button type="button" onClick={() => onToggle(d)}>
                <X className="size-3" />
              </button>
            </Badge>
          ))}
          <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={onClear}>
            清空
          </Button>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-auto rounded-md border">
        {isLoading ? (
          <div className="flex h-32 items-center justify-center">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : isError ? (
          <div className="p-3 text-center text-xs text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : devices.length === 0 ? (
          <div className="p-6 text-center text-xs text-muted-foreground">暂无设备</div>
        ) : (
          <ul className="divide-y">
            {devices.map((d) => {
              const checked = selected.has(d.id)
              return (
                <li key={d.id}>
                  <button
                    type="button"
                    onClick={() => onToggle(d)}
                    className={cn(
                      'flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-xs transition-colors hover:bg-muted',
                      checked && 'bg-primary/5'
                    )}
                  >
                    <span
                      className={cn(
                        'flex size-4 shrink-0 items-center justify-center rounded border',
                        checked ? 'border-primary bg-primary text-primary-foreground' : 'border-input'
                      )}
                    >
                      {checked && '✓'}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">
                        {d.deviceName || d.name || d.sn}
                      </span>
                      <span className="block truncate font-mono text-[11px] text-muted-foreground">
                        {d.sn}
                      </span>
                    </span>
                    <span
                      className={cn(
                        'size-1.5 shrink-0 rounded-full',
                        d.isOnline ? 'bg-emerald-500' : 'bg-muted-foreground/40'
                      )}
                    />
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </Card>
  )
}

function CommandColumn({
  selectedCommandId,
  onSelectCommand,
}: {
  selectedCommandId: string | null
  onSelectCommand: (cmd: GroupTreeCommand) => void
}) {
  const { data, isLoading, isError, error, isFetching, refetch } = useGroupTree()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState('')

  const topGroups = useMemo<GroupTreeNode[]>(
    () => (data ?? []).filter((g) => !g.path.includes('.')),
    [data]
  )

  const q = filter.trim().toLowerCase()

  const toggleGroup = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <Card className="flex min-h-[28rem] flex-col p-3">
      <div className="mb-2 flex items-center justify-between">
        <h3 className="text-sm font-semibold">2 · 选择命令</h3>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          {isFetching ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <RefreshCcw className="size-3.5" />
          )}
        </Button>
      </div>

      <div className="relative mb-2">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="h-8 pl-8 text-xs"
          placeholder="过滤命令名 / 命令码"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-auto rounded-md border">
        {isLoading ? (
          <div className="flex h-32 items-center justify-center">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : isError ? (
          <div className="p-3 text-center text-xs text-destructive">
            加载失败：{error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : topGroups.length === 0 ? (
          <div className="p-6 text-center text-xs text-muted-foreground">暂无命令</div>
        ) : (
          <ul className="py-1">
            {topGroups.map((g) => {
              const cmds = (g.commands ?? []).filter((c) =>
                q
                  ? c.displayName.toLowerCase().includes(q) ||
                    c.commandCode.toLowerCase().includes(q)
                  : true
              )
              if (q && cmds.length === 0) return null
              const open = q ? true : expanded.has(g.id)
              return (
                <li key={g.id}>
                  <button
                    type="button"
                    onClick={() => toggleGroup(g.id)}
                    className="flex w-full items-center gap-1 px-2 py-1.5 text-left text-xs font-medium hover:bg-muted"
                  >
                    {open ? (
                      <ChevronDown className="size-3.5 shrink-0" />
                    ) : (
                      <ChevronRight className="size-3.5 shrink-0" />
                    )}
                    <span className="truncate">{g.displayName}</span>
                    <Badge variant="muted" className="ml-auto">
                      {(g.commands ?? []).length}
                    </Badge>
                  </button>
                  {open && (
                    <ul className="pl-5">
                      {cmds.map((c) => (
                        <li key={c.id}>
                          <button
                            type="button"
                            onClick={() => onSelectCommand(c)}
                            className={cn(
                              'flex w-full items-center gap-1.5 px-2 py-1 text-left text-xs transition-colors hover:bg-muted',
                              selectedCommandId === c.id && 'bg-primary/5 text-primary'
                            )}
                          >
                            <Badge variant="outline" className="shrink-0">
                              {c.operationType}
                            </Badge>
                            <span className="truncate">{c.displayName}</span>
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </Card>
  )
}

function TerminalColumn({
  command,
  deviceSns,
  onExecuted,
}: {
  command: GroupTreeCommand | null
  deviceSns: string[]
  onExecuted: (task: MMLTask) => void
}) {
  const { data: subFields, isLoading: sfLoading } = useCommandSubFields(
    command?.id
  )
  const execute = useExecuteStatements()
  const [selectedSf, setSelectedSf] = useState<Set<string>>(new Set())
  const [execErr, setExecErr] = useState<string | null>(null)
  const [lastTask, setLastTask] = useState<MMLTask | null>(null)

  const fields = subFields ?? []
  const isLst = command?.operationType === 'LST'

  const toggleSf = (id: string) => {
    setSelectedSf((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const canExecute =
    Boolean(command) && deviceSns.length > 0 && !execute.isPending

  const handleExecute = () => {
    if (!command) return
    setExecErr(null)
    const statement: Statement = {
      uid: crypto.randomUUID(),
      commandId: command.id,
      commandCode: command.commandCode,
      logicalCode: command.logicalCode,
      operationType: command.operationType,
      logicalNameI18n: command.logicalNameI18n ?? {},
      subFields: fields,
      selectedSubFieldIds: Array.from(selectedSf),
      values: {},
      unknownCodes: [],
      targetObject: command.targetObject,
    }
    execute.mutate(
      {
        statements: [statement],
        deviceSns,
        taskName: `Console-${command.commandCode}-${Date.now()}`,
        executeType: 'immediate',
      },
      {
        onSuccess: (task) => {
          setLastTask(task)
          onExecuted(task)
        },
        onError: (e) =>
          setExecErr(e instanceof Error ? e.message : '执行失败'),
      }
    )
  }

  return (
    <Card className="flex min-h-[28rem] flex-col p-3">
      <div className="mb-2 flex items-center gap-2">
        <Terminal className="size-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold">3 · 操作终端</h3>
      </div>

      {!command ? (
        <div className="flex flex-1 items-center justify-center text-xs text-muted-foreground">
          请先在中间栏选择一个命令
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-3">
          <div className="rounded-md border bg-muted/40 p-2.5 text-xs">
            <div className="flex items-center gap-2">
              <Badge variant="outline">{command.operationType}</Badge>
              <span className="font-medium">{command.displayName}</span>
            </div>
            <div className="mt-1 font-mono text-[11px] text-muted-foreground">
              {command.commandCode}
            </div>
          </div>

          {isLst && (
            <div className="min-h-0 flex-1 overflow-auto rounded-md border">
              <div className="border-b px-2.5 py-1.5 text-xs font-medium text-muted-foreground">
                查询字段（勾选 = 返回项；不勾选 = 全部返回）
              </div>
              {sfLoading ? (
                <div className="flex h-24 items-center justify-center">
                  <Loader2 className="size-4 animate-spin text-muted-foreground" />
                </div>
              ) : fields.length === 0 ? (
                <div className="p-4 text-center text-xs text-muted-foreground">
                  该命令无可选字段
                </div>
              ) : (
                <ul className="divide-y">
                  {fields.map((sf) => {
                    const checked = selectedSf.has(sf.id)
                    return (
                      <li key={sf.id}>
                        <button
                          type="button"
                          onClick={() => toggleSf(sf.id)}
                          className={cn(
                            'flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-xs hover:bg-muted',
                            checked && 'bg-primary/5'
                          )}
                        >
                          <span
                            className={cn(
                              'flex size-4 shrink-0 items-center justify-center rounded border',
                              checked
                                ? 'border-primary bg-primary text-primary-foreground'
                                : 'border-input'
                            )}
                          >
                            {checked && '✓'}
                          </span>
                          <span className="min-w-0 flex-1 truncate">{sf.label}</span>
                          <span className="shrink-0 font-mono text-[10px] text-muted-foreground">
                            {sf.mmlCode}
                          </span>
                        </button>
                      </li>
                    )
                  })}
                </ul>
              )}
            </div>
          )}

          {!isLst && (
            <div className="rounded-md border border-amber-500/30 bg-amber-500/10 px-2.5 py-2 text-xs text-amber-600 dark:text-amber-400">
              {command.operationType} 操作需填写下发值，本控制台仅支持 LST 查询的端到端执行；请到 v1 控制台执行写操作。
            </div>
          )}

          {execErr && (
            <div className="rounded-md border border-destructive/30 bg-destructive/10 px-2.5 py-2 text-xs text-destructive">
              {execErr}
            </div>
          )}

          <div className="flex items-center gap-2">
            <Button
              size="sm"
              disabled={!canExecute || !isLst}
              onClick={handleExecute}
            >
              {execute.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Terminal className="size-4" />
              )}
              执行（{deviceSns.length} 台设备）
            </Button>
            {deviceSns.length === 0 && (
              <span className="text-xs text-muted-foreground">请先选择设备</span>
            )}
          </div>

          {lastTask && (
            <div className="rounded-md border bg-card p-2.5 text-xs">
              <div className="font-medium text-emerald-600 dark:text-emerald-400">
                已提交任务：{lastTask.taskName}
              </div>
              <div className="mt-0.5 text-muted-foreground">
                状态 {lastTask.status} · 设备 {lastTask.totalDevices ?? deviceSns.length}
              </div>
            </div>
          )}
        </div>
      )}
    </Card>
  )
}

export default function Console() {
  const navigate = useNavigate()
  const [selectedDevices, setSelectedDevices] = useState<Map<string, Device>>(
    new Map()
  )
  const [selectedCommand, setSelectedCommand] = useState<GroupTreeCommand | null>(
    null
  )
  const [recentTasks, setRecentTasks] = useState<MMLTask[]>([])

  const toggleDevice = (d: Device) => {
    setSelectedDevices((prev) => {
      const next = new Map(prev)
      if (next.has(d.id)) next.delete(d.id)
      else next.set(d.id, d)
      return next
    })
  }

  const deviceSns = useMemo(
    () => Array.from(selectedDevices.values()).map((d) => d.sn),
    [selectedDevices]
  )

  const taskCols = ['任务名称', '状态', '设备数', '操作']

  return (
    <PageShell
      title="MML 命令树控制台"
      description="选择设备 → 浏览命令树 → 执行查询命令"
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <DeviceColumn
          selected={selectedDevices}
          onToggle={toggleDevice}
          onClear={() => setSelectedDevices(new Map())}
        />
        <CommandColumn
          selectedCommandId={selectedCommand?.id ?? null}
          onSelectCommand={setSelectedCommand}
        />
        <TerminalColumn
          command={selectedCommand}
          deviceSns={deviceSns}
          onExecuted={(task) =>
            setRecentTasks((prev) => [task, ...prev].slice(0, 10))
          }
        />
      </div>

      {recentTasks.length > 0 && (
        <div className="mt-6">
          <h3 className="mb-2 text-sm font-semibold">本次会话已提交任务</h3>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  {taskCols.map((c) => (
                    <TableHead key={c}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentTasks.length === 0 ? (
                  <EmptyRow colSpan={taskCols.length}>暂无</EmptyRow>
                ) : (
                  recentTasks.map((t) => (
                    <TableRow key={t.id}>
                      <TableCell className="font-medium">{t.taskName}</TableCell>
                      <TableCell>
                        <Badge variant="default">{t.status}</Badge>
                      </TableCell>
                      <TableCell className="tabular-nums">
                        {t.totalDevices ?? deviceSns.length}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => navigate('/mml')}
                        >
                          查看任务记录
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>
        </div>
      )}
    </PageShell>
  )
}
