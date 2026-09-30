import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { CheckCircle2, Shield, FileText, Phone, AlertTriangle, ChevronRight } from 'lucide-react'

const steps = [
    {
        icon: <Shield className="w-6 h-6" />,
        title: "Don't panic — most NRS queries are routine",
        body: "NRS assessment notices are common and do not automatically mean you did something wrong. They often arise from data mismatches between employer PAYE submissions and your filed return. Take a deep breath — you have time to respond.",
        color: 'blue',
    },
    {
        icon: <FileText className="w-6 h-6" />,
        title: "Gather your evidence immediately",
        body: "Collect your Form H1 (Annual PAYE Statement), payslips, bank statements, receipts for deductions claimed (NHF, pension, life insurance), and any prior NRS correspondence. Strong documentation is your most powerful defence.",
        color: 'indigo',
    },
    {
        icon: <CheckCircle2 className="w-6 h-6" />,
        title: "Review what the NRS is questioning",
        body: "The notice will specify exactly which line items are disputed — usually income figures, PAYE credits, or disallowed deductions. Compare these against your filed return line by line. Most cases resolve because a credit was incorrectly attributed.",
        color: 'emerald',
    },
    {
        icon: <Phone className="w-6 h-6" />,
        title: "Respond formally within 30 days",
        body: "Under NTA 2025 s.68, you have 30 days from the date of assessment to object. File a formal written objection with your TaxPayer Identification Number (TIN), the assessment reference number, and a clear statement of each point in dispute. Late objections may be rejected.",
        color: 'orange',
    },
    {
        icon: <AlertTriangle className="w-6 h-6" />,
        title: "Escalate if needed — you have rights",
        body: "If your objection is rejected, you can appeal to the Tax Appeal Tribunal (TAT) within 30 days of the NRS decision. You may also engage a certified Nigerian tax practitioner (CITN member) to represent you. The TAT is independent and has overturned many NRS assessments.",
        color: 'rose',
    },
]

const colorMap: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-700 border-blue-100',
    indigo: 'bg-indigo-50 text-indigo-700 border-indigo-100',
    emerald: 'bg-emerald-50 text-emerald-700 border-emerald-100',
    orange: 'bg-orange-50 text-orange-700 border-orange-100',
    rose: 'bg-rose-50 text-rose-700 border-rose-100',
}

const faqs = [
    {
        q: "What is a Desk Review vs. a Full Audit under NTA 2025?",
        a: "A Desk Review (s.44) is conducted without visiting you — NRS requests documents by post or email. A Full Audit (s.45) involves a formal examination at your premises or an NRS office. Desk Reviews are far more common and usually resolve within 60 days."
    },
    {
        q: "Can the NRS access my bank accounts?",
        a: "Yes — under NTA 2025 s.52, the NRS Commissioner can issue a third-party information notice requiring banks to disclose financial records. This power is exercised judiciously, usually only when prior document requests have been ignored."
    },
    {
        q: "Will claiming deductions trigger an audit?",
        a: "Legitimately claimed, properly documented deductions do not trigger audits. Audits are typically triggered by statistical anomalies (e.g., deductions that seem disproportionate to reported income) or mismatches between your return and third-party data like employer PAYE submissions."
    },
    {
        q: "What penalties can I face?",
        a: "Under NTA 2025 s.74, late filing attracts ₦25,000 per month. Understatement of income carries 10–100% of tax unpaid depending on whether the NRS deems it negligent or deliberate. Voluntary disclosure before an audit significantly reduces penalties."
    },
]

export function AuditSupportPage() {
    return (
        <DashboardLayout>
            <div className="space-y-12 pb-24 animate-in fade-in slide-in-from-bottom-4 duration-500">

                {/* Header */}
                <div className="max-w-2xl">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-rose-50 border border-rose-200 text-rose-700 text-xs font-bold uppercase tracking-widest mb-4">
                        <Shield className="w-3.5 h-3.5" /> Audit Support Centre
                    </div>
                    <h1 className="text-4xl font-black text-neutral-900 tracking-tight leading-tight">
                        Received an NRS assessment? Here's exactly what to do.
                    </h1>
                    <p className="text-neutral-500 mt-3 text-lg leading-relaxed">
                        We've prepared a clear, step-by-step NTA 2025 guide so you know your rights and obligations. Most queries resolve without you ever needing to visit an NRS office.
                    </p>
                </div>

                {/* 5 Steps */}
                <div className="space-y-5">
                    {steps.map((step, idx) => (
                        <div
                            key={idx}
                            className={`flex items-start gap-5 p-6 rounded-2xl border ${colorMap[step.color]} transition-all hover:shadow-md`}
                        >
                            <div className="w-12 h-12 rounded-xl bg-white/60 flex items-center justify-center shrink-0 shadow-sm">
                                {step.icon}
                            </div>
                            <div>
                                <div className="flex items-center gap-3 mb-1.5">
                                    <span className="text-xs font-bold opacity-60 uppercase tracking-widest">Step {idx + 1}</span>
                                </div>
                                <h3 className="font-bold text-[17px] leading-snug mb-2">{step.title}</h3>
                                <p className="text-sm leading-relaxed opacity-80">{step.body}</p>
                            </div>
                        </div>
                    ))}
                </div>

                {/* FAQ */}
                <div>
                    <h2 className="text-2xl font-bold text-neutral-900 mb-6">Frequently asked questions</h2>
                    <div className="space-y-3">
                        {faqs.map((faq, idx) => (
                            <details key={idx} className="group bg-white border border-neutral-200 rounded-xl overflow-hidden shadow-sm">
                                <summary className="flex items-center justify-between px-6 py-4 cursor-pointer list-none font-semibold text-neutral-900 text-[15px]">
                                    {faq.q}
                                    <ChevronRight className="w-4 h-4 text-neutral-400 transition-transform group-open:rotate-90 shrink-0" />
                                </summary>
                                <div className="px-6 pb-5 text-sm text-neutral-600 leading-relaxed border-t border-neutral-100 pt-4">
                                    {faq.a}
                                </div>
                            </details>
                        ))}
                    </div>
                </div>

                {/* CTA */}
                <div className="bg-gradient-to-br from-neutral-900 to-neutral-800 text-white rounded-2xl p-8 flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
                    <div>
                        <h3 className="text-xl font-bold mb-1">Still worried? Talk to a verified Nigerian Tax Pro</h3>
                        <p className="text-neutral-400 text-sm">Our network of CITN-certified tax practitioners can represent you directly before the NRS.</p>
                    </div>
                    <button className="shrink-0 px-6 h-12 bg-white text-neutral-900 font-bold rounded-xl hover:bg-neutral-100 transition-colors whitespace-nowrap flex items-center gap-2">
                        Connect a Tax Pro
                        <ChevronRight className="w-4 h-4" />
                    </button>
                </div>

            </div>
        </DashboardLayout>
    )
}
