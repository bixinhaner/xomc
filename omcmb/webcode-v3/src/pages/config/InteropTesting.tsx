import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  FlaskConical,
  Play,
  Loader2,
  CheckCircle2,
  XCircle,
  ListChecks,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useTestCases, useRunTests } from '@core/hooks/api/useInterop'
import type { TestCase, RunTestsResponse } from '@core/services/api/interopApi'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

// ===========================================================================
// CONFIG · 互操作测试
// 真实测试用例目录（useTestCases，按 category 分组）+ 选设备 →
// useRunTests 执行 → 结果（passed/failed + 每条用例详情）回显。
// ===========================================================================
export default function InteropTesting() {
  const casesQ = useTestCases()
  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const run = useRunTests()

  const grouped: Record<string, TestCase[]> = casesQ.data ?? {}
  const categories = Object.keys(grouped)
  const devices = devicesQ.data?.items ?? []

  const [category, setCategory] = useState('')
  const [deviceSn, setDeviceSn] = useState('')
  const [result, setResult] = useState<RunTestsResponse | null>(null)
  const [err, setErr] = useState('')

  const effectiveCategory = category && grouped[category] ? category : categories[0] || ''
  const effectiveSn = deviceSn || devices[0]?.sn || ''
  const cases = effectiveCategory ? grouped[effectiveCategory] ?? [] : []

  const totalCases = useMemo(
    () => Object.values(grouped).reduce((s, arr) => s + arr.length, 0),
    [grouped],
  )

  const runTests = () => {
    if (!effectiveSn) return
    setErr('')
    setResult(null)
    run.mutate(
      { deviceSn: effectiveSn, categories: effectiveCategory ? [effectiveCategory] : undefined },
      {
        onSuccess: (r) => setResult(r),
        onError: (e: Error) => setErr(e.message || '测试执行失败'),
      },
    )
  }

  const passRate = result && result.total > 0 ? Math.round((result.passed / result.total) * 100) : 0

  return (
    <PageShell
      code="F10"
      title="INTEROP TEST · 互操作测试"
      subtitle="TR-069 CONFORMANCE SUITE"
      isFetching={casesQ.isFetching || devicesQ.isFetching}
      bare
      toolbar={
        <>
          <select
            className="neon-input w-48"
            value={effectiveSn}
            onChange={(e) => setDeviceSn(e.target.value)}
          >
            {devices.length === 0 ? <option value="">无设备</option> : null}
            {devices.map((d) => (
              <option key={d.id} value={d.sn}>
                {d.sn}
              </option>
            ))}
          </select>
          <NeonButton
            icon={run.isPending ? <Loader2 className="animate-spin" /> : <Play />}
            onClick={runTests}
            disabled={!effectiveSn || run.isPending}
          >
            运行测试
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => void casesQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="TEST CASES · 用例" value={totalCases} color="#00f0ff" />
        <StatCard label="CATEGORIES · 分类" value={categories.length} color="#a855f7" />
        <StatCard label="PASSED · 通过" value={result?.passed ?? 0} color="#00ff88" />
        <StatCard
          label="FAILED · 失败"
          value={result?.failed ?? 0}
          color={result && result.failed > 0 ? '#ff2d6f' : '#525a78'}
        />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {categories.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => setCategory(c)}
            className={`chip ${c === effectiveCategory ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            <FlaskConical className="size-3" />
            {c} · {grouped[c].length}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-[1fr_420px] gap-3">
        {/* 测试用例目录 */}
        <GlassPanel
          title={effectiveCategory ? `CASES · ${effectiveCategory}` : 'CASES · 测试用例'}
          meta={`${cases.length}`}
          className="min-h-0"
        >
          {casesQ.isLoading ? (
            <HudLoading />
          ) : casesQ.isError ? (
            <HudError error={casesQ.error} />
          ) : cases.length === 0 ? (
            <HudEmpty icon={ListChecks} text="无测试用例" />
          ) : (
            <div className="max-h-[54vh] space-y-1.5 overflow-auto p-2">
              {cases.map((tc) => {
                const r = result?.results.find((x) => x.testCaseId === tc.id)
                return (
                  <div
                    key={tc.id}
                    className="rounded-sm border px-3 py-2"
                    style={{
                      borderColor: r
                        ? r.passed
                          ? 'rgba(0,255,136,0.35)'
                          : 'rgba(255,45,111,0.4)'
                        : 'rgba(0,240,255,0.1)',
                    }}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="min-w-0 flex-1 truncate text-xs text-cyan-100/90">{tc.name}</span>
                      {r ? (
                        r.passed ? (
                          <StatusBadge status="ok" label={`PASS ${r.durationMs}ms`} />
                        ) : (
                          <StatusBadge status="critical" label="FAIL" />
                        )
                      ) : (
                        <span className="font-mono text-[10px] text-cyan-300/40">{tc.steps.length} 步</span>
                      )}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/50">{tc.description || tc.id}</div>
                    {r && !r.passed && r.error ? (
                      <div className="mt-1 truncate font-mono text-[10px] text-rose-300/80">{r.error}</div>
                    ) : null}
                  </div>
                )
              })}
            </div>
          )}
        </GlassPanel>

        {/* 结果面板 */}
        <GlassPanel title="RESULT · 测试结果" meta={result ? result.deviceSn : undefined} className="min-h-0">
          <div className="p-4">
            {err ? (
              <div className="mb-3 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                {err}
              </div>
            ) : null}
            {run.isPending ? (
              <HudLoading text="RUNNING TESTS…" />
            ) : result ? (
              <>
                <div className="mb-4 flex items-center gap-4">
                  <RadialGauge
                    value={passRate}
                    label="通过率"
                    size={110}
                    color={result.failed > 0 ? '#ffaa00' : '#00ff88'}
                    unit="%"
                  />
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-2 font-mono text-sm">
                      <CheckCircle2 className="size-4 text-emerald-400" />
                      <span className="text-emerald-300">{result.passed}</span>
                      <span className="text-cyan-300/40">通过</span>
                    </div>
                    <div className="flex items-center gap-2 font-mono text-sm">
                      <XCircle className="size-4 text-rose-400" />
                      <span className="text-rose-300">{result.failed}</span>
                      <span className="text-cyan-300/40">失败</span>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/55">共 {result.total} 项</div>
                  </div>
                </div>
                <div className="max-h-[34vh] space-y-1 overflow-auto">
                  {result.results.map((r) => (
                    <div
                      key={r.testCaseId}
                      className="flex items-center gap-2 rounded-sm border border-cyan-500/10 px-3 py-1.5"
                    >
                      {r.passed ? (
                        <CheckCircle2 className="size-3.5 shrink-0 text-emerald-400" />
                      ) : (
                        <XCircle className="size-3.5 shrink-0 text-rose-400" />
                      )}
                      <span className="min-w-0 flex-1 truncate text-[11px] text-cyan-100/85">{r.testName}</span>
                      <span className="shrink-0 font-mono text-[10px] text-cyan-300/45">{r.durationMs}ms</span>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <HudEmpty icon={FlaskConical} text="选设备并运行测试查看结果" />
            )}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
  )
}
