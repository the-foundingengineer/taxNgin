import { useState } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { useNavigate } from 'react-router-dom'
import {
    Home, ShieldCheck, Building2, Users,
    Accessibility, Briefcase, Check, ChevronRight, ChevronLeft
} from 'lucide-react'
import { fmtNaira } from '@/lib/pit'

// ── Wizard Step Definitions ──────────────────────────────────────────────────
const STEPS = [
    { id: 'nhf', icon: Home, title: 'NHF Contributions', subtitle: 'National Housing Fund (s.30(2)(b))' },
    { id: 'nsitf', icon: Briefcase, title: 'NSITF Contributions', subtitle: 'Employee Compensation Scheme (s.30(2)(d))' },
    { id: 'insurance', icon: ShieldCheck, title: 'Life Insurance Premiums', subtitle: 'Qualifying Life Assurance (s.30(4))' },
    { id: 'mortgage', icon: Building2, title: 'Mortgage Interest', subtitle: 'Owner-Occupied Home Loan (s.30(5))' },
    { id: 'rent', icon: Home, title: 'Rent Relief', subtitle: 'Primary Residence Rent (s.30(3))' },
    { id: 'dependants', icon: Users, title: 'Child & Dependant Relief', subtitle: '₦250k per qualifying child (s.30(6))' },
    { id: 'disability', icon: Accessibility, title: 'Disability Relief', subtitle: '₦500k relief for qualifying taxpayers (s.30(7))' },
]

function ProgressBar({ step, total }: { step: number; total: number }) {
    return (
        <div className="flex items-center gap-2 mb-10">
            {Array.from({ length: total }).map((_, i) => (
                <div key={i} className={`flex-1 h-1.5 rounded-full transition-all duration-500 ${i < step ? 'bg-blue-600' : i === step ? 'bg-blue-300' : 'bg-neutral-200'}`} />
            ))}
        </div>
    )
}

function AmountField({ label, value, onChange, sublabel }: {
    label: string; value: string; onChange: (v: string) => void; sublabel?: string
}) {
    return (
        <div>
            <label className="block text-sm font-bold text-neutral-700 mb-1.5">{label}</label>
            {sublabel && <p className="text-xs text-neutral-500 mb-2">{sublabel}</p>}
            <div className="relative">
                <span className="absolute left-4 top-3.5 text-neutral-400 font-bold">₦</span>
                <input
                    type="number" min="0" value={value} onChange={e => onChange(e.target.value)}
                    className="w-full bg-white border border-neutral-300 rounded-xl pl-8 pr-4 py-3.5 text-[16px] font-medium focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                    placeholder="0"
                />
            </div>
        </div>
    )
}

function YesNoField({ label, sublabel, value, onChange }: {
    label: string; sublabel?: string; value: boolean | undefined; onChange: (v: boolean) => void
}) {
    return (
        <div>
            <label className="block text-sm font-bold text-neutral-700 mb-1">{label}</label>
            {sublabel && <p className="text-xs text-neutral-500 mb-3">{sublabel}</p>}
            <div className="flex gap-3">
                {([true, false] as const).map(opt => (
                    <button key={String(opt)} onClick={() => onChange(opt)}
                        className={`flex-1 h-12 rounded-xl font-bold border-2 transition-all ${value === opt ? 'border-blue-600 bg-blue-50 text-blue-700' : 'border-neutral-200 bg-white text-neutral-500 hover:border-neutral-400'}`}>
                        {opt ? 'Yes' : 'No'}
                    </button>
                ))}
            </div>
        </div>
    )
}

export function DeductionsWizardPage() {
    const navigate = useNavigate()
    // Load existing profile from localStorage — no Zustand store dependency needed here
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const existing: Record<string, any> = (() => {
        try { return JSON.parse(localStorage.getItem('tc_profile') ?? '{}') } catch { return {} }
    })()

    const [step, setStep] = useState(0)

    const [nhfContributes, setNhfContributes] = useState<boolean | undefined>(existing.makes_nhf_contributions as boolean | undefined)
    const [nhfAmount, setNhfAmount] = useState(String(existing.nhf_contribution_amount ?? ''))
    const [nsitfContributes, setNsitfContributes] = useState<boolean | undefined>(existing.makes_nsitf_contributions as boolean | undefined)
    const [nsitfAmount, setNsitfAmount] = useState(String(existing.nsitf_contribution_amount ?? ''))
    const [hasInsurance, setHasInsurance] = useState<boolean | undefined>(existing.has_life_insurance as boolean | undefined)
    const [insuranceAmt, setInsuranceAmt] = useState(String(existing.qualifying_premiums_total ?? ''))
    const [hasMortgage, setHasMortgage] = useState<boolean | undefined>(existing.has_home_loan as boolean | undefined)
    const [mortgageOwnerOccupied, setMortgageOwnerOccupied] = useState<boolean | undefined>(existing.home_loan_property_owner_occupied as boolean | undefined)
    const [mortgageInterest, setMortgageInterest] = useState(String(existing.home_loan_interest_paid ?? ''))
    const [paysRent, setPaysRent] = useState<boolean | undefined>(existing.pays_own_rent as boolean | undefined)
    const [rentOwn, setRentOwn] = useState<boolean | undefined>(existing.rent_is_own_residence as boolean | undefined)
    const [annualRent, setAnnualRent] = useState(String(existing.annual_rent_paid ?? ''))
    const [childCount, setChildCount] = useState(String(existing.dependant_count ?? '0'))
    const [hasDisability, setHasDisability] = useState<boolean | undefined>(existing.has_disability_relief as boolean | undefined)

    function computeSavingPreview(): number {
        let total = 0
        if (nhfContributes) total += Number(nhfAmount) * 0.15
        if (nsitfContributes) total += Number(nsitfAmount) * 0.15
        if (hasInsurance) total += Number(insuranceAmt) * 0.15
        if (hasMortgage && mortgageOwnerOccupied) total += Number(mortgageInterest) * 0.15
        if (paysRent && rentOwn) total += Math.min(Number(annualRent) * 0.20, 500_000) * 0.15
        total += Math.min(Number(childCount), 4) * 250_000 * 0.15
        if (hasDisability) total += 500_000 * 0.15
        return Math.round(total)
    }

    function handleNext() {
        if (step < STEPS.length - 1) { setStep(s => s + 1); return }
        const updates = {
            makes_nhf_contributions: nhfContributes,
            nhf_contribution_amount: Number(nhfAmount),
            makes_nsitf_contributions: nsitfContributes,
            nsitf_contribution_amount: Number(nsitfAmount),
            has_life_insurance: hasInsurance,
            qualifying_premiums_total: Number(insuranceAmt),
            has_home_loan: hasMortgage,
            home_loan_property_owner_occupied: mortgageOwnerOccupied,
            home_loan_interest_paid: Number(mortgageInterest),
            pays_own_rent: paysRent,
            rent_is_own_residence: rentOwn,
            annual_rent_paid: Number(annualRent),
            dependant_count: Number(childCount),
            has_dependants: Number(childCount) > 0 ? 'yes' : 'no',
            has_disability_relief: hasDisability,
        }
        localStorage.setItem('tc_profile', JSON.stringify({ ...existing, ...updates }))
        navigate('/ready-check')
    }

    const currentStep = STEPS[step]
    const Icon = currentStep.icon
    const saving = computeSavingPreview()

    const renderStepContent = () => {
        switch (currentStep.id) {
            case 'nhf': return (
                <div className="space-y-6">
                    <YesNoField label="Do you contribute to the National Housing Fund (NHF)?" sublabel="Most FIRS-registered employees contribute 2.5% of basic salary monthly through their employer." value={nhfContributes} onChange={setNhfContributes} />
                    {nhfContributes && <AmountField label="Annual NHF Contribution Amount" value={nhfAmount} onChange={setNhfAmount} sublabel="Total NHF deducted from your payslip in the tax year" />}
                </div>
            )
            case 'nsitf': return (
                <div className="space-y-6">
                    <YesNoField label="Does your employer deduct NSITF contributions on your behalf?" sublabel="1% of your basic salary is typically deducted for the Employees' Compensation Scheme." value={nsitfContributes} onChange={setNsitfContributes} />
                    {nsitfContributes && <AmountField label="Annual NSITF Contribution Amount" value={nsitfAmount} onChange={setNsitfAmount} />}
                </div>
            )
            case 'insurance': return (
                <div className="space-y-6">
                    <YesNoField label="Do you pay premiums on a qualifying life assurance policy?" sublabel="Policies issued by NAICOM-licensed insurers on your own life qualify." value={hasInsurance} onChange={setHasInsurance} />
                    {hasInsurance && <AmountField label="Annual Qualifying Premiums Paid" value={insuranceAmt} onChange={setInsuranceAmt} />}
                </div>
            )
            case 'mortgage': return (
                <div className="space-y-6">
                    <YesNoField label="Do you have a mortgage or home loan?" value={hasMortgage} onChange={setHasMortgage} />
                    {hasMortgage && <>
                        <YesNoField label="Is the property your owner-occupied primary residence?" sublabel="Only mortgages on your home qualify — investment properties do not." value={mortgageOwnerOccupied} onChange={setMortgageOwnerOccupied} />
                        {mortgageOwnerOccupied && <AmountField label="Annual Mortgage Interest Paid" value={mortgageInterest} onChange={setMortgageInterest} />}
                    </>}
                </div>
            )
            case 'rent': return (
                <div className="space-y-6">
                    <YesNoField label="Do you pay rent out of your own income?" value={paysRent} onChange={setPaysRent} />
                    {paysRent && <>
                        <YesNoField label="Is the rented property your primary residence?" value={rentOwn} onChange={setRentOwn} />
                        {rentOwn && <AmountField label="Annual Rent Paid" value={annualRent} onChange={setAnnualRent} sublabel="20% of this amount (capped at ₦500,000) will be deducted from your taxable income" />}
                    </>}
                </div>
            )
            case 'dependants': return (
                <div className="space-y-6">
                    <div>
                        <label className="block text-sm font-bold text-neutral-700 mb-1.5">How many qualifying children do you have?</label>
                        <p className="text-xs text-neutral-500 mb-3">Children under 16, or in full-time education of any age. Max 4 children qualify for relief.</p>
                        <div className="flex items-center gap-4">
                            <button onClick={() => setChildCount(c => String(Math.max(0, Number(c) - 1)))} className="w-12 h-12 bg-neutral-100 rounded-xl text-2xl font-bold hover:bg-neutral-200 transition-colors">-</button>
                            <span className="text-4xl font-black w-12 text-center text-neutral-900">{childCount}</span>
                            <button onClick={() => setChildCount(c => String(Math.min(10, Number(c) + 1)))} className="w-12 h-12 bg-blue-600 rounded-xl text-2xl font-bold text-white hover:bg-blue-700 transition-colors">+</button>
                        </div>
                        {Number(childCount) > 0 && (
                            <div className="mt-4 p-4 bg-blue-50 border border-blue-100 rounded-xl">
                                <p className="text-sm font-semibold text-blue-700">
                                    Estimated Child Relief: {fmtNaira(Math.min(Number(childCount), 4) * 250_000)} deducted from taxable income
                                </p>
                            </div>
                        )}
                    </div>
                </div>
            )
            case 'disability': return (
                <div className="space-y-6">
                    <YesNoField label="Do you or a financially dependent family member have a qualifying disability?" sublabel="A qualifying disability is one that substantially limits a major life activity and is certified by a licensed medical practitioner." value={hasDisability} onChange={setHasDisability} />
                    {hasDisability && (
                        <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-xl">
                            <p className="text-sm font-bold text-emerald-800">✓ ₦500,000 Disability Relief will be applied to your return</p>
                        </div>
                    )}
                </div>
            )
            default: return null
        }
    }

    return (
        <DashboardLayout>
            <div className="max-w-2xl mx-auto pb-24 animate-in fade-in duration-500">
                <div className="pt-8 pb-10">
                    <p className="text-sm font-bold text-blue-600 uppercase tracking-widest mb-2">Deductions & Reliefs Wizard</p>
                    <h1 className="text-3xl font-black text-neutral-900 tracking-tight">Find every naira you're entitled to.</h1>
                    <p className="text-neutral-500 mt-2">We'll walk through each NTA s.30 relief section by section.</p>
                </div>

                <ProgressBar step={step} total={STEPS.length} />

                <div key={step} className="animate-in fade-in slide-in-from-right-4 duration-400">
                    <div className="bg-white border border-neutral-200 rounded-3xl p-8 shadow-sm mb-6">
                        <div className="flex items-center gap-3 mb-6">
                            <div className="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center shrink-0">
                                <Icon className="w-5 h-5 text-white" />
                            </div>
                            <div>
                                <h2 className="text-lg font-bold text-neutral-900">{currentStep.title}</h2>
                                <p className="text-xs text-neutral-500">{currentStep.subtitle}</p>
                            </div>
                            <span className="ml-auto text-sm text-neutral-400 font-medium">{step + 1}/{STEPS.length}</span>
                        </div>
                        {renderStepContent()}
                    </div>

                    {saving > 0 && (
                        <div className="bg-emerald-50 border border-emerald-200 rounded-2xl px-6 py-4 mb-6 flex items-center gap-3">
                            <Check className="w-5 h-5 text-emerald-600 shrink-0" />
                            <p className="text-sm font-bold text-emerald-800">
                                Estimated tax saving so far: <span className="text-emerald-700">{fmtNaira(saving)}</span>
                            </p>
                        </div>
                    )}

                    <div className="flex items-center gap-4">
                        {step > 0 && (
                            <button onClick={() => setStep(s => s - 1)} className="flex items-center gap-2 h-12 px-6 bg-neutral-100 hover:bg-neutral-200 rounded-xl font-bold text-neutral-700 transition-colors">
                                <ChevronLeft className="w-4 h-4" /> Back
                            </button>
                        )}
                        <button onClick={handleNext} className="flex-1 flex items-center justify-center gap-2 h-12 bg-blue-600 hover:bg-blue-700 rounded-xl font-bold text-white transition-colors shadow-sm">
                            {step === STEPS.length - 1 ? 'Save & Run ReadyCheck' : 'Next'} <ChevronRight className="w-4 h-4" />
                        </button>
                    </div>
                </div>
            </div>
        </DashboardLayout>
    )
}
