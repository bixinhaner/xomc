/**
 * GroupTreeSelect — KPI 指标分组「真·树形选择器」(v2 webcode-v2 / shadcn-Tailwind)。
 *
 * 2026-06-22:把所有「按分组筛选 / 归属分组 / 父分组」下拉从扁平 `<Select>` 升级
 *   为可展开收起的树形选择器,与「分组管理」表格的展示方式保持一致(树结构、可
 *   展开收起)。默认全部展开,与现状 `defaultExpandAllRows` 表格行为对齐。
 *
 * v2 没有 Radix Popover / ScrollArea,本组件自包含实现:
 *   · button 触发 + **createPortal 渲染到 document.body** 的下拉面板(避免被父级
 *     fixed/overflow 浮层切割,如 KpiIndicatorsDetail 的 IndicatorForm 弹窗);
 *   · 监听 document mousedown 关闭外部点击 + Esc 关闭;
 *   · 内部递归 Tree(按 children 渲染),展开/收起箭头按 expandedKeys 控制,
 *     新出现的可展开节点自动加入展开集合(默认全部展开);
 *   · 选中高亮 + 关闭面板;allowClear 时显示 X 按钮(右侧);
 *   · ARIA: role="combobox" + aria-expanded;面板 role="tree";节点 role="treeitem"
 *     + aria-expanded/aria-selected/aria-level。
 *
 * 数据契约:value 是 group.id 字符串(与原 Select 行为一致);后端 listGroups 返回的
 *   嵌套 children 经 frontend-core 的 groupTreeData() 转为 GroupTreeNode[]。
 */
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Check, ChevronDown, ChevronRight, Loader2, X } from 'lucide-react'

import { useIndicatorGroups } from '@core/hooks/api/useIndicatorsLibrary'
import { groupTreeData } from '@core/services/api/indicatorLibraryApi'
import type { GroupTreeNode } from '@core/services/api/indicatorLibraryApi'
import type { DeviceType } from '@core/types/indicatorLibrary'

import { cn } from '@/lib/utils'

interface Props {
  deviceType: DeviceType
  operatorCode?: string
  /** 仅 KpiIndicatorsDetail 详情态分组筛选用:按 platform 过滤分组。 */
  platform?: string
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
  /** 用于 label 关联;Form 字段对齐 input 时 useful。 */
  id?: string
  /** 用于 form 校验显示边框(对齐 v2 现有 Input 的 aria-invalid 风格)。 */
  invalid?: boolean
}

/** 把树拉平为 id→title 索引,用于显示选中态的 label(避免每次 walk)。 */
function collectTitles(nodes: GroupTreeNode[] | undefined, into: Map<string, string>): void {
  ;(nodes || []).forEach((n) => {
    into.set(n.value, n.title)
    if (n.children) collectTitles(n.children, into)
  })
}

/** 收集所有非叶节点 key(用于"默认全部展开")。 */
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
        className={cn(
          'group flex cursor-pointer items-center gap-1 rounded-sm px-1.5 py-1 text-sm',
          'hover:bg-accent',
          isSelected && 'bg-accent font-medium',
        )}
        style={{ paddingLeft: depth * 16 + 4 }}
        onClick={() => onSelect(node.value)}
      >
        {/* 展开/收起箭头 — 无 children 时占位保宽度对齐。 */}
        {hasChildren ? (
          <button
            type="button"
            aria-label={expanded ? '收起' : '展开'}
            className="flex size-4 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:text-foreground"
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
        {isSelected ? <Check className="size-3.5 shrink-0 text-primary" /> : null}
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
  platform,
  value,
  onChange,
  excludeId,
  disabled,
  allowClear,
  placeholder,
  className,
  id,
  invalid,
}: Props) {
  const triggerRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [panelPos, setPanelPos] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 0,
  })

  const { data, isLoading } = useIndicatorGroups(deviceType, operatorCode, platform)
  const treeData = useMemo(
    () => groupTreeData(data?.items, { excludeId }),
    [data, excludeId],
  )

  // 默认全部展开:数据加载后初始化为所有非叶节点的 key 集合;数据变化时合并保留用户手动收起。
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
  // 跟踪上次"自动加入"过的 expandable key 集合,避免数据变化重复触发覆盖用户操作。
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

  // 计算下拉面板位置(用 portal 渲染到 body,避开父级 fixed/overflow 容器切割)
  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return
    const updatePos = () => {
      const rect = triggerRef.current!.getBoundingClientRect()
      setPanelPos({ top: rect.bottom + 4, left: rect.left, width: rect.width })
    }
    updatePos()
    window.addEventListener('scroll', updatePos, true) // true: 捕获阶段,父级 scroll 也响应
    window.addEventListener('resize', updatePos)
    return () => {
      window.removeEventListener('scroll', updatePos, true)
      window.removeEventListener('resize', updatePos)
    }
  }, [open])

  // 点击外部 / Esc 关闭面板(同时认 trigger + portal panel 两个根)
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
    <div className={cn('relative', className)}>
      <button
        ref={triggerRef}
        type="button"
        id={id}
        role="combobox"
        aria-expanded={open}
        aria-haspopup="tree"
        disabled={disabled}
        onClick={() => !disabled && setOpen((v) => !v)}
        className={cn(
          'flex h-9 w-full items-center justify-between gap-1.5 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm',
          'focus:outline-none focus:ring-1 focus:ring-ring',
          disabled && 'cursor-not-allowed opacity-50',
          invalid && 'border-destructive focus:ring-destructive',
        )}
      >
        <span className={cn('flex-1 truncate text-left', !selectedLabel && 'text-muted-foreground')}>
          {selectedLabel ?? placeholder ?? '请选择'}
        </span>
        {isLoading ? (
          <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" />
        ) : null}
        {allowClear && value && !disabled ? (
          <span
            role="button"
            aria-label="清空"
            tabIndex={-1}
            onClick={handleClear}
            className="flex size-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X className="size-3.5" />
          </span>
        ) : null}
        <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
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
                zIndex: 9999, // 高于所有浮层(KpiIndicatorsDetail IndicatorForm z-[60] 等)
              }}
              className="max-h-80 min-w-60 overflow-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
            >
              {treeData.length === 0 ? (
                <div className="px-3 py-2 text-sm text-muted-foreground">—</div>
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
