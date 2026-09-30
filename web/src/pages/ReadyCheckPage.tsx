import { useEffect, useState } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { useNavigate } from 'react-router-dom'
import { runReadyCheck, ReadyCheckItem } from '@/hooks/useReadyCheck'
import { CheckCircle2, XCircle, AlertCircle, ArrowRight, ClipboardCheck } from 'lucide-react'
import { fmtNaira } from '@/lib/pit'
import { computePITEstimate } from '@/hooks/usePITEstimate'

const STATUS_META: Record<'pass' | 'warn' | 'fail', { icon: typeof CheckCircle2; colour: string; label: string }> = {
    pass: { icon: CheckCircle2, colour: 'text-emerald-500', label: 'Complete' },
    warn: { icon: AlertCircle, colour: 'text-amber-500', label: 'Recommended' },
    fail: { icon: XCircle, colour: 'text-rose-500', label: 'Required' },
}

function CheckRow({ item }: { item: ReadyCheckItem }) {
    const meta = STATUS_META[item.status]
    const IconEl = meta.icon
    const nav = useNavigate()
    return (
        <div className={`flex items-start gap-4 p-4 rounded-2xl border transition-all ${item.status === 'fail' ? 'bg-rose-50 border-rose-200' :
            item.status === 'warn' ? 'bg-amber-50 border-amber-200' :
                'bg-emerald-50 border-emerald-200'
            }`}>
            <IconEl className={`w-5 h-5 mt-0.5 shrink-0 ${meta.colour}`} />
            <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-0.5">
                    <p className="text-sm font-bold text-neutral-900">{item.label}</p>
                    <span className={`text-[10px] font-black uppercase tracking-wider ${meta.colour}`}>{meta.label}</span>
                </div>
                <p className="text-xs text-neutral-500">{item.description}</p>
            </div>
            {item.status !== 'pass' && item.fixRoute && (
                <button onClick={() => nav(item.fixRoute!)} className="shrink-0 text-xs font-bold text-blue-600 hover:text-blue-800 flex items-center gap-1 mt-0.5 whitespace-nowrap">
                    Fix it <ArrowRight className="w-3 h-3" />
                </button>
            )}
        </div>
    )
}

export function ReadyCheckPage() {
    const navigate = useNavigate()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const [profile, setProfile] = useState<Record<string, any>>({})
    const [confirmed, setConfirmed] = useState(false)

    useEffect(() => {
        try {
            const raw = localStorage.getItem('tc_profile')
            if (raw) setProfile(JSON.parse(raw))
        } catch { /* ignore */ }
    }, [])

    function handleProceed() {
        const updated = { ...profile, declaration_confirmed: true }
        localStorage.setItem('tc_profile', JSON.stringify(updated))
        setProfile(updated)
        navigate('/efile')
    }

    const result = runReadyCheck(profile)
    const estimate = computePITEstimate(profile)

    const categories = [...new Set(result.items.map(i => i.category))]

    return (
        <DashboardLayout>
            <div className="max-w-2xl mx-auto pb-24 animate-in fade-in duration-500">

                {/* Header */}
                <div className="pt-8 pb-8 flex items-start gap-4">
                    <div className="w-14 h-14 bg-blue-600 rounded-2xl flex items-center justify-center shrink-0 mt-1">
                        <ClipboardCheck className="w-7 h-7 text-white" />
                    </div>
                    <div>
                        <p className="text-sm font-bold text-blue-600 uppercase tracking-widest mb-1">NRS Filing Readiness</p>
                        <h1 className="text-3xl font-black text-neutral-900">Ready to file?</h1>
                        <p className="text-neutral-500 mt-1">We've checked your return against NRS submission requirements.</p>
                    </div>
                </div>

                {/* Score ring + summary banner */}
                <div className={`rounded-3xl p-6 flex items-center gap-6 mb-8 ${result.canFile ? 'bg-emerald-600' : 'bg-neutral-900'}`}>
                    {/* Score ring */}
                    <div className="relative w-20 h-20 shrink-0">
                        <svg className="w-full h-full -rotate-90" viewBox="0 0 36 36">
                            <circle className="fill-none stroke-white/20" strokeWidth="3.5" r="15.9" cx="18" cy="18" />
                            <circle
                                className="fill-none stroke-white transition-all duration-700"
                                strokeWidth="3.5"
                                strokeLinecap="round"
                                strokeDasharray={`${result.score} ${100 - result.score}`}
                                r="15.9" cx="18" cy="18"
                            />
                        </svg>
                        <span className="absolute inset-0 flex items-center justify-center text-xl font-black text-white">{result.score}%</span>
                    </div>
                    <div className="text-white">
                        <h2 className="text-2xl font-black leading-tight">
                            {result.canFile ? 'Excellent! Ready to file.' : 'Almost ready to file.'}
                        </h2>
                        <p className="text-sm opacity-80 mt-1">
                            {result.canFile
                                ? 'All required fields are complete. Review below and proceed to e-file.'
                                : `${result.items.filter(i => i.status === 'fail').length} required issue(s) must be fixed before filing.`}
                        </p>
                    </div>
                </div>

                {/* Return estimate snapshot */}
                {estimate.hasData && (
                    <div className="bg-white border border-neutral-200 rounded-2xl p-5 mb-8 flex items-center justify-between">
                        <div>
                            <p className="text-xs font-bold text-neutral-400 uppercase tracking-wide">Your 2024 Tax Position</p>
                            <p className={`text-2xl font-black mt-1 ${estimate.balance >= 0 ? 'text-emerald-700' : 'text-rose-700'}`}>
                                {estimate.balance >= 0 ? `₦${estimate.balance.toLocaleString()} Refund` : `₦${Math.abs(estimate.balance).toLocaleString()} Due`}
                            </p>
                        </div>
                        <div className="text-right text-sm text-neutral-500 space-y-0.5">
                            <p>Tax Payable: <strong>{fmtNaira(estimate.taxPayable)}</strong></p>
                            <p>PAYE Paid:   <strong>{fmtNaira(estimate.taxAlreadyPaid)}</strong></p>
                        </div>
                    </div>
                )}

                {/* Check list by category */}
                <div className="space-y-6 mb-8">
                    {categories.map(cat => (
                        <div key={cat}>
                            <h3 className="text-xs font-black text-neutral-400 uppercase tracking-widest mb-3">{cat}</h3>
                            <div className="space-y-2">
                                {result.items.filter(i => i.category === cat).map(item => (
                                    <CheckRow key={item.id} item={item} />
                                ))}
                            </div>
                        </div>
                    ))}
                </div>

                {/* Declaration */}
                <div className="bg-neutral-50 border border-neutral-200 rounded-2xl p-5 mb-6">
                    <label className="flex items-start gap-3 cursor-pointer">
                        <input type="checkbox" checked={confirmed} onChange={e => setConfirmed(e.target.checked)}
                            className="mt-0.5 w-5 h-5 accent-blue-600 rounded" />
                        <p className="text-sm text-neutral-700 leading-relaxed">
                            I declare that all information provided is accurate and complete to the best of my knowledge. I understand that submitting a false return is an offence under NTA 2025 s.68.
                        </p>
                    </label>
                </div>

                <button
                    disabled={!result.canFile || !confirmed}
                    onClick={handleProceed}
                    className="w-full h-14 bg-blue-600 hover:bg-blue-700 disabled:bg-neutral-300 disabled:cursor-not-allowed text-white font-black text-lg rounded-2xl transition-colors flex items-center justify-center gap-3 shadow-lg shadow-blue-200"
                >
                    Proceed to E-File <ArrowRight className="w-5 h-5" />
                </button>
                {!confirmed && <p className="text-center text-xs text-neutral-400 mt-3">You must accept the declaration above to continue</p>}
            </div>
        </DashboardLayout>
    )
}
