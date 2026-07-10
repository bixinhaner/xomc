import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCcw, FolderTree, ChevronRight, Layers, Settings2, X, Plus } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useCreateGroup, useDeviceGroups, useUpdateGroup } from '@core/hooks/api/useDevices'
import type { DeviceGroup, NameFilterItem } from '@core/types/device'
import { buildDeviceGroupSubtreeCountMap } from '@core/utils/deviceGroupCounts'
import { StateGate, StatCard } from './_shared'

function parseNumberRanges(value: string): number[] {
  const values = new Set<number>()
  for (const token of value.split(',')) {
    const match = token.trim().match(/^(\d+)(?:-(\d+))?$/)
    if (!match) continue
    const start = Number(match[1])
    const end = Number(match[2] ?? match[1])
    if (start > end || end > 65535) continue
    for (let current = start; current <= end; current += 1) values.add(current)
  }
  return [...values].sort((a, b) => a - b)
}

type RulePayload = {
  name?: string
  parent_id?: string
  source_group_id?: string
  matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber'
  name_rule_list?: NameFilterItem[]
  lac_list?: number[]
  tac_list?: number[]
  serial_number_list?: string[]
}

function RuleEditor({ group, groups, creating = false, pending, onClose, onSave }: {
  group: DeviceGroup
  groups: DeviceGroup[]
  creating?: boolean
  pending: boolean
  onClose: () => void
  onSave: (data: RulePayload) => void
}) {
  const [name, setName] = useState(group.name)
  const [parentId, setParentId] = useState(group.parentId ?? '')
  const [sourceGroupId, setSourceGroupId] = useState(group.sourceGroupId ?? '')
  const [mode, setMode] = useState<'deviceName' | 'lac' | 'tac' | 'serialNumber'>(group.matchingMode ?? 'deviceName')
  const [nameRules, setNameRules] = useState<NameFilterItem[]>(group.nameRuleList?.length
    ? group.nameRuleList
    : [{ id: `rule-${Date.now()}`, condition: 'contain', value: '' }])
  const [rangeValue, setRangeValue] = useState(
    (group.matchingMode === 'lac' ? group.lacList : group.tacList)?.join(',') ?? ''
  )
  const [serialValue, setSerialValue] = useState(group.serialNumberList?.join(',') ?? '')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
      <div className="max-h-[90vh] w-full max-w-xl overflow-auto border border-cyan-500/30 bg-[#07111d] p-5 shadow-[0_0_40px_rgba(0,240,255,0.12)]">
        <div className="mb-5 flex items-center justify-between">
          <div>
            <div className="font-display text-sm text-cyan-100">AUTO GROUP RULE</div>
            <div className="font-mono text-[10px] text-cyan-300/50">{creating ? 'NEW LEVEL-2 GROUP' : group.name}</div>
          </div>
          <button type="button" onClick={onClose} className="text-cyan-300/60 hover:text-cyan-100"><X className="size-4" /></button>
        </div>
        <div className="space-y-4 font-mono text-xs">
          {creating && <>
            <label className="block space-y-1"><span className="text-cyan-300/60">GROUP NAME · 分组名称</span><input className="h-9 w-full border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={name} onChange={(event) => setName(event.target.value)} /></label>
            <label className="block space-y-1"><span className="text-cyan-300/60">PARENT GROUP · 上级分组</span><select className="h-9 w-full border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={parentId} onChange={(event) => setParentId(event.target.value)}><option value="">请选择</option>{groups.filter((item) => !item.parentId).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          </>}
          <label className="block space-y-1">
            <span className="text-cyan-300/60">SOURCE GROUP · 源设备组</span>
            <select className="h-9 w-full border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={sourceGroupId} onChange={(event) => setSourceGroupId(event.target.value)}>
              <option value="">请选择</option>
              {groups.filter((item) => item.parentId && item.id !== group.id).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
            </select>
          </label>
          <label className="block space-y-1">
            <span className="text-cyan-300/60">MATCH MODE · 匹配方式</span>
            <select className="h-9 w-full border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={mode} onChange={(event) => setMode(event.target.value as typeof mode)}>
              <option value="deviceName">设备名称</option><option value="lac">LAC</option><option value="tac">TAC</option><option value="serialNumber">设备序列号</option>
            </select>
          </label>
          {mode === 'deviceName' && (
            <div className="space-y-2">
              <span className="text-cyan-300/60">NAME CONDITIONS · 名称条件</span>
              {nameRules.map((rule, index) => (
                <div key={rule.id} className="flex gap-2">
                  {index > 0 && <select className="w-20 border border-cyan-500/20 bg-black/30 px-1 text-cyan-100" value={rule.andOr ?? 'and'} onChange={(event) => setNameRules((items) => items.map((item) => item.id === rule.id ? { ...item, andOr: event.target.value as 'and' | 'or' } : item))}><option value="and">并且</option><option value="or">或者</option></select>}
                  <select className="w-24 border border-cyan-500/20 bg-black/30 px-1 text-cyan-100" value={rule.condition} onChange={(event) => setNameRules((items) => items.map((item) => item.id === rule.id ? { ...item, condition: event.target.value as NameFilterItem['condition'] } : item))}><option value="contain">包含</option><option value="startWith">开头是</option><option value="endWith">结尾是</option></select>
                  <input className="h-9 min-w-0 flex-1 border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={rule.value} onChange={(event) => setNameRules((items) => items.map((item) => item.id === rule.id ? { ...item, value: event.target.value } : item))} />
                  <button type="button" disabled={nameRules.length === 1} onClick={() => setNameRules((items) => items.filter((item) => item.id !== rule.id))} className="px-2 text-cyan-300/60 disabled:opacity-20"><X className="size-3.5" /></button>
                </div>
              ))}
              <button type="button" disabled={nameRules.length >= 10} onClick={() => setNameRules((items) => [...items, { id: `rule-${Date.now()}`, condition: 'contain', value: '', andOr: 'and' }])} className="flex items-center gap-1 text-cyan-300/70"><Plus className="size-3.5" />ADD CONDITION</button>
            </div>
          )}
          {(mode === 'lac' || mode === 'tac') && <label className="block space-y-1"><span className="text-cyan-300/60">{mode.toUpperCase()}</span><input className="h-9 w-full border border-cyan-500/20 bg-black/30 px-2 text-cyan-100" value={rangeValue} onChange={(event) => setRangeValue(event.target.value)} placeholder="1,81,100-110" /></label>}
          {mode === 'serialNumber' && <label className="block space-y-1"><span className="text-cyan-300/60">SERIAL NUMBERS · 设备序列号</span><textarea className="min-h-20 w-full border border-cyan-500/20 bg-black/30 p-2 text-cyan-100" value={serialValue} onChange={(event) => setSerialValue(event.target.value)} /></label>}
        </div>
        <div className="mt-6 flex justify-end gap-2">
          <NeonButton onClick={onClose}>CANCEL</NeonButton>
          <NeonButton disabled={!sourceGroupId || pending || (creating && (!name.trim() || !parentId))} onClick={() => onSave({
            ...(creating ? { name: name.trim(), parent_id: parentId } : {}),
            source_group_id: sourceGroupId,
            matching_mode: mode,
            name_rule_list: mode === 'deviceName' ? nameRules.filter((rule) => rule.value.trim()) : [],
            lac_list: mode === 'lac' ? parseNumberRanges(rangeValue) : [],
            tac_list: mode === 'tac' ? parseNumberRanges(rangeValue) : [],
            serial_number_list: mode === 'serialNumber' ? [...new Set(serialValue.split(/[\s,;]+/).filter(Boolean))] : [],
          })}>SAVE RULE</NeonButton>
        </div>
      </div>
    </div>
  )
}

export default function FleetDeviceGroup() {
  const navigate = useNavigate()
  const { data, isLoading, isError, error, isFetching, refetch } = useDeviceGroups()
  const updateGroup = useUpdateGroup()
  const createGroup = useCreateGroup()
  const [editingRuleGroup, setEditingRuleGroup] = useState<DeviceGroup | null>(null)
  const [creatingRuleGroup, setCreatingRuleGroup] = useState(false)
  const groups: DeviceGroup[] = useMemo(() => data?.groups ?? [], [data])
  const reportedTotal = data?.stats?.totalDevices

  // 组织为两级树：L1（parentId == null）→ children
  const tree = useMemo(() => {
    const roots = groups.filter((g) => !g.parentId)
    return roots.map((root) => ({
      root,
      children: groups.filter((g) => g.parentId === root.id),
    }))
  }, [groups])

  const totalDevices = useMemo(
    () => groups.reduce((acc, g) => (g.parentId ? acc + g.deviceCount : acc), 0),
    [groups]
  )
  const subtreeCounts = useMemo(
    () => buildDeviceGroupSubtreeCountMap(groups),
    [groups]
  )

  return (
    <PageShell
      code="F06"
      title="GROUPS · 设备分组"
      subtitle="HIERARCHY · TWO-LEVEL CLUSTERS"
      isFetching={isFetching}
      toolbar={
        <div className="flex gap-2">
          <NeonButton icon={<Plus />} onClick={() => setCreatingRuleGroup(true)}>NEW L2 RULE</NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>REFRESH</NeonButton>
        </div>
      }
    >
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3">
        <StatCard label="ROOT GROUPS" value={tree.length} color="#00f0ff" />
        <StatCard label="TOTAL GROUPS" value={groups.length} color="#a855f7" />
        <StatCard label="DEVICES IN GROUPS" value={reportedTotal ?? totalDevices} color="#00ff88" />
      </div>

      <StateGate
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={groups.length === 0}
        loadingLabel="LOADING GROUPS…"
        emptyLabel="NO GROUPS DEFINED"
      >
        <div className="space-y-3">
          {tree.map(({ root, children }) => (
            <GlassPanel
              key={root.id}
              title={
                <span className="flex items-center gap-2">
                  <FolderTree className="size-3.5" />
                  {root.name}
                </span>
              }
              meta={`${subtreeCounts.get(root.id) ?? root.deviceCount} DEV${root.builtIn ? ' · 内置' : ''}`}
            >
              {children.length === 0 ? (
                <div className="px-3 py-3 font-mono text-[11px] text-cyan-300/40">
                  NO SUBGROUPS
                </div>
              ) : (
                <div className="divide-y divide-cyan-500/8">
                  {children.map((c) => (
                    <div key={c.id} className="flex items-center transition-colors hover:bg-cyan-500/5">
                    <button type="button" onClick={() => navigate(`/device/list?groupId=${c.id}`)} className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left">
                      <Layers className="size-3.5 text-cyan-400/70" />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-mono text-xs text-cyan-100">{c.name}</div>
                        {c.networkType || c.productClass ? (
                          <div className="truncate font-mono text-[10px] text-cyan-300/50">
                            {[c.networkType, c.productClass].filter(Boolean).join(' · ')}
                          </div>
                        ) : null}
                      </div>
                      <span className="font-display text-sm font-bold text-cyan-200">
                        {c.deviceCount}
                      </span>
                      <ChevronRight className="size-3.5 text-cyan-300/40" />
                    </button>
                    {c.builtIn !== 1 && <button type="button" title="编辑自动归属规则" onClick={() => setEditingRuleGroup(c)} className="mr-3 p-1.5 text-cyan-300/50 hover:text-cyan-100"><Settings2 className="size-3.5" /></button>}
                    </div>
                  ))}
                </div>
              )}
            </GlassPanel>
          ))}
        </div>
      </StateGate>
      {editingRuleGroup && <RuleEditor
        group={editingRuleGroup}
        groups={groups}
        pending={updateGroup.isPending}
        onClose={() => setEditingRuleGroup(null)}
        onSave={(data) => updateGroup.mutate({ id: editingRuleGroup.id, data }, { onSuccess: () => setEditingRuleGroup(null) })}
      />}
      {creatingRuleGroup && <RuleEditor
        key="create-rule-group"
        group={{ id: '', name: '', parentId: null, deviceCount: 0, description: '', builtIn: 0 }}
        groups={groups}
        creating
        pending={createGroup.isPending}
        onClose={() => setCreatingRuleGroup(false)}
        onSave={(data) => {
          if (!data.name || !data.parent_id) return
          createGroup.mutate({ ...data, name: data.name, parent_id: data.parent_id }, { onSuccess: () => setCreatingRuleGroup(false) })
        }}
      />}
    </PageShell>
  )
}
