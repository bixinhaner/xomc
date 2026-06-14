import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useGroupById } from '@core/hooks/api/useSystem'

import { StateBlock, MiniStat, FieldRow } from './_shared'

export default function GroupDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: group, isLoading, isError, error, isFetching } = useGroupById(id)

  return (
    <PageShell
      code="F06"
      title="GROUP · 用户组档案"
      subtitle={id}
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/system/groups')}>
          返回清单
        </NeonButton>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={!group}
        emptyLabel="GROUP NOT FOUND · 用户组不存在"
      >
        {group ? (
          <div className="space-y-5">
            <div className="flex items-center gap-3">
              <div className="font-display text-xl font-bold text-cyan-100">{group.groupName}</div>
              <div className="ml-auto">
                <StatusBadge
                  status={group.builtIn > 0 ? 'inactive' : 'online'}
                  label={group.builtIn > 0 ? '内置用户组' : '自定义用户组'}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
              <MiniStat label="成员数 · USERS" value={group.userCount ?? 0} color="#00ff88" />
              <MiniStat label="角色数 · ROLES" value={group.roleCount ?? 0} color="#5b9eff" />
              <MiniStat
                label="类型 · TYPE"
                value={group.builtIn > 0 ? '内置' : '自定义'}
                color={group.builtIn > 0 ? '#a855f7' : '#00f0ff'}
              />
            </div>

            <div className="grid gap-x-8 md:grid-cols-2">
              <div>
                <FieldRow label="用户组名">{group.groupName}</FieldRow>
                <FieldRow label="描述">{group.description || '—'}</FieldRow>
              </div>
              <div>
                <FieldRow label="更新人">{group.updUser || '—'}</FieldRow>
                <FieldRow label="更新时间">{group.updTime ? formatTime(group.updTime) : '—'}</FieldRow>
              </div>
            </div>
          </div>
        ) : null}
      </StateBlock>
    </PageShell>
  )
}
