import type { OpsCommandRecord } from '@core/mock/data/opsTools'
import { formatTime } from '@/lib/format'
import { Drawer, Field, SectionTitle } from './Modal'

function durationText(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)} s` : `${ms} ms`
}

export function CommandDetailDrawer({
  record,
  open,
  onClose,
}: {
  record: OpsCommandRecord | null
  open: boolean
  onClose: () => void
}) {
  if (!open || !record) return null
  const ok = record.success

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={620}
      title={record.commandText}
      badge={
        <span className="chip shrink-0" style={{ color: ok ? '#00ff88' : '#ff2d6f' }}>
          {ok ? '成功' : '失败'}
        </span>
      }
    >
      <div className="space-y-4 p-4">
        <div className="glass rounded-sm">
          <SectionTitle>执行信息 · EXEC</SectionTitle>
          <Field label="指令">
            <span className="font-mono text-emerald-300/85">{record.commandText}</span>
          </Field>
          <Field label="设备名称">{record.deviceName}</Field>
          <Field label="设备 SN">
            <span className="font-mono">{record.deviceSn}</span>
          </Field>
          <Field label="操作人">{record.operator}</Field>
          <Field label="执行时间">{formatTime(record.executeTime)}</Field>
          <Field label="耗时">
            <span
              style={{
                color:
                  record.duration > 5000
                    ? '#ff2d6f'
                    : record.duration > 2000
                      ? '#ffaa00'
                      : '#00ff88',
              }}
            >
              {durationText(record.duration)}
            </span>
          </Field>
        </div>

        <div className="glass rounded-sm">
          <SectionTitle>输出结果 · OUTPUT</SectionTitle>
          <div className="terminal max-h-80 overflow-auto p-3 text-[12px]">
            {ok ? (
              record.output ? (
                record.output.split('\n').map((line, i) => (
                  <div key={i} className="text-emerald-300/85">
                    {line}
                  </div>
                ))
              ) : (
                <div className="text-emerald-300/45">// 无输出</div>
              )
            ) : (
              <>
                {record.errorMessage ? (
                  <div className="text-rose-300">{record.errorMessage}</div>
                ) : null}
                <div className="text-emerald-300/45">// 执行失败，无标准输出</div>
              </>
            )}
          </div>
        </div>
      </div>
    </Drawer>
  )
}
