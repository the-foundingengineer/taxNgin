import { MoneyDisplay } from '@/components/shared/MoneyDisplay'
import type { AssessmentSnapshot } from '@/types/api'
import { cn } from '@/lib/utils'

interface TaxSummaryProps {
  assessment: AssessmentSnapshot
  className?: string
}

interface RowProps {
  label: string
  kobo: number
  emphasis?: boolean
  signed?: boolean
}

function Row({ label, kobo, emphasis, signed }: RowProps) {
  return (
    <div className={cn('flex items-center justify-between py-3 border-b border-neutral-100 last:border-0', emphasis && 'font-semibold')}>
      <span className={cn('text-sm', emphasis ? 'text-neutral-900' : 'text-neutral-600')}>{label}</span>
      <MoneyDisplay kobo={kobo} mono signed={signed} className={cn(emphasis ? 'text-neutral-900' : 'text-neutral-700')} />
    </div>
  )
}

export function TaxSummary({ assessment, className }: TaxSummaryProps) {
  const isRefund = assessment.BalanceKobo < 0
  const isBalanced = assessment.BalanceKobo === 0

  return (
    <div className={cn('divide-y divide-neutral-100', className)}>
      <Row label="Chargeable income" kobo={assessment.ChargeableIncomeKobo} />
      <Row label="Tax payable"        kobo={assessment.TaxPayableKobo} />
      {assessment.TotalCreditsKobo > 0 && (
        <Row label="Tax credits (PAYE &amp; other)" kobo={-assessment.TotalCreditsKobo} signed />
      )}
      <div className={cn(
        'flex items-center justify-between py-4 rounded-lg px-4 mt-3',
        isRefund   ? 'bg-success-light' :
        isBalanced ? 'bg-neutral-100' :
                     'bg-warning-light',
      )}>
        <span className="font-semibold text-neutral-900">
          {isRefund ? 'Refund due' : isBalanced ? 'Nothing owed' : 'Balance to pay'}
        </span>
        <MoneyDisplay
          kobo={Math.abs(assessment.BalanceKobo)}
          mono
          className={cn(
            'text-xl font-bold',
            isRefund ? 'text-success-dark' : isBalanced ? 'text-neutral-700' : 'text-warning-dark',
          )}
        />
      </div>
    </div>
  )
}
