import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Inbox,
  Play,
  Plus,
  XCircle,
  X,
  ScrollText,
  Tag as TagIcon,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { useCreateMMLScript, useCreateMMLTask, useMMLScriptById, useMMLScripts } from '@core/hooks/api/useMML'
import { useUserStore } from '@core/store/userStore'
import type { MMLScript, MMLScriptStatus, MMLTaskPlanItem } from '@core/types/mml'
import { parseMmlScriptPlan } from '@core/utils/mmlScriptPlanParser'

// ─────────────────────────────────────────────────────────────
// SCRIPT VAULT · mml_scripts 脚本库（real：useMMLScripts）
// v1 路由 mml/script（ScriptTask）—— 脚本库列表 + 详情。
// ─────────────────────────────────────────────────────────────

const STATUS_TONE: Record<MMLScriptStatus, string> = {
  active: 'ok',
  archived: 'off',
  pending: 'off',
  running: 'warning',
  paused: 'minor',
  completed: 'ok',
  failed: 'critical',
  cancelled: 'offline',
}

function scriptTone(status: string): string {
  return STATUS_TONE[status as MMLScriptStatus] ?? 'unknown'
}

const PAGE_SIZE = 20

export function MMLScriptPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [viewing, setViewing] = useState<MMLScript | null>(null)
  const [executing, setExecuting] = useState<MMLScript | null>(null)
  const [creating, setCreating] = useState(false)

  const params = useMemo(
    () => ({ page, pageSize: PAGE_SIZE, ...(search.trim() ? { search: search.trim() } : {}) }),
    [page, search]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMMLScripts(params)
  const scripts = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 当前页状态分桶（态势带）。
  const statusCounts = useMemo(() => {
    const m = new Map<string, number>()
    for (const s of scripts) m.set(s.status, (m.get(s.status) ?? 0) + 1)
    return m
  }, [scripts])

  return (
    <PageShell
      code="F06"
      title="SCRIPT VAULT · 脚本库"
      subtitle="MAN-MACHINE LANGUAGE · mml_scripts BATCH SCRIPTS"
      bare
      isFetching={isFetching}
      toolbar={
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="脚本名称"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            {isFetching ? 'SYNC…' : 'REFRESH'}
          </NeonButton>
          <NeonButton icon={<Plus />} onClick={() => setCreating(true)}>
            新建脚本
          </NeonButton>
        </div>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-3 gap-3 md:grid-cols-5">
          <StatCard label="TOTAL" value={total} color="#00f0ff" trend={[4, 6, 5, 8, 7, 9, 11]} />
          <StatCard label="活跃" value={statusCounts.get('active') ?? 0} color="#00ff88" />
          <StatCard label="执行中" value={statusCounts.get('running') ?? 0} color="#ffaa00" />
          <StatCard label="已完成" value={statusCounts.get('completed') ?? 0} color="#00ff88" />
          <StatCard label="失败" value={statusCounts.get('failed') ?? 0} color="#ff2d6f" />
        </div>

        <GlassPanel
          title="SCRIPT VAULT · 脚本库"
          meta={`PAGE ${page}/${totalPages}`}
          className="min-h-0 flex-1 overflow-hidden"
        >
          <div className="h-full overflow-auto">
            <div className="sticky top-0 z-10 grid grid-cols-[2fr_2.4fr_1fr_1fr_1.4fr] gap-3 border-b border-cyan-500/20 bg-[#03050d]/85 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/60 backdrop-blur">
              <span>脚本名</span>
              <span>描述 / 标签</span>
              <span>创建人</span>
              <span>状态</span>
              <span>更新时间</span>
            </div>

            {isLoading ? (
              <CenterState>
                <Loader2 className="size-4 animate-spin" />
                <span>SYNCING SCRIPT VAULT…</span>
              </CenterState>
            ) : isError ? (
              <CenterState tone="err">
                <XCircle className="size-5" />
                <span>FAILURE · {error instanceof Error ? error.message : '未知错误'}</span>
              </CenterState>
            ) : scripts.length === 0 ? (
              <CenterState>
                <Inbox className="size-6" />
                <span>无脚本</span>
              </CenterState>
            ) : (
              scripts.map((s) => (
                <button
                  type="button"
                  key={s.id}
                  onClick={() => setViewing(s)}
                  className="grid w-full grid-cols-[2fr_2.4fr_1fr_1fr_1.4fr] items-center gap-3 border-b border-cyan-500/8 px-3 py-2.5 text-left hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <ScrollText className="size-3.5 shrink-0 text-cyan-300/45" />
                      <span className="truncate font-display text-sm font-bold text-cyan-100">
                        {s.scriptName}
                      </span>
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/45">{s.id}</div>
                  </div>
                  <div className="min-w-0">
                    <div className="truncate text-xs text-cyan-100/75">{s.description || '—'}</div>
                    {s.tags && s.tags.length > 0 ? (
                      <div className="mt-0.5 flex flex-wrap gap-1">
                        {s.tags.slice(0, 4).map((tag) => (
                          <span
                            key={tag}
                            className="rounded-sm border border-cyan-500/25 bg-cyan-500/8 px-1.5 py-0.5 font-mono text-[9px] text-cyan-300/75"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  <div className="truncate text-xs text-cyan-100/80">{s.creator || '—'}</div>
                  <div>
                    <StatusBadge status={scriptTone(s.status)} label={s.status} />
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/70">{formatTime(s.updateTime)}</div>
                </button>
              ))
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
      </div>

      {viewing && (
        <ScriptDetailDrawer
          script={viewing}
          onClose={() => setViewing(null)}
          onExecute={(scriptToRun) => setExecuting(scriptToRun)}
        />
      )}
      {creating && (
        <ScriptCreateDialog
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false)
            void refetch()
          }}
        />
      )}
      {executing && <ScriptExecuteDialog script={executing} onClose={() => setExecuting(null)} />}
    </PageShell>
  )
}

function ScriptCreateDialog({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const username = useUserStore((s) => s.currentUser?.username) ?? ''
  const createScript = useCreateMMLScript()
  const [scriptName, setScriptName] = useState('')
  const [description, setDescription] = useState('')
  const [tagsText, setTagsText] = useState('')
  const [content, setContent] = useState('')
  const [error, setError] = useState('')

  const parsed = useMemo(() => parseMmlScriptPlan(content, { format: 'auto' }), [content])

  const submit = async () => {
    setError('')
    if (!scriptName.trim()) {
      setError('请输入脚本名称')
      return
    }
    if (!content.trim()) {
      setError('请输入脚本内容')
      return
    }
    try {
      await createScript.mutateAsync({
        scriptName: scriptName.trim(),
        description: description.trim(),
        content,
        creator: username,
        tags: tagsText
          .split(/[,\s;]+/)
          .map((tag) => tag.trim())
          .filter(Boolean),
        status: 'active',
        type: 'manual',
        progress: 0,
      })
      onCreated()
    } catch (e) {
      setError(e instanceof Error ? e.message : '保存失败')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-strong flex max-h-[90vh] w-[min(860px,calc(100vw-32px))] flex-col overflow-hidden border border-cyan-500/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="font-display text-base font-bold text-cyan-100">新建脚本</div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-100"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
          <label className="block">
            <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">脚本名称</span>
            <input className="neon-input w-full" value={scriptName} onChange={(e) => setScriptName(e.target.value)} />
          </label>
          <label className="block">
            <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">描述</span>
            <input className="neon-input w-full" value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
          <label className="block">
            <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">标签</span>
            <input className="neon-input w-full" value={tagsText} placeholder="逗号或空格分隔" onChange={(e) => setTagsText(e.target.value)} />
          </label>
          <label className="block">
            <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">脚本内容</span>
            <textarea
              className="neon-input min-h-44 w-full resize-y font-mono text-xs leading-relaxed"
              value={content}
              placeholder={'LST DEVICE_INFO;1202000091177SP0005\nMOD DEVICE_INFO:USER_LABEL=Site-A;1202000091177SP0006\nLST DEVICE_INFO;1202000091177SP0006'}
              onChange={(e) => setContent(e.target.value)}
            />
          </label>

          {content.trim() ? (
            <ScriptPlanPreview parsedMode={parsed.executeMode} planItems={parsed.planItems} commandCount={parsed.commands.length} deviceCount={parsed.deviceSns.length} warnings={parsed.warnings} />
          ) : null}

          {error ? <div className="font-mono text-[11px] text-rose-300">{error}</div> : null}
        </div>

        <div className="flex justify-end gap-2 border-t border-cyan-500/20 px-4 py-3">
          <NeonButton onClick={onClose}>CANCEL</NeonButton>
          <NeonButton icon={createScript.isPending ? <Loader2 className="animate-spin" /> : <Plus />} onClick={() => void submit()} disabled={createScript.isPending}>
            SAVE
          </NeonButton>
        </div>
      </div>
    </div>
  )
}

function ScriptPlanPreview({
  parsedMode,
  planItems,
  commandCount,
  deviceCount,
  warnings,
}: {
  parsedMode: 'common' | 'device_bound'
  planItems: MMLTaskPlanItem[]
  commandCount: number
  deviceCount: number
  warnings: string[]
}) {
  const isDeviceBound = parsedMode === 'device_bound'
  return (
    <div className="border border-cyan-500/15 bg-cyan-500/4">
      <div className="flex items-center justify-between border-b border-cyan-500/15 px-3 py-2">
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">解析预览</span>
        <span className="font-mono text-[10px] text-cyan-100/80">
          {isDeviceBound ? '按设备编排' : '公共脚本'}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 px-3 py-2">
        <MiniStat label="MODE" value={isDeviceBound ? 'DEVICE-BOUND' : 'COMMON'} />
        <MiniStat
          label="PLAN"
          value={isDeviceBound ? `${planItems.length} ROWS / ${deviceCount} DEVICES` : `${commandCount} COMMANDS`}
        />
      </div>
      {warnings.length > 0 ? (
        <div className="px-3 pb-2 font-mono text-[11px] text-amber-200">
          已同时检测到带 SN 和不带 SN 的脚本行；执行时将按设备计划行处理。
        </div>
      ) : null}
      {isDeviceBound ? (
        <div className="max-h-56 overflow-auto border-t border-cyan-500/15">
          {planItems.slice(0, 30).map((item) => (
            <div
              key={`${item.lineNo}-${item.deviceSn}-${item.order}`}
              className="grid grid-cols-[76px_160px_1fr] gap-2 border-b border-cyan-500/8 px-3 py-1.5 font-mono text-[11px] text-cyan-100/85 last:border-b-0"
            >
              <span>#{item.lineNo}/{item.order}</span>
              <span className="truncate text-cyan-300/75">{item.deviceSn}</span>
              <span className="truncate">{item.command.commandCode}</span>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function ScriptDetailDrawer({
  script,
  onClose,
  onExecute,
}: {
  script: MMLScript
  onClose: () => void
  onExecute: (script: MMLScript) => void
}) {
  const { data, isFetching } = useMMLScriptById(script.id)
  const detailScript = data ?? script
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-strong flex h-full w-full max-w-[640px] flex-col overflow-hidden border-l border-cyan-500/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="truncate font-display text-base font-bold text-cyan-100">{detailScript.scriptName}</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/50">{detailScript.id}</div>
          </div>
          <div className="flex items-center gap-2">
            <NeonButton icon={<Play />} onClick={() => onExecute(detailScript)}>
              EXEC
            </NeonButton>
            <button
              type="button"
              onClick={onClose}
              className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-100"
            >
              <X className="size-4" />
            </button>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b border-cyan-500/15 px-4 py-3">
          <div>
            <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">状态</div>
            <div className="mt-0.5">
              <StatusBadge status={scriptTone(detailScript.status)} label={detailScript.status} />
            </div>
          </div>
          <Meta label="类型" value={detailScript.type} />
          <Meta label="创建人" value={detailScript.creator || '—'} />
          <Meta label="进度" value={`${detailScript.progress ?? 0}%`} />
          <Meta label="创建时间" value={formatTime(detailScript.createTime)} />
          <Meta label="更新时间" value={formatTime(detailScript.updateTime)} />
        </div>

        {detailScript.description ? (
          <div className="border-b border-cyan-500/15 px-4 py-3">
            <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">描述</div>
            <div className="text-sm text-cyan-100/85">{detailScript.description}</div>
          </div>
        ) : null}

        {detailScript.tags && detailScript.tags.length > 0 ? (
          <div className="flex flex-wrap items-center gap-1.5 border-b border-cyan-500/15 px-4 py-3">
            <TagIcon className="size-3.5 text-cyan-300/45" />
            {detailScript.tags.map((tag) => (
              <span
                key={tag}
                className="rounded-sm border border-cyan-500/25 bg-cyan-500/8 px-2 py-0.5 font-mono text-[10px] text-cyan-300/80"
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-1.5 flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>脚本内容</span>
            {isFetching ? <span>LOADING…</span> : null}
          </div>
          {detailScript.content?.trim() ? (
            <pre className="overflow-auto whitespace-pre-wrap rounded-sm border border-emerald-500/20 bg-black/40 p-3 font-mono text-[12px] leading-relaxed text-emerald-300/90">
              {detailScript.content}
            </pre>
          ) : (
            <div className="rounded-sm border border-dashed border-cyan-500/20 bg-cyan-500/5 px-3 py-8 text-center font-mono text-xs uppercase tracking-[0.18em] text-cyan-300/55">
              暂无脚本内容
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function ScriptExecuteDialog({ script, onClose }: { script: MMLScript; onClose: () => void }) {
  const { data, isFetching } = useMMLScriptById(script.id)
  const detailScript = data ?? script
  const [taskName, setTaskName] = useState(`执行脚本: ${script.scriptName}`)
  const [deviceInput, setDeviceInput] = useState('')
  const [error, setError] = useState('')
  const createTask = useCreateMMLTask()
  const parsed = useMemo(
    () => parseMmlScriptPlan(detailScript.content ?? '', { format: 'auto' }),
    [detailScript.content]
  )
  const isDeviceBound = parsed.executeMode === 'device_bound'

  const submit = async () => {
    setError('')
    const deviceSns = isDeviceBound ? parsed.deviceSns : parseDeviceInput(deviceInput)
    if (!isDeviceBound && deviceSns.length === 0) {
      setError('请输入设备 SN')
      return
    }
    if (parsed.commands.length === 0) {
      setError('脚本内容为空')
      return
    }
    await createTask.mutateAsync({
      taskName: taskName.trim() || `执行脚本: ${detailScript.scriptName}`,
      scriptId: detailScript.id,
      deviceSns,
      commands: parsed.commands,
      executeMode: parsed.executeMode,
      planItems: isDeviceBound ? parsed.planItems : undefined,
      creator: '',
      executeType: 'immediate',
      offlineRetry: false,
      offlineRetryWait: 60,
      failedRetry: false,
      failedRetryCount: 3,
      failedRetryInterval: 5,
    })
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-strong flex max-h-[86vh] w-[min(760px,calc(100vw-32px))] flex-col overflow-hidden border border-cyan-500/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="font-display text-base font-bold text-cyan-100">EXEC SCRIPT</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/50">{detailScript.scriptName}</div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-100"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
          <label className="block">
            <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">任务名</span>
            <input className="neon-input w-full" value={taskName} onChange={(e) => setTaskName(e.target.value)} />
          </label>
          <div className="grid grid-cols-2 gap-3">
            <MiniStat label="MODE" value={isDeviceBound ? 'DEVICE-BOUND' : 'COMMON'} />
            <MiniStat
              label="PLAN"
              value={isDeviceBound ? `${parsed.planItems.length} ROWS / ${parsed.deviceSns.length} DEVICES` : `${parsed.commands.length} COMMANDS`}
            />
          </div>

          {!isDeviceBound ? (
            <label className="block">
              <span className="mb-1 block font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/55">设备 SN</span>
              <input
                className="neon-input w-full"
                value={deviceInput}
                placeholder="SN1,SN2"
                onChange={(e) => setDeviceInput(e.target.value)}
              />
            </label>
          ) : (
            <div className="border border-cyan-500/15 bg-cyan-500/4">
              <div className="border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                PLAN PREVIEW
              </div>
              <div className="max-h-56 overflow-auto">
                {parsed.planItems.slice(0, 20).map((item) => (
                  <div key={`${item.lineNo}-${item.deviceSn}-${item.order}`} className="grid grid-cols-[76px_160px_1fr] gap-2 border-b border-cyan-500/8 px-3 py-1.5 font-mono text-[11px] text-cyan-100/85 last:border-b-0">
                    <span>#{item.lineNo}/{item.order}</span>
                    <span className="truncate text-cyan-300/75">{item.deviceSn}</span>
                    <span className="truncate">{item.command.commandCode}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {error ? <div className="font-mono text-[11px] text-rose-300">{error}</div> : null}
          {isFetching ? <div className="font-mono text-[10px] text-cyan-300/55">LOADING SCRIPT…</div> : null}
        </div>

        <div className="flex justify-end gap-2 border-t border-cyan-500/20 px-4 py-3">
          <NeonButton onClick={onClose}>CANCEL</NeonButton>
          <NeonButton icon={createTask.isPending ? <Loader2 className="animate-spin" /> : <Play />} onClick={() => void submit()} disabled={createTask.isPending}>
            EXEC
          </NeonButton>
        </div>
      </div>
    </div>
  )
}

function parseDeviceInput(value: string): string[] {
  return Array.from(new Set(value.split(/[,\s;]+/).map((v) => v.trim()).filter(Boolean)))
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-cyan-500/15 bg-cyan-500/4 px-3 py-2">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</div>
      <div className="mt-0.5 truncate font-mono text-xs text-cyan-100/90">{value}</div>
    </div>
  )
}

function StatCard({
  label,
  value,
  color,
  trend,
}: {
  label: string
  value: number
  color: string
  trend?: number[]
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="truncate font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/60">{label}</div>
      <div className="flex items-end justify-between gap-1">
        <div className="font-display text-2xl font-bold leading-tight" style={{ color, textShadow: `0 0 8px ${color}` }}>
          {value}
        </div>
        {trend ? <Sparkline data={trend} color={color} width={54} height={22} /> : null}
      </div>
    </div>
  )
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</div>
      <div className="truncate text-xs text-cyan-100/90">{value}</div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function CenterState({
  children,
  tone = 'cyan',
}: {
  children: React.ReactNode
  tone?: 'cyan' | 'err'
}) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-2 py-14 font-mono text-xs uppercase tracking-[0.2em] ${
        tone === 'err' ? 'text-rose-300/80' : 'text-cyan-300/60'
      }`}
    >
      {children}
    </div>
  )
}

export default MMLScriptPage
