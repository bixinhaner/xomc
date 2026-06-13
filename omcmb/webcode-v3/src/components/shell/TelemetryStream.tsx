import { GlassPanel } from '@/components/ui/GlassPanel'
import { DataStream } from '@/components/viz/DataStream'

export function TelemetryStream() {
  return (
    <GlassPanel
      title="TELEMETRY · 实时事件流"
      meta="LIVE"
      className="h-full"
      strong
    >
      <div className="h-full overflow-hidden">
        <DataStream max={28} />
      </div>
    </GlassPanel>
  )
}
