import { ReactNode } from 'react'

interface AuthLayoutProps {
    children: ReactNode
    title: string
    subtitle: string
}

export function AuthLayout({ children, title, subtitle }: AuthLayoutProps) {
    // Extract the short title for the mobile header (strip "Sign in to" / "Sign up for")
    const mobileTitle = title.replace(/^Sign in to |^Sign up for /, '')

    return (
        <div className="min-h-screen w-full flex font-sans overflow-hidden" style={{ backgroundColor: '#f5f4ef' }}>

            {/* ══════════════════════════════════════════════
                LEFT / MAIN COLUMN
                - On mobile: full-width with dark curved header at top
                - On desktop: half-width white form panel
            ══════════════════════════════════════════════ */}
            <div className="flex-1 flex flex-col lg:bg-white lg:justify-center lg:items-center lg:overflow-y-auto">

                {/* ── Mobile: dark blue curved header (hidden lg+) ── */}
                <div
                    className="lg:hidden relative w-full flex flex-col px-6 pt-10 pb-20 text-white shrink-0"
                    style={{
                        backgroundColor: '#1C4ED8',
                        borderBottomLeftRadius: '48px',
                        borderBottomRightRadius: '48px',
                    }}
                >
                    {/* Subtle circle pattern */}
                    <div className="absolute top-4 right-6 opacity-10 pointer-events-none">
                        <svg width="80" height="80" viewBox="0 0 80 80">
                            <circle cx="40" cy="40" r="30" stroke="white" strokeWidth="0.8" fill="none" />
                            <circle cx="40" cy="40" r="18" stroke="white" strokeWidth="0.8" fill="none" />
                        </svg>
                    </div>
                    {/* Back nav placeholder */}
                    <div className="mb-6 text-white/60">
                        <svg width="24" height="24" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                        </svg>
                    </div>
                    <h1 className="text-4xl font-bold leading-tight">{mobileTitle}</h1>
                    <p className="text-white/60 text-sm mt-1">{subtitle}</p>
                </div>

                {/* ── Mobile: TC logo bubble sits on the seam (hidden lg+) ── */}
                <div className="lg:hidden flex justify-center -mt-8 z-10 shrink-0">
                    <div
                        className="w-16 h-16 rounded-full flex items-center justify-center shadow-lg"
                        style={{
                            background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)',
                            border: '4px solid #f5f4ef',
                        }}
                    >
                        <span className="text-white font-black text-lg tracking-tight">TC</span>
                    </div>
                </div>

                {/* ── Desktop: wordmark + page heading (hidden on mobile) ── */}
                <div className="hidden lg:block w-full max-w-[420px]">
                    <div className="flex items-center gap-2 mb-16 select-none">
                        <div
                            className="w-7 h-7 rounded-full flex items-center justify-center"
                            style={{ background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)' }}
                        >
                            <span className="transform rotate-45 block w-3 h-3 border-2 border-white border-t-0 border-l-0 rounded-sm" />
                        </div>
                        <span className="text-xl font-bold tracking-tight text-neutral-900">Tax Check.</span>
                    </div>
                    <div className="text-center mb-8">
                        <h1 className="text-[28px] leading-tight font-bold text-neutral-900 mb-2">{title}</h1>
                        <p className="text-neutral-500 text-[15px]">{subtitle}</p>
                    </div>
                </div>

                {/* ── Form children — rendered ONCE for both layouts ── */}
                <div className="w-full max-w-[420px] px-6 lg:px-0 pt-6 lg:pt-0 pb-4">
                    {children}
                </div>

                {/* ── Footer ── */}
                <div className="w-full max-w-[420px] px-6 lg:px-0 mt-10 mb-8 flex justify-between text-xs text-neutral-400 font-medium">
                    <span className="cursor-pointer hover:text-neutral-600 transition-colors">Privacy Policy</span>
                    <span>Copyright {new Date().getFullYear()}</span>
                </div>
            </div>

            {/* ══════════════════════════════════════════════
                RIGHT VISUAL PANEL — desktop only
            ══════════════════════════════════════════════ */}
            <div
                className="hidden lg:flex w-1/2 relative flex-col items-center justify-center p-12 overflow-hidden"
                style={{ backgroundColor: '#1C4ED8' }}
            >
                {/* Background shapes */}
                <div className="absolute top-[10%] left-[15%] w-16 h-16 border border-white/10 rotate-12" />
                <div className="absolute top-[15%] right-[10%] opacity-20">
                    <svg width="40" height="40" viewBox="0 0 40 40">
                        <circle cx="20" cy="20" r="1" fill="white" />
                        <circle cx="10" cy="10" r="1" fill="white" />
                        <circle cx="30" cy="10" r="1" fill="white" />
                        <circle cx="10" cy="30" r="1" fill="white" />
                        <circle cx="30" cy="30" r="1" fill="white" />
                        <path d="M10 10l20 20M10 30L30 10" stroke="white" strokeWidth="0.5" strokeDasharray="2 2" />
                    </svg>
                </div>

                {/* Tagline — above the mockup */}
                <div className="text-center mb-10 z-10 px-4">
                    <h2 className="text-3xl font-bold text-amber-300 mb-2">File smarter, save more</h2>
                    <p className="text-white/70 text-sm max-w-xs mx-auto">
                        Claim legal relief structures and secure your TCC up to 5 days early.
                    </p>
                </div>

                {/* Floating CSS UI Mockup */}
                <div className="relative w-full max-w-[460px] h-[460px]">

                    {/* Main Card */}
                    <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white rounded-2xl w-full h-[380px] shadow-[0_30px_60px_rgba(0,0,0,0.15)] flex flex-col p-6">
                        <div className="flex justify-between items-center mb-6">
                            <h3 className="font-bold text-neutral-800">File Returns</h3>
                            <div className="w-1.5 h-1.5 rounded-full bg-neutral-300 shadow-[6px_0_0_#d4d4d4,12px_0_0_#d4d4d4]" />
                        </div>
                        <p className="text-xs text-neutral-500 mb-6 text-center max-w-[280px] mx-auto">
                            Please enter the tax assessment information that you wish to review.
                        </p>
                        <div className="mb-4">
                            <span className="text-xs font-semibold text-neutral-800">Recent Filings</span>
                            <div className="flex gap-4 mt-3">
                                <div className="flex flex-col items-center gap-1">
                                    <div className="w-10 h-10 rounded-full border border-dashed border-neutral-300 flex items-center justify-center text-neutral-400">+</div>
                                    <span className="text-[10px] text-neutral-500">New</span>
                                </div>
                                {['2025', '2024', '2023', '2022'].map((yr, i) => (
                                    <div key={yr} className="flex flex-col items-center gap-1">
                                        <div className="w-10 h-10 rounded-full bg-neutral-100 border border-neutral-200 overflow-hidden">
                                            <img src={`https://i.pravatar.cc/100?img=${i + 15}`} alt="Avatar" className="w-full h-full object-cover opacity-80" />
                                        </div>
                                        <span className="text-[10px] text-neutral-500 font-medium">Yr {yr}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                        <div>
                            <span className="text-xs font-semibold text-neutral-800 flex justify-between">Choose Method <span className="text-info">Add +</span></span>
                            <div className="mt-3 relative w-full h-24 bg-gradient-to-r from-blue-600 to-info rounded-xl overflow-hidden text-white p-4 shadow-lg">
                                <div className="absolute -right-4 -top-8 w-24 h-24 bg-white/10 rounded-full" />
                                <div className="absolute right-12 -bottom-6 w-16 h-16 bg-white/10 rounded-full" />
                                <div className="flex justify-between w-full opacity-90 relative z-10 text-[10px]">
                                    <span>Tax Check</span><span>CARD</span>
                                </div>
                                <div className="mt-4 absolute bottom-4 left-4 z-10">
                                    <span className="text-[10px] opacity-75 block mb-0.5">Balance</span>
                                    <span className="text-lg font-bold">₦124,098.00</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Floating Balance Card */}
                    <div className="absolute top-8 -right-8 w-64 bg-white rounded-xl shadow-[0_20px_40px_rgba(0,0,0,0.12)] p-5 animate-[float_6s_ease-in-out_infinite]">
                        <span className="text-xs text-neutral-500 block mb-1">Total Liability</span>
                        <div className="flex justify-between items-end">
                            <span className="text-2xl font-bold text-info">₦56,476.00</span>
                            <span className="text-xs font-medium text-success bg-success/10 px-2 py-0.5 rounded-full mb-1">↗ 2.05%</span>
                        </div>
                        <span className="text-[10px] text-neutral-400 block mt-2">February 05, 2026</span>
                    </div>

                    {/* Floating Action Card */}
                    <div className="absolute bottom-12 -left-12 w-[180px] bg-white rounded-xl shadow-[0_20px_40px_rgba(0,0,0,0.1)] p-4 flex items-center gap-3 animate-[float_8s_ease-in-out_infinite_reverse]">
                        <div className="w-10 h-10 rounded-full bg-neutral-100 flex items-center justify-center text-xl">🤝</div>
                        <div>
                            <span className="text-xs font-bold text-neutral-800 block">TCC Ready</span>
                            <span className="text-[10px] text-success font-medium block">All cleared</span>
                        </div>
                    </div>
                </div>

                {/* Slider Dots */}
                <div className="flex gap-2 justify-center mt-10 z-10">
                    <div className="w-6 h-1.5 bg-white rounded-full" />
                    <div className="w-1.5 h-1.5 bg-white/40 rounded-full" />
                    <div className="w-1.5 h-1.5 bg-white/40 rounded-full" />
                </div>
            </div>

            <style>{`
        @keyframes float {
          0%   { transform: translateY(0px) rotate(0deg); }
          50%  { transform: translateY(-12px) rotate(1deg); }
          100% { transform: translateY(0px) rotate(0deg); }
        }
      `}</style>
        </div>
    )
}
