import { useState } from 'react'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { Wallet, Info, Plus } from 'lucide-react'
import { fmtNaira } from '@/lib/pit'

export function CryptoCGTPage() {
    const [sales, setSales] = useState([{ id: 1, asset: '', proceeds: '', cost: '' }])

    const calculateTotal = () => {
        let totalGain = 0
        sales.forEach(s => {
            const p = Number(s.proceeds) || 0
            const c = Number(s.cost) || 0
            if (p > c) totalGain += (p - c)
        })
        return totalGain
    }

    const totalGain = calculateTotal()
    const cgtLiability = totalGain * 0.10 // 10% CGT on digital assets under SEC/NTA rules

    return (
        <DashboardLayout>
            <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500 max-w-4xl mx-auto pb-24">

                {/* Header */}
                <div className="flex items-center gap-4">
                    <div className="w-16 h-16 rounded-2xl bg-[#F7931A]/10 text-[#F7931A] flex items-center justify-center shrink-0">
                        <Wallet className="w-8 h-8" />
                    </div>
                    <div>
                        <h1 className="text-3xl font-black text-neutral-900 tracking-tight">Crypto & Digital Assets</h1>
                        <p className="text-neutral-500 font-medium mt-1">Report cryptocurrency and NFT disposals for Capital Gains Tax (10%).</p>
                    </div>
                </div>

                <div className="bg-blue-50 border border-blue-200 text-blue-800 p-4 rounded-xl flex items-start gap-3 text-sm">
                    <Info className="w-5 h-5 shrink-0 mt-0.5" />
                    <div>
                        <strong className="block mb-1">2025 NTA Crypto Regulations</strong>
                        Gains realised from the disposal of digital assets (including cryptocurrencies, stablecoins, and NFTs) are now chargeable to Capital Gains Tax (CGT) at a flat rate of 10%. Only net gains are taxed. Losses cannot be deducted from regular income.
                    </div>
                </div>

                <div className="bg-white border border-neutral-200 rounded-3xl p-6 md:p-8 shadow-sm">
                    <div className="flex items-center justify-between mb-6">
                        <h2 className="text-xl font-bold text-neutral-900">Your Disposals (Sales)</h2>
                        <button
                            onClick={() => setSales([...sales, { id: Date.now(), asset: '', proceeds: '', cost: '' }])}
                            className="flex items-center gap-1.5 text-sm font-bold text-blue-600 hover:text-blue-700 bg-blue-50 px-3 py-1.5 rounded-lg transition-colors"
                        >
                            <Plus className="w-4 h-4" /> Add Asset
                        </button>
                    </div>

                    <div className="space-y-4">
                        {sales.map((sale, index) => (
                            <div key={sale.id} className="grid grid-cols-1 md:grid-cols-12 gap-4 items-end bg-neutral-50 p-4 rounded-2xl border border-neutral-100">
                                <div className="md:col-span-5">
                                    <label className="block text-xs font-bold text-neutral-500 uppercase tracking-wide mb-1.5">Asset Name (e.g. BTC, ETH)</label>
                                    <input
                                        type="text"
                                        value={sale.asset}
                                        onChange={(e) => {
                                            const newSales = [...sales]
                                            newSales[index].asset = e.target.value
                                            setSales(newSales)
                                        }}
                                        className="w-full bg-white border border-neutral-300 rounded-xl px-4 py-2.5 text-[15px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                                        placeholder="Bitcoin (BTC)"
                                    />
                                </div>
                                <div className="md:col-span-3">
                                    <label className="block text-xs font-bold text-neutral-500 uppercase tracking-wide mb-1.5">Proceeds (Sale Price)</label>
                                    <div className="relative">
                                        <span className="absolute left-3 top-2.5 text-neutral-400">₦</span>
                                        <input
                                            type="number"
                                            value={sale.proceeds}
                                            onChange={(e) => {
                                                const newSales = [...sales]
                                                newSales[index].proceeds = e.target.value
                                                setSales(newSales)
                                            }}
                                            className="w-full bg-white border border-neutral-300 rounded-xl pl-8 pr-4 py-2.5 text-[15px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                                            placeholder="0.00"
                                        />
                                    </div>
                                </div>
                                <div className="md:col-span-3">
                                    <label className="block text-xs font-bold text-neutral-500 uppercase tracking-wide mb-1.5">Cost Basis (Buy Price)</label>
                                    <div className="relative">
                                        <span className="absolute left-3 top-2.5 text-neutral-400">₦</span>
                                        <input
                                            type="number"
                                            value={sale.cost}
                                            onChange={(e) => {
                                                const newSales = [...sales]
                                                newSales[index].cost = e.target.value
                                                setSales(newSales)
                                            }}
                                            className="w-full bg-white border border-neutral-300 rounded-xl pl-8 pr-4 py-2.5 text-[15px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                                            placeholder="0.00"
                                        />
                                    </div>
                                </div>
                                <div className="md:col-span-1 flex justify-center pb-2">
                                    <button
                                        onClick={() => setSales(sales.filter(s => s.id !== sale.id))}
                                        className="text-neutral-400 hover:text-rose-500 transition-colors"
                                    >
                                        Delete
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>

                    <div className="mt-8 pt-8 border-t border-neutral-200">
                        <div className="flex flex-col md:flex-row items-center justify-between gap-6 bg-gradient-to-r from-neutral-900 to-neutral-800 rounded-2xl p-6">
                            <div>
                                <p className="text-neutral-400 text-sm font-medium mb-1">Total Net Chargeable Gain</p>
                                <div className="text-3xl font-black text-white">{fmtNaira(totalGain)}</div>
                            </div>
                            <div className="h-12 w-px bg-white/20 hidden md:block" />
                            <div>
                                <p className="text-blue-300 text-sm font-medium mb-1 flex items-center gap-2">Estimated CGT Liability (10%)</p>
                                <div className="text-3xl font-black text-blue-400">{fmtNaira(cgtLiability)}</div>
                            </div>
                            <button className="h-12 px-6 bg-blue-600 hover:bg-blue-500 text-white font-bold rounded-xl transition-colors w-full md:w-auto">
                                Save Asset Profile
                            </button>
                        </div>
                    </div>

                </div>
            </div>
        </DashboardLayout>
    )
}
