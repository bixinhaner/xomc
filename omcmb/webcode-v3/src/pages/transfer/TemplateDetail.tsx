import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2, ShieldCheck, Wrench, ChevronRight, Cpu } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import { useUnifiedFileTransferTaskTypes } from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferTaskType,
  TransferStepId,
} from '@core/types/unifiedFileTransfer'

// 步骤链中文标签 —— 与后端 TransferStepId 联合类型一一对应（穷尽）。
const STEP_LABEL: Record<TransferStepId, string> = {
  CHECK_PERMISSION: '权限校验',
  CHECK_ONLINE: '在线检查',
  CHECK_CONFLICT: '冲突检查',
  PRE_VALIDATE: '前置校验',
  SEND_RPC: '下发 RPC',
  WAIT_RPC_RESPONSE: '等待 RPC 响应',
  WAIT_FILE_TRANSFER: '等待文件传输',
  WAIT_TRANSFER_COMPLETE: '等待传输完成',
  WAIT_INFORM_EVENT: '等待 Inform 事件',
  WAIT_REBOOT_COMPLETE: '等待重启完成',
}

export default function TransferTemplateDetailPage() {
  const { typeCode = '' } = useParams<{ typeCode: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching } = useUnifiedFileTransferTaskTypes({
    refetchOnMount: 'always',
  })
  const type = useMemo<UnifiedFileTransferTaskType | undefined>(
    () => (data ?? []).find((t) => t.typeCode === typeCode),
    [data, typeCode],
  )

  const back = (
    <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/transfer/template-management')}>
      BACK
    </NeonButton>
  )

  if (isLoading) {
    return (
      <PageShell code="F06" title="TEMPLATE · 模板详情" subtitle="LOADING" toolbar={back}>
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      </PageShell>
    )
  }

  if (isError || !type) {
    return (
      <PageShell code="F06" title="TEMPLATE · 模板详情" subtitle="ERROR" toolbar={back}>
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          {isError
            ? `FAILURE · ${error instanceof Error ? error.message : '未知错误'}`
            : 'NOT FOUND · 模板类型不存在'}
        </div>
      </PageShell>
    )
  }

  // #492：优先展示「适用产品」（产品英文名），空则回退旧平台范围。
  const scopeItems = (type.products ?? []).length > 0 ? (type.products ?? []) : type.platformScope

  return (
    <PageShell
      code="F06"
      title={`TEMPLATE · ${type.displayName}`}
      subtitle={`${type.category} · ${type.rpcType}`}
      isFetching={isFetching}
      toolbar={back}
    >
      {/* 摘要带 */}
      <div className="mb-4 grid grid-cols-12 gap-3">
        <GlassPanel title="STATS · 30D 统计" className="col-span-12 lg:col-span-4">
          <div className="flex items-center justify-around gap-2 px-3 py-4">
            <RadialGauge value={type.successRate30d} label="成功率" size={108} color="#00ff88" />
            <div className="flex flex-col gap-3">
              <Stat label="30D 任务" value={`${type.taskCount30d}`} color="#00f0ff" />
              <Stat label="步骤数" value={`${type.stepChain.length}`} color="#5b9eff" />
              <Stat label="适用产品" value={`${scopeItems.length}`} color="#a855f7" />
            </div>
          </div>
        </GlassPanel>

        <GlassPanel title="DEFINITION · 模板定义" meta="REGISTRY" className="col-span-12 lg:col-span-8">
          <dl className="grid grid-cols-1 sm:grid-cols-2">
            <Field
              label="来源"
              valueNode={
                <span className="flex items-center justify-end gap-1.5">
                  {type.builtIn ? (
                    <ShieldCheck className="size-3.5 text-cyan-300/70" />
                  ) : (
                    <Wrench className="size-3.5 text-fuchsia-300/70" />
                  )}
                  <StatusBadge
                    status={type.builtIn ? 'active' : 'ok'}
                    label={type.builtIn ? 'BUILTIN' : 'CUSTOM'}
                  />
                </span>
              }
            />
            <Field
              label="启用状态"
              valueNode={
                <StatusBadge
                  status={type.enabled ? 'online' : 'offline'}
                  label={type.enabled ? '已启用' : '已停用'}
                />
              }
            />
            <Field label="类型编码" value={type.typeCode} />
            <Field label="业务分类" value={type.categoryLabel} />
            <Field label="RPC 类型" value={type.rpcType} />
            <Field label="权限码" value={type.permissionCode} />
            <Field label="文件类型" value={type.fileTypeLabel || type.fileType} />
            <Field
              label="文件类型可编辑"
              valueNode={
                <StatusBadge status={type.fileTypeEditable ? 'ok' : 'offline'} label={type.fileTypeEditable ? '是' : '否'} />
              }
            />
            {type.postTcEventCode && <Field label="TC 后事件码" value={type.postTcEventCode} />}
            {type.delaySeconds !== undefined && <Field label="延迟(秒)" value={`${type.delaySeconds}`} />}
            <Field label="最近编辑" value={type.lastEditor} />
            <Field label="更新时间" value={formatTime(type.updatedAt)} />
          </dl>
          {type.description && (
            <div className="border-t border-cyan-500/10 px-3.5 py-3">
              <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
                DESCRIPTION · 描述
              </div>
              <p className="text-xs leading-relaxed text-cyan-100/80">{type.description}</p>
            </div>
          )}
        </GlassPanel>
      </div>

      {/* 步骤链流水线 */}
      <GlassPanel title="STEP CHAIN · 执行步骤链" meta={`${type.stepChain.length} STEPS`} className="mb-4">
        {type.stepChain.length === 0 ? (
          <div className="px-4 py-6 text-center font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/45">
            NO STEPS · 未配置步骤链
          </div>
        ) : (
          <div className="flex flex-wrap items-center gap-2 p-3.5">
            {type.stepChain.map((step, i) => (
              <div key={`${step}-${i}`} className="flex items-center gap-2">
                <div className="flex items-center gap-2 rounded-sm border border-cyan-500/25 bg-cyan-500/[0.04] px-3 py-1.5">
                  <span className="font-display text-xs font-bold text-cyan-300/60">{i + 1}</span>
                  <div>
                    <div className="font-display text-xs font-bold text-cyan-100">{STEP_LABEL[step]}</div>
                    <div className="font-mono text-[9px] uppercase tracking-[0.1em] text-cyan-300/45">{step}</div>
                  </div>
                </div>
                {i < type.stepChain.length - 1 && (
                  <ChevronRight className="size-4 shrink-0 text-cyan-300/40" />
                )}
              </div>
            ))}
          </div>
        )}
      </GlassPanel>

      {/* 文件模板 + 平台范围 */}
      <div className="grid grid-cols-12 gap-4">
        <GlassPanel title="FILE TEMPLATES · 文件模板" className="col-span-12 lg:col-span-7">
          <dl className="divide-y divide-cyan-500/8">
            <Field label="URL 模板" value={type.urlTemplate} mono />
            <Field label="目标文件名模板" value={type.targetFileNameTemplate} mono />
            <Field label="文件名模板" value={type.fileNameTemplate} mono />
            <Field label="文件大小字段" value={type.fileSizeField} mono />
            <Field label="校验字段" value={type.checksumField} mono />
            <Field label="传输路径" value={type.transportPath} mono />
            <Field label="原始模式" value={type.rawMode} mono />
            {type.firmwareFileType !== undefined && (
              <Field label="固件库文件类型" value={`${type.firmwareFileType}`} mono />
            )}
          </dl>
        </GlassPanel>

        <GlassPanel
          title="PRODUCTS · 适用产品"
          meta={`${scopeItems.length} ITEMS`}
          className="col-span-12 lg:col-span-5"
        >
          {scopeItems.length === 0 ? (
            <div className="flex items-center gap-2 px-4 py-6 font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
              <Cpu className="size-3.5" />
              ALL PRODUCTS · 不限产品
            </div>
          ) : (
            <div className="flex flex-wrap gap-2 p-3.5">
              {scopeItems.map((p) => (
                <span key={p} className="chip text-cyan-200">
                  {p}
                </span>
              ))}
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function Stat({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <div
        className="font-display text-lg font-bold leading-none"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-300/55">{label}</div>
    </div>
  )
}

function Field({
  label,
  value,
  valueNode,
  mono,
}: {
  label: string
  value?: string
  valueNode?: React.ReactNode
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-3.5 py-2">
      <dt className="shrink-0 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </dt>
      <dd
        className={`min-w-0 truncate text-right text-xs text-cyan-100/90 ${mono ? 'font-mono text-[11px]' : ''}`}
      >
        {valueNode ?? (value && value.trim() ? value : <span className="text-cyan-300/40">—</span>)}
      </dd>
    </div>
  )
}
