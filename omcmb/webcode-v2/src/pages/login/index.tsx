import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Activity, Loader2 } from 'lucide-react'

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

import { authApi } from '@core/services/api/authApi'
import { useUserStore } from '@core/store/userStore'
import { usePublicOmcName, resolveOmcName } from '@core/hooks/api/useOmcName'

export function LoginPage() {
  const navigate = useNavigate()
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

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const tokens = await authApi.login(username, password)
      setTokenPair(tokens)
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
      const msg = err instanceof Error ? err.message : '登录失败'
      setError(msg)
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
