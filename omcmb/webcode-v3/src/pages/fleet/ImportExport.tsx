import { useMemo, useState } from 'react'
import { Download, Upload, Loader2, CheckCircle2, FileText } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useDeviceList } from '@core/hooks/api/useDevices'
import { deviceApi } from '@core/services/api/deviceApi'
import type {
  BatchImportDevice,
  BatchImportResponse,
} from '@core/types/device'
import { ErrorBlock, StatCard } from './_shared'

// 导出：拉取一页（最多 500）真实设备清单，前端生成 CSV。
const EXPORT_PAGE_SIZE = 500

export default function FleetImportExport() {
  // ----- 导出 -----
  const { data, isLoading, isFetching } = useDeviceList({ page: 1, pageSize: EXPORT_PAGE_SIZE })
  const devices = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0

  const exportCsv = () => {
    const header = ['serial_number', 'device_name', 'oui', 'carrier', 'product_class', 'ip_address']
    const lines = devices.map((d) =>
      [d.sn, d.name || d.deviceName, d.oui, d.carrier, d.productClass, d.ipAddress]
        .map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`)
        .join(',')
    )
    const blob = new Blob([[header.join(','), ...lines].join('\n')], {
      type: 'text/csv;charset=utf-8',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `fleet-export-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  // ----- 导入 -----
  const [raw, setRaw] = useState('')
  const [importing, setImporting] = useState(false)
  const [result, setResult] = useState<BatchImportResponse | null>(null)
  const [importError, setImportError] = useState<unknown>(null)

  // 解析 CSV：每行 serial_number,device_name,remark（首列必填）。
  const parsed: BatchImportDevice[] = useMemo(() => {
    return raw
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l && !l.toLowerCase().startsWith('serial_number'))
      .map((l) => {
        const [sn, name, remark] = l.split(',').map((c) => c.trim())
        return { serial_number: sn, device_name: name || undefined, remark: remark || undefined }
      })
      .filter((d) => d.serial_number)
  }, [raw])

  const doImport = async () => {
    if (parsed.length === 0) return
    setImporting(true)
    setImportError(null)
    setResult(null)
    try {
      const res = await deviceApi.batchImportDevices({ devices: parsed })
      setResult(res)
    } catch (e) {
      setImportError(e)
    } finally {
      setImporting(false)
    }
  }

  return (
    <PageShell
      code="F06"
      title="IMPORT / EXPORT · 导入导出"
      subtitle="BULK UNIT TRANSFER · CSV"
      isFetching={isFetching}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* 导出 */}
        <GlassPanel strong title="EXPORT · 导出清单" meta={<FileText className="size-3.5" />}>
          <div className="space-y-3 p-4">
            <div className="grid grid-cols-2 gap-3">
              <StatCard label="TOTAL UNITS" value={total} color="#00f0ff" />
              <StatCard
                label="EXPORTABLE"
                value={devices.length}
                color="#00ff88"
                hint={`≤ ${EXPORT_PAGE_SIZE} / 批`}
              />
            </div>
            <p className="font-mono text-[11px] text-cyan-300/55">
              导出当前设备清单（SN / 名称 / OUI / 运营商 / 型号 / IP）为 CSV，
              可二次编辑后回传导入。
            </p>
            <NeonButton
              icon={isLoading ? <Loader2 className="animate-spin" /> : <Download />}
              disabled={isLoading || devices.length === 0}
              onClick={exportCsv}
            >
              EXPORT CSV
            </NeonButton>
          </div>
        </GlassPanel>

        {/* 导入 */}
        <GlassPanel strong title="IMPORT · 批量更新" meta={<Upload className="size-3.5" />}>
          <div className="space-y-3 p-4">
            <p className="font-mono text-[11px] text-cyan-300/55">
              每行一条：<span className="text-cyan-200">serial_number,device_name,remark</span>
              （仅更新已注册设备的名称 / 备注，首列必填）。
            </p>
            <textarea
              className="neon-input h-40 w-full resize-none font-mono text-[11px]"
              placeholder={'CPE-D8E1F4,北京朝阳-01,主用\nCPE-A1B0E9,北京海淀-02'}
              value={raw}
              onChange={(e) => setRaw(e.target.value)}
            />
            <div className="flex items-center gap-3">
              <NeonButton
                icon={importing ? <Loader2 className="animate-spin" /> : <Upload />}
                disabled={parsed.length === 0 || importing}
                onClick={() => void doImport()}
              >
                {importing ? 'IMPORTING…' : `IMPORT (${parsed.length})`}
              </NeonButton>
              {parsed.length === 0 && raw.trim() && (
                <span className="font-mono text-[10px] text-amber-300/70">无有效行</span>
              )}
            </div>

            {result && (
              <div className="space-y-1.5 rounded-sm border border-cyan-500/20 bg-cyan-500/[0.04] p-3">
                <div className="flex items-center gap-2 font-mono text-[11px] text-emerald-300">
                  <CheckCircle2 className="size-4" />
                  导入完成：{result.succeeded} 成功 / {result.failed} 失败 / {result.total} 总计
                </div>
                {result.errors.length > 0 && (
                  <div className="max-h-32 space-y-0.5 overflow-auto font-mono text-[10px] text-rose-300/80">
                    {result.errors.map((er, i) => (
                      <div key={`${er.row}-${i}`}>
                        行 {er.row} {er.sn ? `(${er.sn})` : ''}: {er.reason}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {importError ? <ErrorBlock error={importError} /> : null}
          </div>
        </GlassPanel>
      </div>
    </PageShell>
  )
}
