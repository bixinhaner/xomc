import { useMemo, useState } from 'react'
import { Loader2, Play, Search, Terminal } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useMMLCommands, useExecuteMMLCommand } from '@core/hooks/api/useMML'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { MMLCommand, MMLTask } from '@core/types/mml'

// ============================================================
// 命令行模式 — 对照 v1 config/CommandMode（MML 命令控制台）。
// 真实数据：useMMLCommands 命令目录 + useExecuteMMLCommand 执行（创建 MMLTask）。
// 三栏：命令列表 / 命令详情+参数表单 / 执行结果。
// ============================================================

export default function CommandMode() {
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<MMLCommand | null>(null)
  const [deviceSn, setDeviceSn] = useState('')
  const [paramValues, setParamValues] = useState<Record<string, string>>({})
  const [lastTask, setLastTask] = useState<MMLTask | null>(null)

  const commandsQuery = useMMLCommands({ keyword, page: 1, pageSize: 100 })
  const deviceQuery = useDeviceList({ page: 1, pageSize: 200 })
  const execute = useExecuteMMLCommand()

  const commands = commandsQuery.data?.items ?? []
  const devices = deviceQuery.data?.items ?? []

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return commands
    return commands.filter(
      (c) =>
        c.commandName.toLowerCase().includes(kw) ||
        c.commandCode.toLowerCase().includes(kw)
    )
  }, [commands, keyword])

  const pickCommand = (c: MMLCommand) => {
    setSelected(c)
    setParamValues({})
    setLastTask(null)
  }

  const handleExecute = () => {
    if (!selected || !deviceSn) return
    const params: Record<string, string | number | boolean> = {}
    for (const [k, v] of Object.entries(paramValues)) {
      if (v !== '') params[k] = v
    }
    execute.mutate(
      { commandCode: selected.commandCode, deviceSns: [deviceSn], params },
      { onSuccess: (task) => setLastTask(task) }
    )
  }

  return (
    <PageShell
      title="命令行模式"
      description="MML 命令控制台 — 选命令、填参数、对设备执行"
      isFetching={commandsQuery.isFetching}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[280px_1fr_1fr]">
        {/* 命令列表 */}
        <div className="flex flex-col rounded-lg border bg-card">
          <div className="border-b px-3 py-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="h-8 pl-8"
                placeholder="搜索命令"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
            </div>
          </div>
          <div className="max-h-[60vh] flex-1 overflow-auto">
            {commandsQuery.isLoading ? (
              <div className="p-4 text-center text-sm text-muted-foreground">
                <Loader2 className="mx-auto size-4 animate-spin" />
              </div>
            ) : commandsQuery.isError ? (
              <div className="p-4 text-sm text-destructive">
                加载失败：
                {commandsQuery.error instanceof Error ? commandsQuery.error.message : '未知错误'}
              </div>
            ) : filtered.length === 0 ? (
              <div className="p-4 text-sm text-muted-foreground">暂无命令</div>
            ) : (
              filtered.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => pickCommand(c)}
                  className={cn(
                    'block w-full border-b px-3 py-2 text-left last:border-b-0 hover:bg-muted/50',
                    selected?.id === c.id && 'bg-primary/5'
                  )}
                >
                  <div className="text-sm font-medium">{c.commandName}</div>
                  <div className="font-mono text-xs text-muted-foreground">{c.commandCode}</div>
                </button>
              ))
            )}
          </div>
        </div>

        {/* 命令详情 + 参数 */}
        <div className="rounded-lg border bg-card p-4">
          {selected ? (
            <div className="space-y-4">
              <div>
                <div className="text-base font-semibold">{selected.commandName}</div>
                <div className="mt-1 flex items-center gap-2">
                  <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                    {selected.commandCode}
                  </code>
                  {selected.category && <Badge variant="muted">{selected.category}</Badge>}
                </div>
                {selected.description && (
                  <p className="mt-2 text-sm text-muted-foreground">{selected.description}</p>
                )}
              </div>

              <div>
                <Label className="mb-1.5 block">目标设备</Label>
                <Select value={deviceSn || 'none'} onValueChange={(v) => setDeviceSn(v === 'none' ? '' : v)}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择设备 SN" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">请选择设备</SelectItem>
                    {devices
                      .filter((d) => d.sn)
                      .map((d) => (
                        <SelectItem key={d.id} value={d.sn}>
                          {d.sn}
                          {d.name && d.name !== d.sn ? ` (${d.name})` : ''}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              </div>

              {selected.params.length > 0 ? (
                <div className="space-y-3">
                  <Label className="block">命令参数</Label>
                  {selected.params.map((param) => (
                    <div key={param.name}>
                      <Label className="mb-1 block text-xs">
                        {param.name}
                        {param.required && <span className="ml-1 text-destructive">*</span>}
                      </Label>
                      <Input
                        className="h-8"
                        placeholder={param.description || param.name}
                        value={paramValues[param.name] ?? ''}
                        onChange={(e) =>
                          setParamValues((prev) => ({ ...prev, [param.name]: e.target.value }))
                        }
                      />
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">该命令无需参数</p>
              )}

              <Button
                className="w-full"
                disabled={!deviceSn || execute.isPending}
                onClick={handleExecute}
              >
                {execute.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Play className="size-4" />
                )}
                执行
              </Button>
            </div>
          ) : (
            <div className="flex h-full min-h-48 items-center justify-center text-sm text-muted-foreground">
              请从左侧选择命令
            </div>
          )}
        </div>

        {/* 执行结果 */}
        <div className="flex flex-col rounded-lg border bg-card">
          <div className="flex items-center gap-2 border-b px-3 py-2 text-sm font-medium">
            <Terminal className="size-4" /> 执行结果
          </div>
          <div className="flex-1 overflow-auto p-3 font-mono text-xs">
            {execute.isError ? (
              <div className="text-destructive">
                执行失败：
                {execute.error instanceof Error ? execute.error.message : '未知错误'}
              </div>
            ) : lastTask ? (
              <div className="space-y-1">
                <div>
                  <span className="text-muted-foreground">任务 ID：</span>
                  {lastTask.id}
                </div>
                <div>
                  <span className="text-muted-foreground">任务名：</span>
                  {lastTask.taskName}
                </div>
                <div>
                  <span className="text-muted-foreground">状态：</span>
                  <Badge variant="secondary">{lastTask.status}</Badge>
                </div>
                <div>
                  <span className="text-muted-foreground">设备数：</span>
                  {lastTask.totalDevices}
                </div>
                <div>
                  <span className="text-muted-foreground">命令：</span>
                  {lastTask.commands.join(', ') || '—'}
                </div>
              </div>
            ) : (
              <div className="text-muted-foreground">尚未执行命令</div>
            )}
          </div>
        </div>
      </div>
    </PageShell>
  )
}
