import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const LIFE_EVENTS = [
    { id: 'married', icon: 'favorite', label: 'Got married', sub: 'Joint filing may apply' },
    { id: 'child', icon: 'child_care', label: 'Had or adopted a child', sub: 'Child allowance relief' },
    { id: 'dependent', icon: 'family_restroom', label: 'Supporting a dependent', sub: 'Parents, relatives' },
    { id: 'bought_home', icon: 'cottage', label: 'Bought a home', sub: 'Mortgage interest relief' },
    { id: 'school_fees', icon: 'menu_book', label: 'Paid school fees', sub: 'Education deductions' },
    { id: 'charity', icon: 'volunteer_activism', label: 'Donated to charity', sub: 'Charitable contributions' },
    { id: 'life_assur', icon: 'health_and_safety', label: 'Has life insurance / NHF', sub: 'Premium deductions' },
    { id: 'pension_con', icon: 'savings', label: 'Made pension contributions', sub: 'RSA / voluntary contrib.' },
    { id: 'lost_job', icon: 'work_off', label: 'Lost a job this year', sub: 'Unemployment considerations' },
    { id: 'started_biz', icon: 'rocket_launch', label: 'Started a new business', sub: 'Startup deductions' },
    { id: 'sick', icon: 'medical_services', label: 'Major medical expenses', sub: 'Healthcare deductions' },
    { id: 'none', icon: 'sentiment_satisfied', label: 'None of these', sub: 'Straightforward year' },
]

export interface LifeEventsData {
    lifeEvents: string[]
}

interface Props {
    defaultValues?: LifeEventsData
    onNext: (data: LifeEventsData) => void
    onBack: () => void
}

export function StepLifeEvents({ defaultValues, onNext, onBack }: Props) {
    const [selected, setSelected] = useState<Set<string>>(
        new Set(defaultValues?.lifeEvents ?? [])
    )

    const toggle = (id: string) => {
        setSelected(prev => {
            const next = new Set(prev)
            // "none" is exclusive
            if (id === 'none') {
                return new Set(['none'])
            }
            next.delete('none')
            next.has(id) ? next.delete(id) : next.add(id)
            return next
        })
    }

    // Compute a rough relief figure for the "insight pill"
    const reliefMap: Record<string, number> = {
        married: 200000, child: 250000, dependent: 150000, bought_home: 300000,
        school_fees: 180000, charity: 100000, life_assur: 120000, pension_con: 200000,
        lost_job: 50000, started_biz: 200000, sick: 80000,
    }
    const estimatedRelief = Array.from(selected)
        .filter(k => k !== 'none')
        .reduce((sum, k) => sum + (reliefMap[k] ?? 0), 0)

    return (
        <div>
            {/* Heading */}
            <div className="mb-8">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-5"
                    style={{ background: 'rgba(28,78,216,0.07)', color: '#1C4ED8', border: '1px solid rgba(28,78,216,0.14)' }}>
                    <span className="material-symbols-outlined text-[16px]">auto_awesome</span> Life events
                </div>
                <h1 className="text-[28px] lg:text-[34px] font-bold text-neutral-900 leading-tight mb-2">
                    Did any of these happen<br className="hidden sm:block" /> this year?
                </h1>
                <p className="text-neutral-500 text-[15px]">
                    Life events unlock deductions and reliefs under Nigerian tax law (PITA).
                </p>
            </div>

            {/* Relief insight — shown when selections made */}
            {estimatedRelief > 0 && (
                <div
                    className="flex items-center gap-3 p-4 rounded-xl mb-6 transition-all duration-300"
                    style={{
                        background: 'linear-gradient(135deg, rgba(28,78,216,0.06) 0%, rgba(69,180,194,0.06) 100%)',
                        border: '1px solid rgba(28,78,216,0.15)',
                    }}
                >
                    <span className="material-symbols-outlined text-2xl text-amber-500 shrink-0">lightbulb</span>
                    <div>
                        <p className="text-sm font-semibold text-neutral-800">
                            Based on your events, you may qualify for up to{' '}
                            <span style={{ color: '#1C4ED8' }}>
                                ₦{estimatedRelief.toLocaleString()} in reliefs
                            </span>
                        </p>
                        <p className="text-xs text-neutral-500 mt-0.5">We'll verify the exact figures as you file</p>
                    </div>
                </div>
            )}

            {/* Card grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 mb-8">
                {LIFE_EVENTS.map(evt => {
                    const isSelected = selected.has(evt.id)
                    return (
                        <button
                            key={evt.id}
                            type="button"
                            onClick={() => toggle(evt.id)}
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
                            <span className="material-symbols-outlined text-[26px] mb-2 leading-none" style={{ color: isSelected ? '#1C4ED8' : '#4b5563' }}>{evt.icon}</span>
                            <span
                                className="text-[12px] font-semibold leading-tight block"
                                style={{ color: isSelected ? '#1C4ED8' : '#1a1a1a' }}
                            >
                                {evt.label}
                            </span>
                            <span className="text-[11px] text-neutral-400 mt-0.5 leading-tight">{evt.sub}</span>
                        </button>
                    )
                })}
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
                    onClick={() => onNext({ lifeEvents: Array.from(selected) })}
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
