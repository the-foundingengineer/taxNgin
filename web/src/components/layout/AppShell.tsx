import { Link, useLocation } from 'react-router-dom'
import { LayoutDashboard, FileText, ArrowUpCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

interface AppShellProps {
  children: React.ReactNode
  /** Hide the bottom nav (e.g. inside the filing wizard) */
  hideNav?: boolean
}

export function AppShell({ children, hideNav }: AppShellProps) {
  return (
    <div className="min-h-screen bg-neutral-50">
      {/* Top bar */}
      <header className="sticky top-0 z-40 bg-white border-b border-neutral-200">
        <div className="max-w-md mx-auto px-4 h-14 flex items-center justify-between">
          <Link to="/dashboard" className="flex items-center gap-2">
            <span className="text-brand-400 font-bold text-xl tracking-tight">Tax Check</span>
          </Link>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-md mx-auto px-4 py-6 pb-24">
        {children}
      </main>

      {/* Bottom nav — hidden during wizard */}
      {!hideNav && <BottomNav />}
    </div>
  )
}

function BottomNav() {
  const { pathname } = useLocation()
  const items = [
    { to: '/dashboard', label: 'Returns', icon: LayoutDashboard },
    { to: '/returns/new', label: 'New return', icon: FileText },
    { to: '/upgrade', label: 'Upgrade', icon: ArrowUpCircle },
  ]
  return (
    <nav className="fixed bottom-0 inset-x-0 bg-white border-t border-neutral-200 z-40">
      <div className="max-w-md mx-auto flex">
        {items.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className={cn(
              'flex-1 flex flex-col items-center gap-1 py-3 text-xs font-medium transition-colors',
              pathname.startsWith(to) && to !== '/returns/new'
                ? 'text-brand-500'
                : 'text-neutral-400 hover:text-neutral-600',
            )}
          >
            <Icon className="h-5 w-5" />
            {label}
          </Link>
        ))}
      </div>
    </nav>
  )
}
