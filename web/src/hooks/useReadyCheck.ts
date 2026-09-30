/**
 * useReadyCheck.ts
 * ─────────────────
 * NRS filing gate-check engine.
 * Reads tc_profile from localStorage and returns a list of
 * pass/fail checks that must all pass before e-filing.
 */

export type CheckStatus = 'pass' | 'fail' | 'warn'

export interface ReadyCheckItem {
    id: string
    category: string
    label: string
    description: string
    status: CheckStatus
    fixRoute?: string      // where to go to fix it
}

export interface ReadyCheckResult {
    canFile: boolean        // true iff no 'fail' items
    score: number           // 0-100 completeness score
    items: ReadyCheckItem[]
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function runReadyCheck(p: Partial<Record<string, any>>): ReadyCheckResult {
    const items: ReadyCheckItem[] = []

    // ── Identity ───────────────────────────────────────────────────────────────
    items.push({
        id: 'tin', category: 'Identity',
        label: 'Tax Identification Number (TIN)',
        description: 'Your FIRS-issued TIN is required on every return.',
        status: p.tin && String(p.tin).replace(/\D/g, '').length >= 10 ? 'pass' : 'fail',
        fixRoute: '/profile',
    })

    items.push({
        id: 'name', category: 'Identity',
        label: 'Full Legal Name',
        description: 'First and last name must match your FIRS records.',
        status: p.first_name && p.last_name ? 'pass' : 'fail',
        fixRoute: '/profile',
    })

    items.push({
        id: 'dob', category: 'Identity',
        label: 'Date of Birth',
        description: 'Required for taxpayer verification.',
        status: p.date_of_birth ? 'pass' : 'warn',
        fixRoute: '/profile',
    })

    // ── Income ─────────────────────────────────────────────────────────────────
    const hasIncome =
        Number(p.employment_gross_salary ?? 0) > 0 ||
        Number(p.freelance_gross_income ?? 0) > 0 ||
        Number(p.business_net_profit ?? 0) > 0 ||
        Number(p.rental_income_annual ?? 0) > 0

    items.push({
        id: 'income', category: 'Income',
        label: 'At Least One Income Source',
        description: 'You must declare at least one income source.',
        status: hasIncome ? 'pass' : 'fail',
        fixRoute: '/deductions',
    })

    items.push({
        id: 'paye', category: 'Income',
        label: 'PAYE / Employer TIN (if employed)',
        description: 'Employees must include their employer\'s FIRS TIN from Form H1.',
        status: p.work_situation?.includes('employee')
            ? (p.employer_tin ? 'pass' : 'warn')
            : 'pass',
        fixRoute: '/profile',
    })

    // ── Deductions ─────────────────────────────────────────────────────────────
    items.push({
        id: 'cra', category: 'Deductions',
        label: 'Consolidated Relief Allowance Confirmed',
        description: 'CRA is automatically computed from your gross income.',
        status: hasIncome ? 'pass' : 'warn',
        fixRoute: '/deductions',
    })

    items.push({
        id: 'pension', category: 'Deductions',
        label: 'Pension Contribution Declared',
        description: 'Pension deduction status must be explicitly confirmed.',
        status: p.makes_pension_contributions !== undefined ? 'pass' : 'warn',
        fixRoute: '/deductions',
    })

    // ── Banking ────────────────────────────────────────────────────────────────
    items.push({
        id: 'bank', category: 'Banking',
        label: 'Bank Account Number (for Refunds)',
        description: 'Needed if you are owed a refund after filing.',
        status: p.bank_account_number && String(p.bank_account_number).length >= 10
            ? 'pass' : 'warn',
        fixRoute: '/profile',
    })

    // ── Declaration ────────────────────────────────────────────────────────────
    items.push({
        id: 'declaration', category: 'Declaration',
        label: 'Information Accuracy Declaration',
        description: 'You must confirm all information is true and complete.',
        status: p.declaration_confirmed === true ? 'pass' : 'fail',
        fixRoute: '/efile',
    })

    const failed = items.filter(i => i.status === 'fail').length
    const passed = items.filter(i => i.status === 'pass').length
    const score = Math.round((passed / items.length) * 100)

    return {
        canFile: failed === 0,
        score,
        items,
    }
}
