import { useEffect, useState } from 'react'
import { Power, Wifi, ShieldCheck } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { PulseHex } from '@/components/viz/PulseHex'
import { useUserStore } from '@core/store/userStore'
import { useAppStore } from '@core/store/appStore'
import { usePublicOmcName, resolveOmcName } from '@core/hooks/api/useOmcName'
import { useSystemTimezone } from '@core/hooks/api/useSystemTimezone'
import { useSystemClock } from '@core/hooks/useSystemClock'

export function HUDStatusBar() {
  const navigate = useNavigate()
  const user = useUserStore((s) => s.currentUser)
  const clearAuth = useUserStore((s) => s.clearAuth)

  // HUD 左上角主标题跟随「OMC 名称」配置（公开通道），空回退 'STARFORGE'。
  const { omcName } = usePublicOmcName()
  const storedOmcName = useAppStore((s) => s.omcName)
  const brandTitle = resolveOmcName(omcName ?? storedOmcName, 'STARFORGE')

  // 系统时区（#459 子单 D）：登录后拉取写入 appStore，供顶部只读时钟 + 时间筛选/显示。
  useSystemTimezone()
  // 只读时钟：按系统时区每秒刷新「时区 + 实时时间」，替代原浏览器本地 new Date()。
  const { timezoneLabel, time: clockTime, date: clockDate } = useSystemClock()

  const [vitals, setVitals] = useState({ cpu: 38, mem: 56, net: 72, sess: 64, alm: 21 })

  useEffect(() => {
    const t = setInterval(() => {
      setVitals((v) => ({
        cpu: clamp(v.cpu + jitter()),
        mem: clamp(v.mem + jitter() * 0.5),
        net: clamp(v.net + jitter()),
        sess: clamp(v.sess + jitter() * 0.6),
        alm: clamp(v.alm + jitter() * 0.4, 0, 60),
      }))
    }, 1800)
    return () => clearInterval(t)
  }, [])

  const onLogout = () => {
    clearAuth()
    navigate('/login')
  }

  return (
    <header className="relative z-20 flex h-14 items-center gap-6 border-b border-cyan-500/15 bg-gradient-to-b from-[#03050d]/90 to-[#03050d]/40 px-6 backdrop-blur-sm">
      {/* LOGO + 系统脉搏 */}
      <div className="flex items-center gap-3">
        <div className="relative flex size-9 items-center justify-center hex bg-cyan-500/20 border border-cyan-400/60 shadow-[0_0_18px_rgba(0,240,255,0.55)]">
          <Power className="size-4 text-cyan-300" />
        </div>
        <div>
          <div className="font-display text-sm font-bold tracking-[0.2em] text-cyan-100">
            {brandTitle}
          </div>
          <div className="font-mono text-[10px] tracking-[0.3em] text-cyan-300/60">
            OMC · v3.0 · BRIDGE
          </div>
        </div>
      </div>

      <div className="h-7 w-px bg-cyan-500/20" />

      {/* 系统脉冲六边形阵列 */}
      <div className="flex items-center gap-3">
        <PulseHex label="CPU" value={vitals.cpu} color="#00f0ff" />
        <PulseHex label="MEM" value={vitals.mem} color="#a855f7" />
        <PulseHex label="NET" value={vitals.net} color="#00ff88" />
        <PulseHex label="SES" value={vitals.sess} color="#ffaa00" />
        <PulseHex
          label="ALM"
          value={vitals.alm}
          color={vitals.alm > 40 ? '#ff2d6f' : '#5b9eff'}
        />
      </div>

      <div className="ml-auto flex items-center gap-6">
        {/* 链路状态 */}
        <div className="hidden items-center gap-3 text-[10px] uppercase tracking-[0.2em] md:flex">
          <span className="flex items-center gap-1 text-emerald-300/90">
            <Wifi className="size-3" /> ACS LINK · UP
          </span>
          <span className="flex items-center gap-1 text-cyan-300/80">
            <ShieldCheck className="size-3" /> SECURE · TLS1.3
          </span>
        </div>

        {/* 时钟（#459 子单 D）：只读，按系统时区显示「时区 + 实时时间」 */}
        <div className="text-right" title={`当前系统时区: ${timezoneLabel}`}>
          <div className="lcd text-2xl leading-none">{clockTime}</div>
          <div className="font-mono text-[10px] tracking-[0.2em] text-cyan-300/55">
            {clockDate} · {timezoneLabel}
          </div>
        </div>

        <div className="h-7 w-px bg-cyan-500/20" />

        {/* 用户 */}
        <button
          type="button"
          onClick={onLogout}
          className="group flex items-center gap-2 rounded-sm border border-cyan-500/20 bg-cyan-500/5 px-3 py-1.5 hover:border-cyan-400/70 hover:bg-cyan-500/10"
        >
          <span className="size-2 rounded-full bg-emerald-400 shadow-[0_0_8px_currentColor]" />
          <span className="font-mono text-xs text-cyan-100">
            {user?.displayName || user?.username || 'OPERATOR'}
          </span>
          <span className="text-[9px] uppercase tracking-[0.2em] text-cyan-300/60 group-hover:text-rose-300">
            EJECT
          </span>
        </button>
      </div>
    </header>
  )
}

function clamp(v: number, lo = 5, hi = 95) {
  return Math.max(lo, Math.min(hi, v))
}
function jitter() {
  return (Math.random() - 0.5) * 16
}
