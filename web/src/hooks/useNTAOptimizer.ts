/**
 * useNTAOptimizer
 * ---------------
 * Scans a taxpayer profile against every NTA 2025 s.30 relief and
 * returns categorised "found" vs "potentially missed" opportunities.
 * Works from either the Zustand store (onboarding flow) or a raw
 * profile object (Dashboard, loaded from localStorage).
 */

import { computeRentRelief, pos } from '@/lib/pit'

export type ReliefStatus = 'found' | 'missed' | 'not_applicable'

export interface Relief {
    id: string
    icon: string
    title: string
    basis: string          // NTA legal reference
    status: ReliefStatus
    estimatedSaving: number   // ₦ tax saving at marginal rate
    description: string
    actionLabel?: string   // CTA text if missed
}

export interface OptimizerResult {
    found: Relief[]
    missed: Relief[]
    totalFoundSaving: number
    totalMissedSaving: number
}

const MARGINAL_RATE = 0.21   // approximate mid-band rate for savings estimate

function saving(amount: number) {
    return Math.round(amount * MARGINAL_RATE)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function runNTAOptimizer(p: Partial<Record<string, any>>): OptimizerResult {
    const reliefs: Relief[] = []

    // ── 1. Consolidated Relief Allowance (CRA) — always applicable ──────────
    const grossIncome = pos(p.employment_gross_salary) || pos(p.employment_basic_salary) || pos(p.freelance_gross_income)
    const cra = Math.max(200_000, grossIncome * 0.01) + grossIncome * 0.20
    reliefs.push({
        id: 'cra',
        icon: 'person',
        title: 'Consolidated Relief Allowance (CRA)',
        basis: 'NTA 2025 s.30(1)',
        status: grossIncome > 0 ? 'found' : 'not_applicable',
        estimatedSaving: saving(cra),
        description: `₦200,000 + 20% of gross income automatically deducted from your taxable income.`,
    })

    // ── 2. Pension Contribution (RSS/PFA) ────────────────────────────────────
    const pensionAmount = pos(p.pension_contribution_amount)
    const pensionStatus: ReliefStatus = p.makes_pension_contributions === true
        ? 'found'
        : (grossIncome > 0 ? 'missed' : 'not_applicable')
    reliefs.push({
        id: 'pension',
        icon: 'savings',
        title: 'Pension Contribution (8% Employee RSS)',
        basis: 'NTA 2025 s.30(2)(a)',
        status: pensionStatus,
        estimatedSaving: pensionStatus === 'found' ? saving(pensionAmount) : saving(grossIncome * 0.08),
        description: 'Mandatory 8% of monthly emolument to your Retirement Savings Account (RSA) is fully deductible.',
        actionLabel: pensionStatus === 'missed' ? 'Confirm your pension contributions' : undefined,
    })

    // ── 3. NHF Contribution ──────────────────────────────────────────────────
    const nhfAmount = pos(p.nhf_contribution_amount)
    const nhfStatus: ReliefStatus = p.makes_nhf_contributions === true
        ? 'found'
        : (p.work_situation?.includes('employee') ? 'missed' : 'not_applicable')
    reliefs.push({
        id: 'nhf',
        icon: 'home',
        title: 'National Housing Fund (NHF) Contribution',
        basis: 'NTA 2025 s.30(2)(b)',
        status: nhfStatus,
        estimatedSaving: nhfStatus === 'found' ? saving(nhfAmount) : saving(grossIncome * 0.025),
        description: '2.5% of monthly basic salary contributed to NHF is fully tax deductible.',
        actionLabel: nhfStatus === 'missed' ? 'Confirm your NHF contributions' : undefined,
    })

    // ── 4. NHIS Contribution ─────────────────────────────────────────────────
    const nhisAmount = pos(p.nhis_contribution_amount)
    const nhisStatus: ReliefStatus = p.makes_nhis_contributions === true ? 'found'
        : (p.work_situation?.includes('employee') ? 'missed' : 'not_applicable')
    reliefs.push({
        id: 'nhis',
        icon: 'health_and_safety',
        title: 'NHIS Health Insurance Contribution',
        basis: 'NTA 2025 s.30(2)(c)',
        status: nhisStatus,
        estimatedSaving: nhisStatus === 'found' ? saving(nhisAmount) : saving(grossIncome * 0.015),
        description: 'Your mandatory 1.5% NHIS employee contribution is tax deductible.',
        actionLabel: nhisStatus === 'missed' ? 'Add your NHIS contributions' : undefined,
    })

    // ── 5. Rent Relief ───────────────────────────────────────────────────────
    const rentAmount = pos(p.annual_rent_paid)
    const rentEligible = p.pays_own_rent === true && p.rent_is_own_residence === true
    const rentRelief = rentEligible ? computeRentRelief(rentAmount) : 0
    const rentStatus: ReliefStatus = rentEligible ? 'found'
        : (p.pays_own_rent === false ? 'not_applicable' : 'missed')
    reliefs.push({
        id: 'rent',
        icon: 'apartment',
        title: 'Rent Relief (Owner-Occupied Residence)',
        basis: 'NTA 2025 s.30(3)',
        status: rentStatus,
        estimatedSaving: rentStatus === 'found' ? saving(rentRelief) : saving(200_000),
        description: '20% of annual rent for your primary residence (capped at ₦500,000) is deductible.',
        actionLabel: rentStatus === 'missed' ? 'Confirm your rental details' : undefined,
    })

    // ── 6. Life Insurance Premium Relief ────────────────────────────────────
    const insuranceAmount = pos(p.qualifying_premiums_total)
    const insuranceStatus: ReliefStatus = p.has_life_insurance === true ? 'found' : 'missed'
    reliefs.push({
        id: 'insurance',
        icon: 'shield',
        title: 'Qualifying Life Insurance Premiums',
        basis: 'NTA 2025 s.30(4)',
        status: insuranceStatus,
        estimatedSaving: insuranceStatus === 'found' ? saving(insuranceAmount) : saving(120_000),
        description: 'Premiums paid on qualifying life assurance policies are deductible.',
        actionLabel: insuranceStatus === 'missed' ? 'Do you have a life insurance policy?' : undefined,
    })

    // ── 7. Home Loan Interest Relief ─────────────────────────────────────────
    const loanInterest = pos(p.home_loan_interest_paid)
    const loanStatus: ReliefStatus = (p.has_home_loan === true && p.home_loan_property_owner_occupied === true)
        ? 'found'
        : (p.has_home_loan === true ? 'missed' : 'not_applicable')
    reliefs.push({
        id: 'mortgage',
        icon: 'real_estate_agent',
        title: 'Mortgage Interest Relief (Owner-Occupied)',
        basis: 'NTA 2025 s.30(5)',
        status: loanStatus,
        estimatedSaving: loanStatus === 'found' ? saving(loanInterest) : saving(180_000),
        description: 'Interest paid on a loan secured on your owner-occupied residential property is deductible.',
        actionLabel: loanStatus === 'missed' ? 'Mark property as owner-occupied' : undefined,
    })

    // ── 8. Dependant Child Relief ─────────────────────────────────────────────
    const childCount = pos(p.dependant_count)
    const hasDependants = p.has_dependants === 'yes'
    const hasChildren = hasDependants && (p.dependant_types as string[] | undefined)?.some(t => ['child', 'stepchild', 'adopted_child', 'foster_child'].includes(t))
    const childReliefAmount = Math.min(childCount, 4) * 250_000  // ₦250k per child, max 4
    reliefs.push({
        id: 'child_relief',
        icon: 'child_care',
        title: 'Dependant Child Relief',
        basis: 'NTA 2025 s.30(6)',
        status: hasChildren ? 'found' : (hasDependants ? 'missed' : 'not_applicable'),
        estimatedSaving: hasChildren ? saving(childReliefAmount) : saving(250_000),
        description: '₦250,000 relief per qualifying child (up to 4). Children must be under 16 or in full-time education.',
        actionLabel: !hasChildren && hasDependants ? 'Confirm child dependant types' : undefined,
    })

    // ── 9. Disability Relief ─────────────────────────────────────────────────
    reliefs.push({
        id: 'disability',
        icon: 'accessible',
        title: 'Disability / Incapacitated Person Relief',
        basis: 'NTA 2025 s.30(7)',
        status: 'missed',
        estimatedSaving: saving(500_000),
        description: 'A ₦500,000 relief is available if you or a dependant has a qualifying disability. Answer one question to check.',
        actionLabel: 'Do you qualify for disability relief?',
    })

    // ── 10. NSITF Contribution ───────────────────────────────────────────────
    reliefs.push({
        id: 'nsitf',
        icon: 'work_history',
        title: 'NSITF Employee Contribution',
        basis: 'NTA 2025 s.30(2)(d)',
        status: p.work_situation?.includes('employee') ? 'missed' : 'not_applicable',
        estimatedSaving: saving(grossIncome * 0.01),
        description: '1% of monthly basic salary contributed to NSITF (Employees Compensation Scheme) is deductible.',
        actionLabel: 'Confirm your NSITF contributions',
    })

    const found = reliefs.filter(r => r.status === 'found')
    const missed = reliefs.filter(r => r.status === 'missed')

    return {
        found,
        missed,
        totalFoundSaving: found.reduce((s, r) => s + r.estimatedSaving, 0),
        totalMissedSaving: missed.reduce((s, r) => s + r.estimatedSaving, 0),
    }
}
