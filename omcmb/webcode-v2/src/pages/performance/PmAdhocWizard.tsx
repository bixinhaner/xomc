import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Save } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
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

import {
  usePmAdhocDetail,
  useCreatePmAdhoc,
  useUpdatePmAdhoc,
} from '@core/hooks/api/usePmAdhoc'
import type {
  AdhocDimension,
  AdhocMode,
  CreateAdhocTaskInput,
  UpdateAdhocTaskInput,
} from '@core/types/pmAdhoc'
import { isGranularityDimensionSupported } from '@core/utils/pmAdhocConstraints'

// ============================================================
// 自定义聚合任务 新建/编辑 — 对齐 v1 /performance/pm-adhoc/new 与 /:id/edit
//   - /new：建任务（CreateAdhocTaskInput）。
//   - /:id/edit：useParams 取 id → usePmAdhocDetail 真实加载回填 → UpdateAdhocTaskInput。
//   v2 无穿梭框/设备选择弹窗，设备 SN 与指标路径用逗号分隔输入。
// ============================================================

const MODE_OPTIONS: { label: string; value: AdhocMode }[] = [
  { label: '单次(oneshot)', value: 'oneshot' },
  { label: '持续(continuous)', value: 'continuous' },
]

const DIMENSION_OPTIONS: { label: string; value: AdhocDimension }[] = [
  { label: '按设备', value: 'device' },
  { label: '自选组聚合', value: 'aggregate_group' },
  { label: '按产品', value: 'product' },
  { label: '按频段', value: 'band' },
  { label: '全网汇总', value: 'network' },
  { label: '按设备组', value: 'device_group' },
]

const GRANULARITY_OPTIONS = ['15min', 'hourly', 'daily']
const TECH_OPTIONS: { label: string; value: string }[] = [
  { label: '不限', value: '' },
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
]

function splitCsv(s: string): string[] {
  return s
    .split(/[,\n]/)
    .map((x) => x.trim())
    .filter(Boolean)
}

export function PmAdhocWizardPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = Boolean(id)

  const detail = usePmAdhocDetail(id)
  const createTask = useCreatePmAdhoc()
  const updateTask = useUpdatePmAdhoc()

  // 表单状态
  const [name, setName] = useState('')
  const [mode, setMode] = useState<AdhocMode>('oneshot')
  const [dimension, setDimension] = useState<AdhocDimension>('device')
  const [technology, setTechnology] = useState('')
  const [granularity, setGranularity] = useState('15min')
  const [deviceSnsText, setDeviceSnsText] = useState('')
  const [metricPathsText, setMetricPathsText] = useState('')
  const [windowStart, setWindowStart] = useState('')
  const [windowEnd, setWindowEnd] = useState('')
  const [hydrated, setHydrated] = useState(false)

  // 编辑模式：详情到手回填一次
  const task = detail.data
  useEffect(() => {
    if (!isEdit || !task || hydrated) return
    setName(task.name)
    setMode(task.mode)
    setDimension(task.dimension)
    setTechnology(task.technology ?? '')
    setGranularity(task.granularities[0] ?? '15min')
    setDeviceSnsText(task.deviceSns.join(', '))
    setMetricPathsText(task.metricPaths.join(', '))
    setWindowStart(task.windowStart ? task.windowStart.slice(0, 16) : '')
    setWindowEnd(task.windowEnd ? task.windowEnd.slice(0, 16) : '')
    setHydrated(true)
  }, [isEdit, task, hydrated])

  const deviceSns = useMemo(() => splitCsv(deviceSnsText), [deviceSnsText])
  const metricPaths = useMemo(() => splitCsv(metricPathsText), [metricPathsText])

  const toIso = (local: string): string | undefined => {
    if (!local) return undefined
    const d = new Date(local)
    return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
  }

  // #363：编辑模式维度不可改，取任务原维度做组合校验；新建模式取表单维度。
  const effectiveDimension: AdhocDimension = isEdit ? (task?.dimension ?? dimension) : dimension
  // #363：(粒度, 维度) 组合守门——15min × 设备组不支持。
  const granDimSupported = isGranularityDimensionSupported(granularity, effectiveDimension)

  const canSubmit =
    name.trim().length > 0 &&
    metricPaths.length > 0 &&
    granDimSupported &&
    !createTask.isPending &&
    !updateTask.isPending

  const handleSubmit = () => {
    if (isEdit && id) {
      const input: UpdateAdhocTaskInput = {
        name: name.trim(),
        metricPaths,
        deviceSns,
        granularities: [granularity],
        windowStart: toIso(windowStart),
        windowEnd: toIso(windowEnd),
      }
      updateTask.mutate(
        { id, input },
        { onSuccess: () => navigate('/performance/pm-adhoc') }
      )
    } else {
      const input: CreateAdhocTaskInput = {
        name: name.trim(),
        mode,
        dimension,
        deviceSns,
        metricPaths,
        granularities: [granularity],
        technology: technology || undefined,
        windowStart: mode === 'oneshot' ? toIso(windowStart) : undefined,
        windowEnd: mode === 'oneshot' ? toIso(windowEnd) : undefined,
      }
      createTask.mutate(input, { onSuccess: () => navigate('/performance/pm-adhoc') })
    }
  }

  const submitErr = createTask.error ?? updateTask.error

  // 编辑模式且加载中/失败的早退处理
  if (isEdit && detail.isLoading) {
    return (
      <PageShell title="编辑聚合任务" description="加载任务详情…">
        <Card className="p-6 text-sm text-muted-foreground">加载中…</Card>
      </PageShell>
    )
  }
  if (isEdit && detail.isError) {
    return (
      <PageShell title="编辑聚合任务" description="加载失败">
        <Card className="p-6 text-sm text-destructive">
          加载失败：{detail.error instanceof Error ? detail.error.message : '未知错误'}
        </Card>
      </PageShell>
    )
  }
  if (isEdit && !detail.isLoading && !task) {
    return (
      <PageShell title="编辑聚合任务" description="未找到任务">
        <Card className="p-6 text-sm text-muted-foreground">未找到该任务（ID: {id}）</Card>
      </PageShell>
    )
  }

  return (
    <PageShell
      title={isEdit ? '编辑聚合任务' : '新建聚合任务'}
      description={isEdit ? `任务 ID ${id}` : '配置 PM 自定义聚合任务的设备、指标与维度'}
      toolbar={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate('/performance/pm-adhoc')}>
            <ArrowLeft className="size-4" /> 返回
          </Button>
          <Button size="sm" disabled={!canSubmit} onClick={handleSubmit}>
            <Save className="size-4" /> {isEdit ? '保存修改' : '创建任务'}
          </Button>
        </div>
      }
    >
      <Card className="max-w-3xl space-y-5 p-6">
        {isEdit && task?.isBuiltin && (
          <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300">
            内置任务仅可修改指标集，模式 / 维度 / 制式不可改。
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="adhoc-name">任务名称 *</Label>
          <Input
            id="adhoc-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="例：核心区 LTE RRC 成功率日聚合"
          />
        </div>

        {!isEdit && (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div className="flex flex-col gap-1.5">
              <Label>模式</Label>
              <Select value={mode} onValueChange={(v) => setMode(v as AdhocMode)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODE_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>聚合维度</Label>
              <Select
                value={dimension}
                onValueChange={(v) => {
                  const next = v as AdhocDimension
                  setDimension(next)
                  // #363：切到设备组若当前粒度 15min（不支持）→ 自动回落 hourly。
                  if (!isGranularityDimensionSupported(granularity, next)) {
                    setGranularity('hourly')
                  }
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {DIMENSION_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>制式</Label>
              <Select
                value={technology || 'any'}
                onValueChange={(v) => setTechnology(v === 'any' ? '' : v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TECH_OPTIONS.map((o) => (
                    <SelectItem key={o.value || 'any'} value={o.value || 'any'}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        )}

        {isEdit && task && (
          <div className="flex flex-wrap gap-1.5 text-xs">
            <Badge variant="outline">模式 {task.mode === 'continuous' ? '持续' : '单次'}</Badge>
            <Badge variant="outline">
              维度 {DIMENSION_OPTIONS.find((d) => d.value === task.dimension)?.label ?? task.dimension}
            </Badge>
            <Badge variant="outline">制式 {task.technology || '不限'}</Badge>
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <Label>粒度</Label>
          <Select value={granularity} onValueChange={setGranularity}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {GRANULARITY_OPTIONS.map((g) => {
                // #363：设备组维度不支持 15min 粒度，禁用该选项。
                const disabled = !isGranularityDimensionSupported(g, effectiveDimension)
                return (
                  <SelectItem key={g} value={g} disabled={disabled}>
                    {disabled ? `${g}（设备组不支持）` : g}
                  </SelectItem>
                )
              })}
            </SelectContent>
          </Select>
          {!granDimSupported && (
            <span className="text-xs text-destructive">
              设备组维度最细为小时，不支持 15 分钟粒度，请改用小时及以上。
            </span>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="adhoc-devices">设备 SN（逗号或换行分隔）</Label>
          <Input
            id="adhoc-devices"
            value={deviceSnsText}
            onChange={(e) => setDeviceSnsText(e.target.value)}
            placeholder="ENB00001, ENB00002"
          />
          <span className="text-xs text-muted-foreground">已识别 {deviceSns.length} 个设备</span>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="adhoc-metrics">指标路径 *（逗号或换行分隔）</Label>
          <Input
            id="adhoc-metrics"
            value={metricPathsText}
            onChange={(e) => setMetricPathsText(e.target.value)}
            placeholder="K1001, K1002"
          />
          <span className="text-xs text-muted-foreground">已识别 {metricPaths.length} 个指标</span>
        </div>

        {(!isEdit ? mode === 'oneshot' : true) && (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="adhoc-start">窗口起（本地时间）</Label>
              <Input
                id="adhoc-start"
                type="datetime-local"
                value={windowStart}
                onChange={(e) => setWindowStart(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="adhoc-end">窗口止（本地时间）</Label>
              <Input
                id="adhoc-end"
                type="datetime-local"
                value={windowEnd}
                onChange={(e) => setWindowEnd(e.target.value)}
              />
            </div>
          </div>
        )}

        {submitErr && (
          <div className="text-sm text-destructive">
            提交失败：{submitErr instanceof Error ? submitErr.message : '未知错误'}
          </div>
        )}
      </Card>
    </PageShell>
  )
}

export default PmAdhocWizardPage
