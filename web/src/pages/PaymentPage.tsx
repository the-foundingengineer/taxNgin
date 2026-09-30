import { useState, useEffect } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { useNavigate } from 'react-router-dom'
import { usePayment } from '@/hooks/usePayment'
import { computePITEstimate } from '@/hooks/usePITEstimate'
import { fmtNaira } from '@/lib/pit'
import { ArrowRight, Loader2, CheckCircle2, Building, CreditCard } from 'lucide-react'

export function PaymentPage() {
    const navigate = useNavigate()
    const { status, session, error, initiatePayment, markPaid } = usePayment()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const [profile, setProfile] = useState<Record<string, any>>({})

    useEffect(() => {
        try {
            const raw = localStorage.getItem('tc_profile')
            if (raw) setProfile(JSON.parse(raw))
        } catch { /* ignore */ }
    }, [])

    const estimate = computePITEstimate(profile)
    const taxDue = Math.max(0, -(estimate.balance))

    // ── Paid / success ────────────────────────────────────────────────────────
    if (status === 'success') {
        return (
            <DashboardLayout>
                <div className="max-w-xl mx-auto pb-24 pt-12 text-center animate-in fade-in duration-500">
                    <div className="w-24 h-24 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-6">
                        <CheckCircle2 className="w-14 h-14 text-emerald-500" />
                    </div>
                    <h1 className="text-3xl font-black text-neutral-900 mb-2">Payment Confirmed!</h1>
                    <p className="text-neutral-500 mb-2">Your tax payment of <strong>{fmtNaira(session?.amount ?? taxDue)}</strong> has been received.</p>
                    <p className="text-neutral-400 text-sm mb-8">Provider: {session?.provider === 'remita' ? 'Remita' : 'Paystack'} · Ref: {session?.rrr ?? localStorage.getItem('tc_efile_ref') ?? '—'}</p>
                    <button onClick={() => navigate('/tcc')} className="w-full h-14 bg-blue-600 text-white font-bold rounded-2xl flex items-center justify-center gap-2 text-lg">
                        Get Your TCC <ArrowRight className="w-5 h-5" />
                    </button>
                </div>
            </DashboardLayout>
        )
    }

    // ── Redirecting (demo mode) ────────────────────────────────────────────────
    if (status === 'redirecting') {
        return (
            <DashboardLayout>
                <div className="max-w-xl mx-auto pb-24 pt-12 text-center animate-in fade-in duration-500">
                    {session?.provider === 'remita' ? (
                        <>
                            <div className="w-20 h-20 bg-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-6">
                                <Building className="w-10 h-10 text-white" />
                            </div>
                            <h2 className="text-2xl font-black text-neutral-900 mb-2">Your Remita RRR</h2>
                            <p className="text-neutral-500 mb-6">Use this code at any Nigerian bank, or on the Remita portal.</p>
                            <div className="bg-blue-50 border-2 border-blue-600 rounded-2xl p-8 mb-6">
                                <p className="text-5xl font-black text-blue-700 tracking-widest font-mono">{session.rrr}</p>
                                <p className="text-lg font-bold text-blue-500 mt-2">Amount: {fmtNaira(session.amount)}</p>
                            </div>
                            <p className="text-xs text-neutral-400 mb-8">This RRR is valid for 3 banking days. After paying, click Confirm below.</p>
                        </>
                    ) : (
                        <>
                            <div className="w-20 h-20 bg-emerald-600 rounded-2xl flex items-center justify-center mx-auto mb-6">
                                <CreditCard className="w-10 h-10 text-white" />
                            </div>
                            <h2 className="text-2xl font-black text-neutral-900 mb-2">Paystack Checkout</h2>
                            <p className="text-neutral-500 mb-8">In production, a Paystack modal would appear here for card/USSD/bank transfer payment of <strong>{fmtNaira(session?.amount ?? taxDue)}</strong>.</p>
                        </>
                    )}
                    <button onClick={markPaid} className="w-full h-14 bg-emerald-600 hover:bg-emerald-700 text-white font-black text-lg rounded-2xl flex items-center justify-center gap-2">
                        <CheckCircle2 className="w-6 h-6" /> Confirm Payment Received
                    </button>
                </div>
            </DashboardLayout>
        )
    }

    // ── Default: choose provider ───────────────────────────────────────────────
    return (
        <DashboardLayout>
            <div className="max-w-2xl mx-auto pb-24 animate-in fade-in duration-500">
                <div className="pt-8 pb-8">
                    <p className="text-sm font-bold text-blue-600 uppercase tracking-widest mb-2">Step 4 of 4 — Payment</p>
                    <h1 className="text-3xl font-black text-neutral-900">Pay your tax balance.</h1>
                    <p className="text-neutral-500 mt-1">Choose your preferred payment method below.</p>
                </div>

                {/* Amount due banner */}
                <div className="bg-rose-600 text-white rounded-3xl p-8 mb-8">
                    <p className="text-rose-200 text-sm font-bold uppercase tracking-wide mb-1">Total Balance Due</p>
                    <p className="text-5xl font-black">{fmtNaira(taxDue)}</p>
                    <p className="text-rose-200 text-sm mt-2">For assessment year 2024 (Jan–Dec)</p>
                    <p className="text-rose-300 text-xs mt-1">NRS Reference: {localStorage.getItem('tc_efile_ref') ?? 'Pending filing'}</p>
                </div>

                {/* Provider cards */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
                    {/* Remita */}
                    <button
                        onClick={() => initiatePayment(taxDue, 'remita')}
                        disabled={status === 'initiating'}
                        className="text-left bg-white border-2 border-neutral-200 hover:border-blue-500 rounded-3xl p-6 transition-all shadow-sm hover:shadow-lg group"
                    >
                        <div className="w-12 h-12 bg-blue-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                            <Building className="w-6 h-6 text-white" />
                        </div>
                        <h3 className="text-lg font-black text-neutral-900 mb-1">Pay via Remita</h3>
                        <p className="text-sm text-neutral-500 leading-relaxed">Generate a Remita Retrieval Reference (RRR) and pay at any Nigerian bank, bank app, or the Remita portal.</p>
                        <div className="mt-4 flex items-center text-blue-600 font-bold text-sm gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            Get RRR <ArrowRight className="w-4 h-4" />
                        </div>
                    </button>

                    {/* Paystack */}
                    <button
                        onClick={() => initiatePayment(taxDue, 'paystack')}
                        disabled={status === 'initiating'}
                        className="text-left bg-white border-2 border-neutral-200 hover:border-emerald-500 rounded-3xl p-6 transition-all shadow-sm hover:shadow-lg group"
                    >
                        <div className="w-12 h-12 bg-emerald-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                            <CreditCard className="w-6 h-6 text-white" />
                        </div>
                        <h3 className="text-lg font-black text-neutral-900 mb-1">Pay via Paystack</h3>
                        <p className="text-sm text-neutral-500 leading-relaxed">Pay instantly with debit card, bank transfer, USSD, or QR code. Instant confirmation.</p>
                        <div className="mt-4 flex items-center text-emerald-600 font-bold text-sm gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            Pay Now <ArrowRight className="w-4 h-4" />
                        </div>
                    </button>
                </div>

                {status === 'initiating' && (
                    <div className="flex items-center justify-center gap-3 text-neutral-500">
                        <Loader2 className="w-5 h-5 animate-spin" /> Initiating payment session…
                    </div>
                )}
                {error && <p className="text-center text-rose-600 text-sm mt-4">{error}</p>}
            </div>
        </DashboardLayout>
    )
}
