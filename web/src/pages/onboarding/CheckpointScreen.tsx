import { useEffect, useState } from 'react'
import { useOnboardingStore } from '@/stores/onboardingStore'
import type { CheckpointConfig } from '@/stores/onboardingStore'

/**
 * Full-screen interstitial that fires between major onboarding sections.
 * Matches auth design language: #1C4ED8 blue accents, white panel, glassy cards.
 */
export function CheckpointScreen({ config, onContinue }: { config: CheckpointConfig; onContinue: () => void }) {
    const profile = useOnboardingStore((s) => s.profile)
    const [visible, setVisible] = useState(false)

    // Slight delay for mount animation
    useEffect(() => {
        const t = setTimeout(() => setVisible(true), 60)
        return () => clearTimeout(t)
    }, [])

    const pills = buildPills(config.section, profile)

    return (
        <>
            <style>{`
                @keyframes checkpointFade {
                    from { opacity: 0; transform: translateY(18px) scale(0.98); }
                    to   { opacity: 1; transform: translateY(0)   scale(1); }
                }
                @keyframes iconPop {
                    0%   { transform: scale(0.5); opacity: 0; }
                    60%  { transform: scale(1.12); opacity: 1; }
                    100% { transform: scale(1);    opacity: 1; }
                }
                @keyframes checkpointRing {
                    0%   { transform: scale(1);    opacity: 0.35; }
                    100% { transform: scale(1.9);  opacity: 0; }
                }
            `}</style>

            <div
                className="flex flex-col items-center justify-center text-center px-6 py-10"
                style={{
                    opacity: visible ? 1 : 0,
                    animation: visible ? 'checkpointFade 0.55s cubic-bezier(0.34,1.56,0.64,1) both' : undefined,
                }}
            >
                {/* Animated icon orb */}
                <div className="relative mb-8">
                    {/* Pulse rings */}
                    <div
                        className="absolute inset-0 rounded-full"
                        style={{
                            background: 'rgba(28,78,216,0.15)',
                            animation: 'checkpointRing 2s ease-out infinite',
                        }}
                    />
                    <div
                        className="absolute inset-0 rounded-full"
                        style={{
                            background: 'rgba(28,78,216,0.10)',
                            animation: 'checkpointRing 2s ease-out 0.4s infinite',
                        }}
                    />
                    {/* Icon orb */}
                    <div
                        className="relative w-20 h-20 rounded-full flex items-center justify-center"
                        style={{
                            background: 'linear-gradient(135deg, #1C4ED8 0%, #45B4C2 100%)',
                            boxShadow: '0 16px 48px rgba(28,78,216,0.35)',
                            animation: 'iconPop 0.6s cubic-bezier(0.34,1.56,0.64,1) 0.1s both',
                        }}
                    >
                        <span className="material-symbols-outlined text-[36px] text-white select-none">
                            {config.icon}
                        </span>
                    </div>
                </div>

                {/* Section complete badge */}
                <div
                    className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold mb-4"
                    style={{
                        background: 'rgba(28,78,216,0.07)',
                        color: '#1C4ED8',
                        border: '1px solid rgba(28,78,216,0.15)',
                    }}
                >
                    <span className="material-symbols-outlined text-[13px]">check_circle</span>
                    Section complete
                </div>

                {/* Headline */}
                <h1 className="text-[30px] lg:text-[36px] font-bold text-neutral-900 leading-tight mb-3">
                    {config.title}
                </h1>
                <p className="text-neutral-500 text-[15px] max-w-sm mx-auto mb-10">
                    {config.subtitle}
                </p>

                {/* "What's coming" card */}
                {pills.length > 0 && (
                    <div
                        className="w-full max-w-md rounded-2xl p-5 mb-10 text-left"
                        style={{
                            background: 'white',
                            border: '1.5px solid rgba(28,78,216,0.1)',
                            boxShadow: '0 8px 32px rgba(28,78,216,0.07)',
                        }}
                    >
                        <p className="text-[11px] font-bold uppercase tracking-widest text-neutral-400 mb-4">
                            What's coming next
                        </p>
                        <div className="space-y-3">
                            {pills.map((pill, i) => (
                                <div key={i} className="flex items-center gap-3">
                                    <div
                                        className="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
                                        style={{ background: 'rgba(28,78,216,0.07)' }}
                                    >
                                        <span
                                            className="material-symbols-outlined text-[16px]"
                                            style={{ color: '#1C4ED8' }}
                                        >
                                            {pill.icon}
                                        </span>
                                    </div>
                                    <div>
                                        <p className="text-sm font-semibold text-neutral-800 leading-tight">
                                            {pill.label}
                                        </p>
                                        {pill.sub && (
                                            <p className="text-[12px] text-neutral-400 mt-0.5">{pill.sub}</p>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* CTA */}
                <button
                    type="button"
                    onClick={onContinue}
                    className="h-13 px-10 rounded-xl text-white font-semibold text-[15px] flex items-center gap-2.5 transition-all hover:gap-4 hover:shadow-lg"
                    style={{
                        height: '52px',
                        background: 'linear-gradient(135deg, #1C4ED8 0%, #45B4C2 100%)',
                        boxShadow: '0 8px 24px rgba(28,78,216,0.32)',
                    }}
                >
                    <span>Let's keep going</span>
                    <span className="material-symbols-outlined text-[20px]">arrow_forward</span>
                </button>

                {/* Security footnote */}
                <div className="flex items-center gap-1.5 mt-8">
                    <span className="material-symbols-outlined text-[14px] text-neutral-300">lock</span>
                    <span className="text-[12px] text-neutral-400">Your information is encrypted end-to-end</span>
                </div>
            </div>
        </>
    )
}

// ---------------------------------------------------------------------------
// Personalised "what's coming" pill builder
// ---------------------------------------------------------------------------

interface Pill {
    icon: string
    label: string
    sub?: string
}

function buildPills(section: string, profile: Record<string, unknown>): Pill[] {
    const situation = (Array.isArray(profile.work_situation) ? profile.work_situation : []) as string[]
    const hasNin = profile.has_nin === 'yes'

    if (section === 'income') {
        const pills: Pill[] = []
        if (situation.includes('employee')) {
            pills.push({ icon: 'receipt_long', label: 'Your PAYE slip figures', sub: 'Based on Form H1 layout' })
        }
        if (situation.includes('freelancer')) {
            pills.push({ icon: 'manage_accounts', label: 'Withholding tax (WHT) credits', sub: 'Project-level deductions' })
        }
        if (situation.includes('landlord')) {
            pills.push({ icon: 'home_work', label: 'Rental income & Schedule E', sub: 'Property relief calculations' })
        }
        if (situation.includes('business_owner')) {
            pills.push({ icon: 'trending_up', label: 'Business profit & loss', sub: 'Allowable expenses & depreciation' })
        }
        if (!hasNin) {
            pills.push({
                icon: 'badge', label: "We'll help you register your TIN", sub: 'Quick process, no queue'
            })
        }
        if (pills.length === 0) {
            pills.push({ icon: 'payments', label: 'Your income sources', sub: 'All categories covered' })
        }
        return pills.slice(0, 3)
    }

    if (section === 'deductions') {
        const pills: Pill[] = [
            { icon: 'savings', label: 'Pension contributions (RSA)', sub: 'Up to ₦ 1.2M tax-free' },
            { icon: 'home', label: 'Rent relief claim', sub: '20% of income, max ₦500k' },
            { icon: 'health_and_safety', label: 'NHF & life assurance premiums', sub: 'Statutory deductions' },
        ]
        return pills.slice(0, 3)
    }

    return [
        { icon: 'check_circle', label: `${section.charAt(0).toUpperCase() + section.slice(1)} complete`, sub: 'Continue to the next section' },
    ]
}
