import { useEffect, useState } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { runNTAOptimizer, OptimizerResult, Relief } from '@/hooks/useNTAOptimizer'
import { fmtNaira } from '@/lib/pit'
import { ArrowRight, CheckCircle2, AlertCircle } from 'lucide-react'

export function OptimizerPage() {
    const [result, setResult] = useState<OptimizerResult | null>(null)

    useEffect(() => {
        try {
            const raw = localStorage.getItem('tc_profile')
            const profile = raw ? JSON.parse(raw) : {}
            setResult(runNTAOptimizer(profile))
        } catch {
            setResult(runNTAOptimizer({}))
        }
    }, [])

    if (!result) return null

    const totalSaving = result.totalFoundSaving + result.totalMissedSaving

    return (
        <DashboardLayout>
            <div className="space-y-10 pb-24 animate-in fade-in slide-in-from-bottom-4 duration-500">

                {/* Header */}
                <div>
                    <p className="text-sm font-bold text-blue-600 uppercase tracking-widest mb-2">NTA Optimizer</p>
                    <h1 className="text-4xl font-black text-neutral-900 tracking-tight">Your tax relief scan</h1>
                    <p className="text-neutral-500 mt-2 text-lg">
                        We checked your profile against every NTA 2025 s.30 relief. Here's what we found.
                    </p>
                </div>

                {/* Score banner */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="col-span-1 md:col-span-3 bg-gradient-to-r from-blue-700 to-blue-900 text-white rounded-2xl p-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
                        <div>
                            <p className="text-blue-300 text-xs font-bold uppercase tracking-widest mb-1">Total reliefs identified</p>
                            <h2 className="text-5xl font-black tracking-tight">{fmtNaira(totalSaving)}</h2>
                            <p className="text-blue-200 mt-1">in potential tax savings across {result.found.length + result.missed.length} applicable reliefs</p>
                        </div>
                        <div className="flex gap-6 shrink-0">
                            <div className="text-center">
                                <p className="text-3xl font-black text-emerald-400">{result.found.length}</p>
                                <p className="text-xs text-blue-300 mt-0.5">Captured</p>
                            </div>
                            <div className="w-px bg-white/20" />
                            <div className="text-center">
                                <p className="text-3xl font-black text-amber-400">{result.missed.length}</p>
                                <p className="text-xs text-blue-300 mt-0.5">Potentially missed</p>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Missed reliefs — show first, most important */}
                {result.missed.length > 0 && (
                    <section>
                        <div className="flex items-center gap-3 mb-5">
                            <AlertCircle className="w-5 h-5 text-amber-500" />
                            <h2 className="text-xl font-bold text-neutral-900">
                                Potentially unclaimed — {fmtNaira(result.totalMissedSaving)} at stake
                            </h2>
                        </div>
                        <div className="space-y-4">
                            {result.missed.map(r => <ReliefCard key={r.id} relief={r} />)}
                        </div>
                    </section>
                )}

                {/* Found reliefs */}
                {result.found.length > 0 && (
                    <section>
                        <div className="flex items-center gap-3 mb-5">
                            <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                            <h2 className="text-xl font-bold text-neutral-900">
                                Already captured — {fmtNaira(result.totalFoundSaving)} locked in
                            </h2>
                        </div>
                        <div className="space-y-4">
                            {result.found.map(r => <ReliefCard key={r.id} relief={r} />)}
                        </div>
                    </section>
                )}
            </div>
        </DashboardLayout>
    )
}

function ReliefCard({ relief }: { relief: Relief }) {
    const isFound = relief.status === 'found'

    return (
        <div className={`bg-white rounded-2xl border p-6 shadow-sm flex items-start gap-5 transition-all hover:shadow-md ${isFound ? 'border-emerald-200' : 'border-amber-200'
            }`}>
            {/* Icon */}
            <div className={`w-12 h-12 rounded-xl flex items-center justify-center shrink-0 ${isFound ? 'bg-emerald-50 text-emerald-600' : 'bg-amber-50 text-amber-600'
                }`}>
                <span className="material-symbols-outlined text-[22px]">{relief.icon}</span>
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-4 flex-wrap">
                    <div>
                        <h3 className="font-bold text-neutral-900 text-[15px] leading-tight">{relief.title}</h3>
                        <p className="text-xs text-neutral-400 mt-0.5 font-mono">{relief.basis}</p>
                    </div>
                    <div className={`shrink-0 px-3 py-1 rounded-full text-xs font-bold ${isFound
                            ? 'bg-emerald-100 text-emerald-700'
                            : 'bg-amber-100 text-amber-700'
                        }`}>
                        {isFound ? `Est. saving: ${fmtNaira(relief.estimatedSaving)}` : `Up to ${fmtNaira(relief.estimatedSaving)} available`}
                    </div>
                </div>
                <p className="text-sm text-neutral-600 mt-2 leading-relaxed">{relief.description}</p>

                {relief.actionLabel && (
                    <button className="mt-3 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-800 transition-colors">
                        {relief.actionLabel}
                        <ArrowRight className="w-3.5 h-3.5" />
                    </button>
                )}
            </div>

            {/* Status dot */}
            <div className="shrink-0 pt-1">
                {isFound
                    ? <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                    : <AlertCircle className="w-5 h-5 text-amber-400" />
                }
            </div>
        </div>
    )
}
