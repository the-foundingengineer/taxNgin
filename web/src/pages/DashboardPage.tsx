import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { computePITEstimate, PITEstimate } from '@/hooks/usePITEstimate'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { fmtNaira } from '@/lib/pit'
import { ArrowRight, CheckCircle2, FileText, Sparkles, Download, LifeBuoy, Banknote } from 'lucide-react'
import { ProfileImportModal, downloadProfile } from '@/components/shared/ProfileTransfer'
import { runNTAOptimizer } from '@/hooks/useNTAOptimizer'

export function DashboardPage() {
  const navigate = useNavigate()
  const [estimate, setEstimate] = useState<PITEstimate | null>(null)
  const [firstName, setFirstName] = useState('Taxpayer')
  const [showImport, setShowImport] = useState(false)
  const [missedCount, setMissedCount] = useState(0)

  useEffect(() => {
    try {
      const raw = localStorage.getItem('tc_profile')
      if (raw) {
        const profile = JSON.parse(raw)
        setEstimate(computePITEstimate(profile))
        if (profile.first_name) setFirstName(profile.first_name)
        const opt = runNTAOptimizer(profile)
        setMissedCount(opt.missed.length)
      }
    } catch (e) {
      console.error('Failed to parse local profile', e)
    }
  }, [])

  return (
    <DashboardLayout>
      {showImport && <ProfileImportModal onClose={() => setShowImport(false)} />}
      <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-20">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold text-neutral-900 tracking-tight">Welcome back, {firstName}</h1>
          <p className="text-neutral-500 font-medium">Let's continue working on your 2025 tax return.</p>
        </div>

        {estimate && <LiveEstimateHero estimate={estimate} />}

        {/* Phase B: Smart Tools Strip */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <SmartToolCard
            icon={<Sparkles className="w-5 h-5" />}
            color="blue"
            title="NTA Optimizer"
            desc={missedCount > 0 ? `${missedCount} reliefs may be unclaimed` : 'All reliefs captured ✓'}
            badge={missedCount > 0 ? `${missedCount} found` : undefined}
            onClick={() => navigate('/optimizer')}
          />
          <SmartToolCard
            icon={<Download className="w-5 h-5" />}
            color="indigo"
            title="Import last year"
            desc="Pre-fill from a previous Tax Check profile"
            onClick={() => setShowImport(true)}
            secondaryAction={{ label: 'Export this year', onClick: () => downloadProfile() }}
          />
          <SmartToolCard
            icon={<LifeBuoy className="w-5 h-5" />}
            color="rose"
            title="Audit Support"
            desc="Got an NRS assessment? We'll guide you."
            onClick={() => navigate('/help/audit')}
          />
        </div>

        {estimate && estimate.hasData && <WHTCreditWidget estimate={estimate} />}

        {/* Lower Dashboard Widgets */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 pt-2">
          <div className="lg:col-span-2">
            <ProgressChecklist />
          </div>
          <div className="lg:col-span-1 border-t lg:border-t-0 lg:border-l border-neutral-200/60 pt-8 lg:pt-0 lg:pl-8">
            <DocumentVault />
          </div>
        </div>
      </div>
    </DashboardLayout>
  )
}

interface SmartToolCardProps {
  icon: React.ReactNode
  color: 'blue' | 'indigo' | 'rose'
  title: string
  desc: string
  badge?: string
  onClick: () => void
  secondaryAction?: { label: string; onClick: () => void }
}

function SmartToolCard({ icon, color, title, desc, badge, onClick, secondaryAction }: SmartToolCardProps) {
  const colorCls = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200 hover:border-blue-400',
    indigo: 'bg-indigo-50 text-indigo-700 border-indigo-200 hover:border-indigo-400',
    rose: 'bg-rose-50 text-rose-700 border-rose-200 hover:border-rose-400',
  }[color]
  const iconCls = {
    blue: 'bg-blue-100 text-blue-600',
    indigo: 'bg-indigo-100 text-indigo-600',
    rose: 'bg-rose-100 text-rose-600',
  }[color]

  return (
    <div className={`relative border rounded-2xl p-5 cursor-pointer transition-all shadow-sm hover:shadow-md ${colorCls}`} onClick={onClick}>
      {badge && (
        <span className="absolute -top-2.5 -right-2 px-2 py-0.5 rounded-full text-[10px] font-bold bg-amber-400 text-white shadow">
          {badge}
        </span>
      )}
      <div className={`w-10 h-10 rounded-xl flex items-center justify-center mb-3 ${iconCls}`}>{icon}</div>
      <h3 className="font-bold text-[15px] leading-tight">{title}</h3>
      <p className="text-xs mt-1 opacity-75 leading-relaxed">{desc}</p>
      {secondaryAction && (
        <button
          onClick={e => { e.stopPropagation(); secondaryAction.onClick() }}
          className="mt-3 text-[11px] font-bold underline underline-offset-2 opacity-60 hover:opacity-100 transition-opacity"
        >
          {secondaryAction.label}
        </button>
      )}
    </div>
  )
}

function WHTCreditWidget({ estimate }: { estimate: PITEstimate }) {
  const { taxAlreadyPaid, taxPayable } = estimate
  if (taxAlreadyPaid === 0) return null
  const pct = Math.min(100, Math.round((taxAlreadyPaid / Math.max(taxPayable, 1)) * 100))

  return (
    <div className="bg-white border border-neutral-200 rounded-2xl p-6 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <Banknote className="w-5 h-5 text-indigo-600" />
            <h3 className="font-bold text-[16px] text-neutral-900">WHT & PAYE Credits</h3>
          </div>
          <p className="text-sm text-neutral-500 mt-0.5">Tax already paid on your behalf this year</p>
        </div>
        <div className="text-right">
          <p className="text-2xl font-black text-indigo-700">{fmtNaira(taxAlreadyPaid)}</p>
          <p className="text-xs text-neutral-400 mt-0.5">of {fmtNaira(taxPayable)} total liability</p>
        </div>
      </div>
      <div className="h-2.5 bg-neutral-100 rounded-full overflow-hidden">
        <div
          className="h-full rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 transition-all duration-700"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="text-xs text-neutral-500 mt-2">{pct}% of your liability is already cleared via PAYE/WHT credits</p>
    </div>
  )
}

function LiveEstimateHero({ estimate }: { estimate: PITEstimate }) {
  const isRefund = estimate.balance >= 0
  // Styling based on refund vs due
  const bgClass = isRefund
    ? 'bg-gradient-to-br from-[#0f766e] to-[#042f2e]' // Deep teal/emerald for refund
    : 'bg-gradient-to-br from-[#be123c] to-[#4c0519]' // Deep rose/burgundy for due

  return (
    <div className={`relative overflow-hidden rounded-[24px] text-white p-8 lg:p-12 shadow-2xl ${bgClass}`}>

      {/* Background design elements */}
      <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-white/10 blur-[100px] rounded-full translate-x-1/2 -translate-y-1/2" />
      <div className="absolute bottom-0 left-0 w-[300px] h-[300px] bg-black/20 blur-[80px] rounded-full -translate-x-1/3 translate-y-1/3" />

      <div className="relative z-10 flex flex-col md:flex-row items-start md:items-center justify-between gap-10">

        {/* Left side: The number */}
        <div className="space-y-4">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/10 border border-white/20 backdrop-blur-md">
            <div className={`w-2 h-2 rounded-full ${isRefund ? 'bg-emerald-400' : 'bg-rose-400'} animate-pulse`} />
            <span className="text-xs font-bold tracking-widest uppercase text-white/90">
              {isRefund ? 'Estimated Refund' : 'Estimated Tax Due'}
            </span>
          </div>

          <div>
            <h2 className="text-[56px] leading-[1.1] font-black tracking-tighter drop-shadow-sm">
              {fmtNaira(Math.abs(estimate.balance))}
            </h2>
            <p className="text-white/70 font-medium mt-2 text-lg">
              {isRefund
                ? "This is how much you're getting back so far."
                : "This is how much you owe so far."}
            </p>
          </div>

          <p className="text-sm text-white/50 pt-2 max-w-sm">
            Based on {fmtNaira(estimate.totalIncome)} total income and {fmtNaira(estimate.totalDeductions)} in eligible deductions according to Nigerian Tax Act provisions.
          </p>
        </div>

        {/* Right side: Primary CTA Card mapped neatly inside the hero like TurboTax */}
        <div className="w-full md:w-[340px] shrink-0 bg-white shadow-xl rounded-2xl p-6 text-neutral-900 border border-white/20 relative">
          <div className="absolute -top-3 -right-3 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center shadow-md">
            <span className="text-xs font-bold">1</span>
          </div>
          <h3 className="font-bold text-lg mb-1">Pick up where you left off</h3>
          <p className="text-sm text-neutral-500 mb-6 leading-relaxed">
            It's time to review your deductions and reliefs to make sure we aren't missing any tax breaks.
          </p>
          <button onClick={() => window.location.href = '#'} className="w-full h-12 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-xl flex items-center justify-center gap-2 transition-colors shadow-sm">
            Review Deductions
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

function ProgressChecklist() {
  const steps = [
    { label: 'Personal Information', desc: 'Identity, family & residency', status: 'completed' },
    { label: 'Employment & Income', desc: 'Salary, freelance & business', status: 'completed' },
    { label: 'Deductions & Reliefs', desc: 'Pension, NHF, & life insurance', status: 'active' },
    { label: 'Review & Optimization', desc: 'Final checks for your refund', status: 'pending' },
    { label: 'File Return & Pay', desc: 'Secure submission to NRS', status: 'pending' },
  ]

  return (
    <div className="bg-white rounded-2xl border border-neutral-200 shadow-sm overflow-hidden">
      <div className="p-6 border-b border-neutral-100">
        <h3 className="font-bold text-[17px] text-neutral-900">Your filing progress</h3>
        <p className="text-sm text-neutral-500">Just a few steps left to complete your 2025 return.</p>
      </div>
      <div className="p-6 space-y-7">
        {steps.map((step, idx) => {
          const isLast = idx === steps.length - 1
          return (
            <div key={idx} className="relative flex items-start gap-4">
              {/* Vertical Line Connector */}
              {!isLast && (
                <div className={`absolute top-8 left-3.5 w-px h-full -ml-[0.5px] ${step.status === 'completed' ? 'bg-emerald-500' : 'bg-neutral-200'}`} style={{ height: 'calc(100% + 12px)' }} />
              )}

              {/* Status Icon */}
              <div className="relative z-10 pt-0.5">
                {step.status === 'completed' && (
                  <div className="w-7 h-7 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center">
                    <CheckCircle2 className="w-4 h-4" strokeWidth={3} />
                  </div>
                )}
                {step.status === 'active' && (
                  <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center ring-4 ring-white">
                    <div className="w-2.5 h-2.5 bg-blue-600 rounded-full animate-pulse" />
                  </div>
                )}
                {step.status === 'pending' && (
                  <div className="w-7 h-7 rounded-full bg-neutral-100 border-2 border-neutral-200 flex items-center justify-center ring-4 ring-white" />
                )}
              </div>

              {/* Text Content */}
              <div className={`flex-1 ${step.status === 'pending' ? 'opacity-50' : ''}`}>
                <h4 className={`text-[15px] font-semibold ${step.status === 'active' ? 'text-blue-700' : 'text-neutral-900'}`}>
                  {step.label}
                </h4>
                <p className="text-sm text-neutral-500 mt-0.5">{step.desc}</p>
              </div>

              {/* Action text */}
              <div className="pt-1">
                {step.status === 'completed' && <span className="text-xs font-bold text-emerald-600">Done</span>}
                {step.status === 'active' && <button className="text-xs font-bold text-blue-600 hover:underline">Continue</button>}
                {step.status === 'pending' && <span className="text-xs font-semibold text-neutral-400">Locked</span>}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function DocumentVault() {
  return (
    <div className="space-y-5">
      <div>
        <h3 className="font-bold text-[17px] text-neutral-900">Your documents</h3>
        <p className="text-sm text-neutral-500">Secure storage for your tax files</p>
      </div>

      <div className="space-y-3">
        <div className="bg-white p-4 rounded-xl border border-neutral-200 shadow-sm flex items-start gap-3 hover:border-blue-300 transition-colors cursor-pointer group">
          <div className="w-10 h-10 rounded-lg bg-red-50 text-red-500 flex items-center justify-center shrink-0 group-hover:bg-red-100">
            <FileText className="w-5 h-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-neutral-900 leading-tight">Form H1 (Annual PAYE)</h4>
            <p className="text-[11px] text-neutral-500 mt-0.5 border border-emerald-200 bg-emerald-50 text-emerald-700 px-1.5 py-[1px] rounded inline-flex items-center">
              Uploaded
            </p>
            <p className="text-[11px] text-neutral-400 mt-1">TechCorp Nigeria Ltd</p>
          </div>
        </div>

        <div className="bg-white p-4 rounded-xl border border-neutral-200 shadow-sm flex items-start gap-3 hover:border-blue-300 transition-colors cursor-pointer group">
          <div className="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center shrink-0 group-hover:bg-blue-100">
            <FileText className="w-5 h-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-neutral-900 leading-tight">NTA Assessment 2024</h4>
            <p className="text-[11px] text-neutral-500 mt-0.5 border border-neutral-200 bg-neutral-100 text-neutral-600 px-1.5 py-[1px] rounded inline-flex items-center">
              Archived
            </p>
            <p className="text-[11px] text-neutral-400 mt-1">Auto-generated</p>
          </div>
        </div>
      </div>

      <button className="w-full py-2.5 border-2 border-dashed border-neutral-300 rounded-xl text-sm font-semibold text-neutral-500 hover:border-neutral-400 hover:text-neutral-700 transition-colors">
        + Upload document
      </button>
    </div>
  )
}
