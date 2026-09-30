import { useState, useEffect, useRef } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { fmtNaira } from '@/lib/pit'
import { computePITEstimate } from '@/hooks/usePITEstimate'
import { Download, Share2, QrCode, BadgeCheck } from 'lucide-react'

// QR code via a free CDN-based generator (no npm dependency needed)
function QrCodeDisplay({ value }: { value: string }) {
    const encoded = encodeURIComponent(value)
    return (
        <img
            src={`https://api.qrserver.com/v1/create-qr-code/?size=120x120&data=${encoded}`}
            alt="NRS Verification QR Code"
            className="w-24 h-24 rounded-xl border border-neutral-200 bg-white p-1"
        />
    )
}

export function TCCPage() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const [profile, setProfile] = useState<Record<string, any>>({})
    const ref = localStorage.getItem('tc_efile_ref') ?? 'NRS-2024-DEMO'
    const filedAt = localStorage.getItem('tc_filed_at') ?? new Date().toISOString()
    const certRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        try {
            const raw = localStorage.getItem('tc_profile')
            if (raw) setProfile(JSON.parse(raw))
        } catch { /* ignore */ }
    }, [])

    const estimate = computePITEstimate(profile)
    const yearFrom = '1 January 2024'
    const yearTo = '31 December 2024'
    const issued = new Date(filedAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'long', year: 'numeric' })
    const expires = new Date(new Date(filedAt).setFullYear(new Date(filedAt).getFullYear() + 1)).toLocaleDateString('en-NG', { day: 'numeric', month: 'long', year: 'numeric' })

    const fullName = `${profile.first_name ?? ''} ${profile.last_name ?? ''}`.trim() || 'Nigerian Taxpayer'
    const tin = profile.tin ?? '—'
    const stateTax = profile.state_of_residence ?? 'Federal Capital Territory'

    const verifyUrl = `https://nrs.gov.ng/verify?ref=${ref}&tin=${tin}`

    function handlePrint() {
        window.print()
    }

    return (
        <DashboardLayout>
            <div className="max-w-3xl mx-auto pb-24 animate-in fade-in duration-500">

                {/* Title bar */}
                <div className="flex items-center justify-between pt-8 pb-6">
                    <div>
                        <p className="text-sm font-bold text-emerald-600 uppercase tracking-widest mb-1">Filing Complete</p>
                        <h1 className="text-3xl font-black text-neutral-900">Tax Clearance Certificate</h1>
                    </div>
                    <div className="flex items-center gap-3">
                        <button onClick={handlePrint} className="flex items-center gap-2 h-10 px-4 bg-neutral-100 hover:bg-neutral-200 rounded-xl font-bold text-sm text-neutral-700 transition-colors">
                            <Download className="w-4 h-4" /> Download
                        </button>
                        <button
                            onClick={() => navigator.share?.({ title: 'My Tax Clearance Certificate', url: verifyUrl })}
                            className="flex items-center gap-2 h-10 px-4 bg-blue-600 hover:bg-blue-700 rounded-xl font-bold text-sm text-white transition-colors"
                        >
                            <Share2 className="w-4 h-4" /> Share
                        </button>
                    </div>
                </div>

                {/* ── THE TCC CARD ── */}
                <div ref={certRef} className="bg-white border-2 border-emerald-600 rounded-3xl overflow-hidden shadow-xl print:shadow-none">

                    {/* Green header stripe */}
                    <div className="bg-gradient-to-r from-emerald-700 to-emerald-600 px-8 py-6 text-white flex items-center justify-between">
                        <div className="flex items-center gap-4">
                            {/* NRS Logo placeholder */}
                            <div className="w-14 h-14 bg-white rounded-full flex items-center justify-center shrink-0">
                                <span className="text-emerald-700 font-black text-xl">NRS</span>
                            </div>
                            <div>
                                <p className="font-black text-xl leading-tight">Tax Clearance Certificate</p>
                                <p className="text-emerald-200 text-sm font-medium">Nigeria Revenue Service · NTA 2025 s.85</p>
                            </div>
                        </div>
                        <BadgeCheck className="w-10 h-10 text-emerald-300 shrink-0" />
                    </div>

                    {/* Certificate body */}
                    <div className="px-8 py-8 space-y-6">
                        <p className="text-neutral-600 leading-relaxed text-base">
                            This is to certify that <strong className="text-neutral-900">{fullName}</strong> with Tax Identification Number <strong className="text-neutral-900">{tin}</strong> has satisfied all income tax obligations under the Nigeria Tax Act 2025 for the period:
                        </p>

                        <div className="bg-emerald-50 border border-emerald-200 rounded-2xl p-5 text-center">
                            <p className="text-5xl font-black text-emerald-800 mb-1">2024</p>
                            <p className="text-emerald-700 font-bold">{yearFrom} — {yearTo}</p>
                        </div>

                        {/* 3-column data grid */}
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <DataCell label="Taxpayer Name" value={fullName} />
                            <DataCell label="TIN" value={tin} />
                            <DataCell label="State / Zone" value={stateTax} />
                            <DataCell label="Assessment Year" value="2024" />
                            <DataCell label="Gross Income" value={fmtNaira(Number(profile.employment_gross_salary ?? 0) + Number(profile.freelance_gross_income ?? 0))} />
                            <DataCell label="Tax Payable" value={fmtNaira(estimate.taxPayable)} />
                            <DataCell label="Tax Paid (PAYE)" value={fmtNaira(estimate.taxAlreadyPaid)} />
                            <DataCell label="Balance Paid" value={fmtNaira(Math.max(0, estimate.taxPayable - estimate.taxAlreadyPaid))} />
                            <DataCell label="Filing Reference" value={ref} mono />
                        </div>

                        {/* QR + verification */}
                        <div className="flex flex-col md:flex-row items-center gap-6 bg-neutral-50 rounded-2xl p-6 border border-neutral-200">
                            {/* QR Code */}
                            <div className="shrink-0 text-center">
                                <QrCodeDisplay value={verifyUrl} />
                                <p className="text-[10px] text-neutral-400 mt-2 font-medium">Scan to Verify</p>
                            </div>
                            <div className="flex-1">
                                <div className="flex items-center gap-2 mb-2">
                                    <QrCode className="w-5 h-5 text-neutral-600" />
                                    <h3 className="font-bold text-neutral-900">NRS Online Verification</h3>
                                </div>
                                <p className="text-sm text-neutral-600 leading-relaxed mb-3">
                                    Scan the QR code or visit the URL below to verify the authenticity of this certificate on the NRS portal:
                                </p>
                                <p className="text-xs text-blue-600 font-mono break-all">{verifyUrl}</p>
                            </div>
                        </div>

                        {/* Footer */}
                        <div className="border-t border-neutral-200 pt-6 flex flex-col md:flex-row items-start md:items-end justify-between gap-4">
                            <div>
                                <p className="text-sm text-neutral-600"><strong>Date of Issue:</strong> {issued}</p>
                                <p className="text-sm text-neutral-600"><strong>Valid Until:</strong> {expires}</p>
                                <p className="text-xs text-neutral-400 mt-2">This certificate was issued electronically and is valid without a physical signature under NTA 2025 s.85(4).</p>
                            </div>
                            <div className="text-right">
                                <div className="h-px w-48 bg-neutral-900 mb-1" />
                                <p className="text-xs text-neutral-500">Authorised Officer, NRS</p>
                                <p className="text-xs text-neutral-400">Digital issuance via TaxCheck</p>
                            </div>
                        </div>
                    </div>
                </div>

                <p className="text-center text-xs text-neutral-400 mt-6">
                    This certificate has been generated in accordance with NTA 2025 s.85. Keep a digital copy for your records.
                </p>
            </div>
        </DashboardLayout>
    )
}

function DataCell({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
    return (
        <div className="bg-neutral-50 rounded-xl p-4 border border-neutral-100">
            <p className="text-[10px] font-black uppercase tracking-widest text-neutral-400 mb-1">{label}</p>
            <p className={`text-sm font-bold text-neutral-900 break-all ${mono ? 'font-mono' : ''}`}>{value}</p>
        </div>
    )
}
