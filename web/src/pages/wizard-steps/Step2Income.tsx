import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { WizardShell } from '@/components/wizard/WizardShell'
import { MoneyDisplay } from '@/components/shared/MoneyDisplay'
import { IncomeForm } from '@/components/income/IncomeForm'
import { returnsApi } from '@/lib/api/returns'
import { returnKey } from '@/hooks/useReturn'
import type { ReturnProjection, DeclareIncomeRequest } from '@/types/api'

const CATEGORY_LABELS: Record<string, string> = {
  EMPLOYMENT:      'Employment',
  BUSINESS:        'Business',
  PROFESSIONAL:    'Professional',
  RENT_RECEIVED:   'Rental',
  DIVIDEND:        'Dividends',
  INTEREST:        'Interest',
  ROYALTY:         'Royalties',
  DIGITAL_ASSET_GAIN: 'Digital assets',
  CAPITAL_GAIN:    'Capital gain',
  FOREIGN:         'Foreign',
  PENSION_RECEIVED:'Pension',
  GRATUITY:        'Gratuity',
  OTHER:           'Other',
}

interface Step2IncomeProps {
  returnId: string
  proj: ReturnProjection
  onNext: () => void
}

export function Step2Income({ returnId, proj, onNext }: Step2IncomeProps) {
  const [showForm, setShowForm] = useState(false)
  const qc = useQueryClient()

  const income = Object.values(proj.Income ?? {})
  const hasIncome = income.length > 0

  const addMutation = useMutation({
    mutationFn: (req: DeclareIncomeRequest) => returnsApi.declareIncome(returnId, req),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  const removeMutation = useMutation({
    mutationFn: (id: string) => returnsApi.removeIncome(returnId, id),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  return (
    <WizardShell
      step={2}
      returnId={returnId}
      onNext={onNext}
      nextDisabled={!hasIncome && !proj.Income}
      nextLabel="Continue to reliefs"
    >
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Income</h2>
          <p className="text-neutral-500 text-sm mt-1">
            Add all income earned during the {proj.YearOfAssessment} tax year.
          </p>
        </div>

        {income.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center">
              <p className="text-neutral-400 text-sm mb-4">No income added yet</p>
              <Button variant="outline" onClick={() => setShowForm(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add income
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-2">
            {income.map(line => (
              <Card key={line.ID}>
                <CardContent className="py-3 flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-neutral-900 text-sm">
                      {CATEGORY_LABELS[line.Category] ?? line.Category}
                    </p>
                    <p className="text-xs text-neutral-500 truncate">{line.SourceRef}</p>
                  </div>
                  <MoneyDisplay kobo={line.AmountKobo} mono className="text-sm font-semibold" />
                  <button
                    onClick={() => removeMutation.mutate(line.ID)}
                    disabled={removeMutation.isPending}
                    className="text-neutral-300 hover:text-danger transition-colors p-1"
                    aria-label="Remove income line"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </CardContent>
              </Card>
            ))}

            <Button variant="outline" className="w-full" onClick={() => setShowForm(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Add another income source
            </Button>
          </div>
        )}

        <IncomeForm
          open={showForm}
          onClose={() => setShowForm(false)}
          onSubmit={async (req) => { await addMutation.mutateAsync(req) }}
        />
      </div>
    </WizardShell>
  )
}
