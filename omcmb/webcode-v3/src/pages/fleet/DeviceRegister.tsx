import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, PlusCircle, Loader2, CheckCircle2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { useCreateDevice, useProductClasses } from '@core/hooks/api/useDevices'
import type { CarrierCode, CreateDeviceInput, DeviceTechnology } from '@core/types/device'
import { ErrorBlock } from './_shared'

const CARRIERS: { v: CarrierCode; t: string }[] = [
  { v: 'cmcc', t: '中国移动' },
  { v: 'ctcc', t: '中国电信' },
  { v: 'cucc', t: '中国联通' },
]
const TECHS: { v: DeviceTechnology; t: string }[] = [
  { v: 'lte', t: 'LTE · 4G' },
  { v: 'nr', t: '5G NR · SA' },
]

export default function FleetDeviceRegister() {
  const navigate = useNavigate()
  const create = useCreateDevice()
  const { data: productClasses } = useProductClasses()

  const [form, setForm] = useState<CreateDeviceInput>({
    serialNumber: '',
    oui: '',
    carrier: 'cmcc',
    technology: 'lte',
  })
  const [done, setDone] = useState(false)

  const set = <K extends keyof CreateDeviceInput>(k: K, v: CreateDeviceInput[K]) =>
    setForm((p) => ({ ...p, [k]: v }))

  const valid =
    form.serialNumber.trim().length > 0 && /^[0-9A-Fa-f]{6}$/.test(form.oui.trim())

  const submit = () => {
    setDone(false)
    create.mutate(
      {
        ...form,
        serialNumber: form.serialNumber.trim(),
        oui: form.oui.trim(),
        productClass: form.productClass?.trim() || undefined,
        deviceName: form.deviceName?.trim() || undefined,
        ipAddress: form.ipAddress?.trim() || undefined,
      },
      { onSuccess: () => setDone(true) }
    )
  }

  return (
    <PageShell
      code="F06"
      title="REGISTER · 设备注册"
      subtitle="MANUAL UNIT ENROLLMENT · TR069 PARTITION"
      toolbar={
        <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/device/list')}>
          FLEET
        </NeonButton>
      }
    >
      <div className="mx-auto max-w-3xl">
        <GlassPanel strong title="NEW UNIT · 新建设备">
          <div className="grid grid-cols-1 gap-4 p-4 md:grid-cols-2">
            <Field label="SERIAL NUMBER *" hint="设备唯一序列号（必填）">
              <input
                className="neon-input w-full"
                value={form.serialNumber}
                onChange={(e) => set('serialNumber', e.target.value)}
                placeholder="如 CPE-D8E1F4"
              />
            </Field>
            <Field label="OUI *" hint="厂商 OUI，6 位十六进制">
              <input
                className="neon-input w-full"
                value={form.oui}
                onChange={(e) => set('oui', e.target.value)}
                placeholder="如 00A1B2"
              />
            </Field>
            <Field label="CARRIER *" hint="运营商分区（不可变）">
              <select
                className="neon-input w-full"
                value={form.carrier}
                onChange={(e) => set('carrier', e.target.value as CarrierCode)}
              >
                {CARRIERS.map((c) => (
                  <option key={c.v} value={c.v} className="bg-[#03050d]">
                    {c.t}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="TECHNOLOGY *" hint="制式">
              <select
                className="neon-input w-full"
                value={form.technology}
                onChange={(e) => set('technology', e.target.value as DeviceTechnology)}
              >
                {TECHS.map((t) => (
                  <option key={t.v} value={t.v} className="bg-[#03050d]">
                    {t.t}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="PRODUCT CLASS" hint="产品型号（可空）">
              <input
                className="neon-input w-full"
                value={form.productClass ?? ''}
                onChange={(e) => set('productClass', e.target.value)}
                list="fleet-pc-list"
                placeholder="如 PM-B4860"
              />
              <datalist id="fleet-pc-list">
                {(productClasses ?? []).map((pc) => (
                  <option key={pc} value={pc} />
                ))}
              </datalist>
            </Field>
            <Field label="DEVICE NAME" hint="显示名（可空）">
              <input
                className="neon-input w-full"
                value={form.deviceName ?? ''}
                onChange={(e) => set('deviceName', e.target.value)}
              />
            </Field>
            <Field label="IP ADDRESS" hint="管理 IP（可空）">
              <input
                className="neon-input w-full"
                value={form.ipAddress ?? ''}
                onChange={(e) => set('ipAddress', e.target.value)}
                placeholder="如 100.64.18.31"
              />
            </Field>
            <Field label="SITE ID" hint="站点编号（可空）">
              <input
                className="neon-input w-full"
                value={form.siteId ?? ''}
                onChange={(e) => set('siteId', e.target.value)}
              />
            </Field>
          </div>

          <div className="flex flex-wrap items-center gap-3 border-t border-cyan-500/15 px-4 py-3">
            <NeonButton
              icon={create.isPending ? <Loader2 className="animate-spin" /> : <PlusCircle />}
              disabled={!valid || create.isPending}
              onClick={submit}
            >
              {create.isPending ? 'REGISTERING…' : 'REGISTER UNIT'}
            </NeonButton>
            {!valid && (
              <span className="font-mono text-[11px] text-amber-300/70">
                需填写 SN 与合法 OUI（6 位十六进制）
              </span>
            )}
            {done && (
              <span className="flex items-center gap-1.5 font-mono text-[11px] text-emerald-300">
                <CheckCircle2 className="size-4" /> 注册成功
                <button
                  type="button"
                  className="ml-2 underline decoration-dotted"
                  onClick={() => navigate('/device/list')}
                >
                  返回舰队
                </button>
              </span>
            )}
          </div>

          {create.isError && (
            <div className="px-4 pb-4">
              <ErrorBlock error={create.error} />
            </div>
          )}
        </GlassPanel>
      </div>
    </PageShell>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/60">
        {label}
      </span>
      {children}
      {hint ? <span className="font-mono text-[10px] text-cyan-300/40">{hint}</span> : null}
    </label>
  )
}
