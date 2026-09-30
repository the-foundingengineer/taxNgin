import { useEffect, useState } from 'react'
import type { DeductionToast } from '@/hooks/useDeductionMilestones'

interface ContainerProps {
    toasts:    DeductionToast[]
    onDismiss: (key: string) => void
    autoDismissMs: number
}

export function MilestoneToastContainer({ toasts, onDismiss, autoDismissMs }: ContainerProps) {
    if (toasts.length === 0) return null
    return (
        <>
            <style>{`
                @keyframes toastSlideIn {
                    from { opacity: 0; transform: translateX(110%); }
                    to   { opacity: 1; transform: translateX(0); }
                }
                @keyframes toastProgress {
                    from { width: 100%; }
                    to   { width: 0%; }
                }
            `}</style>
            <div className="fixed bottom-6 right-4 lg:right-6 z-50 flex flex-col gap-3 items-end pointer-events-none">
                {toasts.map((t) => (
                    <ToastCard
                        key={t.key}
                        toast={t}
                        onDismiss={onDismiss}
                        autoDismissMs={autoDismissMs}
                    />
                ))}
            </div>
        </>
    )
}

function ToastCard({
    toast,
    onDismiss,
    autoDismissMs,
}: {
    toast: DeductionToast
    onDismiss: (key: string) => void
    autoDismissMs: number
}) {
    const [leaving, setLeaving] = useState(false)

    useEffect(() => {
        const leave  = setTimeout(() => setLeaving(true), autoDismissMs - 400)
        const remove = setTimeout(() => onDismiss(toast.key), autoDismissMs)
        return () => { clearTimeout(leave); clearTimeout(remove) }
    }, [toast.key, onDismiss, autoDismissMs])

    function handleDismiss() {
        setLeaving(true)
        setTimeout(() => onDismiss(toast.key), 350)
    }

    return (
        <div
            className="pointer-events-auto relative overflow-hidden bg-white rounded-2xl"
            style={{
                minWidth: '270px',
                maxWidth: '330px',
                border: '1px solid rgba(28,78,216,0.14)',
                boxShadow: '0 8px 32px rgba(28,78,216,0.14), 0 2px 8px rgba(0,0,0,0.06)',
                animation: leaving
                    ? 'toastSlideIn 0.35s cubic-bezier(0.55,0,1,0.45) reverse forwards'
                    : 'toastSlideIn 0.45s cubic-bezier(0.34,1.56,0.64,1) both',
            }}
        >
            {/* Top gradient accent bar */}
            <div
                className="h-[3px] w-full"
                style={{ background: 'linear-gradient(90deg, #1C4ED8 0%, #45B4C2 100%)' }}
            />

            <div className="flex items-start gap-3 px-4 pt-3 pb-3.5">
                {/* Icon */}
                <div
                    className="w-9 h-9 rounded-full flex items-center justify-center shrink-0"
                    style={{ background: 'linear-gradient(135deg, #1C4ED8 0%, #45B4C2 100%)' }}
                >
                    <span className="material-symbols-outlined text-[18px] text-white select-none">
                        {toast.icon}
                    </span>
                </div>

                {/* Copy */}
                <div className="flex-1 min-w-0 pt-0.5">
                    <p className="text-[13px] font-bold text-neutral-900 leading-tight">
                        {toast.title}
                    </p>
                    <p className="text-[12px] font-semibold mt-0.5" style={{ color: '#d97706' }}>
                        {toast.subtitle}
                    </p>
                </div>

                {/* Close */}
                <button
                    type="button"
                    onClick={handleDismiss}
                    className="shrink-0 transition-colors hover:text-neutral-500 mt-0.5"
                    style={{ color: '#d1d5db' }}
                    aria-label="Dismiss"
                >
                    <span className="material-symbols-outlined text-[16px]">close</span>
                </button>
            </div>

            {/* Auto-dismiss progress bar */}
            <div
                className="absolute bottom-0 left-0 h-[2px] rounded-full"
                style={{
                    background: 'rgba(28,78,216,0.2)',
                    animation: `toastProgress ${autoDismissMs}ms linear forwards`,
                }}
            />
        </div>
    )
}
