/** Safely coerce an unknown profile value to a positive number, or 0. */
export function pos(v: unknown): number {
    const x = Number(v)
    return isFinite(x) && x > 0 ? x : 0
}

// NTA 2025 Fourth Schedule — individual PIT bands.
// Each entry is the SIZE of the band, not the cumulative threshold.
const BANDS: { size: number; rate: number }[] = [
    { size: 800_000,    rate: 0.00 },
    { size: 2_200_000,  rate: 0.15 },
    { size: 9_000_000,  rate: 0.18 },
    { size: 13_000_000, rate: 0.21 },
    { size: 25_000_000, rate: 0.23 },
    { size: Infinity,   rate: 0.25 },
]

const RENT_RELIEF_RATE = 0.20
const RENT_RELIEF_CAP  = 500_000

export function computePIT(chargeableIncome: number): number {
    let remaining = Math.max(0, chargeableIncome)
    let tax = 0
    for (const { size, rate } of BANDS) {
        const slice = Math.min(remaining, size)
        tax += slice * rate
        remaining -= slice
        if (remaining <= 0) break
    }
    return Math.round(tax)
}

export function computeRentRelief(annualRent: number): number {
    return Math.min(annualRent * RENT_RELIEF_RATE, RENT_RELIEF_CAP)
}

export function fmtNaira(n: number): string {
    const abs = Math.abs(n)
    if (abs === 0) return '₦0'
    if (abs >= 1_000_000) {
        const m = abs / 1_000_000
        const s = m % 1 === 0 ? m.toFixed(0) : m.toFixed(2).replace(/\.?0+$/, '')
        return `₦${s}m`
    }
    if (abs >= 1_000) {
        const k = abs / 1_000
        const s = k % 1 === 0 ? k.toFixed(0) : k.toFixed(1).replace(/\.?0+$/, '')
        return `₦${s}k`
    }
    return `₦${Math.round(abs).toLocaleString()}`
}
