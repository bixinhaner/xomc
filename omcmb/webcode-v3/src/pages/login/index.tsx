import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Lock, ScanLine, Loader2, ChevronRight, Rocket } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'
import { authApi } from '@core/services/api/authApi'
import { useUserStore } from '@core/store/userStore'
import type { User } from '@core/types/system'

export function LoginPage() {
  const navigate = useNavigate()
  const setTokenPair = useUserStore((s) => s.setTokenPair)
  const login = useUserStore((s) => s.login)

  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const tokens = await authApi.login(username, password)
      setTokenPair(tokens)
      const me = await authApi.getMe()
      login(me)
      navigate('/bridge')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'AUTH FAILED'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  /** DEMO 模式：注入虚拟凭据，直接登舰桥（无需后端） */
  const onDemo = () => {
    setTokenPair({
      access_token: 'demo-access-token',
      refresh_token: 'demo-refresh-token',
      expires_at: new Date(Date.now() + 24 * 3600 * 1000).toISOString(),
    })
    const fakeUser: User = {
      id: 'demo',
      username: 'demo',
      displayName: 'COMMANDER',
      email: 'demo@starforge.local',
      phone: '',
      role: 'admin',
      status: 'active',
      lastLoginTime: new Date().toISOString(),
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
    }
    login(fakeUser)
    navigate('/bridge')
  }

  return (
    <div className="relative h-screen w-screen overflow-hidden">
      <div className="starforge-backdrop">
        <div className="starforge-stars" />
      </div>

      {/* 中央识别环 */}
      <div className="relative z-10 flex h-full w-full items-center justify-center">
        {/* 旋转外圈 */}
        <div className="pointer-events-none absolute size-[680px] max-w-[92vw] max-h-[92vh] rounded-full border border-cyan-400/15 animate-sweep" />
        <div
          className="pointer-events-none absolute size-[520px] max-w-[80vw] max-h-[80vh] rounded-full border border-cyan-400/25"
          style={{ animation: 'sweep 12s linear reverse infinite' }}
        />
        <div className="pointer-events-none absolute size-[400px] max-w-[68vw] max-h-[68vh] rounded-full border-2 border-dashed border-cyan-400/30 animate-sweep" />
        <div className="pointer-events-none absolute size-[260px] max-w-[55vw] max-h-[55vh] rounded-full border border-cyan-400/40 shadow-[0_0_50px_rgba(0,240,255,0.3)] animate-pulseRing" />

        {/* 中央表单 */}
        <div className="relative z-20 w-[440px] max-w-[90vw] glass-strong rounded-sm p-8">
          <div className="scanline" />

          <div className="relative mb-6 flex items-center justify-between">
            <div className="flex items-center gap-2 text-cyan-300">
              <ScanLine className="size-5" />
              <span className="font-display text-sm font-bold tracking-[0.25em]">
                STARFORGE · OMC v3
              </span>
            </div>
            <span className="font-mono text-[10px] tracking-[0.2em] text-cyan-300/55">
              SECURE LINK
            </span>
          </div>

          <div className="relative mb-6">
            <div className="font-display text-3xl font-bold text-cyan-100 text-glow">
              身份验证
            </div>
            <div className="mt-1 font-mono text-[11px] tracking-[0.18em] text-cyan-300/55">
              OPERATOR CREDENTIAL REQUIRED · 请刷虹膜或输入凭据
            </div>
          </div>

          <form onSubmit={onSubmit} className="relative space-y-4">
            <div>
              <label className="mb-1 block font-mono text-[10px] uppercase tracking-[0.25em] text-cyan-300/70">
                CALLSIGN · 账号
              </label>
              <input
                className="neon-input w-full"
                value={username}
                autoComplete="username"
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>

            <div>
              <label className="mb-1 block font-mono text-[10px] uppercase tracking-[0.25em] text-cyan-300/70">
                PASSPHRASE · 密码
              </label>
              <div className="relative">
                <input
                  type="password"
                  className="neon-input w-full pl-9"
                  value={password}
                  autoComplete="current-password"
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
                <Lock className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-cyan-300/55" />
              </div>
            </div>

            {error && (
              <div className="border border-rose-400/40 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-300">
                <span className="font-bold uppercase tracking-[0.2em]">ERR</span>
                <span className="ml-2 text-rose-200/85">{error}</span>
              </div>
            )}

            <NeonButton
              type="submit"
              className="w-full justify-center !py-3 !text-sm"
              disabled={loading}
              icon={loading ? <Loader2 className="animate-spin" /> : <ChevronRight />}
            >
              {loading ? 'LINKING…' : '登入指挥舰桥'}
            </NeonButton>

            <button
              type="button"
              onClick={onDemo}
              className="group flex w-full items-center justify-center gap-2 border border-cyan-500/15 bg-cyan-500/5 px-3 py-2 font-mono text-[11px] uppercase tracking-[0.22em] text-cyan-300/65 transition-all hover:border-cyan-400/45 hover:bg-cyan-500/10 hover:text-cyan-200"
            >
              <Rocket className="size-3.5 transition-transform group-hover:translate-x-0.5" />
              DEMO · 无需后端 · 直接进入舰桥
            </button>
          </form>

          <div className="relative mt-5 flex items-center justify-between font-mono text-[10px] tracking-[0.18em] text-cyan-300/45">
            <span>BRIDGE-04 · CHANNEL 7547</span>
            <span className="animate-flicker">SIGNAL ◉ STRONG</span>
          </div>
        </div>
      </div>
    </div>
  )
}
