import { useOnboardingStore } from '@/stores/onboardingStore'
import { computePIT, computeRentRelief } from '@/lib/pit'

// Safely coerce an unknown profile value to a positive number.
function n(v: unknown): number {
    const x = Number(v)
    return isFinite(x) && x > 0 ? x : 0
}

export interface PITEstimate {
    totalIncome: number
    totalDeductions: number
    totalReliefs: number         // alias for totalDeductions — used in E-File summary
    chargeableIncome: number
    taxPayable: number
    taxAlreadyPaid: number
    /** Positive = refund due to taxpayer. Negative = amount still owed. */
    balance: number
    hasData: boolean
}

export function computePITEstimate(p: Partial<any>): PITEstimate {

    // ── Income ──────────────────────────────────────────────────────────────
    const grossSalary = n(p.employment_gross_salary) || n(p.employment_basic_salary)
    const bonuses = p.received_bonuses === true ? n(p.bonus_amount) : 0
    const commissions = p.received_commissions === true ? n(p.commission_amount) : 0
    const allowances = p.received_allowances === true ? n(p.taxable_allowances_total) : 0

    const freelance = n(p.freelance_gross_income)
    const bizProfit = Math.max(0, n(p.business_total_revenue) - n(p.business_allowable_expenses))
    const profNet = Math.max(0,
        n(p.professional_gross_income) -
        (p.has_professional_expenses === true ? n(p.professional_expenses_total) : 0)
    )
    const rental = n(p.total_rental_income)
    const interest = p.received_interest_income === true ? n(p.total_interest_received) : 0
    const dividends = p.received_dividends === true ? n(p.total_dividends_received) : 0
    const foreign = p.earned_foreign_income === true ? n(p.total_foreign_income) : 0
    const pensionInc = p.receives_pension_income === true ? n(p.pension_income_amount) : 0
    const other = p.has_other_income === true ? n(p.other_income_total) : 0

    const totalIncome =
        grossSalary + bonuses + commissions + allowances +
        freelance + bizProfit + profNet + rental +
        interest + dividends + foreign + pensionInc + other

    // ── Eligible deductions (NTA s.30) ───────────────────────────────────────
    const pension = p.makes_pension_contributions === true ? n(p.pension_contribution_amount) : 0
    const nhis = p.makes_nhis_contributions === true ? n(p.nhis_contribution_amount) : 0
    const nhf = p.makes_nhf_contributions === true ? n(p.nhf_contribution_amount) : 0
    const rentRelief = (p.pays_own_rent === true && p.rent_is_own_residence === true)
        ? computeRentRelief(n(p.annual_rent_paid)) : 0
    const insurance = p.has_life_insurance === true ? n(p.qualifying_premiums_total) : 0
    const loanInt = (p.has_home_loan === true && p.home_loan_property_owner_occupied === true)
        ? n(p.home_loan_interest_paid) : 0

    const totalDeductions = pension + nhis + nhf + rentRelief + insurance + loanInt

    // ── PIT ──────────────────────────────────────────────────────────────────
    const chargeableIncome = Math.max(0, totalIncome - totalDeductions)
    const taxPayable = computePIT(chargeableIncome)

    // ── Tax already paid / credits ────────────────────────────────────────────
    const payeCredit = p.employment_paye_deducted === true ? n(p.employment_paye_amount) : 0
    const freelanceWHT = p.freelance_wht_deducted === true ? n(p.freelance_wht_amount) : 0
    const profWHT = p.professional_wht_deducted === true ? n(p.professional_wht_amount) : 0
    const tenantWHT = p.tenant_wht_deducted === true ? n(p.tenant_wht_amount) : 0
    const interestWHT = p.interest_tax_deducted === true ? n(p.interest_tax_amount) : 0
    const directTax = n(p.direct_assessment_paid)

    const taxAlreadyPaid = payeCredit + freelanceWHT + profWHT + tenantWHT + interestWHT + directTax

    return {
        totalIncome,
        totalDeductions,
        totalReliefs: totalDeductions,
        chargeableIncome,
        taxPayable,
        taxAlreadyPaid,
        balance: taxAlreadyPaid - taxPayable,
        hasData: totalIncome > 0,
    }
}

export function usePITEstimate(): PITEstimate {
    const p = useOnboardingStore((s) => s.profile)
    return computePITEstimate(p)
}
