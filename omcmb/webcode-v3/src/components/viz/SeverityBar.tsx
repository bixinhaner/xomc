import type { AlarmSeverity } from '@core/types/common'

const COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}

const HEIGHT: Record<AlarmSeverity, string> = {
  critical: '100%',
  major: '78%',
  minor: '54%',
  warning: '34%',
}

export function SeverityBar({
  severity,
  height = 28,
}: {
  severity: AlarmSeverity
  height?: number
}) {
  return (
    <span
      className="inline-flex items-end"
      style={{ width: 4, height, color: COLOR[severity] }}
    >
      <span className="sev-pillar" style={{ height: HEIGHT[severity] }} />
    </span>
  )
}
