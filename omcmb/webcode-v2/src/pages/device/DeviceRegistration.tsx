import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, Loader2, Save } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useCreateDevice } from '@core/hooks/api/useDevices'
import type {
  CarrierCode,
  CreateDeviceInput,
  DeviceTechnology,
} from '@core/types/device'

// ============================================================
// 设备手动注册 — 预登记表单（SN/OUI/运营商/制式 + 可选网络/站址）
// 对照 v1 webcode/src/pages/device/DeviceRegistration 的业务深度
// 提交走真实 useCreateDevice（POST /devices）；CPE 上线后 Inform 会覆盖占位值
// ============================================================

const CARRIER_OPTIONS: { label: string; value: CarrierCode }[] = [
  { label: '中国移动 (cmcc)', value: 'cmcc' },
  { label: '中国电信 (ctcc)', value: 'ctcc' },
  { label: '中国联通 (cucc)', value: 'cucc' },
]

const TECHNOLOGY_OPTIONS: { label: string; value: DeviceTechnology }[] = [
  { label: 'LTE (4G)', value: 'lte' },
  { label: 'NR (5G)', value: 'nr' },
]

const PRODUCT_CLASS_OPTIONS = ['eNB', 'gNB', 'CPE', 'eGW']

const SN_RE = /^[A-Za-z0-9-_]+$/
const OUI_RE = /^[0-9A-Fa-f]{6}$/
const IPV4_RE = /^(\d{1,3}\.){3}\d{1,3}$/

interface FormState {
  serialNumber: string
  oui: string
  carrier: CarrierCode | ''
  technology: DeviceTechnology | ''
  manufacturer: string
  productClass: string
  modelName: string
  ipAddress: string
  deviceName: string
  siteId: string
  latitude: string
  longitude: string
}

const EMPTY_FORM: FormState = {
  serialNumber: '',
  oui: '',
  carrier: '',
  technology: '',
  manufacturer: '',
  productClass: '',
  modelName: '',
  ipAddress: '',
  deviceName: '',
  siteId: '',
  latitude: '',
  longitude: '',
}

function Field({
  label,
  required,
  error,
  children,
}: {
  label: string
  required?: boolean
  error?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-xs">
        {label}
        {required && <span className="ml-0.5 text-destructive">*</span>}
      </Label>
      {children}
      {error && <span className="text-xs text-destructive">{error}</span>}
    </div>
  )
}

export default function DeviceRegistration() {
  const navigate = useNavigate()
  const createDevice = useCreateDevice()

  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>(
    {}
  )
  const [submittedSn, setSubmittedSn] = useState<string | null>(null)

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    setErrors((prev) => ({ ...prev, [key]: undefined }))
  }

  function validate(): boolean {
    const next: Partial<Record<keyof FormState, string>> = {}
    if (!form.serialNumber.trim()) next.serialNumber = 'SN 必填'
    else if (!SN_RE.test(form.serialNumber.trim()))
      next.serialNumber = 'SN 仅允许 A-Z 0-9 - _'
    if (!form.oui.trim()) next.oui = 'OUI 必填'
    else if (!OUI_RE.test(form.oui.trim())) next.oui = 'OUI 为 6 位十六进制'
    if (!form.carrier) next.carrier = '请选择运营商'
    if (!form.technology) next.technology = '请选择制式'
    if (form.ipAddress.trim() && !IPV4_RE.test(form.ipAddress.trim()))
      next.ipAddress = 'IPv4 点分十进制'
    if (form.latitude.trim() && Number.isNaN(Number(form.latitude)))
      next.latitude = '纬度需为数字'
    if (form.longitude.trim() && Number.isNaN(Number(form.longitude)))
      next.longitude = '经度需为数字'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  function handleSubmit() {
    if (!validate()) return
    const input: CreateDeviceInput = {
      serialNumber: form.serialNumber.trim(),
      oui: form.oui.trim(),
      carrier: form.carrier as CarrierCode,
      technology: form.technology as DeviceTechnology,
      ...(form.manufacturer.trim() ? { manufacturer: form.manufacturer.trim() } : {}),
      ...(form.productClass ? { productClass: form.productClass } : {}),
      ...(form.modelName.trim() ? { modelName: form.modelName.trim() } : {}),
      ...(form.ipAddress.trim() ? { ipAddress: form.ipAddress.trim() } : {}),
      ...(form.deviceName.trim() ? { deviceName: form.deviceName.trim() } : {}),
      ...(form.siteId.trim() ? { siteId: form.siteId.trim() } : {}),
      ...(form.latitude.trim() ? { latitude: Number(form.latitude) } : {}),
      ...(form.longitude.trim() ? { longitude: Number(form.longitude) } : {}),
    }
    createDevice.mutate(
      input as unknown as Parameters<typeof createDevice.mutate>[0],
      {
        onSuccess: () => setSubmittedSn(input.serialNumber),
      }
    )
  }

  function resetForm() {
    setForm(EMPTY_FORM)
    setErrors({})
    setSubmittedSn(null)
  }

  if (submittedSn) {
    return (
      <PageShell title="设备手动注册">
        <Card>
          <CardContent className="flex flex-col items-center gap-4 py-12">
            <CheckCircle2 className="size-14 text-emerald-500" />
            <div className="text-lg font-semibold text-emerald-600 dark:text-emerald-400">
              注册成功
            </div>
            <div className="font-mono text-sm text-muted-foreground">
              {submittedSn}
            </div>
            <div className="mt-2 flex flex-wrap items-center justify-center gap-2">
              <Button onClick={() => navigate(`/devices/detail/${submittedSn}`)}>
                查看设备
              </Button>
              <Button variant="outline" onClick={resetForm}>
                继续注册
              </Button>
              <Button variant="ghost" onClick={() => navigate('/devices')}>
                返回列表
              </Button>
            </div>
          </CardContent>
        </Card>
      </PageShell>
    )
  }

  return (
    <PageShell
      title="设备手动注册"
      description="为尚未自动发现的设备预登记，CPE 上线后将以真实信息覆盖占位值"
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-4">
        {createDevice.isError && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2.5 text-sm text-destructive">
            注册失败：
            {createDevice.error instanceof Error
              ? createDevice.error.message
              : '未知错误'}
          </div>
        )}

        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">基本信息</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-4 p-4 pt-2 md:grid-cols-2">
            <Field label="序列号 (SN)" required error={errors.serialNumber}>
              <Input
                className={cn('font-mono', errors.serialNumber && 'border-destructive')}
                placeholder="例如 BCL2024001234"
                value={form.serialNumber}
                onChange={(e) => set('serialNumber', e.target.value)}
              />
            </Field>
            <Field label="OUI" required error={errors.oui}>
              <Input
                className={cn('font-mono uppercase', errors.oui && 'border-destructive')}
                placeholder="48575A"
                maxLength={6}
                value={form.oui}
                onChange={(e) => set('oui', e.target.value)}
              />
            </Field>
            <Field label="运营商" required error={errors.carrier}>
              <Select
                value={form.carrier}
                onValueChange={(v) => set('carrier', v as CarrierCode)}
              >
                <SelectTrigger className={cn(errors.carrier && 'border-destructive')}>
                  <SelectValue placeholder="请选择运营商" />
                </SelectTrigger>
                <SelectContent>
                  {CARRIER_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field label="制式" required error={errors.technology}>
              <Select
                value={form.technology}
                onValueChange={(v) => set('technology', v as DeviceTechnology)}
              >
                <SelectTrigger className={cn(errors.technology && 'border-destructive')}>
                  <SelectValue placeholder="请选择制式" />
                </SelectTrigger>
                <SelectContent>
                  {TECHNOLOGY_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field label="厂商">
              <Input
                placeholder="Baicells"
                value={form.manufacturer}
                onChange={(e) => set('manufacturer', e.target.value)}
              />
            </Field>
            <Field label="产品类型">
              <Select
                value={form.productClass}
                onValueChange={(v) => set('productClass', v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="可选" />
                </SelectTrigger>
                <SelectContent>
                  {PRODUCT_CLASS_OPTIONS.map((p) => (
                    <SelectItem key={p} value={p}>
                      {p}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field label="设备型号">
              <Input
                placeholder="BBU3910 / AAU5613"
                value={form.modelName}
                onChange={(e) => set('modelName', e.target.value)}
              />
            </Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">
              网络与站址
              <Badge variant="muted" className="ml-2">
                可选
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-4 p-4 pt-2 md:grid-cols-2">
            <Field label="IP 地址" error={errors.ipAddress}>
              <Input
                className={cn('font-mono', errors.ipAddress && 'border-destructive')}
                placeholder="192.168.1.100"
                value={form.ipAddress}
                onChange={(e) => set('ipAddress', e.target.value)}
              />
            </Field>
            <Field label="设备名称 / 站点">
              <Input
                placeholder="可选"
                value={form.deviceName}
                onChange={(e) => set('deviceName', e.target.value)}
              />
            </Field>
            <Field label="站点 ID">
              <Input
                placeholder="SITE-001"
                value={form.siteId}
                onChange={(e) => set('siteId', e.target.value)}
              />
            </Field>
            <div />
            <Field label="纬度" error={errors.latitude}>
              <Input
                className={cn(errors.latitude && 'border-destructive')}
                placeholder="39.9042"
                value={form.latitude}
                onChange={(e) => set('latitude', e.target.value)}
              />
            </Field>
            <Field label="经度" error={errors.longitude}>
              <Input
                className={cn(errors.longitude && 'border-destructive')}
                placeholder="116.4074"
                value={form.longitude}
                onChange={(e) => set('longitude', e.target.value)}
              />
            </Field>
          </CardContent>
        </Card>

        <div className="flex items-center justify-end gap-2">
          <Button variant="ghost" onClick={resetForm}>
            重置
          </Button>
          <Button onClick={handleSubmit} disabled={createDevice.isPending}>
            {createDevice.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Save className="size-4" />
            )}
            提交注册
          </Button>
        </div>
      </div>
    </PageShell>
  )
}
