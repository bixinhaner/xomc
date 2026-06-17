import { useMemo, useState } from 'react'
import { Search, Loader2, ListTree, RefreshCcw, Plus, Pencil, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import {
  useStandardParams,
  useUpsertStandard,
  useDeleteStandard,
} from '@core/hooks/api/useParamModels'
import type { StandardParam, UpsertStandardInput } from '@core/types/paramModel'

import { Modal } from './Modal'
import { StandardParamDialog } from './StandardParamDialog'

const PAGE_SIZE = 30

const ENTRY_OPTIONS = [
  { v: '', t: 'ALL' },
  { v: 'parameter', t: 'PARAMETER' },
  { v: 'object', t: 'OBJECT' },
] as const

/**
 * 标准参数树（TR-069 标准路径字典，对照 v1 product/standard-params）。
 * 翻译器 Translator 以此为标准侧路径，与各产品私有路径双向映射。
 * 服务端一次性全量返回，本页客户端切片分页（与 v1 一致）。
 */
export default function StandardParamsPage() {
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [entryType, setEntryType] = useState('')
  const [page, setPage] = useState(1)

  // ISSUE-488: 新增 / 编辑 / 删除表单状态
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<StandardParam | null>(null)
  const [confirmDel, setConfirmDel] = useState<StandardParam | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = useStandardParams({
    keyword: keyword || undefined,
    entryType: entryType || undefined,
  })

  const upsert = useUpsertStandard()
  const del = useDeleteStandard()

  const openCreate = () => {
    setEditing(null)
    upsert.reset()
    setDialogOpen(true)
  }
  const openEdit = (row: StandardParam) => {
    setEditing(row)
    upsert.reset()
    setDialogOpen(true)
  }
  const submitDialog = (input: UpsertStandardInput, path?: string) => {
    upsert.mutate(
      { input, path },
      {
        onSuccess: () => {
          setDialogOpen(false)
          setEditing(null)
        },
      },
    )
  }

  const all = useMemo<StandardParam[]>(() => data?.items ?? [], [data])
  const total = all.length
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const pageRows = useMemo(
    () => all.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [all, page],
  )

  const paramCount = useMemo(() => all.filter((r) => r.entryType === 'parameter').length, [all])
  const objectCount = useMemo(() => all.filter((r) => r.entryType === 'object').length, [all])

  const applySearch = () => {
    setKeyword(draft.trim())
    setPage(1)
  }

  return (
    <PageShell
      code="F02"
      title="STANDARD PARAM TREE · 标准参数树"
      subtitle="TR-069 CANONICAL PATHS · TRANSLATOR SOURCE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-80 pl-9"
              placeholder="标准路径 / 数据类型"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          {ENTRY_OPTIONS.map((o) => (
            <button
              key={o.v || 'all'}
              type="button"
              onClick={() => {
                setEntryType(o.v)
                setPage(1)
              }}
              className={`chip transition-all ${
                entryType === o.v ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              {o.t}
            </button>
          ))}
          <NeonButton icon={<Search />} onClick={applySearch}>
            SEARCH
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
          <NeonButton icon={<Plus />} onClick={openCreate}>
            新增
          </NeonButton>
        </>
      }
    >
      {/* 概览统计 */}
      <div className="mb-3 grid grid-cols-3 gap-3">
        <Stat label="TOTAL PATHS" color="#00f0ff" value={total} />
        <Stat label="PARAMETER" color="#00ff88" value={paramCount} />
        <Stat label="OBJECT" color="#5b9eff" value={objectCount} />
      </div>

      {/* 列表头 */}
      {pageRows.length > 0 && (
        <div className="mb-1 grid grid-cols-[2.6fr_0.9fr_1fr_0.9fr_1fr_0.7fr_0.7fr_0.8fr] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>STANDARD PATH</span>
          <span>ENTRY</span>
          <span>ACCESS</span>
          <span>TYPE</span>
          <span>CHANGE APPLIES</span>
          <span className="text-right">MIN</span>
          <span className="text-right">MAX</span>
          <span className="text-right">操作</span>
        </div>
      )}

      <div className="space-y-1.5">
        {isLoading ? (
          <LoadingRow />
        ) : isError ? (
          <ErrorRow message={error instanceof Error ? error.message : '未知错误'} />
        ) : pageRows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <ListTree className="size-10 text-cyan-400/50" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              {keyword ? `NO MATCH · 无匹配「${keyword}」` : 'NO PATHS · 暂无标准参数'}
            </div>
          </div>
        ) : (
          pageRows.map((row) => (
            <div
              key={row.standardPath}
              className="fleet-row grid grid-cols-[2.6fr_0.9fr_1fr_0.9fr_1fr_0.7fr_0.7fr_0.8fr] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: row.entryType === 'object' ? '#5b9eff' : '#00ff88' }}
            >
              <code className="min-w-0 truncate font-mono text-xs text-cyan-100/90" title={row.standardPath}>
                {row.standardPath}
              </code>
              <span className="font-mono text-[11px] text-cyan-300/75">{row.entryType || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-300/75">{row.access || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-100/80">{row.dataType || '—'}</span>
              <span className="font-mono text-[11px] text-cyan-300/75">{row.changeApplies || '—'}</span>
              <span className="text-right font-mono text-[11px] text-cyan-300/60">{row.minValue ?? '—'}</span>
              <span className="text-right font-mono text-[11px] text-cyan-300/60">{row.maxValue ?? '—'}</span>
              <span className="flex justify-end gap-1">
                <button
                  type="button"
                  aria-label="编辑"
                  title="编辑"
                  onClick={() => openEdit(row)}
                  className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/70 transition-colors hover:border-cyan-400/60 hover:bg-cyan-500/10 hover:text-cyan-200"
                >
                  <Pencil className="size-3.5" />
                </button>
                <button
                  type="button"
                  aria-label="删除"
                  title="删除"
                  onClick={() => setConfirmDel(row)}
                  className="rounded-sm border border-rose-500/25 p-1 text-rose-300/70 transition-colors hover:border-rose-400/60 hover:bg-rose-500/10 hover:text-rose-200"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </span>
            </div>
          ))
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>

      {/* ISSUE-488: 新增 / 编辑 弹窗 */}
      <StandardParamDialog
        open={dialogOpen}
        editing={editing}
        loading={upsert.isPending}
        error={upsert.isError}
        onSubmit={submitDialog}
        onCancel={() => {
          setDialogOpen(false)
          setEditing(null)
        }}
      />

      {/* ISSUE-488: 删除确认 */}
      <Modal
        open={confirmDel !== null}
        title="删除标准参数"
        subtitle="DELETE STANDARD PARAM"
        width={460}
        onClose={() => setConfirmDel(null)}
        footer={
          <>
            <NeonButton onClick={() => setConfirmDel(null)} disabled={del.isPending}>
              取消
            </NeonButton>
            <NeonButton
              tone="danger"
              icon={del.isPending ? <Loader2 className="animate-spin" /> : <Trash2 />}
              disabled={del.isPending}
              onClick={() => {
                if (!confirmDel) return
                del.mutate(confirmDel.standardPath, {
                  onSuccess: () => setConfirmDel(null),
                })
              }}
            >
              确认删除
            </NeonButton>
          </>
        }
      >
        <p className="text-[13px] text-cyan-100/85">
          确认删除标准参数{' '}
          <code className="font-mono text-cyan-50">{confirmDel?.standardPath}</code> ？此操作不可撤销。
        </p>
        {del.isError ? (
          <p className="mt-2 font-mono text-[11px] text-rose-300">删除失败，请重试</p>
        ) : null}
      </Modal>
    </PageShell>
  )
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">{label}</div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function LoadingRow() {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function ErrorRow({ message }: { message: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {message}
    </div>
  )
}
