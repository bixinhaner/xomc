import type { OpsStep, OpsTemplate } from '@core/mock/data/opsTools'
import { formatTime } from '@/lib/format'
import { Drawer, Field, SectionTitle } from './Modal'

const STEP_COLOR: Record<OpsStep['stepType'], string> = {
  mml: '#00f0ff',
  check: '#00ff88',
  wait: '#ffaa00',
  notify: '#a855f7',
  script: '#5b9eff',
}
const STEP_LABEL: Record<OpsStep['stepType'], string> = {
  mml: 'MML 指令',
  check: '条件校验',
  wait: '等待',
  notify: '通知',
  script: '脚本',
}

function durationText(secs: number): string {
  return secs >= 60 ? `${Math.floor(secs / 60)} 分钟` : `${secs} 秒`
}

export function TemplateDetailDrawer({
  template,
  open,
  onClose,
}: {
  template: OpsTemplate | null
  open: boolean
  onClose: () => void
}) {
  if (!open || !template) return null

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={560}
      title={template.templateName}
      badge={<span className="chip shrink-0 text-[#00f0ff]">{template.category}</span>}
    >
      <div className="space-y-4 p-4">
        <div className="glass rounded-sm">
          <SectionTitle>基本信息 · BASIC</SectionTitle>
          <Field label="模板名称">{template.templateName}</Field>
          <Field label="分类">{template.category}</Field>
          <Field label="适用设备">
            <span className="flex flex-wrap gap-1">
              {template.targetDeviceTypes.map((tp) => (
                <span key={tp} className="chip text-[#5b9eff]">
                  {tp}
                </span>
              ))}
            </span>
          </Field>
          <Field label="预计耗时">{durationText(template.estimatedDuration)}</Field>
          <Field label="使用次数">{template.useCount}</Field>
          <Field label="创建人">{template.creator}</Field>
          <Field label="更新时间">{formatTime(template.updateTime)}</Field>
          {template.tags.length > 0 ? (
            <Field label="标签">
              <span className="flex flex-wrap gap-1">
                {template.tags.map((tg) => (
                  <span key={tg} className="chip text-[#a855f7]">
                    {tg}
                  </span>
                ))}
              </span>
            </Field>
          ) : null}
          <Field label="说明">{template.description || '—'}</Field>
        </div>

        <div className="glass rounded-sm">
          <SectionTitle>执行步骤 · STEPS ({template.steps.length})</SectionTitle>
          {template.steps.length === 0 ? (
            <div className="px-3.5 py-4 font-mono text-[11px] text-cyan-300/45">
              // 该模板尚未配置步骤
            </div>
          ) : (
            <div className="space-y-2 p-3.5">
              {template.steps.map((s) => {
                const c = STEP_COLOR[s.stepType]
                return (
                  <div
                    key={s.stepNo}
                    className="relative rounded-sm border-l-2 bg-cyan-500/[0.03] px-3 py-2"
                    style={{ borderLeftColor: c }}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-[10px] text-cyan-300/45">
                        {String(s.stepNo).padStart(2, '0')}
                      </span>
                      <span className="chip" style={{ color: c }}>
                        {STEP_LABEL[s.stepType]}
                      </span>
                      <span className="font-display text-[13px] text-cyan-100">{s.stepName}</span>
                    </div>
                    {s.description ? (
                      <div className="mt-1 text-[11px] text-cyan-300/70">{s.description}</div>
                    ) : null}
                    {s.command ? (
                      <div className="mt-1.5 rounded-sm bg-black/40 px-2 py-1 font-mono text-[11px] text-emerald-300/85">
                        {s.command}
                      </div>
                    ) : null}
                    {s.condition ? (
                      <div className="mt-1 font-mono text-[10px] text-[#00f0ff]">
                        条件 · {s.condition}
                      </div>
                    ) : null}
                    {s.waitSeconds ? (
                      <div className="mt-1 font-mono text-[10px] text-[#ffaa00]">
                        等待 · {s.waitSeconds}s
                      </div>
                    ) : null}
                    {s.notifyTarget ? (
                      <div className="mt-1 font-mono text-[10px] text-[#a855f7]">
                        通知 · {s.notifyTarget}
                      </div>
                    ) : null}
                    {s.rollbackCommand ? (
                      <div className="mt-1 font-mono text-[10px] text-[#ff2d6f]">
                        回滚 · {s.rollbackCommand}
                      </div>
                    ) : null}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </Drawer>
  )
}
