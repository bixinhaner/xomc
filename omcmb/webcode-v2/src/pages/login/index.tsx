import { useState, useCallback, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Activity, Loader2, RefreshCw, ShieldCheck } from 'lucide-react'
import type { AxiosError } from 'axios'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useT } from '@/hooks/useT'

import { authApi } from '@core/services/api/authApi'
import type { CaptchaChallenge, CaptchaCredentials } from '@core/services/api/authApi'
import { useUserStore } from '@core/store/userStore'
import { usePublicOmcName, resolveOmcName } from '@core/hooks/api/useOmcName'
import { getI18nKeyByBizCode } from '@core/i18n/bizCodeMessages'

export function LoginPage() {
  const navigate = useNavigate()
  const t = useT()
  const setTokenPair = useUserStore((s) => s.setTokenPair)
  const login = useUserStore((s) => s.login)
  // Issue #649：标记必须改密，登录后阻塞 + 强制 logout（v2 暂无内置改密入口）。
  const setMustChangePassword = useUserStore((s) => s.setMustChangePassword)
  const logout = useUserStore((s) => s.logout)

  // 登录页品牌标题跟随「OMC 名称」配置（免登录公开通道），空回退 'OMC · v2'。
  const { omcName } = usePublicOmcName()
  const brandTitle = resolveOmcName(omcName, 'OMC · v2')

  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Issue #649：必须改密时显示阻塞提示卡片，点确认 → 退出登录回登录页。
  const [mustChangeOpen, setMustChangeOpen] = useState(false)
  // Issue #687: 图形验证码状态
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [captchaData, setCaptchaData] = useState<CaptchaChallenge | null>(null)
  const [captchaLoading, setCaptchaLoading] = useState(false)
  const [captchaError, setCaptchaError] = useState(false)
  const [captchaInput, setCaptchaInput] = useState('')

  // Issue #687: 加载验证码图片
  const loadCaptcha = useCallback(async () => {
    setCaptchaLoading(true)
    setCaptchaError(false)
    try {
      const data = await authApi.getCaptcha()
      setCaptchaData(data)
      setCaptchaInput('')
    } catch {
      setCaptchaError(true)
      setCaptchaData(null)
    } finally {
      setCaptchaLoading(false)
    }
  }, [])

  // 验证码状态变为“需要”时自动拉取
  useEffect(() => {
    if (captchaRequired && !captchaData && !captchaLoading) {
      loadCaptcha()
    }
  }, [captchaRequired, captchaData, captchaLoading, loadCaptcha])

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      // Issue #687: 若验证码已触发，附带 captcha 参数
      let captchaCreds: CaptchaCredentials | undefined
      if (captchaRequired && captchaData && captchaInput) {
        captchaCreds = {
          captchaId: captchaData.captchaId,
          captchaAnswer: captchaInput,
        }
      }

      const tokens = await authApi.login(username, password, captchaCreds)
      setTokenPair(tokens)

      // 登录成功后清除验证码状态
      setCaptchaRequired(false)
      setCaptchaData(null)

      const me = await authApi.getMe()
      login(me)
      // Issue #649：v2 没有内置改密 Modal/页面（历史欠债），仅阻塞登录入口并强制
      // 用户退回登录页，告知"请到 v1 修改密码后再来"。后端硬规则仍会拒绝任何业务调用。
      if (tokens.must_change_password) {
        setMustChangePassword(true)
        setMustChangeOpen(true)
        return
      }
      navigate('/dashboard')
    } catch (err) {
      // Issue #687: 检测 biz_code=7010（需要验证码）或 7011（验证码错误）
      // http 拦截器将 biz_code 暴露为 err.bizCode（而非 response.data.biz_code）
      const axiosErr = err as AxiosError<{ biz_code?: number }> & { bizCode?: number; userMessage?: string }
      const bizCode = axiosErr.bizCode ?? axiosErr.response?.data?.biz_code

      if (bizCode === 7010) {
        setCaptchaRequired(true)
        setError(t('login.captcha.required'))
        return
      }
      if (bizCode === 7011) {
        loadCaptcha()
        setError(t('login.captcha.invalid'))
        return
      }

      // Issue #730: 按 biz_code 查语料，fallback 后端 msg 或默认语料
      const i18nKey = getI18nKeyByBizCode(bizCode)
      if (i18nKey) {
        // 带参数的错误码（7012 账号临时锁定 / 7014 IP 限流）：从后端 msg 提取数字
        if (bizCode === 7012) {
          const match = axiosErr.userMessage?.match(/(\d+)/)
          setError(t(i18nKey, { minutes: match ? match[1] : '?' }))
          return
        }
        if (bizCode === 7014) {
          const match = axiosErr.userMessage?.match(/(\d+)/)
          setError(t(i18nKey, { seconds: match ? match[1] : '?' }))
          return
        }
        // 无参数的错误码 — 直接用前端语料
        setError(t(i18nKey))
        return
      }
      setError(axiosErr.userMessage || t('login.failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen grid place-items-center bg-gradient-to-br from-background via-muted/50 to-accent/30 px-4">
      <Card className="w-full max-w-md border-border/60 shadow-xl">
        <CardHeader className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <Activity className="size-6" />
            <span className="text-base font-semibold tracking-wide">{brandTitle}</span>
          </div>
          <CardTitle>欢迎回来</CardTitle>
          <CardDescription>运维与网管控制台 · 新一代界面</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">账号</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            {/* Issue #687: 验证码（仅当后端要求时显示） */}
            {captchaRequired && (
              <div className="space-y-2">
                <Label htmlFor="captcha">验证码</Label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <ShieldCheck className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                    <Input
                      id="captcha"
                      className="pl-9"
                      placeholder="请输入图中字符"
                      maxLength={5}
                      value={captchaInput}
                      onChange={(e) => setCaptchaInput(e.target.value)}
                      required
                    />
                  </div>
                  <button
                    type="button"
                    onClick={loadCaptcha}
                    className="w-[120px] h-10 border rounded-md overflow-hidden flex items-center justify-center bg-muted/50 hover:bg-muted transition-colors"
                    title="点击刷新验证码"
                  >
                    {captchaLoading ? (
                      <Loader2 className="size-5 animate-spin text-muted-foreground" />
                    ) : captchaError ? (
                      <RefreshCw className="size-5 text-destructive" />
                    ) : captchaData ? (
                      <img
                        src={captchaData.image}
                        alt="captcha"
                        className="w-full h-full object-contain"
                      />
                    ) : (
                      <RefreshCw className="size-5 text-muted-foreground" />
                    )}
                  </button>
                </div>
              </div>
            )}

            {error && (
              <p className="text-sm text-destructive" role="alert">
                {error}
              </p>
            )}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="animate-spin" /> 登录中…
                </>
              ) : (
                '登录'
              )}
            </Button>
          </form>

          {mustChangeOpen && (
            <div
              className="mt-4 rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm"
              role="alert"
            >
              <p className="font-semibold text-destructive">需修改密码</p>
              <p className="mt-2 text-foreground">
                为保障账号安全，请立即修改密码。请使用主控制台（v1 皮肤）完成修改后再登录。
              </p>
              <Button
                className="mt-3 w-full"
                variant="destructive"
                onClick={() => {
                  setMustChangeOpen(false)
                  logout()
                }}
              >
                返回登录
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
