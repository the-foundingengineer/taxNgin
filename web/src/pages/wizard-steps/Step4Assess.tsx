import { useEffect } from 'react'
import { AlertCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { WizardShell } from '@/components/wizard/WizardShell'
import { TaxSummary } from '@/components/assessment/TaxSummary'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { useAssess } from '@/hooks/useReturn'
import type { ReturnProjection } from '@/types/api'

interface Step4AssessProps {
  returnId: string
  proj: ReturnProjection
  onNext: () => void
}

export function Step4Assess({ returnId, proj, onNext }: Step4AssessProps) {
  const assess = useAssess(returnId)

  // Auto-run assessment on mount if not yet done or stale.
  useEffect(() => {
    if (!proj.Assessment || proj.AssessmentStale) {
      assess.mutate()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const isLoading = assess.isPending
  const hasAssessment = !!proj.Assessment && !proj.AssessmentStale

  return (
    <WizardShell
      step={4}
      returnId={returnId}
      onNext={onNext}
      nextDisabled={!hasAssessment}
      nextLoading={isLoading}
      nextLabel="Continue to validation"
    >
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Your tax</h2>
          <p className="text-neutral-500 text-sm mt-1">
            We've computed your tax based on your income and reliefs.
          </p>
        </div>

        {assess.isError && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              {assess.error instanceof Error ? assess.error.message : 'Could not compute tax — please try again.'}
            </AlertDescription>
          </Alert>
        )}

        {isLoading && (
          <Card>
            <CardContent className="py-12 flex flex-col items-center gap-4">
              <LoadingSpinner className="h-8 w-8" />
              <p className="text-sm text-neutral-500">Computing your tax…</p>
            </CardContent>
          </Card>
        )}

        {proj.AssessmentStale && !isLoading && (
          <Alert variant="warning">
            <AlertDescription>
              Your income or reliefs changed since the last computation. Tap "Recompute" to update.
            </AlertDescription>
          </Alert>
        )}

        {proj.Assessment && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Tax computation</CardTitle>
            </CardHeader>
            <CardContent className="pt-0">
              <TaxSummary assessment={proj.Assessment} />
            </CardContent>
          </Card>
        )}
      </div>
    </WizardShell>
  )
}
