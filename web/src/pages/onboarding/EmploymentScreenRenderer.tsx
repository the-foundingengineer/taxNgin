import { useOnboardingStore } from '@/stores/onboardingStore'
import { Input } from '@/components/ui/input'
import { ChevronLeft, ChevronRight } from 'lucide-react'

/**
 * Custom renderer for screen 07 (Employment).
 * Lays out fields as a visual Form H1 / PAYE annual statement so users
 * can copy figures directly from the document their employer issued.
 */
export function EmploymentScreenRenderer() {
    const profile = useOnboardingStore((s) => s.profile)
    const setAnswer = useOnboardingStore((s) => s.setAnswer)
    const goNext = useOnboardingStore((s) => s.goNextScreen)
    const goBack = useOnboardingStore((s) => s.goPrevScreen)
    const canAdvance = useOnboardingStore((s) => s.canAdvance())
    const isLast = useOnboardingStore((s) => s.currentScreenIndex >= s.screens.length - 1)
    const currentScreenIndex = useOnboardingStore((s) => s.currentScreenIndex)

    const firstName = (profile.first_name as string | undefined)?.trim() ?? ''
    const lastName = (profile.last_name as string | undefined)?.trim() ?? ''
    const taxYear = (profile.tax_year as string | undefined) ?? '2026'
    const employeeName = [firstName, lastName].filter(Boolean).join(' ') || '—'

    const get = (code: string) => profile[code]
    const set = (code: string, v: unknown) => setAnswer(code, v)
    const bool = (code: string) => get(code) === true
    const str = (code: string) => (get(code) as string | undefined) ?? ''
    const num = (code: string) => (get(code) as number | undefined) ?? 0

    // Auto-calculated fields for the slip
    const basic = num('employment_basic_salary')
    const bonuses = num('bonus_amount')
    const commissions = num('commission_amount')
    const allowances = num('taxable_allowances_total')
    const calcGross = basic + bonuses + commissions + allowances

    return (
        <div>
            {/* Section badge */}
            <div className="mb-6">
                <div
                    className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-5"
                    style={{ background: 'rgba(28,78,216,0.07)', color: '#1C4ED8', border: '1px solid rgba(28,78,216,0.14)' }}
                >
                    <span className="material-symbols-outlined text-[16px]">edit_note</span>
                    Employment details
                </div>
                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    Your job &amp; salary
                </h1>
                <p className="text-neutral-500 text-[15px] leading-relaxed">
                    Your employment details determine your PAYE credit and any remaining tax to pay.
                    Lay your annual PAYE statement (Form H1) beside this screen — the sections match.
                </p>
            </div>

            {/* ── PAYE slip document ── */}
            <div
                className="rounded-2xl overflow-hidden mb-8"
                style={{ border: '1px solid #e5e7eb', boxShadow: '0 4px 24px rgba(0,0,0,0.07)' }}
            >
                {/* Document header */}
                <div
                    className="px-6 py-4 flex items-start justify-between"
                    style={{ background: 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)' }}
                >
                    <div>
                        <div className="flex items-center gap-2 mb-1">
                            <div
                                className="w-5 h-5 rounded-full flex items-center justify-center"
                                style={{ background: 'rgba(255,255,255,0.2)' }}
                            >
                                <span className="transform rotate-45 block w-2 h-2 border-[1.5px] border-white border-t-0 border-l-0 rounded-sm" />
                            </div>
                            <span className="text-white/70 text-[11px] font-bold tracking-wide">TAX CHECK</span>
                        </div>
                        <p className="text-white text-[15px] font-bold">Annual PAYE Statement</p>
                        <p className="text-white/60 text-[11px] mt-0.5">Form H1 · Year of Assessment {taxYear}</p>
                    </div>
                    <div className="text-right">
                        <p className="text-white/50 text-[10px] uppercase tracking-widest mb-1">Employee</p>
                        <p className="text-white text-sm font-semibold">{employeeName}</p>
                        <p className="text-white/50 text-[10px] mt-0.5">1 Jan – 31 Dec {taxYear}</p>
                    </div>
                </div>

                {/* ── Section A: Employer ── */}
                <Section title="A — Employer Information" icon="business">
                    <FormRow label="Employer name" required>
                        <Input
                            value={str('employer_name')}
                            onChange={(e) => set('employer_name', e.target.value)}
                            className="h-10 rounded-xl border-neutral-200 px-3 text-sm"
                            placeholder="e.g. Dangote Industries Ltd"
                        />
                    </FormRow>

                    <FormRow label="How many employers did you have?">
                        <OptionPicker
                            options={[
                                { id: '1', label: '1' },
                                { id: '2', label: '2' },
                                { id: '3', label: '3' },
                                { id: '4_plus', label: '4+' },
                            ]}
                            value={str('employer_count')}
                            onChange={(v) => set('employer_count', v)}
                        />
                    </FormRow>

                    <FormRow label="Nigerian employer?">
                        <OptionPicker
                            options={[
                                { id: 'yes', label: 'Nigerian' },
                                { id: 'both', label: 'Both' },
                                { id: 'no', label: 'Foreign only' },
                            ]}
                            value={str('nigerian_employer')}
                            onChange={(v) => set('nigerian_employer', v)}
                        />
                    </FormRow>

                    <FormRow label="Worked physically in Nigeria?">
                        <OptionPicker
                            options={[
                                { id: 'yes', label: 'Yes' },
                                { id: 'partly', label: 'Partly' },
                                { id: 'no', label: 'No' },
                            ]}
                            value={str('worked_physically_nigeria')}
                            onChange={(v) => set('worked_physically_nigeria', v)}
                        />
                    </FormRow>
                </Section>

                {/* ── Section B: Gross Emoluments ── */}
                <Section
                    title="B — Gross Emoluments"
                    icon="payments"
                    hint="Find these figures under 'Gross Emoluments' on your Form H1"
                >
                    <FormRow label="Basic salary" required>
                        <CurrencyInput
                            value={num('employment_basic_salary')}
                            onChange={(v) => set('employment_basic_salary', v)}
                            placeholder="0"
                        />
                    </FormRow>

                    <FormRow label="Total gross salary">
                        <CurrencyInput
                            value={num('employment_gross_salary')}
                            onChange={(v) => set('employment_gross_salary', v)}
                            placeholder="Basic + all allowances"
                        />
                    </FormRow>

                    <FormRow label="Bonuses received?">
                        <BoolPicker
                            value={get('received_bonuses') as boolean | undefined}
                            onChange={(v) => set('received_bonuses', v)}
                        />
                    </FormRow>
                    {bool('received_bonuses') && (
                        <FormRow label="↳  Bonus amount" indent>
                            <CurrencyInput
                                value={num('bonus_amount')}
                                onChange={(v) => set('bonus_amount', v)}
                            />
                        </FormRow>
                    )}

                    <FormRow label="Commissions received?">
                        <BoolPicker
                            value={get('received_commissions') as boolean | undefined}
                            onChange={(v) => set('received_commissions', v)}
                        />
                    </FormRow>
                    {bool('received_commissions') && (
                        <FormRow label="↳  Commission amount" indent>
                            <CurrencyInput
                                value={num('commission_amount')}
                                onChange={(v) => set('commission_amount', v)}
                            />
                        </FormRow>
                    )}

                    <FormRow label="Allowances received?">
                        <BoolPicker
                            value={get('received_allowances') as boolean | undefined}
                            onChange={(v) => set('received_allowances', v)}
                        />
                    </FormRow>
                    {bool('received_allowances') && (
                        <FormRow label="↳  Taxable allowances total" indent>
                            <CurrencyInput
                                value={num('taxable_allowances_total')}
                                onChange={(v) => set('taxable_allowances_total', v)}
                            />
                        </FormRow>
                    )}

                    {/* Auto-sum footer */}
                    <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between">
                        <span className="text-[12px] font-bold text-neutral-500 uppercase tracking-wider">
                            Calculated Gross
                        </span>
                        <span className="text-[15px] font-bold text-blue-700">
                            ₦{calcGross.toLocaleString('en-NG')}
                        </span>
                    </div>
                </Section>

                {/* ── Section C: Benefits in kind ── */}
                <Section title="C — Benefits in Kind" icon="card_giftcard">
                    <FormRow label="Employer provided accommodation?">
                        <BoolPicker
                            value={get('employer_provided_accommodation') as boolean | undefined}
                            onChange={(v) => set('employer_provided_accommodation', v)}
                        />
                    </FormRow>
                    <FormRow label="Vehicle or other benefits provided?">
                        <BoolPicker
                            value={get('employer_provided_vehicle') as boolean | undefined}
                            onChange={(v) => set('employer_provided_vehicle', v)}
                        />
                    </FormRow>
                    <FormRow label="Any other employer benefits?">
                        <BoolPicker
                            value={get('other_employer_benefits') as boolean | undefined}
                            onChange={(v) => set('other_employer_benefits', v)}
                        />
                    </FormRow>
                </Section>

                {/* ── Section D: PAYE deducted ── */}
                <Section
                    title="D — Tax Deducted (PAYE)"
                    icon="receipt_long"
                    hint="Find 'Tax Deducted' or 'PAYE' at the bottom of your Form H1"
                >
                    <FormRow label="Did your employer deduct PAYE?">
                        <BoolPicker
                            value={get('employment_paye_deducted') as boolean | undefined}
                            onChange={(v) => set('employment_paye_deducted', v)}
                        />
                    </FormRow>
                    {bool('employment_paye_deducted') && (
                        <FormRow label="↳  Total PAYE deducted" indent>
                            <CurrencyInput
                                value={num('employment_paye_amount')}
                                onChange={(v) => set('employment_paye_amount', v)}
                            />
                        </FormRow>
                    )}
                </Section>

                {/* ── Section E: Documents ── */}
                <Section title="E — Supporting Documents" icon="folder_open">
                    <FormRow label="Do you have your annual PAYE statement?">
                        <OptionPicker
                            options={[
                                { id: 'yes', label: 'Yes' },
                                { id: 'no', label: 'No' },
                                { id: 'not_provided', label: 'Not issued' },
                                { id: 'not_sure', label: 'Not sure' },
                            ]}
                            value={str('has_annual_paye_statement')}
                            onChange={(v) => set('has_annual_paye_statement', v)}
                        />
                    </FormRow>
                    {str('has_annual_paye_statement') === 'yes' && (
                        <FormRow label="↳  Upload your statement" indent>
                            <label
                                className="flex items-center gap-3 px-4 py-3 rounded-xl cursor-pointer transition-all hover:border-blue-300"
                                style={{ border: '2px dashed #d1d5db', background: '#fafafa' }}
                            >
                                <span className="material-symbols-outlined text-[22px] text-neutral-400">upload_file</span>
                                <span className="text-sm text-neutral-500">Click to upload Form H1 / PAYE certificate</span>
                            </label>
                        </FormRow>
                    )}
                </Section>
            </div>

            {/* Navigation */}
            <div className="flex items-center gap-3">
                {currentScreenIndex > 0 && (
                    <button
                        type="button"
                        onClick={goBack}
                        className="h-12 px-5 rounded-xl text-sm font-semibold flex items-center gap-1.5 transition-colors"
                        style={{ border: '1.5px solid #e5e7eb', color: '#4b5563', background: 'white' }}
                    >
                        <ChevronLeft className="w-4 h-4" /> Back
                    </button>
                )}
                <button
                    type="button"
                    onClick={goNext}
                    disabled={!canAdvance}
                    className="h-12 px-8 rounded-xl text-white font-semibold text-[15px] flex items-center gap-2 transition-all hover:gap-3 disabled:opacity-40"
                    style={{
                        background: isLast
                            ? 'linear-gradient(135deg, #1C4ED8 0%, #45B4C2 100%)'
                            : 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)',
                    }}
                >
                    {isLast ? (
                        <>
                            <span className="material-symbols-outlined text-[20px]">celebration</span>
                            Let's start filing
                        </>
                    ) : (
                        <>
                            Continue <ChevronRight className="w-4 h-4" />
                        </>
                    )}
                </button>
            </div>
        </div>
    )
}

// ---------------------------------------------------------------------------
// Shared sub-components
// ---------------------------------------------------------------------------

function Section({
    title,
    icon,
    hint,
    children,
}: {
    title: string
    icon: string
    hint?: string
    children: React.ReactNode
}) {
    return (
        <div className="border-t border-neutral-100">
            {/* Section header */}
            <div
                className="flex items-center gap-2.5 px-6 py-3"
                style={{ background: 'rgba(28,78,216,0.03)', borderBottom: '1px solid rgba(28,78,216,0.07)' }}
            >
                <span
                    className="material-symbols-outlined text-[16px]"
                    style={{ color: '#1C4ED8' }}
                >
                    {icon}
                </span>
                <span className="text-[12px] font-bold text-neutral-700 uppercase tracking-wider">{title}</span>
                {hint && (
                    <span className="ml-auto text-[10px] font-medium text-neutral-400 flex items-center gap-1">
                        <span className="material-symbols-outlined text-[12px]">info</span>
                        {hint}
                    </span>
                )}
            </div>
            {/* Section rows */}
            <div className="px-6 py-4 space-y-4 bg-white">
                {children}
            </div>
        </div>
    )
}

function FormRow({
    label,
    required,
    indent,
    children,
}: {
    label: string
    required?: boolean
    indent?: boolean
    children: React.ReactNode
}) {
    return (
        <div className={`flex items-start gap-4 ${indent ? 'pl-4' : ''}`}>
            <div className="flex-1 min-w-0">
                <p className="text-[12px] font-semibold text-neutral-500 mb-1.5 leading-tight">
                    {label}
                    {required && <span className="text-red-400 ml-0.5">*</span>}
                </p>
                {children}
            </div>
        </div>
    )
}

function CurrencyInput({
    value,
    onChange,
    placeholder = '0',
}: {
    value: number | string
    onChange: (v: number | '') => void
    placeholder?: string
}) {
    return (
        <div className="relative max-w-[220px]">
            <span
                className="absolute left-3 top-1/2 -translate-y-1/2 text-sm font-bold"
                style={{ color: '#1C4ED8' }}
            >
                ₦
            </span>
            <Input
                type="number"
                value={value}
                onChange={(e) => onChange(e.target.value === '' ? '' : Number(e.target.value))}
                className="h-10 rounded-xl pl-7 pr-3 text-sm border-neutral-200"
                placeholder={placeholder}
                min={0}
            />
        </div>
    )
}

function BoolPicker({
    value,
    onChange,
}: {
    value: boolean | undefined
    onChange: (v: boolean) => void
}) {
    return (
        <div className="flex gap-2">
            {([true, false] as const).map((opt) => {
                const active = value === opt
                return (
                    <button
                        key={String(opt)}
                        type="button"
                        onClick={() => onChange(opt)}
                        className="h-9 px-5 rounded-xl text-sm font-semibold transition-all duration-150"
                        style={{
                            background: active ? 'rgba(28,78,216,0.08)' : 'white',
                            border: active ? '2px solid #1C4ED8' : '2px solid #e5e7eb',
                            color: active ? '#1C4ED8' : '#6b7280',
                        }}
                    >
                        {opt ? 'Yes' : 'No'}
                    </button>
                )
            })}
        </div>
    )
}

function OptionPicker({
    options,
    value,
    onChange,
}: {
    options: { id: string; label: string }[]
    value: string
    onChange: (id: string) => void
}) {
    return (
        <div className="flex flex-wrap gap-2">
            {options.map((opt) => {
                const active = value === opt.id
                return (
                    <button
                        key={opt.id}
                        type="button"
                        onClick={() => onChange(opt.id)}
                        className="h-9 px-4 rounded-xl text-sm font-semibold transition-all duration-150"
                        style={{
                            background: active ? 'rgba(28,78,216,0.08)' : 'white',
                            border: active ? '2px solid #1C4ED8' : '2px solid #e5e7eb',
                            color: active ? '#1C4ED8' : '#6b7280',
                        }}
                    >
                        {opt.label}
                    </button>
                )
            })}
        </div>
    )
}
