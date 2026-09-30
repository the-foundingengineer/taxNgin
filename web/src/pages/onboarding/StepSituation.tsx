import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const SITUATIONS = [
    { id: 'salaried', icon: 'badge', label: 'Salaried employee', sub: 'Paid via PAYE' },
    { id: 'selfem', icon: 'laptop_mac', label: 'Self-employed', sub: 'Freelance / contract' },
    { id: 'business', icon: 'store', label: 'Business owner', sub: 'You run a registered biz' },
    { id: 'landlord', icon: 'real_estate_agent', label: 'Landlord', sub: 'Earn rental income' },
    { id: 'pensioner', icon: 'elderly', label: 'Pensioner', sub: 'Retired / pension income' },
    { id: 'student', icon: 'school', label: 'Student', sub: 'In full-time education' },
    { id: 'director', icon: 'corporate_fare', label: 'Director / officer', sub: 'Company director' },
    { id: 'trader', icon: 'shopping_cart', label: 'Trader', sub: 'Market / retail trader' },
    { id: 'farmer', icon: 'agriculture', label: 'Farmer', sub: 'Agricultural activity' },
    { id: 'artist', icon: 'palette', label: 'Artist / creative', sub: 'Creator, performer, etc.' },
    { id: 'civil', icon: 'account_balance', label: 'Civil servant', sub: 'Federal or state employee' },
    { id: 'other', icon: 'help', label: 'Other / not sure', sub: 'We\'ll help you figure it out' },
]

export interface SituationData {
    situations: string[]
}

interface Props {
    defaultValues?: SituationData
    onNext: (data: SituationData) => void
    onBack: () => void
}

export function StepSituation({ defaultValues, onNext, onBack }: Props) {
    const [selected, setSelected] = useState<Set<string>>(
        new Set(defaultValues?.situations ?? [])
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
                    <span className="material-symbols-outlined text-[16px]">groups</span> About you
                </div>
                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    Tell us about your situation.
                </h1>
                <p className="text-neutral-500 text-[15px]">
                    Select everything that applies — we'll personalise your filing journey.
                </p>
            </div>

            {/* Card grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-8">
                {SITUATIONS.map(sit => {
                    const isSelected = selected.has(sit.id)
                    return (
                        <button
                            key={sit.id}
                            type="button"
                            onClick={() => toggle(sit.id)}
                            className="relative flex flex-col items-center text-center p-4 rounded-2xl transition-all duration-200 cursor-pointer group"
                            style={{
                                background: isSelected
                                    ? 'rgba(28,78,216,0.05)'
                                    : 'white',
                                border: isSelected
                                    ? '2px solid #1C4ED8'
                                    : '2px solid #e5e5e3',
                                boxShadow: isSelected
                                    ? '0 4px 16px rgba(28,78,216,0.12)'
                                    : '0 1px 4px rgba(0,0,0,0.04)',
                                transform: isSelected ? 'translateY(-1px)' : undefined,
                            }}
                        >
                            {/* Checkmark badge */}
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
                            <span className="material-symbols-outlined text-[26px] mb-2 leading-none" style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}>{sit.icon}</span>
                            <span
                                className="text-[12px] font-semibold leading-tight block"
                                style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                            >
                                {sit.label}
                            </span>
                            <span className="text-[11px] text-neutral-400 mt-0.5 leading-tight">{sit.sub}</span>
                        </button>
                    )
                })}
            </div>

            <p className="text-xs text-neutral-400 mb-6">
                It's OK to pick something even if you're not 100% sure — we'll help you figure it out.
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
                    onClick={() => onNext({ situations: Array.from(selected) })}
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
