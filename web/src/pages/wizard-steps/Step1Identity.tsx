import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { WizardShell } from '@/components/wizard/WizardShell'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const IDENTITY_KEY = (returnId: string) => `tx_identity_${returnId}`

const schema = z.object({
  tax_id:          z.string().min(1, 'Tax ID is required'),
  tax_id_verified: z.boolean(),
  nin:             z.string().length(11, 'NIN must be 11 digits').regex(/^\d+$/, 'Digits only'),
  nil_declaration: z.boolean(),
  duplicate_check_ok: z.boolean(),
  accommodation_type: z.string().optional(),
  ownership_type:  z.string().optional(),
  landlord_name:   z.string().optional(),
  landlord_address: z.string().optional(),
})

export type IdentityData = z.infer<typeof schema>

export function saveIdentity(returnId: string, data: IdentityData) {
  sessionStorage.setItem(IDENTITY_KEY(returnId), JSON.stringify(data))
}

export function loadIdentity(returnId: string): IdentityData | null {
  try {
    const raw = sessionStorage.getItem(IDENTITY_KEY(returnId))
    return raw ? (JSON.parse(raw) as IdentityData) : null
  } catch {
    return null
  }
}

interface Step1IdentityProps {
  returnId: string
  onNext: () => void
}

export function Step1Identity({ returnId, onNext }: Step1IdentityProps) {
  const defaults = loadIdentity(returnId)

  const { register, handleSubmit, setValue, watch, formState: { errors } } = useForm<IdentityData>({
    resolver: zodResolver(schema),
    defaultValues: defaults ?? {
      tax_id: '', tax_id_verified: false, nin: '',
      nil_declaration: false, duplicate_check_ok: false,
      accommodation_type: '', ownership_type: '', landlord_name: '', landlord_address: '',
    },
  })

  const accommodationType = watch('accommodation_type')
  const taxId = watch('tax_id')

  function onValid(data: IdentityData) {
    saveIdentity(returnId, data)
    onNext()
  }

  return (
    <WizardShell
      step={1}
      returnId={returnId}
      onNext={handleSubmit(onValid)}
      nextDisabled={!taxId}
    >
      <div className="space-y-5">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Your identity</h2>
          <p className="text-neutral-500 text-sm mt-1">
            This information is only used during validation — it is not stored.
          </p>
        </div>

        <Card>
          <CardContent className="pt-4 space-y-4">
            <div className="space-y-2">
              <Label>Tax Identification Number (TIN)</Label>
              <Input {...register('tax_id')} placeholder="e.g. LAG-123456789" />
              <p className="text-xs text-neutral-400">
                Find your TIN on a previous LIRS assessment notice or the JTAX portal.
              </p>
              {errors.tax_id && <p className="text-xs text-danger">{errors.tax_id.message}</p>}
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input type="checkbox" {...register('tax_id_verified')} className="mt-0.5 h-4 w-4 accent-brand-400" />
              <span className="text-sm text-neutral-700">I have verified this TIN on the LIRS portal</span>
            </label>

            <div className="space-y-2">
              <Label>National Identity Number (NIN)</Label>
              <Input
                {...register('nin')}
                inputMode="numeric"
                maxLength={11}
                placeholder="11-digit NIN"
              />
              {errors.nin && <p className="text-xs text-danger">{errors.nin.message}</p>}
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input type="checkbox" {...register('nil_declaration')} className="mt-0.5 h-4 w-4 accent-brand-400" />
              <span className="text-sm text-neutral-700">
                I had no taxable income this year (nil declaration)
              </span>
            </label>

            <label className="flex items-start gap-3 cursor-pointer">
              <input type="checkbox" {...register('duplicate_check_ok')} className="mt-0.5 h-4 w-4 accent-brand-400" />
              <span className="text-sm text-neutral-700">
                I confirm I have not already filed a return for this year elsewhere
              </span>
            </label>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-4 space-y-4">
            <h3 className="font-semibold text-neutral-900">Accommodation</h3>
            <p className="text-xs text-neutral-500 -mt-2">
              Required for Page 2 of the LIRS eTax portal.
            </p>

            <div className="space-y-2">
              <Label>Accommodation type</Label>
              <Select
                value={watch('accommodation_type') ?? ''}
                onValueChange={(v) => setValue('accommodation_type', v)}
              >
                <SelectTrigger><SelectValue placeholder="Select type" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Rented">Rented</SelectItem>
                  <SelectItem value="Owner-occupied">Owner-occupied</SelectItem>
                  <SelectItem value="Company-provided">Company-provided</SelectItem>
                  <SelectItem value="Other">Other</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {accommodationType === 'Rented' && (
              <>
                <div className="space-y-2">
                  <Label>Landlord / agent name</Label>
                  <Input {...register('landlord_name')} placeholder="Full name" />
                </div>
                <div className="space-y-2">
                  <Label>Landlord / agent address</Label>
                  <Input {...register('landlord_address')} placeholder="Street address" />
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </WizardShell>
  )
}
