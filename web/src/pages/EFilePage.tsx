import { useState, useEffect } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { useNavigate } from 'react-router-dom'
import { useEFile } from '@/hooks/useEFile'
import { computePITEstimate } from '@/hooks/usePITEstimate'
import { fmtNaira } from '@/lib/pit'
import { Send, Loader2, CheckCircle2, XCircle, FileText, ArrowRight } from 'lucide-react'

function SummaryRow({ label, value, bold }: { label: string; value: string; bold?: boolean }) {
    return (
        <div className="flex items-center justify-between py-3 border-b border-neutral-100 last:border-0">
            <span className="text-sm text-neutral-600">{label}</span>
            <span className={`text-sm ${bold ? 'font-black text-neutral-900' : 'font-medium text-neutral-700'}`}>{value}</span>
        </div>
    )
}

export function EFilePage() {
    const navigate = useNavigate()
    const { status, result, error, submitReturn } = useEFile()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const [profile, setProfile] = useState<Record<string, any>>({})

    useEffect(() => {
        try {
            const raw = localStorage.getItem('tc_profile')
            if (raw) setProfile(JSON.parse(raw))
        } catch { /* ignore */ }
    }, [])

    const estimate = computePITEstimate(profile)

    const taxDue = Math.max(0, -estimate.balance)
    const refund = Math.max(0, estimate.balance)

    async function handleSubmit() {
        await submitReturn(profile)
    }

    // ── Success state ──────────────────────────────────────────────────────────
    if (status === 'success' && result) {
        return (
            <DashboardLayout>
                <div className="max-w-xl mx-auto pb-24 pt-12 animate-in fade-in duration-500 text-center">
                    <div className="w-24 h-24 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-6">
                        <CheckCircle2 className="w-14 h-14 text-emerald-500" />
                    </div>
                    <h1 className="text-3xl font-black text-neutral-900 mb-2">Return Filed!</h1>
                    <p className="text-neutral-500 text-lg mb-8">Your 2024 Income Tax Return has been submitted to the NRS.</p>

                    <div className="bg-white border border-neutral-200 rounded-3xl p-8 text-left mb-6 shadow-sm">
                        <p className="text-xs font-black text-neutral-400 uppercase tracking-widest mb-4">Filing Confirmation</p>
                        <SummaryRow label="NRS Reference Number" value={result.referenceNumber} bold />
                        <SummaryRow label="Filed At" value={new Date(result.filedAt).toLocaleString('en-NG', { timeZone: 'Africa/Lagos' })} />
                        <SummaryRow label="Assessment Year" value="2024 (January - December)" />
                        <SummaryRow label="Tax Payable" value={fmtNaira(estimate.taxPayable)} />
                        <SummaryRow label="PAYE / WHT Credit" value={fmtNaira(estimate.taxAlreadyPaid)} />
                        <SummaryRow
                            label={refund > 0 ? 'Refund Due' : 'Balance to Pay'}
                            value={refund > 0 ? `+${fmtNaira(refund)}` : fmtNaira(taxDue)}
                            bold
                        />
                    </div>

                    <div className="flex flex-col gap-3">
                        {taxDue > 0 && (
                            <button onClick={() => navigate('/payment')} className="w-full h-14 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-2xl flex items-center justify-center gap-2 text-lg">
                                Pay {fmtNaira(taxDue)} Now <ArrowRight className="w-5 h-5" />
                            </button>
                        )}
                        <button onClick={() => navigate('/tcc')} className="w-full h-12 bg-neutral-100 hover:bg-neutral-200 text-neutral-700 font-bold rounded-2xl flex items-center justify-center gap-2">
                            <FileText className="w-5 h-5" /> View Tax Clearance Certificate
                        </button>
                    </div>
                </div>
            </DashboardLayout>
        )
    }

    // ── Error state ────────────────────────────────────────────────────────────
    if (status === 'error') {
        return (
            <DashboardLayout>
                <div className="max-w-xl mx-auto pb-24 pt-12 text-center">
                    <div className="w-24 h-24 bg-rose-100 rounded-full flex items-center justify-center mx-auto mb-6">
                        <XCircle className="w-14 h-14 text-rose-400" />
                    </div>
                    <h1 className="text-2xl font-black text-neutral-900 mb-2">Submission Failed</h1>
                    <p className="text-neutral-500 mb-8">{error}</p>
                    <button onClick={handleSubmit} className="h-12 px-8 bg-blue-600 text-white font-bold rounded-xl">Try Again</button>
                </div>
            </DashboardLayout>
        )
    }

    // ── Default: review & submit ───────────────────────────────────────────────
    return (
        <DashboardLayout>
            <div className="max-w-2xl mx-auto pb-24 animate-in fade-in duration-500">

                <div className="pt-8 pb-8">
                    <p className="text-sm font-bold text-blue-600 uppercase tracking-widest mb-2">Step 3 of 4 — E-File</p>
                    <h1 className="text-3xl font-black text-neutral-900">Review and submit your return.</h1>
                    <p className="text-neutral-500 mt-1">This is a final review before we send your return to the NRS.</p>
                </div>

                {/* Shareable summary card */}
                <div className="bg-white border border-neutral-200 rounded-3xl p-8 shadow-sm mb-6">
                    <div className="flex items-center gap-3 mb-6">
                        <FileText className="w-6 h-6 text-blue-600" />
                        <h2 className="text-lg font-bold text-neutral-900">2024 Income Tax Return — Summary</h2>
                    </div>
                    <SummaryRow label="Taxpayer" value={`${profile.first_name ?? ''} ${profile.last_name ?? ''}`} />
                    <SummaryRow label="TIN" value={profile.tin ?? '—'} />
                    <SummaryRow label="Gross Income" value={fmtNaira(Number(profile.employment_gross_salary ?? 0) + Number(profile.freelance_gross_income ?? 0))} />
                    <SummaryRow label="Total Reliefs & Deductions" value={fmtNaira(estimate.totalReliefs ?? 0)} />
                    <SummaryRow label="Chargeable Income" value={fmtNaira(estimate.chargeableIncome ?? 0)} />
                    <SummaryRow label="Gross Tax Payable" value={fmtNaira(estimate.taxPayable)} />
                    <SummaryRow label="PAYE / WHT Credits" value={`— ${fmtNaira(estimate.taxAlreadyPaid)}`} />
                    <div className={`flex items-center justify-between mt-4 p-4 rounded-xl ${refund > 0 ? 'bg-emerald-50' : 'bg-rose-50'}`}>
                        <span className={`font-black text-lg ${refund > 0 ? 'text-emerald-700' : 'text-rose-700'}`}>
                            {refund > 0 ? 'Estimated Refund' : 'Balance Due'}
                        </span>
                        <span className={`font-black text-2xl ${refund > 0 ? 'text-emerald-700' : 'text-rose-700'}`}>
                            {refund > 0 ? `+${fmtNaira(refund)}` : fmtNaira(taxDue)}
                        </span>
                    </div>
                </div>

                {/* Important notes */}
                <div className="bg-amber-50 border border-amber-200 rounded-2xl p-5 mb-6 text-sm text-amber-800">
                    <strong className="block mb-1">⚠️ Important</strong>
                    Once submitted, you can only amend your return within 90 days under NTA 2025 s.16(3). Ensure all figures are correct before filing.
                </div>

                <button
                    onClick={handleSubmit}
                    disabled={status === 'submitting'}
                    className="w-full h-16 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-black text-xl rounded-2xl transition-colors flex items-center justify-center gap-3 shadow-lg shadow-blue-200"
                >
                    {status === 'submitting' ? (
                        <><Loader2 className="w-6 h-6 animate-spin" /> Submitting to NRS…</>
                    ) : (
                        <><Send className="w-6 h-6" /> Submit Return to NRS</>
                    )}
                </button>
            </div>
        </DashboardLayout>
    )
}
