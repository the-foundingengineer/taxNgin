import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ChevronLeft } from 'lucide-react'

const TAX_YEARS = [
    { id: '2025', label: '2025', sub: 'Current year' },
    { id: '2024', label: '2024', sub: 'Most recent' },
    { id: '2023', label: '2023', sub: '2 years ago' },
    { id: '2022', label: '2022', sub: '3 years ago' },
]

const GOALS = [
    {
        id: 'maximize',
        icon: 'rocket_launch',
        label: 'Maximize my refund',
        sub: "Find every deduction and relief I'm entitled to",
    },
    {
        id: 'accurate',
        icon: 'verified',
        label: 'File accurately',
        sub: 'Just get it right — no surprises from FIRS',
    },
    {
        id: 'tcc',
        icon: 'description',
        label: 'Get my TCC fast',
        sub: 'I need my Tax Clearance Certificate ASAP',
    },
]

export interface GoalData {
    taxYear: string
    goal: string
}

interface Props {
    defaultValues?: GoalData
    onNext: (data: GoalData) => void
    onBack: () => void
}

export function StepGoal({ defaultValues, onNext, onBack }: Props) {
    const [taxYear, setTaxYear] = useState(defaultValues?.taxYear ?? '2024')
    const [goal, setGoal] = useState(defaultValues?.goal ?? '')

    return (
        <div>
            {/* Heading */}
            <div className="mb-8">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-5"
                    style={{ background: 'rgba(28,78,216,0.07)', color: '#1C4ED8', border: '1px solid rgba(28,78,216,0.14)' }}>
                    <span className="material-symbols-outlined text-[16px]">ads_click</span> Final step
                </div>
                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    Set your goal &<br className="hidden sm:block" /> choose your tax year.
                </h1>
                <p className="text-neutral-500 text-[15px]">
                    We'll build your personalised filing experience around this.
                </p>
            </div>

            {/* Tax year selector */}
            <div className="mb-8">
                <p className="text-sm font-semibold text-neutral-700 mb-3">Which tax year are you filing?</p>
                <div className="flex gap-3 flex-wrap">
                    {TAX_YEARS.map(yr => {
                        const isSelected = taxYear === yr.id
                        return (
                            <button
                                key={yr.id}
                                type="button"
                                onClick={() => setTaxYear(yr.id)}
                                className="flex flex-col items-center px-6 py-4 rounded-2xl transition-all duration-200 relative min-w-[90px]"
                                style={{
                                    background: isSelected ? 'rgba(28,78,216,0.06)' : 'white',
                                    border: isSelected ? '2px solid #1C4ED8' : '2px solid #e5e5e3',
                                    boxShadow: isSelected ? '0 4px 14px rgba(28,78,216,0.14)' : '0 1px 4px rgba(0,0,0,0.04)',
                                    transform: isSelected ? 'translateY(-1px)' : undefined,
                                }}
                            >
                                <span
                                    className="text-lg font-bold"
                                    style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                                >
                                    {yr.label}
                                </span>
                                <span className="text-[11px] text-neutral-400 mt-0.5">{yr.sub}</span>
                                {isSelected && (
                                    <div
                                        className="absolute top-2 right-2 w-4 h-4 rounded-full flex items-center justify-center"
                                        style={{ background: '#1C4ED8' }}
                                    >
                                        <svg width="8" height="8" viewBox="0 0 8 8" fill="none">
                                            <path d="M1.5 4l2 2 3-4" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                                        </svg>
                                    </div>
                                )}
                            </button>
                        )
                    })}
                </div>
            </div>

            {/* Goal selector */}
            <div className="mb-10">
                <p className="text-sm font-semibold text-neutral-700 mb-3">What's your primary goal?</p>
                <div className="flex flex-col gap-3">
                    {GOALS.map(g => {
                        const isSelected = goal === g.id
                        return (
                            <button
                                key={g.id}
                                type="button"
                                onClick={() => setGoal(g.id)}
                                className="flex items-center gap-4 p-4 rounded-2xl text-left transition-all duration-200 relative"
                                style={{
                                    background: isSelected ? 'rgba(28,78,216,0.05)' : 'white',
                                    border: isSelected ? '2px solid #1C4ED8' : '2px solid #e5e5e3',
                                    boxShadow: isSelected ? '0 4px 16px rgba(28,78,216,0.12)' : '0 1px 4px rgba(0,0,0,0.04)',
                                    transform: isSelected ? 'translateY(-1px)' : undefined,
                                }}
                            >
                                {/* Checkmark */}
                                {isSelected && (
                                    <div
                                        className="absolute top-3 right-3 w-5 h-5 rounded-full flex items-center justify-center"
                                        style={{ background: '#1C4ED8' }}
                                    >
                                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                                            <path d="M2 5l2.5 2.5 3.5-5" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                                        </svg>
                                    </div>
                                )}
                                <span className="material-symbols-outlined text-[26px] shrink-0" style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}>{g.icon}</span>
                                <div>
                                    <span
                                        className="text-sm font-semibold block"
                                        style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                                    >
                                        {g.label}
                                    </span>
                                    <span className="text-xs text-neutral-400 mt-0.5 block">{g.sub}</span>
                                </div>
                            </button>
                        )
                    })}
                </div>
            </div>

            {/* Navigation */}
            <div className="flex items-center gap-3">
                <Button
                    type="button"
                    variant="outline"
                    onClick={onBack}
                    className="h-12 px-5 rounded-xl border-neutral-200 text-neutral-600 font-semibold flex items-center gap-1.5 hover:bg-neutral-50"
                >
                    <ChevronLeft className="w-4 h-4" /> Back
                </Button>
                <Button
                    type="button"
                    onClick={() => onNext({ taxYear, goal })}
                    disabled={!goal}
                    className="h-12 px-8 rounded-xl text-white font-bold text-[15px] flex items-center gap-2 disabled:opacity-40 transition-all"
                    style={{
                        background: 'linear-gradient(135deg, #1C4ED8 0%, #45B4C2 100%)',
                        boxShadow: goal ? '0 4px 18px rgba(28,78,216,0.35)' : undefined,
                    }}
                >
                    <span className="material-symbols-outlined text-[20px]">celebration</span> Let's start filing
                </Button>
            </div>
        </div>
    )
}
