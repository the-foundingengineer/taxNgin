import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { parseNairaToKobo } from '@/lib/money'
import type { IncomeCategory, DeclareIncomeRequest } from '@/types/api'

const CATEGORIES: { value: IncomeCategory; label: string }[] = [
  { value: 'EMPLOYMENT',      label: 'Employment income (salary, wages)' },
  { value: 'BUSINESS',        label: 'Business income' },
  { value: 'PROFESSIONAL',    label: 'Professional fees' },
  { value: 'RENT_RECEIVED',   label: 'Rental income' },
  { value: 'DIVIDEND',        label: 'Dividends' },
  { value: 'INTEREST',        label: 'Interest income' },
  { value: 'ROYALTY',         label: 'Royalties' },
  { value: 'DIGITAL_ASSET_GAIN', label: 'Crypto / digital assets' },
  { value: 'CAPITAL_GAIN',    label: 'Capital gains' },
  { value: 'FOREIGN',         label: 'Foreign income' },
  { value: 'PENSION_RECEIVED', label: 'Pension received' },
  { value: 'GRATUITY',        label: 'Gratuity / severance' },
  { value: 'OTHER',           label: 'Other income' },
]

const schema = z.object({
  category:   z.string().min(1, 'Select a category'),
  naira:      z.string().min(1, 'Enter an amount'),
  source_ref: z.string().min(1, 'Enter the source (employer name, platform, etc.)'),
  label:      z.string().optional(),
})

type FormValues = z.infer<typeof schema>

interface IncomeFormProps {
  open: boolean
  onClose: () => void
  onSubmit: (req: DeclareIncomeRequest) => Promise<void>
}

export function IncomeForm({ open, onClose, onSubmit }: IncomeFormProps) {
  const { register, handleSubmit, setValue, watch, reset, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { category: '', naira: '', source_ref: '', label: '' },
  })

  async function onValid(values: FormValues) {
    let kobo: number
    try {
      kobo = parseNairaToKobo(values.naira)
    } catch {
      return
    }
    await onSubmit({
      category:   values.category as IncomeCategory,
      amount_kobo: kobo,
      source_ref: values.source_ref,
      label:      values.label ?? '',
    })
    reset()
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { reset(); onClose() } }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add income</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onValid)} className="space-y-4">
          <div className="space-y-2">
            <Label>Income type</Label>
            <Select
              value={watch('category')}
              onValueChange={(v) => setValue('category', v, { shouldValidate: true })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a category" />
              </SelectTrigger>
              <SelectContent>
                {CATEGORIES.map(c => (
                  <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.category && <p className="text-xs text-danger">{errors.category.message}</p>}
          </div>

          <div className="space-y-2">
            <Label>Annual amount (₦)</Label>
            <Input
              {...register('naira')}
              placeholder="e.g. 4,800,000"
              inputMode="decimal"
            />
            {errors.naira && <p className="text-xs text-danger">{errors.naira.message}</p>}
          </div>

          <div className="space-y-2">
            <Label>Source</Label>
            <Input
              {...register('source_ref')}
              placeholder="Employer name, bank, or platform"
            />
            {errors.source_ref && <p className="text-xs text-danger">{errors.source_ref.message}</p>}
          </div>

          <div className="space-y-2">
            <Label>Description (optional)</Label>
            <Input {...register('label')} placeholder="e.g. Annual salary 2026" />
          </div>

          <div className="flex gap-3 pt-2">
            <Button type="button" variant="outline" className="flex-1" onClick={() => { reset(); onClose() }}>
              Cancel
            </Button>
            <Button type="submit" className="flex-1" disabled={isSubmitting}>
              {isSubmitting ? 'Adding…' : 'Add income'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
