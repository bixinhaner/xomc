/**
 * GroupTreeSelect — KPI 指标分组「真·树形选择器」(v3 webcode-v3 / STARFORGE HUD)。
 *
 * 2026-06-22:把所有「按分组筛选 / 归属分组 / 父分组」下拉从原生 `<select>` 升级
 *   为可展开收起的 HUD 风格树形选择器,与「分组管理」表格的展示方式保持一致(树
 *   结构、可展开收起)。默认全部展开。
 *
 * v3 完全自制(无 Radix Popover):button 触发 + **createPortal 渲染到 document.body**
 *   的下拉面板(避免被父级 HudOverlay `max-h-[64vh] overflow-auto` 容器切割);
 *   监听 document mousedown + Esc 关闭外部点击。HUD 风格用 `neon-input` / cyan 色系
 *   / font-mono 大写字距与现有 IndicatorFormModal 等 HUD 浮层对齐。
 *
 * 数据契约:value 是 group.id 字符串(与原 select 行为一致);后端 listGroups 返回的
 *   嵌套 children 经 frontend-core 的 groupTreeData() 转为 GroupTreeNode[]。
 */
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Check, ChevronDown, ChevronRight, Loader2, X } from 'lucide-react'

import { useIndicatorGroups } from '@core/hooks/api/useIndicatorsLibrary'
import { groupTreeData } from '@core/services/api/indicatorLibraryApi'
import type { GroupTreeNode } from '@core/services/api/indicatorLibraryApi'
import type { DeviceType } from '@core/types/indicatorLibrary'

interface Props {
  deviceType: DeviceType
  operatorCode?: string
  /** 选中的 group.id;空串/undefined 表示未选。 */
  value?: string
  /** 选中变更回调;清空时回传 undefined。 */
  onChange: (next: string | undefined) => void
  /** 编辑「父分组」时传被编辑分组的 id,排除自身及整棵子树(避免成环)。 */
  excludeId?: string
  disabled?: boolean
  /** 显示清空按钮(X);列表筛选用,新建/编辑「必填」字段不传。 */
  allowClear?: boolean
  placeholder?: string
  className?: string
}

function collectTitles(nodes: GroupTreeNode[] | undefined, into: Map<string, string>): void {
  ;(nodes || []).forEach((n) => {
    into.set(n.value, n.title)
    if (n.children) collectTitles(n.children, into)
  })
}

function collectExpandable(nodes: GroupTreeNode[] | undefined, into: Set<string>): void {
  ;(nodes || []).forEach((n) => {
    if (n.children && n.children.length > 0) {
      into.add(n.key)
      collectExpandable(n.children, into)
    }
  })
}

interface NodeRowProps {
  node: GroupTreeNode
  depth: number
  selected?: string
  expandedKeys: Set<string>
  onToggle: (key: string) => void
  onSelect: (value: string) => void
}

function NodeRow({ node, depth, selected, expandedKeys, onToggle, onSelect }: NodeRowProps) {
  const hasChildren = !!node.children && node.children.length > 0
  const expanded = hasChildren ? expandedKeys.has(node.key) : false
  const isSelected = selected === node.value

  return (
    <>
      <div
        role="treeitem"
        aria-expanded={hasChildren ? expanded : undefined}
        aria-selected={isSelected}
        aria-level={depth + 1}
        className={
          'group flex cursor-pointer items-center gap-1 rounded-sm px-1.5 py-1 font-mono text-[12px] text-cyan-100 ' +
          'hover:bg-cyan-500/15 ' +
          (isSelected ? 'bg-cyan-500/20 text-cyan-50 ring-1 ring-cyan-400/40' : '')
        }
        style={{ paddingLeft: depth * 16 + 4 }}
        onClick={() => onSelect(node.value)}
      >
        {hasChildren ? (
          <button
            type="button"
            aria-label={expanded ? '收起' : '展开'}
            className="flex size-4 shrink-0 items-center justify-center rounded-sm text-cyan-300/60 hover:text-cyan-100"
            onClick={(e) => {
              e.stopPropagation()
              onToggle(node.key)
            }}
          >
            {expanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
          </button>
        ) : (
          <span className="size-4 shrink-0" aria-hidden />
        )}
        <span className="flex-1 truncate" title={node.title}>
          {node.title}
        </span>
        {isSelected ? <Check className="size-3.5 shrink-0 text-cyan-300" /> : null}
      </div>
      {hasChildren && expanded ? (
        <div role="group">
          {node.children!.map((child) => (
            <NodeRow
              key={child.key}
              node={child}
              depth={depth + 1}
              selected={selected}
              expandedKeys={expandedKeys}
              onToggle={onToggle}
              onSelect={onSelect}
            />
          ))}
        </div>
      ) : null}
    </>
  )
}

export default function GroupTreeSelect({
  deviceType,
  operatorCode,
  value,
  onChange,
  excludeId,
  disabled,
  allowClear,
  placeholder,
  className,
}: Props) {
  const triggerRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [panelPos, setPanelPos] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 0,
  })

  const { data, isLoading } = useIndicatorGroups(deviceType, operatorCode)
  const treeData = useMemo(
    () => groupTreeData(data?.items, { excludeId }),
    [data, excludeId],
  )

  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
  const seenExpandable = useRef<Set<string>>(new Set())
  useEffect(() => {
    const allExpandable = new Set<string>()
    collectExpandable(treeData, allExpandable)
    let mutated = false
    const next = new Set(expandedKeys)
    allExpandable.forEach((k) => {
      if (!seenExpandable.current.has(k)) {
        next.add(k) // 新出现的可展开节点:默认展开
        mutated = true
      }
    })
    if (mutated) setExpandedKeys(next)
    seenExpandable.current = allExpandable
    // 故意只依赖 treeData,不响应 expandedKeys —— 否则用户每次手动收起会被回放为"展开"。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [treeData])

  const titleMap = useMemo(() => {
    const m = new Map<string, string>()
    collectTitles(treeData, m)
    return m
  }, [treeData])

  const selectedLabel = value ? titleMap.get(value) : undefined

  // 计算下拉面板位置(用 portal 渲染到 body,避开 HudOverlay overflow-auto 切割)
  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return
    const updatePos = () => {
      const rect = triggerRef.current!.getBoundingClientRect()
      setPanelPos({ top: rect.bottom + 4, left: rect.left, width: rect.width })
    }
    updatePos()
    window.addEventListener('scroll', updatePos, true)
    window.addEventListener('resize', updatePos)
    return () => {
      window.removeEventListener('scroll', updatePos, true)
      window.removeEventListener('resize', updatePos)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node
      if (triggerRef.current?.contains(t)) return
      if (panelRef.current?.contains(t)) return
      setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const toggleExpand = (key: string) => {
    setExpandedKeys((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const handleSelect = (next: string) => {
    onChange(next)
    setOpen(false)
  }

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation()
    onChange(undefined)
  }

  return (
    <div className={'relative ' + (className ?? '')}>
      <button
        ref={triggerRef}
        type="button"
        role="combobox"
        aria-expanded={open}
        aria-haspopup="tree"
        disabled={disabled}
        onClick={() => !disabled && setOpen((v) => !v)}
        className={
          'neon-input flex w-full items-center justify-between gap-1.5 text-left ' +
          (disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer')
        }
      >
        <span
          className={
            'flex-1 truncate ' +
            (selectedLabel ? 'text-cyan-100' : 'text-cyan-300/50')
          }
        >
          {selectedLabel ?? placeholder ?? '—'}
        </span>
        {isLoading ? (
          <Loader2 className="size-3.5 shrink-0 animate-spin text-cyan-300/60" />
        ) : null}
        {allowClear && value && !disabled ? (
          <span
            role="button"
            aria-label="清空"
            tabIndex={-1}
            onClick={handleClear}
            className="flex size-5 shrink-0 items-center justify-center rounded-sm text-cyan-300/60 hover:bg-cyan-500/15 hover:text-cyan-100"
          >
            <X className="size-3" />
          </span>
        ) : null}
        <ChevronDown className="size-3.5 shrink-0 text-cyan-300/60" />
      </button>

      {open
        ? createPortal(
            <div
              ref={panelRef}
              role="tree"
              style={{
                position: 'fixed',
                top: panelPos.top,
                left: panelPos.left,
                width: panelPos.width,
                zIndex: 9999, // 高于所有 HudOverlay(默认 zIndex 70)
              }}
              className="max-h-80 min-w-60 overflow-auto rounded-sm border border-cyan-500/30 bg-[#03060d]/96 p-1 shadow-[0_0_24px_rgba(0,240,255,0.18)] backdrop-blur-sm"
            >
              {treeData.length === 0 ? (
                <div className="px-3 py-2 font-mono text-[11px] text-cyan-300/50">—</div>
              ) : (
                treeData.map((n) => (
                  <NodeRow
                    key={n.key}
                    node={n}
                    depth={0}
                    selected={value || undefined}
                    expandedKeys={expandedKeys}
                    onToggle={toggleExpand}
                    onSelect={handleSelect}
                  />
                ))
              )}
            </div>,
            document.body,
          )
        : null}
    </div>
  )
}
