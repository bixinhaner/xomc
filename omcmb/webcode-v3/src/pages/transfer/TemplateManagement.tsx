import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  RefreshCcw,
  Loader2,
  ScrollText,
  ChevronRight,
  Power,
  PowerOff,
  Trash2,
  ShieldCheck,
  Wrench,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import {
  useUnifiedFileTransferTaskTypes,
  useUpdateUnifiedFileTransferTaskType,
  useDeleteUnifiedFileTransferTaskType,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferTaskType,
  TransferRpcType,
} from '@core/types/unifiedFileTransfer'
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  filterTaskTypesForCategory,
} from '@core/utils/ufteCategory'

const RPC_BADGE: Record<TransferRpcType, { status: string; label: string }> = {
  DOWNLOAD: { status: 'active', label: 'DOWNLOAD' },
  UPLOAD: { status: 'ok', label: 'UPLOAD' },
  SET_PARAM_VALUES: { status: 'warning', label: 'SET_PARAM' },
}

export default function TransferTemplateManagementPage() {
  const navigate = useNavigate()
  const [category, setCategory] = useState<string>('')
  const [opError, setOpError] = useState<string | null>(null)
  const [busyCode, setBusyCode] = useState<string | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = useUnifiedFileTransferTaskTypes({
    refetchOnMount: 'always',
  })
  const taskTypes = useMemo<UnifiedFileTransferTaskType[]>(() => data ?? [], [data])

  // #483：4G/5G/2G 折叠为单条『设备升级』(device_upgrade)，与任务创建页一致
  // （共享 @core/utils/ufteCategory，三皮肤同一口径）。count 经 filterTaskTypesForCategory
  // 计算——device_upgrade 取成员（4G/5G/2G）合计，其它分类取精确等值。
  const categories = useMemo(
    () =>
      aggregateCategoryOptions(taskTypes).map((opt) => ({
        value: opt.value,
        label: opt.value === DEVICE_UPGRADE_CATEGORY ? '设备升级' : opt.label,
        count: filterTaskTypesForCategory(taskTypes, opt.value).length,
      })),
    [taskTypes],
  )

  const visible = useMemo(
    () => (category ? filterTaskTypesForCategory(taskTypes, category) : taskTypes),
    [taskTypes, category],
  )

  const updateType = useUpdateUnifiedFileTransferTaskType()
  const deleteType = useDeleteUnifiedFileTransferTaskType()

  const toggleEnabled = async (t: UnifiedFileTransferTaskType) => {
    setOpError(null)
    setBusyCode(t.typeCode)
    try {
      await updateType.mutateAsync({
        typeCode: t.typeCode,
        category: t.category,
        categoryLabel: t.categoryLabel,
        displayName: t.displayName,
        description: t.description,
        rpcType: t.rpcType,
        stepChain: t.stepChain,
        postTcEventCode: t.postTcEventCode,
        enabled: !t.enabled,
        platformScope: t.platformScope,
        fileType: t.fileType,
        fileTypeLabel: t.fileTypeLabel,
        fileTypeEditable: t.fileTypeEditable,
        firmwareFileType: t.firmwareFileType,
        urlTemplate: t.urlTemplate,
        targetFileNameTemplate: t.targetFileNameTemplate,
        fileNameTemplate: t.fileNameTemplate,
        fileSizeField: t.fileSizeField,
        checksumField: t.checksumField,
        rawMode: t.rawMode,
        delaySeconds: t.delaySeconds,
        transportPath: t.transportPath,
      })
      await refetch()
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusyCode(null)
    }
  }

  const removeType = async (t: UnifiedFileTransferTaskType) => {
    setOpError(null)
    setBusyCode(t.typeCode)
    try {
      await deleteType.mutateAsync(t.typeCode)
      await refetch()
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '删除失败')
    } finally {
      setBusyCode(null)
    }
  }

  return (
    <PageShell
      code="F06"
      title="TRANSFER TEMPLATES · 模板配置"
      subtitle="UNIFIED FILE TRANSFER TASK-TYPE DEFINITIONS"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {/* 分类筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setCategory('')}
          className={`chip transition-all ${
            category === '' ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
          }`}
        >
          ALL · 全部 ({taskTypes.length})
        </button>
        {categories.map((c) => (
          <button
            key={c.value}
            type="button"
            onClick={() => setCategory(c.value)}
            className={`chip transition-all ${
              category === c.value
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {c.label} ({c.count})
          </button>
        ))}
      </div>

      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : visible.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <ScrollText className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
            NO TEMPLATES · 暂无模板定义
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {visible.map((t) => {
            const rpc = RPC_BADGE[t.rpcType]
            const busy = busyCode === t.typeCode
            return (
              <GlassPanel
                key={t.typeCode}
                className="border-l-2"
                title={
                  <span className="flex items-center gap-2">
                    {t.builtIn ? (
                      <ShieldCheck className="size-3.5 text-cyan-300/70" />
                    ) : (
                      <Wrench className="size-3.5 text-fuchsia-300/70" />
                    )}
                    <span className="truncate">{t.displayName}</span>
                  </span>
                }
                meta={t.builtIn ? 'BUILTIN' : 'CUSTOM'}
              >
                <div className="space-y-2.5 p-3.5">
                  <div className="flex items-center gap-2">
                    <StatusBadge
                      status={t.enabled ? 'online' : 'offline'}
                      label={t.enabled ? '已启用' : '已停用'}
                      className="scale-90"
                    />
                    <StatusBadge status={rpc.status} label={rpc.label} className="scale-90" />
                    <span className="chip text-cyan-300/70">{t.categoryLabel}</span>
                  </div>
                  <p className="line-clamp-2 min-h-[2.5em] text-xs leading-relaxed text-cyan-100/70">
                    {t.description || '—'}
                  </p>
                  <div className="grid grid-cols-2 gap-2 font-mono text-[10px] text-cyan-300/55">
                    <span>STEPS · {t.stepChain.length}</span>
                    <span>FILE · {t.fileTypeLabel || t.fileType || '—'}</span>
                    <span>30D · {t.taskCount30d} 次</span>
                    <span>成功率 · {Math.round(t.successRate30d)}%</span>
                  </div>
                  <div className="flex items-center justify-between gap-1.5 border-t border-cyan-500/10 pt-2.5">
                    <button
                      type="button"
                      onClick={() => navigate(`/transfer/template-management/${encodeURIComponent(t.typeCode)}`)}
                      className="flex items-center gap-1 font-mono text-[11px] text-cyan-300/80 hover:text-cyan-100"
                    >
                      详情 <ChevronRight className="size-3.5" />
                    </button>
                    <div className="flex items-center gap-1.5">
                      <button
                        type="button"
                        title={t.enabled ? '停用' : '启用'}
                        disabled={busy}
                        onClick={() => void toggleEnabled(t)}
                        className="flex size-6 items-center justify-center rounded-sm border border-cyan-500/25 text-cyan-300/75 transition-colors hover:border-cyan-400/60 hover:text-cyan-100 disabled:opacity-40"
                      >
                        {busy ? (
                          <Loader2 className="size-3.5 animate-spin" />
                        ) : t.enabled ? (
                          <PowerOff className="size-3.5" />
                        ) : (
                          <Power className="size-3.5" />
                        )}
                      </button>
                      {!t.builtIn && (
                        <button
                          type="button"
                          title="删除"
                          disabled={busy}
                          onClick={() => void removeType(t)}
                          className="flex size-6 items-center justify-center rounded-sm border border-rose-500/30 text-rose-300/80 transition-colors hover:border-rose-400/70 hover:text-rose-200 disabled:opacity-40"
                        >
                          <Trash2 className="size-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </GlassPanel>
            )
          })}
        </div>
      )}
    </PageShell>
  )
}
