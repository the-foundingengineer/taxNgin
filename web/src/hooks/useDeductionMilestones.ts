import { useEffect, useRef, useState } from 'react'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { computeRentRelief, fmtNaira, pos } from '@/lib/pit'
import type { TaxpayerProfile } from '@/types/onboarding'

export interface DeductionToast {
    key:      string
    icon:     string
    title:    string
    subtitle: string
}

interface MilestoneDef {
    id:       string
    icon:     string
    title:    string
    check:    (p: TaxpayerProfile) => boolean
    subtitle: (p: TaxpayerProfile) => string
}

const MILESTONES: MilestoneDef[] = [
    {
        id:    'rent_relief',
        icon:  'key',
        title: 'Rent relief applied!',
        check: (p) =>
            p.pays_own_rent === true &&
            p.rent_is_own_residence === true &&
            pos(p.annual_rent_paid) > 0,
        subtitle: (p) => {
            const relief = computeRentRelief(pos(p.annual_rent_paid))
            return `${fmtNaira(relief)} deducted from your taxable income`
        },
    },
    {
        id:    'pension',
        icon:  'elderly',
        title: 'Pension deduction added!',
        check: (p) =>
            p.makes_pension_contributions === true &&
            pos(p.pension_contribution_amount) > 0,
        subtitle: (p) =>
            `${fmtNaira(pos(p.pension_contribution_amount))} deducted — NTA s.30`,
    },
    {
        id:    'nhis',
        icon:  'health_and_safety',
        title: 'NHIS deduction added!',
        check: (p) =>
            p.makes_nhis_contributions === true &&
            pos(p.nhis_contribution_amount) > 0,
        subtitle: (p) =>
            `${fmtNaira(pos(p.nhis_contribution_amount))} deducted — NTA s.30`,
    },
    {
        id:    'nhf',
        icon:  'home',
        title: 'NHF deduction added!',
        check: (p) =>
            p.makes_nhf_contributions === true &&
            pos(p.nhf_contribution_amount) > 0,
        subtitle: (p) =>
            `${fmtNaira(pos(p.nhf_contribution_amount))} deducted — NTA s.30`,
    },
    {
        id:    'life_insurance',
        icon:  'favorite',
        title: 'Life insurance deducted!',
        check: (p) =>
            p.has_life_insurance === true &&
            pos(p.qualifying_premiums_total) > 0,
        subtitle: (p) =>
            `${fmtNaira(pos(p.qualifying_premiums_total))} in premiums deducted — NTA s.30`,
    },
    {
        id:    'home_loan',
        icon:  'account_balance',
        title: 'Home loan interest deducted!',
        check: (p) =>
            p.has_home_loan === true &&
            p.home_loan_property_owner_occupied === true &&
            pos(p.home_loan_interest_paid) > 0,
        subtitle: (p) =>
            `${fmtNaira(pos(p.home_loan_interest_paid))} in interest deducted — NTA s.30`,
    },
]

const AUTO_DISMISS_MS = 4500

export function useDeductionMilestones() {
    const profile = useOnboardingStore((s) => s.profile)
    const shown   = useRef(new Set<string>())
    const [toasts, setToasts] = useState<DeductionToast[]>([])

    useEffect(() => {
        const fresh: DeductionToast[] = []
        for (const m of MILESTONES) {
            if (shown.current.has(m.id)) continue
            try {
                if (m.check(profile)) {
                    shown.current.add(m.id)
                    fresh.push({
                        key:      `${m.id}-${Date.now()}`,
                        icon:     m.icon,
                        title:    m.title,
                        subtitle: m.subtitle(profile),
                    })
                }
            } catch {
                // guard against unexpected profile shapes
            }
        }
        if (fresh.length > 0) setToasts((prev) => [...prev, ...fresh])
    }, [profile])

    function dismiss(key: string) {
        setToasts((prev) => prev.filter((t) => t.key !== key))
    }

    return { toasts, dismiss, AUTO_DISMISS_MS }
}
