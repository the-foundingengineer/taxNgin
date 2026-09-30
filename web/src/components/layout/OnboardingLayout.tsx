import { ReactNode, useEffect, useRef } from 'react'
import { usePITEstimate } from '@/hooks/usePITEstimate'
import { fmtNaira } from '@/lib/pit'

interface Step {
    label: string
    icon: string
}

interface OnboardingLayoutProps {
    children: ReactNode
    currentStep: number   // 0-indexed
    totalSteps: number
    steps: Step[]
}

export function OnboardingLayout({
    children,
    currentStep,
    totalSteps,
    steps,
}: OnboardingLayoutProps) {
    const progressPct = ((currentStep + 1) / totalSteps) * 100
    const est = usePITEstimate()

    const activeStepRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        if (activeStepRef.current) {
            activeStepRef.current.scrollIntoView({
                behavior: 'smooth',
                block: 'center',
            })
        }
    }, [currentStep])

    return (
        <div
            className="h-screen w-full flex font-sans overflow-hidden"
            style={{ backgroundColor: '#f5f4ef' }}
        >
            {/* ══ LEFT: Blue nav panel (desktop only) ══ */}
            <div
                className="hidden lg:flex w-[300px] shrink-0 flex-col relative overflow-hidden"
                style={{ backgroundColor: '#1C4ED8' }}
            >
                {/* Background geometry */}
                <div className="absolute top-0 right-0 w-64 h-64 rounded-full opacity-10"
                    style={{ background: 'radial-gradient(circle, white 0%, transparent 70%)', transform: 'translate(30%, -30%)' }} />
                <div className="absolute bottom-0 left-0 w-48 h-48 rounded-full opacity-5"
                    style={{ background: 'radial-gradient(circle, white 0%, transparent 70%)', transform: 'translate(-30%, 30%)' }} />
                <div className="absolute top-[40%] right-6 opacity-10">
                    <svg width="60" height="120" viewBox="0 0 60 120">
                        <line x1="30" y1="0" x2="30" y2="120" stroke="white" strokeWidth="0.5" strokeDasharray="4 4" />
                        <circle cx="30" cy="20" r="2" fill="white" opacity="0.5" />
                        <circle cx="30" cy="60" r="2" fill="white" opacity="0.5" />
                        <circle cx="30" cy="100" r="2" fill="white" opacity="0.5" />
                    </svg>
                </div>

                {/* Wordmark */}
                <div className="px-8 pt-10 pb-0 flex items-center gap-2.5 select-none">
                    <div
                        className="w-7 h-7 rounded-full flex items-center justify-center"
                        style={{ background: 'rgba(255,255,255,0.2)', backdropFilter: 'blur(8px)' }}
                    >
                        <span className="transform rotate-45 block w-3 h-3 border-2 border-white border-t-0 border-l-0 rounded-sm" />
                    </div>
                    <span className="text-white text-[17px] font-bold tracking-tight">Tax Check.</span>
                </div>

                {/* Step rail */}
                <div className="flex-1 flex flex-col py-10 overflow-y-auto no-scrollbar scroll-smooth" style={{ minHeight: 0 }}>
                    <div className="px-8 flex-1 flex flex-col justify-center">
                        <p className="text-white/40 text-xs font-semibold uppercase tracking-widest mb-8">Your Setup</p>
                        <div className="flex flex-col gap-0">
                            {steps.map((step, i) => {
                                const isDone = i < currentStep
                                const isActive = i === currentStep
                                return (
                                    <div
                                        key={i}
                                        ref={isActive ? activeStepRef : null}
                                        className="flex items-start gap-4"
                                    >
                                        {/* Dot + connector line */}
                                        <div className="flex flex-col items-center">
                                            <div
                                                className="relative flex items-center justify-center w-8 h-8 rounded-full transition-all duration-500 shrink-0"
                                                style={{
                                                    background: isDone
                                                        ? 'rgba(255,255,255,0.95)'
                                                        : isActive
                                                            ? 'rgba(255,255,255,0.15)'
                                                            : 'rgba(255,255,255,0.06)',
                                                    border: isActive
                                                        ? '2px solid rgba(255,255,255,0.9)'
                                                        : isDone
                                                            ? '2px solid rgba(255,255,255,0.9)'
                                                            : '2px solid rgba(255,255,255,0.2)',
                                                    boxShadow: isActive ? '0 0 0 5px rgba(255,255,255,0.12)' : 'none',
                                                }}
                                            >
                                                {isDone ? (
                                                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                                                        <path d="M2.5 7l3.5 3.5 5.5-7" stroke="#1C4ED8" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                                                    </svg>
                                                ) : (
                                                    <span
                                                        className="material-symbols-outlined text-[16px] transition-all duration-300"
                                                        style={{
                                                            opacity: isActive ? 1 : 0.4,
                                                            color: 'white',
                                                        }}
                                                    >
                                                        {step.icon}
                                                    </span>
                                                )}
                                                {/* Pulse ring on active */}
                                                {isActive && (
                                                    <span className="absolute inset-0 rounded-full animate-ping"
                                                        style={{ background: 'rgba(255,255,255,0.15)', animationDuration: '2s' }} />
                                                )}
                                            </div>
                                            {/* Connector line */}
                                            {i < steps.length - 1 && (
                                                <div
                                                    className="w-px mt-0 transition-all duration-700"
                                                    style={{
                                                        height: '40px',
                                                        background: isDone
                                                            ? 'rgba(255,255,255,0.7)'
                                                            : 'rgba(255,255,255,0.15)',
                                                    }}
                                                />
                                            )}
                                        </div>

                                        {/* Label */}
                                        <div className="pt-1.5 pb-[40px]" style={{ paddingBottom: i < steps.length - 1 ? undefined : 0 }}>
                                            <span
                                                className="text-sm font-medium block transition-all duration-300 leading-tight"
                                                style={{
                                                    color: isDone
                                                        ? 'rgba(255,255,255,0.9)'
                                                        : isActive
                                                            ? 'white'
                                                            : 'rgba(255,255,255,0.35)',
                                                    fontWeight: isActive ? 700 : 500,
                                                }}
                                            >
                                                {step.label}
                                            </span>
                                            {isDone && (
                                                <span className="text-[11px] text-white/40 font-medium mt-0.5 block">Complete</span>
                                            )}
                                            {isActive && (
                                                <span className="text-[11px] text-white/60 font-medium mt-0.5 block">In progress</span>
                                            )}
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                </div>

                {/* ── Live tax estimate widget ── */}
                <div className="px-8 pb-4">
                    {est.hasData ? (
                        <div
                            className="rounded-xl p-4"
                            style={{ background: 'rgba(255,255,255,0.09)', border: '1px solid rgba(255,255,255,0.18)' }}
                        >
                            <div className="flex items-center gap-1.5 mb-3">
                                <span className="material-symbols-outlined text-[13px]" style={{ color: 'rgba(255,255,255,0.5)' }}>
                                    calculate
                                </span>
                                <span className="text-[9px] font-bold uppercase tracking-widest" style={{ color: 'rgba(255,255,255,0.45)' }}>
                                    Live estimate
                                </span>
                            </div>

                            <div className="space-y-1.5">
                                <Row label="Income" value={fmtNaira(est.totalIncome)} />
                                {est.totalDeductions > 0 && (
                                    <Row label="Deductions" value={`-${fmtNaira(est.totalDeductions)}`} dim />
                                )}
                                <Divider />
                                <Row label="Tax payable" value={fmtNaira(est.taxPayable)} />
                                {est.taxAlreadyPaid > 0 && (
                                    <Row label="Paid (PAYE / WHT)" value={`-${fmtNaira(est.taxAlreadyPaid)}`} dim />
                                )}
                                <Divider />
                                {est.balance >= 0 ? (
                                    <div className="flex justify-between items-center pt-0.5">
                                        <span className="text-[11px] font-bold text-amber-300">Est. refund</span>
                                        <span className="text-[14px] font-bold text-amber-300">{fmtNaira(est.balance)}</span>
                                    </div>
                                ) : (
                                    <div className="flex justify-between items-center pt-0.5">
                                        <span className="text-[11px] font-semibold" style={{ color: 'rgba(255,255,255,0.75)' }}>
                                            Est. tax due
                                        </span>
                                        <span className="text-[14px] font-bold" style={{ color: 'rgba(255,255,255,0.9)' }}>
                                            {fmtNaira(-est.balance)}
                                        </span>
                                    </div>
                                )}
                            </div>
                        </div>
                    ) : (
                        <div
                            className="rounded-xl p-4"
                            style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.09)' }}
                        >
                            <div className="flex items-center gap-1.5 mb-2">
                                <span className="material-symbols-outlined text-[13px]" style={{ color: 'rgba(255,255,255,0.35)' }}>
                                    calculate
                                </span>
                                <span className="text-[9px] font-bold uppercase tracking-widest" style={{ color: 'rgba(255,255,255,0.3)' }}>
                                    Live estimate
                                </span>
                            </div>
                            <p className="text-[11px] leading-relaxed" style={{ color: 'rgba(255,255,255,0.35)' }}>
                                Enter your income and your live tax position will appear here.
                            </p>
                        </div>
                    )}
                </div>

                {/* Bottom tip */}
                <div className="px-8 pb-10">
                    <div
                        className="rounded-xl p-4"
                        style={{ background: 'rgba(255,255,255,0.08)', border: '1px solid rgba(255,255,255,0.1)' }}
                    >
                        <div className="flex items-start gap-3">
                            <span className="material-symbols-outlined text-lg text-white/80 shrink-0">lock</span>
                            <p className="text-white/60 text-xs leading-relaxed">
                                Your information is encrypted and never shared. We're here to get you the most out of every naira.
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            {/* ══ RIGHT: Content panel ══ */}
            <div className="flex-1 flex flex-col overflow-y-auto no-scrollbar scroll-smooth">
                {/* Progress bar */}
                <div className="h-1 w-full bg-neutral-200 shrink-0">
                    <div
                        className="h-full rounded-r-full transition-all duration-700 ease-out"
                        style={{
                            width: `${progressPct}%`,
                            background: 'linear-gradient(90deg, #1C4ED8 0%, #45B4C2 100%)',
                        }}
                    />
                </div>

                {/* Mobile header */}
                <div
                    className="lg:hidden flex items-center justify-between px-6 py-4 shrink-0 border-b border-neutral-100"
                    style={{ backgroundColor: 'white' }}
                >
                    <div className="flex items-center gap-2">
                        <div
                            className="w-6 h-6 rounded-full flex items-center justify-center"
                            style={{ background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)' }}
                        >
                            <span className="transform rotate-45 block w-2.5 h-2.5 border-2 border-white border-t-0 border-l-0 rounded-sm" />
                        </div>
                        <span className="text-sm font-bold tracking-tight text-neutral-900">Tax Check.</span>
                    </div>
                    <span className="text-xs font-semibold text-neutral-400">
                        Step {currentStep + 1} of {totalSteps}
                    </span>
                </div>

                {/* Mobile live estimate strip — only when income data exists */}
                {est.hasData && (
                    <div
                        className="lg:hidden flex items-center justify-between px-6 py-2.5 shrink-0"
                        style={{ backgroundColor: '#1C4ED8' }}
                    >
                        <div className="flex items-center gap-1.5">
                            <span className="material-symbols-outlined text-[13px]" style={{ color: 'rgba(255,255,255,0.6)' }}>
                                calculate
                            </span>
                            <span className="text-[11px] font-medium" style={{ color: 'rgba(255,255,255,0.6)' }}>
                                Live estimate
                            </span>
                        </div>
                        <div className="flex items-center gap-3">
                            <span className="text-[11px]" style={{ color: 'rgba(255,255,255,0.55)' }}>
                                Tax {fmtNaira(est.taxPayable)}
                            </span>
                            {est.balance >= 0 ? (
                                <span className="text-[12px] font-bold text-amber-300">
                                    Refund {fmtNaira(est.balance)}
                                </span>
                            ) : (
                                <span className="text-[12px] font-bold text-white">
                                    Due {fmtNaira(-est.balance)}
                                </span>
                            )}
                        </div>
                    </div>
                )}

                {/* Step counter pill (desktop) */}
                <div className="hidden lg:flex justify-end px-10 pt-8 shrink-0">
                    <div
                        className="px-4 py-1.5 rounded-full text-xs font-semibold"
                        style={{
                            background: 'rgba(28, 78, 216, 0.06)',
                            color: '#1C4ED8',
                            border: '1px solid rgba(28, 78, 216, 0.15)',
                        }}
                    >
                        Step {currentStep + 1} of {totalSteps}
                    </div>
                </div>

                {/* Page content */}
                <div className="flex-1 flex flex-col items-center justify-start px-6 py-10 lg:py-16">
                    <div className="w-full max-w-2xl">
                        {children}
                    </div>
                </div>
            </div>
        </div>
    )
}

// ---------------------------------------------------------------------------
// Estimate widget helpers — scoped to this file
// ---------------------------------------------------------------------------

function Row({ label, value, dim }: { label: string; value: string; dim?: boolean }) {
    return (
        <div className="flex justify-between items-center">
            <span className="text-[11px]" style={{ color: dim ? 'rgba(255,255,255,0.45)' : 'rgba(255,255,255,0.6)' }}>
                {label}
            </span>
            <span className="text-[11px] font-semibold" style={{ color: dim ? 'rgba(255,255,255,0.55)' : 'rgba(255,255,255,0.9)' }}>
                {value}
            </span>
        </div>
    )
}

function Divider() {
    return <div className="border-t my-1.5" style={{ borderColor: 'rgba(255,255,255,0.1)' }} />
}
