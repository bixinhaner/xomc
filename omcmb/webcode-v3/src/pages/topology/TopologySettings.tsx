import { useCallback, useState } from 'react'
import { Save, RotateCcw, Sliders, Check } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'

// 本页设置持久化到 localStorage，刷新后回读生效（不引入新依赖、不动共享 store）。
const STORAGE_KEY = 'starforge.topology.settings'

interface TopoSettings {
  layoutAlgorithm: 'force' | 'tree' | 'circular' | 'hierarchy'
  nodeSize: number
  nodeOpacity: number
  showNodeLabel: boolean
  edgeWidth: number
  edgeStyle: 'solid' | 'dashed' | 'dotted'
  showEdgeLabel: boolean
  edgeArrow: boolean
  showStatusBadge: boolean
  showDeviceType: boolean
  showAlarmCount: boolean
  enableAnimation: boolean
  autoRefreshInterval: number
}

const DEFAULTS: TopoSettings = {
  layoutAlgorithm: 'force',
  nodeSize: 24,
  nodeOpacity: 1,
  showNodeLabel: true,
  edgeWidth: 2,
  edgeStyle: 'solid',
  showEdgeLabel: false,
  edgeArrow: true,
  showStatusBadge: true,
  showDeviceType: true,
  showAlarmCount: true,
  enableAnimation: true,
  autoRefreshInterval: 30,
}

function load(): TopoSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULTS }
    const parsed = JSON.parse(raw) as Partial<TopoSettings>
    return { ...DEFAULTS, ...parsed }
  } catch {
    return { ...DEFAULTS }
  }
}

const LAYOUT_OPTS: { value: TopoSettings['layoutAlgorithm']; label: string }[] = [
  { value: 'force', label: 'FORCE · 力导向' },
  { value: 'tree', label: 'TREE · 树状' },
  { value: 'circular', label: 'RING · 环形' },
  { value: 'hierarchy', label: 'TIER · 分层' },
]
const EDGE_OPTS: { value: TopoSettings['edgeStyle']; label: string }[] = [
  { value: 'solid', label: 'SOLID · 实线' },
  { value: 'dashed', label: 'DASHED · 虚线' },
  { value: 'dotted', label: 'DOTTED · 点线' },
]
const REFRESH_OPTS: { value: number; label: string }[] = [
  { value: 0, label: '关闭' },
  { value: 10, label: '10s' },
  { value: 30, label: '30s' },
  { value: 60, label: '60s' },
  { value: 300, label: '5min' },
]

export default function TopologySettings() {
  const [settings, setSettings] = useState<TopoSettings>(load)
  const [saved, setSaved] = useState(false)

  const update = useCallback(<K extends keyof TopoSettings>(key: K, value: TopoSettings[K]) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
    setSaved(false)
  }, [])

  const handleSave = useCallback(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
      setSaved(true)
    } catch {
      setSaved(false)
    }
  }, [settings])

  const handleReset = useCallback(() => {
    setSettings({ ...DEFAULTS })
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(DEFAULTS))
    } catch {
      /* ignore */
    }
    setSaved(true)
  }, [])

  return (
    <PageShell
      code="F06"
      title="SETTINGS · 拓扑设置"
      subtitle="RENDER PROFILE · 布局 / 节点 / 链路 / 刷新"
      bare
      toolbar={
        <>
          <NeonButton icon={<RotateCcw />} onClick={handleReset}>
            RESET
          </NeonButton>
          <NeonButton icon={saved ? <Check /> : <Save />} onClick={handleSave}>
            {saved ? 'SAVED' : 'SAVE'}
          </NeonButton>
        </>
      }
    >
      {saved && (
        <div className="mb-3 flex items-center gap-2 border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.18em] text-emerald-300">
          <Check className="size-3.5" />
          配置已持久化 · 刷新后回读生效
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {/* 布局 */}
        <GlassPanel title="LAYOUT · 布局算法" meta="GRAPH" strong>
          <div className="space-y-3 p-4">
            <SegRow
              label="布局算法"
              options={LAYOUT_OPTS}
              value={settings.layoutAlgorithm}
              onChange={(v) => update('layoutAlgorithm', v)}
            />
            <SegRow
              label="自动刷新"
              options={REFRESH_OPTS}
              value={settings.autoRefreshInterval}
              onChange={(v) => update('autoRefreshInterval', v)}
            />
          </div>
        </GlassPanel>

        {/* 节点样式 */}
        <GlassPanel title="NODE · 节点样式" meta="STYLE" strong>
          <div className="space-y-3 p-4">
            <SliderRow
              label="节点尺寸"
              min={12}
              max={48}
              step={2}
              value={settings.nodeSize}
              suffix="px"
              onChange={(v) => update('nodeSize', v)}
            />
            <SliderRow
              label="节点不透明度"
              min={0.3}
              max={1}
              step={0.1}
              value={settings.nodeOpacity}
              suffix="×"
              onChange={(v) => update('nodeOpacity', v)}
            />
            <ToggleRow
              label="显示节点标签"
              value={settings.showNodeLabel}
              onChange={(v) => update('showNodeLabel', v)}
            />
          </div>
        </GlassPanel>

        {/* 链路样式 */}
        <GlassPanel title="EDGE · 链路样式" meta="STYLE" strong>
          <div className="space-y-3 p-4">
            <SliderRow
              label="链路宽度"
              min={1}
              max={6}
              step={1}
              value={settings.edgeWidth}
              suffix="px"
              onChange={(v) => update('edgeWidth', v)}
            />
            <SegRow
              label="链路线型"
              options={EDGE_OPTS}
              value={settings.edgeStyle}
              onChange={(v) => update('edgeStyle', v)}
            />
            <ToggleRow
              label="显示链路标签"
              value={settings.showEdgeLabel}
              onChange={(v) => update('showEdgeLabel', v)}
            />
            <ToggleRow
              label="显示箭头"
              value={settings.edgeArrow}
              onChange={(v) => update('edgeArrow', v)}
            />
          </div>
        </GlassPanel>

        {/* 显示选项 */}
        <GlassPanel title="DISPLAY · 显示选项" meta="OVERLAY" strong>
          <div className="space-y-3 p-4">
            <ToggleRow
              label="状态徽标"
              value={settings.showStatusBadge}
              onChange={(v) => update('showStatusBadge', v)}
            />
            <ToggleRow
              label="设备类型"
              value={settings.showDeviceType}
              onChange={(v) => update('showDeviceType', v)}
            />
            <ToggleRow
              label="告警计数"
              value={settings.showAlarmCount}
              onChange={(v) => update('showAlarmCount', v)}
            />
            <ToggleRow
              label="启用动画"
              value={settings.enableAnimation}
              onChange={(v) => update('enableAnimation', v)}
            />
          </div>
        </GlassPanel>
      </div>

      <div className="mt-3 flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/45">
        <Sliders className="size-3.5" />
        设置写入 localStorage：{STORAGE_KEY}
      </div>
    </PageShell>
  )
}

function SegRow<T extends string | number>({
  label,
  options,
  value,
  onChange,
}: {
  label: string
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-cyan-500/8 pb-3">
      <span className="font-mono text-[11px] uppercase tracking-[0.15em] text-cyan-300/70">
        {label}
      </span>
      <div className="flex flex-wrap gap-1.5">
        {options.map((o) => (
          <button
            key={String(o.value)}
            type="button"
            onClick={() => onChange(o.value)}
            className={`chip text-[10px] transition-all ${
              value === o.value
                ? 'text-cyan-200 shadow-[0_0_8px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-200'
            }`}
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  )
}

function SliderRow({
  label,
  min,
  max,
  step,
  value,
  suffix,
  onChange,
}: {
  label: string
  min: number
  max: number
  step: number
  value: number
  suffix?: string
  onChange: (v: number) => void
}) {
  return (
    <div className="border-b border-cyan-500/8 pb-3">
      <div className="mb-1.5 flex items-center justify-between">
        <span className="font-mono text-[11px] uppercase tracking-[0.15em] text-cyan-300/70">
          {label}
        </span>
        <span className="font-display text-sm font-bold text-cyan-200">
          {value}
          {suffix && <span className="ml-0.5 text-[10px] text-cyan-300/55">{suffix}</span>}
        </span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="h-1 w-full cursor-pointer appearance-none rounded-full bg-cyan-500/15 accent-cyan-400"
      />
    </div>
  )
}

function ToggleRow({
  label,
  value,
  onChange,
}: {
  label: string
  value: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between border-b border-cyan-500/8 pb-3">
      <span className="font-mono text-[11px] uppercase tracking-[0.15em] text-cyan-300/70">
        {label}
      </span>
      <button
        type="button"
        role="switch"
        aria-checked={value}
        onClick={() => onChange(!value)}
        className={`relative h-5 w-10 rounded-full transition-colors ${
          value ? 'bg-cyan-500/40' : 'bg-cyan-500/10'
        }`}
      >
        <span
          className="absolute top-0.5 size-4 rounded-full transition-all"
          style={{
            left: value ? 22 : 2,
            background: value ? '#00f0ff' : '#525a78',
            boxShadow: value ? '0 0 8px #00f0ff' : 'none',
          }}
        />
      </button>
    </div>
  )
}
