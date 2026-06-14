import { useMemo, useState } from 'react'
import { Search, Settings2, Variable } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'

// ---------------------------------------------------------------------------
// MR 采集变量字典（对齐 v1 mr/Variables）。
// 说明：v1 该页本身无后端/ @core 数据源（纯前端配置目录），故此处以只读字典呈现，
// 不伪造 @core hook、不做假 CRUD。内容与 v1 静态目录一致，HUD 风格渲染。
// ---------------------------------------------------------------------------

type VarType = 'integer' | 'float' | 'enum' | 'boolean' | 'string'

interface MRVariable {
  id: string
  varName: string
  varType: VarType
  valueRange: string
  defaultValue: string
  description: string
  unit?: string
  category: string
}

const VARIABLES: MRVariable[] = [
  { id: 'var-001', varName: 'MR_PERIOD', varType: 'integer', valueRange: '5-3600', defaultValue: '30', description: 'MR 采集周期（秒）', unit: '秒', category: '采集配置' },
  { id: 'var-002', varName: 'MR_SAMPLE_NUM', varType: 'integer', valueRange: '1-1000', defaultValue: '200', description: '单次 MR 采集样本数', unit: '个', category: '采集配置' },
  { id: 'var-003', varName: 'MR_RSRP_THRESHOLD', varType: 'float', valueRange: '-140 ~ -44', defaultValue: '-110', description: 'RSRP 过滤阈值，低于此值的测量不上报', unit: 'dBm', category: '过滤配置' },
  { id: 'var-004', varName: 'MR_RSRQ_THRESHOLD', varType: 'float', valueRange: '-19.5 ~ -3', defaultValue: '-15', description: 'RSRQ 过滤阈值', unit: 'dB', category: '过滤配置' },
  { id: 'var-005', varName: 'MR_REPORT_MODE', varType: 'enum', valueRange: 'A1/A2/A3/A4/A5/B1/B2', defaultValue: 'A3', description: 'MR 上报触发模式', category: '上报配置' },
  { id: 'var-006', varName: 'MR_INTRA_MEAS_ENABLED', varType: 'boolean', valueRange: 'true/false', defaultValue: 'true', description: '是否启用同频测量', category: '测量配置' },
  { id: 'var-007', varName: 'MR_INTER_MEAS_ENABLED', varType: 'boolean', valueRange: 'true/false', defaultValue: 'true', description: '是否启用异频测量', category: '测量配置' },
  { id: 'var-008', varName: 'MR_MAX_NB_CELLS', varType: 'integer', valueRange: '1-8', defaultValue: '6', description: '最多上报邻区个数', unit: '个', category: '采集配置' },
  { id: 'var-009', varName: 'MR_COMPRESS_FORMAT', varType: 'enum', valueRange: 'NONE/GZIP/ZIP', defaultValue: 'GZIP', description: 'MR 文件压缩格式', category: '传输配置' },
  { id: 'var-010', varName: 'MR_FILE_NAMING', varType: 'string', valueRange: '1-255字符', defaultValue: '{SN}_{TYPE}_{DATE}.xml', description: 'MR 文件命名模板', category: '传输配置' },
]

const TYPE_COLOR: Record<VarType, string> = {
  integer: '#00f0ff',
  float: '#5b9eff',
  enum: '#a855f7',
  boolean: '#00ff88',
  string: '#6b86b6',
}
const TYPE_LABEL: Record<VarType, string> = {
  integer: '整数',
  float: '浮点',
  enum: '枚举',
  boolean: '布尔',
  string: '字符串',
}

const ALL_TYPES: VarType[] = ['integer', 'float', 'enum', 'boolean', 'string']

/**
 * MR 采集变量字典（/mr/variables）。只读配置目录，HUD 风格。
 */
export default function Variables() {
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState<VarType | ''>('')

  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    return VARIABLES.filter((v) => {
      if (typeFilter && v.varType !== typeFilter) return false
      if (kw && !v.varName.toLowerCase().includes(kw) && !v.description.toLowerCase().includes(kw)) {
        return false
      }
      return true
    })
  }, [keyword, typeFilter])

  return (
    <PageShell
      code="F05"
      title="MR VARIABLES · 采集变量字典"
      subtitle="MR COLLECTION PARAMETER CATALOG · READ-ONLY"
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-60 pl-9"
              placeholder="变量名 / 描述"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {(['', ...ALL_TYPES] as const).map((tp) => (
            <button
              key={tp || 'all'}
              type="button"
              onClick={() => setTypeFilter(tp)}
              className={`chip transition-all ${
                typeFilter === tp ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: tp ? TYPE_COLOR[tp] : '#00f0ff' }}
            >
              {tp ? TYPE_LABEL[tp] : 'ALL'}
            </button>
          ))}
        </>
      }
    >
      {rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Variable className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            NO VARIABLES · 无匹配变量
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
          {rows.map((v) => {
            const color = TYPE_COLOR[v.varType]
            return (
              <GlassPanel key={v.id} className="p-0">
                <div className="flex flex-col gap-2 p-3.5">
                  <div className="flex items-center justify-between gap-2">
                    <span className="flex items-center gap-2 font-mono text-sm font-bold text-glow" style={{ color }}>
                      <Settings2 className="size-3.5" />
                      {v.varName}
                    </span>
                    <span className="chip" style={{ color }}>
                      {TYPE_LABEL[v.varType]}
                    </span>
                  </div>
                  <div className="text-[11px] leading-relaxed text-cyan-300/65">{v.description}</div>
                  <div className="grid grid-cols-3 gap-2 border-t border-cyan-500/12 pt-2">
                    <KV label="RANGE · 取值范围" value={v.valueRange} />
                    <KV label="DEFAULT · 默认值" value={v.defaultValue} mono />
                    <KV label="UNIT · 单位" value={v.unit || '—'} />
                  </div>
                  <div className="flex justify-end">
                    <span className="chip text-[#a855f7]">{v.category}</span>
                  </div>
                </div>
              </GlassPanel>
            )
          })}
        </div>
      )}
    </PageShell>
  )
}

function KV({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[9px] uppercase tracking-[0.16em] text-cyan-300/45">{label}</div>
      <div className={`truncate text-xs text-cyan-100/85 ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}
