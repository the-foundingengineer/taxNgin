import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { Sparkles, Check, ArrowRight, Loader2 } from 'lucide-react'
import { billingApi } from '@/lib/api/billing'
import type { BillingTier } from '@/types/api'

export function PricingPage() {
    const [searchParams] = useSearchParams()
    const billingStatus = searchParams.get('billing')
    const [upgrading, setUpgrading] = useState(false)
    const [upgradeError, setUpgradeError] = useState<string | null>(null)

    useEffect(() => {
        if (billingStatus === 'error') {
            setUpgradeError('Payment could not be verified. Please try again or contact support.')
        }
    }, [billingStatus])

    async function handleUpgrade(tier: BillingTier) {
        setUpgrading(true)
        setUpgradeError(null)
        try {
            const callbackUrl = `${window.location.origin}/v1/billing/callback`
            const { authorization_url } = await billingApi.initiateUpgrade(tier, callbackUrl)
            window.location.href = authorization_url
        } catch (err) {
            setUpgradeError(err instanceof Error ? err.message : 'Failed to start payment. Please try again.')
            setUpgrading(false)
        }
    }

    return (
        <DashboardLayout>
            <div className="space-y-12 pb-24 animate-in fade-in slide-in-from-bottom-4 duration-500">

                {billingStatus === 'success' && (
                    <div className="max-w-2xl mx-auto bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-2xl px-6 py-4 text-center font-semibold">
                        Payment confirmed! Your account has been upgraded. It may take a moment to reflect.
                    </div>
                )}
                {upgradeError && (
                    <div className="max-w-2xl mx-auto bg-rose-50 border border-rose-200 text-rose-700 rounded-2xl px-6 py-4 text-center text-sm">
                        {upgradeError}
                    </div>
                )}

                <div className="text-center max-w-2xl mx-auto pt-8">
                    <h1 className="text-4xl lg:text-5xl font-black text-neutral-900 tracking-tight leading-tight">
                        Simple, transparent pricing.
                    </h1>
                    <p className="text-neutral-500 mt-4 text-lg leading-relaxed">
                        No hidden fees. Start for free, and only pay when you file. Maximise your Nigerian tax return with confidence.
                    </p>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 max-w-6xl mx-auto px-4 mt-12 items-center">

                    {/* Free Tier */}
                    <div className="bg-white border border-neutral-200 rounded-3xl p-8 shadow-sm">
                        <h3 className="text-xl font-bold text-neutral-900">Standard Edition</h3>
                        <p className="text-sm text-neutral-500 mt-2">For simple W-2 (Form H1) returns only.</p>
                        <div className="my-6">
                            <span className="text-5xl font-black text-neutral-900">Free</span>
                        </div>
                        <button className="w-full h-12 bg-neutral-100 hover:bg-neutral-200 text-neutral-900 font-bold rounded-xl transition-colors flex items-center justify-center gap-2">
                            Get Started <ArrowRight className="w-4 h-4" />
                        </button>
                        <ul className="mt-8 space-y-4">
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">W-2 (PAYE / Form H1) income</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">Standard dependents relief</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">Direct NRS e-Filing</span></li>
                            <li className="flex items-start gap-3 opacity-40"><Check className="w-5 h-5 opacity-0 shrink-0" /><span className="text-sm font-medium text-neutral-700 line-through">Freelance & Gig income</span></li>
                            <li className="flex items-start gap-3 opacity-40"><Check className="w-5 h-5 opacity-0 shrink-0" /><span className="text-sm font-medium text-neutral-700 line-through">Crypto & Investments</span></li>
                        </ul>
                    </div>

                    {/* Pro Tier (Highlight) */}
                    <div className="bg-neutral-900 rounded-3xl p-8 shadow-2xl relative scale-100 lg:scale-105 z-10 border-4 border-blue-600">
                        <div className="absolute -top-4 inset-x-0 flex justify-center">
                            <div className="bg-gradient-to-r from-blue-500 to-indigo-500 text-white text-xs font-bold uppercase tracking-widest px-4 py-1.5 rounded-full shadow-lg flex items-center gap-1.5">
                                <Sparkles className="w-3.5 h-3.5" /> Most Popular
                            </div>
                        </div>

                        <h3 className="text-xl font-bold text-white">Premium Edition</h3>
                        <p className="text-sm text-neutral-400 mt-2">For freelancers, investors, & crypto.</p>
                        <div className="my-6 flex items-end gap-1">
                            <span className="text-5xl font-black text-white">₦15,000</span>
                        </div>
                        <button
                            onClick={() => handleUpgrade('PREMIUM')}
                            disabled={upgrading}
                            className="w-full h-12 bg-blue-600 hover:bg-blue-500 disabled:opacity-60 text-white font-bold rounded-xl transition-colors flex items-center justify-center gap-2 shadow-lg shadow-blue-900/50"
                        >
                            {upgrading ? <><Loader2 className="w-4 h-4 animate-spin" /> Processing…</> : <>Start Premium <ArrowRight className="w-4 h-4" /></>}
                        </button>
                        <ul className="mt-8 space-y-4">
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-blue-400 shrink-0" /><span className="text-sm font-medium text-white">Everything in Standard, plus:</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-400 shrink-0" /><span className="text-sm font-medium text-neutral-300">Freelance & 1099 equivalent forms</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-400 shrink-0" /><span className="text-sm font-medium text-neutral-300">Crypto, Stocks, & Bonds CGT</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-400 shrink-0" /><span className="text-sm font-medium text-neutral-300">Rental Property Income (WHT offset)</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-400 shrink-0" /><span className="text-sm font-medium text-neutral-300">NTA Deduction Optimizer Engine</span></li>
                        </ul>
                    </div>

                    {/* Business Tier */}
                    <div className="bg-white border border-neutral-200 rounded-3xl p-8 shadow-sm">
                        <h3 className="text-xl font-bold text-neutral-900">Live Full Service</h3>
                        <p className="text-sm text-neutral-500 mt-2">A CITN Tax Pro does it completely for you.</p>
                        <div className="my-6 flex items-end gap-1">
                            <span className="text-2xl font-bold text-neutral-400 line-through mr-1">₦60k</span>
                            <span className="text-5xl font-black text-neutral-900">₦45k</span>
                        </div>
                        <button className="w-full h-12 bg-white border border-neutral-300 hover:bg-neutral-50 text-neutral-900 font-bold rounded-xl transition-colors flex items-center justify-center gap-2">
                            Match with a Pro <ArrowRight className="w-4 h-4" />
                        </button>
                        <ul className="mt-8 space-y-4">
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">Dedicated Nigerian Tax Pro</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">We sign and file for you</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">Full Audit Defense Included</span></li>
                            <li className="flex items-start gap-3"><Check className="w-5 h-5 text-emerald-500 shrink-0" /><span className="text-sm font-medium text-neutral-700">Unlimited Tax Advice Year-Round</span></li>
                        </ul>
                    </div>

                </div>
            </div>
        </DashboardLayout>
    )
}
