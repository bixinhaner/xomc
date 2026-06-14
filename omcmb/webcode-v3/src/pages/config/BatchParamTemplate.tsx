import { useState } from 'react'
import {
  Search,
  RefreshCcw,
  FileStack,
  Send,
  Trash2,
  X,
  Loader2,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useConfigTemplates, useDeleteConfigTemplates } from '@core/hooks/api/useConfig'
import { useDispatchTemplate } from '@core/hooks/api/useTemplate'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigTemplate } from '@core/types/config'
import type { DispatchTemplateResponse } from '@core/services/api/templateApi'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// ===========================================================================
// CONFIG · 按模板批量配置
// 真实模板库（useConfigTemplates）→ 选模板 → 多选设备 → 批量下发（dispatch）
// ===========================================================================
export default function BatchParamTemplate() {
  const [keyword, setKeyword] = useState('')
  const [dispatchTarget, setDispatchTarget] = useState<ConfigTemplate | null>(null)
  const [selectedIds, setSelectedIds] = useState<string[]>([])

  const templatesQ = useConfigTemplates({ page: 1, pageSize: 100 })
  const del = useDeleteConfigTemplates()
  const templates = templatesQ.data?.items ?? []

  const kw = keyword.trim().toLowerCase()
  const rows = kw
    ? templates.filter(
        (t) =>
          t.templateName?.toLowerCase().includes(kw) ||
          t.description?.toLowerCase().includes(kw) ||
          t.creator?.toLowerCase().includes(kw),
      )
    : templates

  const totalParams = templates.reduce((s, t) => s + (t.params?.length ?? 0), 0)

  const toggle = (id: string) =>
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))

  const removeSelected = () => {
    if (selectedIds.length === 0) return
    del.mutate(selectedIds, { onSuccess: () => setSelectedIds([]) })
  }

  return (
    <PageShell
      code="F02"
      title="BATCH BY TEMPLATE · 按模板批量"
      subtitle="TEMPLATE LIBRARY / DISPATCH"
      isFetching={templatesQ.isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="名称 / 描述 / 创建者"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {selectedIds.length > 0 ? (
            <NeonButton
              icon={del.isPending ? <Loader2 className="animate-spin" /> : <Trash2 />}
              onClick={removeSelected}
              disabled={del.isPending}
            >
              删除 {selectedIds.length}
            </NeonButton>
          ) : null}
          <NeonButton icon={<RefreshCcw />} onClick={() => void templatesQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="TEMPLATES · 模板" value={templatesQ.data?.total ?? templates.length} color="#00f0ff" />
        <StatCard label="TOTAL PARAMS · 参数总数" value={totalParams} color="#a855f7" />
        <StatCard label="SELECTED · 已选" value={selectedIds.length} color="#ffaa00" />
      </div>

      {templatesQ.isLoading ? (
        <HudLoading />
      ) : templatesQ.isError ? (
        <HudError error={templatesQ.error} />
      ) : rows.length === 0 ? (
        <HudEmpty icon={FileStack} text="无模板" />
      ) : (
        <div className="space-y-1.5">
          {rows.map((t) => {
            const checked = selectedIds.includes(t.id)
            return (
              <div
                key={t.id}
                className="fleet-row grid grid-cols-[28px_2fr_1.2fr_90px_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: checked ? '#ffaa00' : '#00f0ff' }}
              >
                <button
                  type="button"
                  onClick={() => toggle(t.id)}
                  className={`flex size-4 items-center justify-center rounded-sm border ${
                    checked ? 'border-amber-400 bg-amber-400/20' : 'border-cyan-500/30'
                  }`}
                >
                  {checked ? <CheckCircle2 className="size-3 text-amber-300" /> : null}
                </button>
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {t.templateName || '—'}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {t.description || '无描述'}
                  </div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  <span className="text-cyan-300/45">CREATOR </span>
                  {t.creator || '—'}
                </div>
                <div className="text-center">
                  <span className="font-display text-lg font-bold text-cyan-200" style={{ textShadow: '0 0 6px #00f0ff' }}>
                    {t.params?.length ?? 0}
                  </span>
                  <div className="font-mono text-[9px] uppercase tracking-[0.15em] text-cyan-300/45">PARAMS</div>
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(t.createTime)}</div>
                <div className="text-right">
                  <NeonButton icon={<Send />} onClick={() => setDispatchTarget(t)}>
                    下发
                  </NeonButton>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {dispatchTarget ? (
        <DispatchModal template={dispatchTarget} onClose={() => setDispatchTarget(null)} />
      ) : null}
    </PageShell>
  )
}

function DispatchModal({ template, onClose }: { template: ConfigTemplate; onClose: () => void }) {
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [deviceKw, setDeviceKw] = useState('')
  const [result, setResult] = useState<DispatchTemplateResponse | null>(null)
  const [errMsg, setErrMsg] = useState('')

  const dispatch = useDispatchTemplate()
  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const allDevices = devicesQ.data?.items ?? []
  const kw = deviceKw.trim().toLowerCase()
  const devices = kw
    ? allDevices.filter((d) => d.sn?.toLowerCase().includes(kw) || d.name?.toLowerCase().includes(kw))
    : allDevices

  const toggle = (id: string) =>
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))

  const submit = () => {
    if (selectedIds.length === 0) return
    setErrMsg('')
    dispatch.mutate(
      { templateId: template.id, deviceIds: selectedIds },
      {
        onSuccess: (resp) => setResult(resp),
        onError: (err: Error) => setErrMsg(err.message || '下发失败'),
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02030a]/75 backdrop-blur-sm">
      <div className="w-[680px] max-w-[92vw]">
        <GlassPanel strong title={`DISPATCH · ${template.templateName}`} meta={`${template.params?.length ?? 0} PARAMS`}>
          <div className="max-h-[72vh] overflow-auto p-4">
            {result ? (
              <DispatchResult result={result} />
            ) : (
              <>
                <div className="relative mb-3">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
                  <input
                    className="neon-input w-full pl-9"
                    placeholder="按 SN / 名称筛选设备"
                    value={deviceKw}
                    onChange={(e) => setDeviceKw(e.target.value)}
                  />
                </div>
                {devicesQ.isLoading ? (
                  <HudLoading text="LOADING DEVICES…" />
                ) : devices.length === 0 ? (
                  <HudEmpty text="无可选设备" />
                ) : (
                  <div className="max-h-[40vh] space-y-1 overflow-auto">
                    {devices.map((d) => {
                      const on = selectedIds.includes(d.id)
                      return (
                        <button
                          key={d.id}
                          type="button"
                          onClick={() => toggle(d.id)}
                          className={`flex w-full items-center gap-2 rounded-sm border px-3 py-2 text-left transition-colors ${
                            on ? 'border-cyan-400/50 bg-cyan-500/12' : 'border-cyan-500/10 hover:bg-cyan-500/6'
                          }`}
                        >
                          <span
                            className={`flex size-4 shrink-0 items-center justify-center rounded-sm border ${
                              on ? 'border-cyan-400 bg-cyan-400/20' : 'border-cyan-500/30'
                            }`}
                          >
                            {on ? <CheckCircle2 className="size-3 text-cyan-300" /> : null}
                          </span>
                          <span className="min-w-0 flex-1">
                            <span className="block truncate font-mono text-xs text-cyan-100">{d.sn}</span>
                            {d.name && d.name !== d.sn ? (
                              <span className="block truncate text-[10px] text-cyan-300/55">{d.name}</span>
                            ) : null}
                          </span>
                          {d.productClass ? <span className="chip text-[#a855f7]">{d.productClass}</span> : null}
                        </button>
                      )
                    })}
                  </div>
                )}
                {errMsg ? (
                  <div className="mt-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                    {errMsg}
                  </div>
                ) : null}
              </>
            )}
          </div>
          <div className="flex items-center justify-between border-t border-cyan-500/15 px-4 py-3">
            <span className="font-mono text-[11px] text-cyan-300/55">
              {result ? 'RESULT' : `SELECTED ${selectedIds.length}`}
            </span>
            <div className="flex gap-2">
              <NeonButton icon={<X />} onClick={onClose}>
                {result ? '关闭' : '取消'}
              </NeonButton>
              {!result ? (
                <NeonButton
                  icon={dispatch.isPending ? <Loader2 className="animate-spin" /> : <Send />}
                  onClick={submit}
                  disabled={selectedIds.length === 0 || dispatch.isPending}
                >
                  下发到 {selectedIds.length} 台
                </NeonButton>
              ) : null}
            </div>
          </div>
        </GlassPanel>
      </div>
    </div>
  )
}

function DispatchResult({ result }: { result: DispatchTemplateResponse }) {
  const rows = [
    ...result.dispatched.map((r) => ({ ...r, ok: true, key: `s-${r.deviceId}` })),
    ...result.failed.map((r) => ({ ...r, ok: false, key: `f-${r.deviceId}` })),
  ]
  return (
    <div>
      <div className="mb-3 flex gap-2">
        <span className="chip text-[#00f0ff]">总计 {result.totalDevices}</span>
        <span className="chip text-[#00ff88]">成功 {result.dispatched.length}</span>
        <span className={`chip ${result.failed.length > 0 ? 'text-[#ff2d6f]' : 'text-[#525a78]'}`}>
          失败 {result.failed.length}
        </span>
      </div>
      <div className="space-y-1">
        {rows.map((r) => (
          <div
            key={r.key}
            className="grid grid-cols-[28px_1fr_1.4fr] items-center gap-2 rounded-sm border border-cyan-500/10 px-3 py-2"
          >
            {r.ok ? (
              <CheckCircle2 className="size-4 text-emerald-400" />
            ) : (
              <XCircle className="size-4 text-rose-400" />
            )}
            <code className="truncate font-mono text-[11px] text-cyan-100/85">{r.deviceId}</code>
            <span className="truncate font-mono text-[11px]">
              {r.ok ? (
                <span className="text-cyan-300/70">TASK {r.taskId ?? '—'}</span>
              ) : (
                <span className="text-rose-300">{r.error ?? '失败'}</span>
              )}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
