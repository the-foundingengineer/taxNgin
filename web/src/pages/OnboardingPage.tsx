import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { OnboardingLayout } from '@/components/layout/OnboardingLayout'
import { DynamicStepRenderer } from './onboarding/DynamicStepRenderer'
import { CheckpointScreen } from './onboarding/CheckpointScreen'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { fetchOnboardingSchema } from '@/lib/api/onboarding'

export function OnboardingPage() {
    const navigate = useNavigate()
    const setSchema = useOnboardingStore((s) => s.setSchema)
    const screens = useOnboardingStore((s) => s.screens)
    const currentScreenIndex = useOnboardingStore((s) => s.currentScreenIndex)
    const currentQuestionIndex = useOnboardingStore((s) => s.currentQuestionIndex)
    const direction = useOnboardingStore((s) => s.direction)
    const profile = useOnboardingStore((s) => s.profile)
    const showingCheckpoint = useOnboardingStore((s) => s.showingCheckpoint)
    const dismissCheckpoint = useOnboardingStore((s) => s.dismissCheckpoint)

    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [animKey, setAnimKey] = useState(0)
    const containerRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        fetchOnboardingSchema()
            .then((nodes) => { setSchema(nodes); setLoading(false) })
            .catch((err) => { setError(err.message ?? 'Failed to load'); setLoading(false) })
    }, [setSchema])

    useEffect(() => {
        setAnimKey((k) => k + 1)
        if (containerRef.current) containerRef.current.scrollTop = 0
    }, [currentScreenIndex, currentQuestionIndex, showingCheckpoint])

    // Navigate to dashboard when user advances past the last screen
    useEffect(() => {
        if (!loading && screens.length > 0 && currentScreenIndex >= screens.length && !showingCheckpoint) {
            localStorage.setItem('tc_onboarded', '1')
            localStorage.setItem('tc_profile', JSON.stringify(profile))
            navigate('/dashboard', { replace: true })
        }
    }, [currentScreenIndex, screens, profile, navigate, loading, showingCheckpoint])

    const screenLabels: Record<string, { label: string; icon: string }> = {
        '00': { label: 'Getting started', icon: 'rocket_launch' },
        '01': { label: 'Your identity', icon: 'person' },
        '02': { label: 'Tax identity', icon: 'badge' },
        '06': { label: 'Work situation', icon: 'work' },
        '07': { label: 'Employment', icon: 'payments' },
        '08': { label: 'Freelance', icon: 'terminal' },
    }
    const steps = screens.map((sid) => screenLabels[sid] ?? { label: `Screen ${sid}`, icon: 'help' })

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen">
                <div className="flex flex-col items-center gap-3">
                    <span className="material-symbols-outlined text-[40px] text-blue-600 animate-spin">
                        progress_activity
                    </span>
                    <p className="text-neutral-500 text-sm">Loading your tax profile…</p>
                </div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="flex items-center justify-center h-screen">
                <div className="text-center max-w-sm">
                    <span className="material-symbols-outlined text-[48px] text-red-400 mb-3 block">error</span>
                    <h2 className="text-lg font-bold text-neutral-900 mb-2">Something went wrong</h2>
                    <p className="text-sm text-neutral-500 mb-4">{error}</p>
                    <button
                        onClick={() => window.location.reload()}
                        className="px-6 py-2.5 rounded-xl text-sm font-semibold text-white"
                        style={{ background: 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)' }}
                    >
                        Try again
                    </button>
                </div>
            </div>
        )
    }

    const slideStyle: React.CSSProperties = {
        animation: showingCheckpoint
            ? 'checkpointFade 0.55s cubic-bezier(0.34,1.56,0.64,1) both'
            : `${direction === 'forward' ? 'slideInRight' : 'slideInLeft'} 0.3s cubic-bezier(0.25,0.46,0.45,0.94) both`,
    }

    return (
        <>
            <style>{`
                @keyframes slideInRight {
                    from { opacity: 0; transform: translateX(28px); }
                    to   { opacity: 1; transform: translateX(0); }
                }
                @keyframes slideInLeft {
                    from { opacity: 0; transform: translateX(-28px); }
                    to   { opacity: 1; transform: translateX(0); }
                }
            `}</style>

            <OnboardingLayout
                currentStep={currentScreenIndex}
                totalSteps={screens.length}
                steps={steps}
            >
                <div key={animKey} style={slideStyle} ref={containerRef}>
                    {showingCheckpoint
                        ? <CheckpointScreen config={showingCheckpoint} onContinue={dismissCheckpoint} />
                        : <DynamicStepRenderer />
                    }
                </div>
            </OnboardingLayout>
        </>
    )
}
