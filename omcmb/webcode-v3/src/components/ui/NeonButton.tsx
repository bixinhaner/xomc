import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  tone?: 'cyan' | 'danger'
  icon?: ReactNode
}

export function NeonButton({
  tone = 'cyan',
  icon,
  className,
  children,
  ...rest
}: Props) {
  return (
    <button
      type="button"
      className={cn('neon-btn', tone === 'danger' && 'danger', className)}
      {...rest}
    >
      {icon ? <span className="[&_svg]:size-3.5">{icon}</span> : null}
      {children}
    </button>
  )
}
