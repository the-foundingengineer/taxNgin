import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { WizardShell } from '@/components/wizard/WizardShell'
import { TaxSummary } from '@/components/assessment/TaxSummary'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertCircle, ShieldCheck } from 'lucide-react'
import { useSign } from '@/hooks/useReturn'
import type { ReturnProjection } from '@/types/api'

interface Step6SignProps {
  returnId: string
  proj: ReturnProjection
  onNext: () => void
}

export function Step6Sign({ returnId, proj, onNext }: Step6SignProps) {
  const [declared, setDeclared] = useState(false)
  const sign = useSign(returnId)

  const isSigned = proj.State === 'SIGNED' || proj.State === 'PACK_ISSUED' ||
                   proj.State === 'SELF_ATTESTED' || proj.State === 'ACK_UPLOADED'
  const hasBlockers = (proj.BlockingErrors ?? []).length > 0
  const isStale = proj.AssessmentStale

  const canSign = declared && !hasBlockers && !isStale && !isSigned

  async function handleSign() {
    await sign.mutateAsync()
    onNext()
  }

  return (
    <WizardShell
      step={6}
      returnId={returnId}
      onNext={isSigned ? onNext : handleSign}
      nextDisabled={!isSigned && !canSign}
      nextLoading={sign.isPending}
      nextLabel={isSigned ? 'Continue to pack' : 'Sign and continue'}
    >
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Review and sign</h2>
          <p className="text-neutral-500 text-sm mt-1">
            Review your return, then sign to confirm accuracy.
          </p>
        </div>

        {hasBlockers && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              There are blocking validation issues. Go back to Step 5 and resolve them before signing.
            </AlertDescription>
          </Alert>
        )}

        {isStale && (
          <Alert variant="warning">
            <AlertDescription>
              Your assessment is out of date. Go back to Step 4 and recompute.
            </AlertDescription>
          </Alert>
        )}

        {proj.Assessment && (
          <Card>
            <CardContent className="pt-4">
              <TaxSummary assessment={proj.Assessment} />
            </CardContent>
          </Card>
        )}

        {!isSigned && (
          <Card>
            <CardContent className="pt-4">
              <label className="flex items-start gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={declared}
                  onChange={(e) => setDeclared(e.target.checked)}
                  className="mt-0.5 h-4 w-4 accent-brand-400"
                />
                <span className="text-sm text-neutral-700 leading-relaxed">
                  I declare that to the best of my knowledge, the information I have
                  provided is true, correct, and complete. I understand that making a
                  false declaration is a criminal offence under the Nigerian Tax
                  Administration Act 2025.
                </span>
              </label>
            </CardContent>
          </Card>
        )}

        {isSigned && (
          <Alert variant="success">
            <ShieldCheck className="h-4 w-4" />
            <AlertDescription>
              Your return is signed. Proceed to generate your filing pack.
            </AlertDescription>
          </Alert>
        )}
      </div>
    </WizardShell>
  )
}
