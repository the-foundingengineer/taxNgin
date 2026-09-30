import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const INCOME_SOURCES = [
    { id: 'paye', icon: 'account_balance', label: 'Employment (PAYE)', sub: 'Salary from an employer' },
    { id: 'business', icon: 'trending_up', label: 'Business income', sub: 'Profits from your business' },
    { id: 'rental', icon: 'home_work', label: 'Rental income', sub: 'From property you let out' },
    { id: 'divs', icon: 'pie_chart', label: 'Dividends', sub: 'Shares & equity payouts' },
    { id: 'capgain', icon: 'bar_chart', label: 'Capital gains', sub: 'Property or stock sales' },
    { id: 'pension', icon: 'savings', label: 'Pension / annuity', sub: 'Retirement income' },
    { id: 'freelance', icon: 'terminal', label: 'Freelance / contract', sub: '1099-style project work' },
    { id: 'crypto', icon: 'currency_bitcoin', label: 'Crypto / digital assets', sub: 'Trading or mining gains' },
    { id: 'foreign', icon: 'public', label: 'Foreign income', sub: 'Earned outside Nigeria' },
    { id: 'assets', icon: 'sell', label: 'Sale of assets', sub: 'Equipment, vehicles, etc.' },
]

export interface IncomeData {
    incomeSources: string[]
}

interface Props {
    defaultValues?: IncomeData
    onNext: (data: IncomeData) => void
    onBack: () => void
}

export function StepIncome({ defaultValues, onNext, onBack }: Props) {
    const [selected, setSelected] = useState<Set<string>>(
        new Set(defaultValues?.incomeSources ?? [])
    )

    const toggle = (id: string) => {
        setSelected(prev => {
            const next = new Set(prev)
            next.has(id) ? next.delete(id) : next.add(id)
            return next
        })
    }

    return (
        <div>
            {/* Heading */}
            <div className="mb-8">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-5"
                    style={{ background: 'rgba(28,78,216,0.07)', color: '#1C4ED8', border: '1px solid rgba(28,78,216,0.14)' }}>
                    <span className="material-symbols-outlined text-[16px]">payments</span> Income sources
                </div>
                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    What income did you<br className="hidden sm:block" /> receive this year?
                </h1>
                <p className="text-neutral-500 text-[15px]">
                    Select all that applied — we'll build your return around them.
                </p>
            </div>

            {/* Card grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-8">
                {INCOME_SOURCES.map(src => {
                    const isSelected = selected.has(src.id)
                    return (
                        <button
                            key={src.id}
                            type="button"
                            onClick={() => toggle(src.id)}
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
                                        <path d="M2 5l2.5 2.5 3.5-5" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                                    </svg>
                                </div>
                            )}
                            <span className="material-symbols-outlined text-[26px] mb-2 leading-none" style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}>{src.icon}</span>
                            <span
                                className="text-[12px] font-semibold leading-tight block"
                                style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                            >
                                {src.label}
                            </span>
                            <span className="text-[11px] text-neutral-400 mt-0.5 leading-tight">{src.sub}</span>
                        </button>
                    )
                })}
            </div>

            <p className="text-xs text-neutral-400 mb-6">
                It's OK to pick something even if you're not sure — we'll help you figure it out later.
            </p>

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
                    onClick={() => onNext({ incomeSources: Array.from(selected) })}
                    disabled={selected.size === 0}
                    className="h-12 px-8 rounded-xl text-white font-semibold text-[15px] flex items-center gap-2 transition-all hover:gap-3 disabled:opacity-40"
                    style={{ background: 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)' }}
                >
                    Continue <ChevronRight className="w-4 h-4" />
                </Button>
            </div>
        </div>
    )
}
