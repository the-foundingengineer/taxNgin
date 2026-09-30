import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { WizardShell } from '@/components/wizard/WizardShell'
import { MoneyDisplay } from '@/components/shared/MoneyDisplay'
import { returnsApi } from '@/lib/api/returns'
import { returnKey } from '@/hooks/useReturn'
import { parseNairaToKobo } from '@/lib/money'
import type { ReturnProjection, ReliefCode, ClaimReliefRequest } from '@/types/api'

const RELIEFS: { code: ReliefCode; label: string; hint: string; requiresEvidence: boolean }[] = [
  { code: 'RENT_RELIEF', label: 'Rent relief', hint: '20% of annual rent paid, capped at ₦200,000', requiresEvidence: true },
  { code: 'PENSION', label: 'Pension contribution', hint: 'Your employee pension contribution (not employer)', requiresEvidence: true },
  { code: 'NHF', label: 'National Housing Fund (NHF)', hint: 'Monthly NHF deductions from salary', requiresEvidence: true },
  { code: 'NHIS', label: 'NHIS contribution', hint: 'National Health Insurance contributions', requiresEvidence: true },
  { code: 'LIFE_ASSURANCE', label: 'Life assurance premium', hint: 'For yourself and/or spouse', requiresEvidence: true },
  { code: 'MEDICAL', label: 'Medical expenses', hint: 'Medical bills and health insurance premiums', requiresEvidence: true },
  { code: 'HOUSING_LOAN_INTEREST', label: 'Home loan interest', hint: 'Interest on loan for owner-occupied property only', requiresEvidence: true },
  { code: 'BUSINESS_EXPENSES', label: 'Business expenses', hint: 'Expenses wholly and exclusively for business income', requiresEvidence: true },
]

const schema = z.object({
  code:         z.string().min(1),
  naira:        z.string().min(1, 'Enter the amount'),
  has_evidence: z.boolean(),
})

type FormValues = z.infer<typeof schema>

interface Step3ReliefsProps {
  returnId: string
  proj: ReturnProjection
  onNext: () => void
}

export function Step3Reliefs({ returnId, proj, onNext }: Step3ReliefsProps) {
  const [selectedCode, setSelectedCode] = useState<ReliefCode | null>(null)
  const qc = useQueryClient()

  const addMutation = useMutation({
    mutationFn: (req: ClaimReliefRequest) => returnsApi.claimRelief(returnId, req),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  const removeMutation = useMutation({
    mutationFn: (code: string) => returnsApi.removeRelief(returnId, code),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  const { register, handleSubmit, setValue, reset, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { code: '', naira: '', has_evidence: false },
  })

  function openDialog(code: ReliefCode) {
    setSelectedCode(code)
    setValue('code', code)
    reset({ code, naira: '', has_evidence: false })
  }

  async function onValid(values: FormValues) {
    const kobo = parseNairaToKobo(values.naira)
    await addMutation.mutateAsync({ code: values.code as ReliefCode, amount_kobo: kobo, has_evidence: values.has_evidence })
    setSelectedCode(null)
    reset()
  }

  return (
    <WizardShell step={3} returnId={returnId} onNext={onNext} nextLabel="Continue to tax computation">
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Reliefs</h2>
          <p className="text-neutral-500 text-sm mt-1">
            Reliefs reduce your taxable income. Add any that apply to you.
          </p>
        </div>

        <div className="space-y-2">
          {RELIEFS.map(r => {
            const existing = (proj.Reliefs ?? {})[r.code]
            return (
              <Card key={r.code}>
                <CardContent className="py-3 flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-neutral-900 text-sm">{r.label}</p>
                    <p className="text-xs text-neutral-400">{r.hint}</p>
                  </div>
                  {existing ? (
                    <div className="flex items-center gap-2">
                      <MoneyDisplay kobo={existing.AmountKobo} mono className="text-sm font-semibold" />
                      <button
                        onClick={() => removeMutation.mutate(r.code)}
                        className="text-neutral-300 hover:text-danger transition-colors p-1"
                        aria-label="Remove relief"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  ) : (
                    <Button size="sm" variant="ghost-brand" onClick={() => openDialog(r.code)}>
                      <Plus className="h-3.5 w-3.5 mr-1" />
                      Add
                    </Button>
                  )}
                </CardContent>
              </Card>
            )
          })}
        </div>

        <Dialog open={selectedCode !== null} onOpenChange={(o) => { if (!o) { setSelectedCode(null); reset() } }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                {RELIEFS.find(r => r.code === selectedCode)?.label ?? 'Add relief'}
              </DialogTitle>
            </DialogHeader>
            <form onSubmit={handleSubmit(onValid)} className="space-y-4">
              <div className="space-y-2">
                <Label>Amount (₦)</Label>
                <Input {...register('naira')} inputMode="decimal" placeholder="e.g. 48,000" />
                {errors.naira && <p className="text-xs text-danger">{errors.naira.message}</p>}
              </div>
              <label className="flex items-start gap-3 cursor-pointer">
                <input type="checkbox" {...register('has_evidence')} className="mt-0.5 h-4 w-4 accent-brand-400" />
                <span className="text-sm text-neutral-700">I have supporting evidence (receipt, statement, etc.)</span>
              </label>
              <div className="flex gap-3">
                <Button type="button" variant="outline" className="flex-1" onClick={() => { setSelectedCode(null); reset() }}>
                  Cancel
                </Button>
                <Button type="submit" className="flex-1" disabled={isSubmitting}>
                  {isSubmitting ? 'Adding…' : 'Add relief'}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </WizardShell>
  )
}
