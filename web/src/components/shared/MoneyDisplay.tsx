import { formatNaira } from '@/lib/money'
import { cn } from '@/lib/utils'

interface MoneyDisplayProps {
  kobo: number
  className?: string
  /** Show a + prefix for positive values (useful for refunds/credits) */
  signed?: boolean
  /** Use monospace font for tabular alignment */
  mono?: boolean
}

/** The only place in the app where kobo is divided for display. */
export function MoneyDisplay({ kobo, className, signed, mono }: MoneyDisplayProps) {
  const text = signed && kobo > 0 ? `+${formatNaira(kobo)}` : formatNaira(kobo)
  return (
    <span className={cn(mono && 'font-mono tabular-nums', className)}>
      {text}
    </span>
  )
}
