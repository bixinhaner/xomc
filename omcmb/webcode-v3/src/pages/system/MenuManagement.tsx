import { useMemo } from 'react'
import { RefreshCcw, ListTree } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useMenuTree } from '@core/hooks/api/useMenus'
import { flattenMenus } from '@core/types/menu'
import type { Menu, MenuType } from '@core/types/menu'

import { StateBlock, MiniStat } from './_shared'

const MENU_TYPE_COLOR: Record<MenuType, string> = {
  directory: '#00f0ff',
  menu: '#00ff88',
  button: '#a855f7',
}

export default function MenuManagement() {
  const { data, isLoading, isError, error, isFetching, refetch } = useMenuTree()
  const tree = data ?? []

  const flat = useMemo(() => flattenMenus(tree), [tree])
  const dirCount = flat.filter((m) => m.type === 'directory').length
  const menuCount = flat.filter((m) => m.type === 'menu').length
  const btnCount = flat.filter((m) => m.type === 'button').length
  const disabledCount = flat.filter((m) => m.status === 'disabled').length

  return (
    <PageShell
      code="F06"
      title="MENUS · 菜单矩阵"
      subtitle="MENU TREE · RBAC"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="节点总数" value={flat.length} color="#00f0ff" icon={<ListTree className="size-3.5" />} />
          <MiniStat label="目录 · DIR" value={dirCount} color="#00f0ff" />
          <MiniStat label="菜单 · MENU" value={menuCount} color="#00ff88" />
          <MiniStat label="按钮 / 禁用" value={`${btnCount} / ${disabledCount}`} color="#a855f7" />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={tree.length === 0}
              emptyLabel="NO MENUS · 无菜单"
            >
              <div className="space-y-0.5">
                {tree.map((m) => (
                  <MenuNode key={m.id} node={m} depth={0} />
                ))}
              </div>
            </StateBlock>
          </div>
        </div>
      </div>
    </PageShell>
  )
}

function MenuNode({ node, depth }: { node: Menu; depth: number }) {
  const color = MENU_TYPE_COLOR[node.type] ?? '#6b86b6'
  const disabled = node.status === 'disabled'
  return (
    <>
      <div
        className="flex items-center gap-2 border-b border-cyan-500/8 py-1.5 hover:bg-cyan-500/5"
        style={{ paddingLeft: 8 + depth * 18 }}
      >
        <span
          className="size-1.5 shrink-0 rounded-full"
          style={{ background: color, boxShadow: `0 0 6px ${color}` }}
        />
        <span
          className={`font-display text-sm ${disabled ? 'text-cyan-300/35 line-through' : 'text-cyan-100'}`}
        >
          {node.name}
        </span>
        <span className="chip" style={{ color }}>
          {node.type}
        </span>
        {node.routePath ? (
          <span className="truncate font-mono text-[10px] text-cyan-300/45">{node.routePath}</span>
        ) : null}
        {node.permissionKey ? (
          <span className="ml-auto font-mono text-[10px] text-cyan-300/40">{node.permissionKey}</span>
        ) : null}
      </div>
      {node.children?.map((c) => (
        <MenuNode key={c.id} node={c} depth={depth + 1} />
      ))}
    </>
  )
}
