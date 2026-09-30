import { useOnboardingStore } from '@/stores/onboardingStore'
import { useShallow } from 'zustand/shallow'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { SchemaNode } from '@/types/onboarding'

// Question types where clicking an option auto-advances — no Continue button shown.
const AUTO_ADVANCE = new Set(['boolean', 'single_select', 'sentiment_picker'])

export function DynamicStepRenderer() {
    const question = useOnboardingStore(
        useShallow((s) => { const qs = s.getVisibleQuestions(); return qs[0] ?? null }),
    )
    const profile = useOnboardingStore((s) => s.profile)
    const setAnswer = useOnboardingStore((s) => s.setAnswer)
    const goNext = useOnboardingStore((s) => s.goNextScreen)
    const goBack = useOnboardingStore((s) => s.goPrevScreen)
    const canAdvance = useOnboardingStore((s) => s.canAdvance())
    const isLast = useOnboardingStore((s) => s.currentScreenIndex >= s.screens.length - 1)
    const currentScreenIndex = useOnboardingStore((s) => s.currentScreenIndex)
    const currentQuestionIndex = useOnboardingStore((s) => s.currentQuestionIndex)
    const progress = useOnboardingStore((s) => s.questionProgress())

    if (!question) return null

    const firstName = (profile.first_name as string | undefined)?.trim() ?? ''
    function interpolate(text: string): string {
        if (!firstName) return text.replace(/\{first_name\},?\s*/g, '').trim()
        return text.replace(/\{first_name\}/g, firstName)
    }

    const screenLabels: Record<string, string> = {
        '00': 'Getting started',
        '01': 'Your identity',
        '02': 'Tax identity',
        '03': 'Address & residence',
        '04': 'Marital & family',
        '05': 'Dependants',
        '06': 'Your work situation',
        '07': 'Employment details',
        '08': 'Freelance & gig income',
        '09': 'Business income',
        '10': 'Professional income',
        '11': 'Rental income',
        '12': 'Your own rent',
        '13': 'Interest & investment',
        '14': 'Shares & securities',
        '15': 'Crypto & digital assets',
        '16': 'Foreign income',
        '17': 'Pension',
        '18': 'NHIS & NHF',
        '19': 'Life insurance',
        '20': 'Home loan',
        '21': 'Other income',
        '22': 'Asset disposals',
        '23': 'Tax already paid',
        '24': 'Previous filings',
        '25': 'Your documents',
        '26': 'Data verification',
        '27': 'Tax calculation',
        '28': 'Deductions review',
        '29': 'Declaration',
        '30': 'How to file',
    }

    const screenSubtitles: Record<string, string> = {
        '00': "Let's figure out exactly what we can do for you.",
        '01': "A few details about you — takes about 2 minutes.",
        '02': "Your Tax ID lets us pre-fill parts of your return and verify your record with the NRS.",
        '03': "Your residence determines whether you're taxed on Nigerian or worldwide income.",
        '04': "Your marital status can affect your tax position — we just need the basics.",
        '05': "Dependants you support may be relevant to your filing — don't skip this.",
        '06': "Each income type you select opens the deductions that go with it.",
        '07': "Your employment details determine your PAYE credit and any remaining tax to pay.",
        '08': "Freelance expenses are deductible — let's make sure none are missed.",
        '09': "Business income comes with allowable expenses that reduce what you owe.",
        '10': "WHT deducted by clients becomes a direct credit against your personal income tax.",
        '11': "Rental income is taxable, but allowable property expenses reduce the charge.",
        '12': "You could claim up to ₦500,000 in rent relief under NTA s.30 — let's check.",
        '13': "Investment income is taxable. Withholding tax already deducted becomes a credit.",
        '14': "Nigerian share gains have specific exemption thresholds — we'll apply them automatically.",
        '15': "The NRS has specific rules for crypto. Transaction type determines the tax treatment.",
        '16': "As a Nigerian resident, your worldwide income is within the Nigerian tax charge.",
        '17': "Qualifying pension contributions are an eligible deduction that directly reduce your taxable income.",
        '18': "NHIS and NHF contributions are eligible deductions under NTA s.30 — claim every kobo.",
        '19': "Life insurance and deferred annuity premiums for you or your spouse are deductible.",
        '20': "Interest on qualifying home-development loans is an eligible deduction under NTA s.30.",
        '21': "Prizes, grants, compensation and other receipts must be declared — and some are exempt.",
        '22': "Asset disposals can trigger chargeable gains — or qualify for specific exemptions.",
        '23': "PAYE and WHT already paid become credits that reduce your final tax bill.",
        '24': "Outstanding assessments and prior filings affect what we need to include in this return.",
        '25': "Good documentation means a faster, cleaner return — and stronger protection if you're ever queried.",
        '26': "Let's make sure everything is accurate before we lock in your return.",
        '27': "This is your estimated tax position based on everything you've told us.",
        '28': "Review every deduction — nothing you're entitled to should be left on the table.",
        '29': "Your declaration makes this return legally valid under the NTAA 2025.",
        '30': "Choose how you'd like to proceed and we'll guide you from here.",
    }

    const screenId = question.screen
    const screenLabel = screenLabels[screenId] ?? `Section ${screenId}`
    const screenSubtitle = interpolate(screenSubtitles[screenId] ?? '')
    const canGoBack = currentScreenIndex > 0 || currentQuestionIndex > 0
    const isAutoAdvance = AUTO_ADVANCE.has(question.type)

    const handleAnswer = (v: unknown) => {
        setAnswer(question.code, v)
        if (isAutoAdvance) {
            setTimeout(goNext, 180)
        }
    }

    return (
        <div>
            {/* Section badge with within-screen progress */}
            <div className="mb-8">
                <div
                    className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-5"
                    style={{
                        background: 'rgba(28,78,216,0.07)',
                        color: '#1C4ED8',
                        border: '1px solid rgba(28,78,216,0.14)',
                    }}
                >
                    <span className="material-symbols-outlined text-[16px]">edit_note</span>
                    {screenLabel}
                    {progress.total > 1 && (
                        <span className="opacity-50 font-normal">· {progress.current}/{progress.total}</span>
                    )}
                </div>

                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    {interpolate(question.label)}
                </h1>

                {/* Screen context shown only on the first question of each screen */}
                {screenSubtitle && currentQuestionIndex === 0 && (
                    <p className="text-neutral-500 text-[15px] leading-relaxed">{screenSubtitle}</p>
                )}

                {/* Hint shown inline — no toggle needed at single-question density */}
                {question.hint && (
                    <p className="mt-2 text-neutral-400 text-[13px] leading-relaxed">
                        {interpolate(question.hint)}
                    </p>
                )}
            </div>

            {/* Single question input */}
            <div className="mb-10">
                <InputField
                    question={{
                        ...question,
                        label: interpolate(question.label),
                        hint: question.hint ? interpolate(question.hint) : undefined,
                    }}
                    value={profile[question.code]}
                    onChange={handleAnswer}
                />
            </div>

            {/* Navigation — Continue hidden for auto-advance types */}
            <div className="flex items-center gap-3">
                {canGoBack && (
                    <Button
                        type="button"
                        variant="outline"
                        onClick={goBack}
                        className="h-12 px-5 rounded-xl border-neutral-200 text-neutral-600 font-semibold flex items-center gap-1.5 hover:bg-neutral-50"
                    >
                        <ChevronLeft className="w-4 h-4" /> Back
                    </Button>
                )}
                {!isAutoAdvance && (
                    <Button
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
                                <span className="material-symbols-outlined text-[20px]">celebration</span>{' '}
                                Let's start filing
                            </>
                        ) : (
                            <>
                                Continue <ChevronRight className="w-4 h-4" />
                            </>
                        )}
                    </Button>
                )}
            </div>
        </div>
    )
}

// ---------------------------------------------------------------------------
// Per-question input renderer
// ---------------------------------------------------------------------------

interface InputFieldProps {
    question: SchemaNode
    value: unknown
    onChange: (v: unknown) => void
}

function InputField({ question, value, onChange }: InputFieldProps) {
    switch (question.type) {
        case 'text':
            return (
                <Input
                    value={(value as string) ?? ''}
                    onChange={(e) => onChange(e.target.value)}
                    className="h-12 rounded-xl border-neutral-200 px-4 text-[15px]"
                    placeholder={`Enter ${question.label.toLowerCase()}`}
                />
            )

        case 'date':
            return (
                <Input
                    type="date"
                    value={(value as string) ?? ''}
                    onChange={(e) => onChange(e.target.value)}
                    className="h-12 rounded-xl border-neutral-200 px-4 text-[15px]"
                />
            )

        case 'number':
            return (
                <Input
                    type="number"
                    value={(value as number) ?? ''}
                    onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
                    className="h-12 rounded-xl border-neutral-200 px-4 text-[15px]"
                    placeholder="0"
                    min={0}
                />
            )

        case 'currency':
            return (
                <div className="relative">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 text-neutral-400 text-sm font-semibold">
                        ₦
                    </span>
                    <Input
                        type="number"
                        value={(value as number) ?? ''}
                        onChange={(e) => onChange(e.target.value === '' ? '' : Number(e.target.value))}
                        className="h-12 rounded-xl border-neutral-200 pl-8 pr-4 text-[15px]"
                        placeholder="0"
                    />
                </div>
            )

        case 'boolean':
            return (
                <div className="flex gap-3">
                    {[true, false].map((opt) => {
                        const isSelected = value === opt
                        return (
                            <button
                                key={String(opt)}
                                type="button"
                                onClick={() => onChange(opt)}
                                className="flex-1 py-3 rounded-xl text-sm font-semibold transition-all duration-200"
                                style={{
                                    background: isSelected ? 'rgba(28,78,216,0.06)' : 'white',
                                    border: isSelected ? '2px solid #1C4ED8' : '2px solid #e5e5e3',
                                    color: isSelected ? '#1C4ED8' : '#4b5563',
                                }}
                            >
                                {opt ? 'Yes' : 'No'}
                            </button>
                        )
                    })}
                </div>
            )

        case 'single_select':
            return (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {(question.options ?? []).map((opt) => {
                        const isSelected = value === opt.id
                        return (
                            <button
                                key={opt.id}
                                type="button"
                                onClick={() => onChange(opt.id)}
                                className="relative flex items-center gap-3 p-4 rounded-2xl text-left transition-all duration-200"
                                style={{
                                    background: isSelected ? 'rgba(28,78,216,0.05)' : 'white',
                                    border: isSelected ? '2px solid #1C4ED8' : '2px solid #e5e5e3',
                                    boxShadow: isSelected
                                        ? '0 4px 16px rgba(28,78,216,0.12)'
                                        : '0 1px 4px rgba(0,0,0,0.04)',
                                }}
                            >
                                {opt.icon && (
                                    <span
                                        className="material-symbols-outlined text-[24px]"
                                        style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}
                                    >
                                        {opt.icon}
                                    </span>
                                )}
                                <div>
                                    <span
                                        className="text-sm font-semibold block"
                                        style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                                    >
                                        {opt.label}
                                    </span>
                                    {opt.sub && (
                                        <span className="text-xs text-neutral-400 mt-0.5 block">{opt.sub}</span>
                                    )}
                                </div>
                                {isSelected && (
                                    <div
                                        className="absolute top-2 right-2 w-5 h-5 rounded-full flex items-center justify-center"
                                        style={{ background: '#1C4ED8' }}
                                    >
                                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                                            <path
                                                d="M2 5l2.5 2.5 3.5-5"
                                                stroke="white"
                                                strokeWidth="1.5"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            />
                                        </svg>
                                    </div>
                                )}
                            </button>
                        )
                    })}
                </div>
            )

        case 'sentiment_picker':
            return (
                <div className="flex gap-3 mt-4">
                    {(question.options ?? []).map((opt) => {
                        const isSelected = value === opt.id
                        return (
                            <button
                                key={opt.id}
                                type="button"
                                onClick={() => onChange(opt.id)}
                                className="flex-1 flex flex-col items-center justify-center py-6 rounded-2xl transition-all duration-300"
                                style={{
                                    background: isSelected ? 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)' : 'white',
                                    border: isSelected ? '2px solid transparent' : '2px solid #e5e5e3',
                                    boxShadow: isSelected
                                        ? '0 12px 32px rgba(28,78,216,0.25)'
                                        : '0 2px 8px rgba(0,0,0,0.03)',
                                    transform: isSelected ? 'scale(1.02) translateY(-2px)' : 'scale(1)',
                                }}
                            >
                                <span
                                    className="material-symbols-outlined text-[42px] mb-3 transition-colors"
                                    style={{ color: isSelected ? 'white' : '#9ca3af' }}
                                >
                                    {opt.icon}
                                </span>
                                <span
                                    className="text-[13px] font-bold tracking-wide"
                                    style={{ color: isSelected ? 'white' : '#4b5563' }}
                                >
                                    {opt.label}
                                </span>
                            </button>
                        )
                    })}
                </div>
            )

        case 'multi_select': {
            const selected: string[] = Array.isArray(value) ? (value as string[]) : []
            const toggle = (optId: string) => {
                const next = selected.includes(optId)
                    ? selected.filter((x) => x !== optId)
                    : [...selected, optId]
                onChange(next)
            }
            return (
                <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
                    {(question.options ?? []).map((opt) => {
                        const isSelected = selected.includes(opt.id)
                        return (
                            <button
                                key={opt.id}
                                type="button"
                                onClick={() => toggle(opt.id)}
                                className="relative flex flex-col items-center text-center p-4 rounded-2xl transition-all duration-200 cursor-pointer"
                                style={{
                                    background: isSelected ? 'rgba(28,78,216,0.05)' : 'white',
                                    border: isSelected ? '2px solid #1C4ED8' : '2px solid #e5e5e3',
                                    boxShadow: isSelected
                                        ? '0 4px 16px rgba(28,78,216,0.12)'
                                        : '0 1px 4px rgba(0,0,0,0.04)',
                                    transform: isSelected ? 'translateY(-1px)' : undefined,
                                }}
                            >
                                {isSelected && (
                                    <div
                                        className="absolute top-2 right-2 w-5 h-5 rounded-full flex items-center justify-center"
                                        style={{ background: '#1C4ED8' }}
                                    >
                                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                                            <path
                                                d="M2 5l2.5 2.5 3.5-5"
                                                stroke="white"
                                                strokeWidth="1.5"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            />
                                        </svg>
                                    </div>
                                )}
                                {opt.icon && (
                                    <span
                                        className="material-symbols-outlined text-[26px] mb-2 leading-none"
                                        style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}
                                    >
                                        {opt.icon}
                                    </span>
                                )}
                                <span
                                    className="text-[12px] font-semibold leading-tight block"
                                    style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                                >
                                    {opt.label}
                                </span>
                                {opt.sub && (
                                    <span className="text-[11px] text-neutral-400 mt-0.5 leading-tight">
                                        {opt.sub}
                                    </span>
                                )}
                            </button>
                        )
                    })}
                </div>
            )
        }

        case 'file':
            return (
                <div
                    className="flex items-center justify-center p-8 rounded-2xl cursor-pointer transition-all hover:border-blue-300"
                    style={{ border: '2px dashed #d1d5db', background: '#fafafa' }}
                >
                    <div className="text-center">
                        <span className="material-symbols-outlined text-[32px] text-neutral-400 mb-2 block">
                            upload_file
                        </span>
                        <p className="text-sm text-neutral-500">Click or drag to upload</p>
                    </div>
                </div>
            )

        default:
            return (
                <div className="text-sm text-neutral-400">
                    Unsupported field type: {question.type}
                </div>
            )
    }
}
