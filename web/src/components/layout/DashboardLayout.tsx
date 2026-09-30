import { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Home, FileText, User, HelpCircle, Bell, Tag, Users, Wallet, ClipboardList, Send } from 'lucide-react'
import { cn } from '@/lib/utils'

interface DashboardLayoutProps {
    children: ReactNode
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
    const { pathname } = useLocation()

    const navItems = [
        { path: '/dashboard', label: 'Tax Home', icon: <Home className="w-5 h-5" /> },
        { path: '/profile', label: 'Your Profile', icon: <User className="w-5 h-5" /> },
        { path: '/deductions', label: 'Deductions', icon: <ClipboardList className="w-5 h-5" /> },
        { path: '/crypto', label: 'Assets & Crypto', icon: <Wallet className="w-5 h-5" /> },
        { path: '/documents', label: 'Documents', icon: <FileText className="w-5 h-5" /> },
        { path: '/efile', label: 'E-File', icon: <Send className="w-5 h-5" /> },
        { path: '/experts', label: 'Tax Pros', icon: <Users className="w-5 h-5" /> },
        { path: '/pricing', label: 'Upgrade', icon: <Tag className="w-5 h-5" /> },
        { path: '/help', label: 'Help', icon: <HelpCircle className="w-5 h-5" /> },
    ]

    return (
        <div className="h-screen w-full flex font-sans overflow-hidden bg-neutral-50 text-neutral-900">
            {/* ══ LEFT: Desktop Sidebar ══ */}
            <div className="hidden lg:flex w-64 flex-col bg-white border-r border-neutral-200 shrink-0">
                {/* Logo */}
                <div className="h-20 flex items-center px-8 shrink-0">
                    <div className="flex items-center gap-2.5 select-none">
                        <div
                            className="w-7 h-7 rounded-full flex items-center justify-center shrink-0"
                            style={{ background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)' }}
                        >
                            <span className="transform rotate-45 block w-3 h-3 border-2 border-white border-t-0 border-l-0 rounded-sm" />
                        </div>
                        <span className="text-neutral-900 text-[17px] font-bold tracking-tight">Tax Check.</span>
                    </div>
                </div>

                {/* Nav links */}
                <nav className="flex-1 px-4 py-6 space-y-1 overflow-y-auto no-scrollbar">
                    {navItems.map((item) => {
                        const active = pathname === item.path || (item.path !== '/' && pathname.startsWith(item.path) && item.path !== '/dashboard')
                        // For dashboard strictly require exact match to not highlight on every subpage
                        const isStrictActive = item.path === '/dashboard' ? pathname === '/dashboard' : active

                        return (
                            <Link
                                key={item.path}
                                to={item.path}
                                className={cn(
                                    "flex items-center gap-3 px-4 py-3 rounded-xl text-[14px] font-semibold transition-all",
                                    isStrictActive
                                        ? "bg-blue-50 text-blue-700"
                                        : "text-neutral-500 hover:text-neutral-900 hover:bg-neutral-100/50"
                                )}
                            >
                                {item.icon}
                                {item.label}
                            </Link>
                        )
                    })}
                </nav>

                {/* Bottom of sidebar */}
                <div className="p-6 shrink-0 border-t border-neutral-100">
                    <p className="text-xs text-neutral-400 font-medium tracking-wide text-center">NTA 2025 Edition</p>
                </div>
            </div>

            {/* ══ RIGHT: Main Panel ══ */}
            <div className="flex-1 flex flex-col min-w-0">
                {/* Top header */}
                <header className="h-20 bg-white/50 backdrop-blur-md border-b border-neutral-200/60 flex items-center justify-between px-8 shrink-0 lg:px-12 z-10 sticky top-0">
                    <div className="lg:hidden flex items-center gap-2">
                        {/* Mobile logo fallback */}
                        <div
                            className="w-6 h-6 rounded-full flex items-center justify-center"
                            style={{ background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)' }}
                        >
                            <span className="transform rotate-45 block w-2.5 h-2.5 border-[1.5px] border-white border-t-0 border-l-0 rounded-[1px]" />
                        </div>
                        <span className="font-bold text-sm">Tax Check</span>
                    </div>
                    <div className="hidden lg:block">
                        <h2 className="text-sm font-semibold text-neutral-400">Dashboard</h2>
                    </div>

                    <div className="flex items-center gap-4">
                        <button className="w-10 h-10 rounded-full flex items-center justify-center text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 transition-colors">
                            <Bell className="w-5 h-5" />
                        </button>
                        <div className="h-6 w-px bg-neutral-200" />
                        <button className="flex items-center gap-2.5 select-none hover:opacity-80 transition-opacity">
                            <div className="w-8 h-8 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center text-xs font-bold ring-2 ring-white">
                                {localStorage.getItem('tc_profile') ? JSON.parse(localStorage.getItem('tc_profile') as string).first_name?.[0] || 'U' : 'U'}
                            </div>
                        </button>
                    </div>
                </header>

                {/* Body Canvas */}
                <main className="flex-1 overflow-y-auto no-scrollbar scroll-smooth">
                    <div className="w-full max-w-[960px] mx-auto px-6 lg:px-12 py-10 lg:py-12">
                        {children}
                    </div>
                </main>
            </div>

            {/* Mobile Bottom nav - stubbed */}
            <nav className="lg:hidden fixed bottom-0 inset-x-0 bg-white border-t border-neutral-200 z-40 h-16 flex">
                <Link to="/dashboard" className="flex-1 flex flex-col items-center justify-center text-blue-700">
                    <Home className="w-5 h-5" />
                </Link>
                <Link to="/documents" className="flex-1 flex flex-col items-center justify-center text-neutral-400">
                    <FileText className="w-5 h-5" />
                </Link>
            </nav>
        </div>
    )
}
