import { useNavigate } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const STEP_LABELS = [
  'Identity',
  'Income',
  'Reliefs',
  'Compute',
  'Validate',
  'Sign',
  'File',
]

interface WizardShellProps {
  step: number          // 1-based
  totalSteps?: number
  returnId: string
  children: React.ReactNode
  onNext?: () => void
  onBack?: () => void
  nextLabel?: string
  nextDisabled?: boolean
  nextLoading?: boolean
  hideNext?: boolean
}

export function WizardShell({
  step,
  totalSteps = 7,
  children,
  onNext,
  onBack,
  nextLabel = 'Continue',
  nextDisabled,
  nextLoading,
  hideNext,
}: WizardShellProps) {
  const navigate = useNavigate()
  const progress = ((step - 1) / totalSteps) * 100

  function handleBack() {
    if (onBack) { onBack(); return }
    navigate(-1)
  }

  return (
    <div className="min-h-screen bg-neutral-50 flex flex-col">
      {/* Slim progress bar — pinned to very top */}
      <div className="fixed top-0 inset-x-0 z-50 h-1.5 bg-neutral-200">
        <div
          className="h-full bg-brand-400 transition-all duration-500"
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* Header */}
      <div className="max-w-md mx-auto w-full px-4 pt-8 pb-2 flex items-center gap-3">
        <button
          onClick={handleBack}
          className="text-neutral-400 hover:text-neutral-700 transition-colors p-1"
          aria-label="Go back"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="flex-1">
          <p className="text-xs text-neutral-400 font-medium uppercase tracking-wider">
            Step {step} of {totalSteps}
          </p>
          <p className="text-sm font-semibold text-neutral-700">
            {STEP_LABELS[step - 1]}
          </p>
        </div>
        {/* Step dots */}
        <div className="flex gap-1">
          {Array.from({ length: totalSteps }).map((_, i) => (
            <div
              key={i}
              className={cn(
                'rounded-full transition-all',
                i + 1 < step  ? 'w-2 h-2 bg-brand-400' :
                i + 1 === step ? 'w-3 h-2 bg-brand-400' :
                                  'w-2 h-2 bg-neutral-300',
              )}
            />
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 max-w-md mx-auto w-full px-4 py-4">
        {children}
      </div>

      {/* Fixed bottom CTA */}
      {!hideNext && (
        <div className="sticky bottom-0 bg-neutral-50 border-t border-neutral-200 px-4 py-4 max-w-md mx-auto w-full">
          <Button
            className="w-full"
            onClick={onNext}
            disabled={nextDisabled || nextLoading}
          >
            {nextLoading ? 'Working…' : nextLabel}
          </Button>
        </div>
      )}
    </div>
  )
}
