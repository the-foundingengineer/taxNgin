import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { WizardShell } from '@/components/wizard/WizardShell'
import { ValidationPanel } from '@/components/validation/ValidationPanel'
import { Card, CardContent } from '@/components/ui/card'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { returnsApi } from '@/lib/api/returns'
import { returnKey } from '@/hooks/useReturn'
import { loadIdentity } from './Step1Identity'
import type { ReturnProjection } from '@/types/api'

interface Step5ValidateProps {
  returnId: string
  proj: ReturnProjection
  onNext: () => void
}

export function Step5Validate({ returnId, proj, onNext }: Step5ValidateProps) {
  const qc = useQueryClient()
  const hasRun = proj.BlockingErrors !== null && proj.Version > 1
  const isValidated = proj.State === 'VALIDATED'

  const validate = useMutation({
    mutationFn: () => {
      const identity = loadIdentity(returnId)
      return returnsApi.validate(returnId, {
        nin:              identity?.nin ?? '',
        tax_id:           identity?.tax_id ?? '',
        tax_id_verified:  identity?.tax_id_verified ?? false,
        nil_declaration:  identity?.nil_declaration ?? false,
        duplicate_check_ok: identity?.duplicate_check_ok ?? false,
      })
    },
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  return (
    <WizardShell
      step={5}
      returnId={returnId}
      onNext={onNext}
      nextDisabled={!isValidated}
      nextLabel="Continue to sign"
    >
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Validation</h2>
          <p className="text-neutral-500 text-sm mt-1">
            We'll check your return against the LIRS validation rules.
          </p>
        </div>

        {!hasRun && !validate.isPending && (
          <Card>
            <CardContent className="py-8 flex flex-col items-center gap-4">
              <p className="text-sm text-neutral-500 text-center">
                Ready to validate. This checks your TIN, NIN, income, and reliefs against LIRS rules.
              </p>
              <Button onClick={() => validate.mutate()} disabled={validate.isPending}>
                Run validation
              </Button>
            </CardContent>
          </Card>
        )}

        {validate.isPending && (
          <Card>
            <CardContent className="py-8 flex flex-col items-center gap-3">
              <LoadingSpinner className="h-6 w-6" />
              <p className="text-sm text-neutral-500">Validating your return…</p>
            </CardContent>
          </Card>
        )}

        {hasRun && !validate.isPending && (
          <>
            <ValidationPanel
              blockingCodes={proj.BlockingErrors ?? []}
              warningCodes={proj.Warnings ?? []}
            />
            {!isValidated && (
              <Button variant="outline" className="w-full" onClick={() => validate.mutate()}>
                Re-run validation
              </Button>
            )}
          </>
        )}
      </div>
    </WizardShell>
  )
}
