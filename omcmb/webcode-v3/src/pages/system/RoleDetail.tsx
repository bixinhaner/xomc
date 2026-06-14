import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Key, Server, ListTree } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useRoleById } from '@core/hooks/api/useSystem'

import { StateBlock, FieldRow } from './_shared'

export default function RoleDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: role, isLoading, isError, error, isFetching } = useRoleById(id)

  return (
    <PageShell
      code="F06"
      title="ROLE · 角色档案"
      subtitle={id}
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/system/roles')}>
          返回清单
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!role}
        emptyLabel="ROLE NOT FOUND · 角色不存在"
      >
        {role ? (
          <div className="space-y-5">
            <div className="flex items-center gap-3">
              <div className="font-display text-xl font-bold text-cyan-100">{role.roleName}</div>
              <span className="font-mono text-xs text-cyan-300/60">{role.roleCode}</span>
              <div className="ml-auto">
                <StatusBadge
                  status={role.builtIn > 0 ? 'inactive' : 'online'}
                  label={role.builtIn > 0 ? '内置角色' : '自定义角色'}
                />
              </div>
            </div>

            <div className="grid gap-x-8 md:grid-cols-2">
              <div>
                <FieldRow label="角色名">{role.roleName}</FieldRow>
                <FieldRow label="编码 · CODE">{role.roleCode}</FieldRow>
                <FieldRow label="描述">{role.description || '—'}</FieldRow>
                <FieldRow label="绑定用户数">{role.userCount ?? 0}</FieldRow>
                <FieldRow label="批量操作">{role.batchOperation > 0 ? '允许' : '禁止'}</FieldRow>
              </div>
              <div>
                <FieldRow label="设备组">
                  {role.builtIn > 0
                    ? '全部'
                    : role.deviceGroupIds && role.deviceGroupIds.length > 0
                      ? `${role.deviceGroupIds.length} 组`
                      : '未绑定'}
                </FieldRow>
                <FieldRow label="网络制式">
                  {role.networkTypes && role.networkTypes.length > 0
                    ? role.networkTypes.join(' · ')
                    : '全部'}
                </FieldRow>
                <FieldRow label="创建人">{role.createUser || '—'}</FieldRow>
                <FieldRow label="创建时间">
                  {role.createTime ? formatTime(role.createTime) : '—'}
                </FieldRow>
                <FieldRow label="更新时间">
                  {role.updateTime ? formatTime(role.updateTime) : '—'}
                </FieldRow>
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              <div className="glass rounded-sm p-3">
                <div className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
                  <Key className="size-3.5 text-cyan-300" /> 功能权限 · {role.permissions?.length ?? 0}
                </div>
                <div className="flex max-h-40 flex-wrap gap-1.5 overflow-auto">
                  {role.permissions && role.permissions.length > 0 ? (
                    role.permissions.map((p) => (
                      <span key={p} className="chip text-cyan-200">
                        {p}
                      </span>
                    ))
                  ) : (
                    <span className="font-mono text-xs text-cyan-300/45">无</span>
                  )}
                </div>
              </div>
              <div className="glass rounded-sm p-3">
                <div className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
                  <Server className="size-3.5 text-[#a855f7]" /> API 权限 · {role.apiPermissions?.length ?? 0}
                </div>
                <div className="max-h-40 space-y-1 overflow-auto">
                  {role.apiPermissions && role.apiPermissions.length > 0 ? (
                    role.apiPermissions.map((a, i) => (
                      <div key={`${a.method}-${a.path}-${i}`} className="font-mono text-[11px] text-cyan-100/80">
                        <span className="text-[#a855f7]">{a.method}</span> {a.path}
                      </div>
                    ))
                  ) : (
                    <span className="font-mono text-xs text-cyan-300/45">无</span>
                  )}
                </div>
              </div>
              <div className="glass rounded-sm p-3">
                <div className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
                  <ListTree className="size-3.5 text-[#00ff88]" /> 菜单权限 · {role.menuIds?.length ?? 0}
                </div>
                <div className="font-mono text-xs text-cyan-100/70">
                  {role.menuIds && role.menuIds.length > 0
                    ? `已授予 ${role.menuIds.length} 个菜单节点`
                    : '无'}
                </div>
              </div>
            </div>
          </div>
        ) : null}
      </StateBlock>
    </PageShell>
  )
}
