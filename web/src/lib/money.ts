// Money utilities — integer arithmetic only.
//
// All internal values are kobo (int64 on the backend, number in JS which is
// safe as an integer up to 2^53−1 ≈ ₦90 trillion). Never divide by 100 for
// accumulation — only at the final display boundary.

/** Format kobo as a Naira string: 480_000_000 → "₦4,800,000" */
export function formatNaira(kobo: number): string {
  const sign = kobo < 0 ? '-' : ''
  const abs = Math.abs(kobo)
  const naira = Math.trunc(abs / 100)
  const cents = abs % 100
  const nairaFormatted = naira.toLocaleString('en-NG')
  if (cents === 0) return `${sign}₦${nairaFormatted}`
  return `${sign}₦${nairaFormatted}.${cents.toString().padStart(2, '0')}`
}

/** Parse a user-typed Naira string to kobo: "4,800,000" → 480_000_000 */
export function parseNairaToKobo(input: string): number {
  const cleaned = input.replace(/[₦,\s]/g, '')
  if (cleaned === '' || cleaned === '-') return 0
  const parts = cleaned.split('.')
  const nairaStr = parts[0] ?? '0'
  const centsStr = (parts[1] ?? '00').padEnd(2, '0').slice(0, 2)
  const naira = parseInt(nairaStr, 10)
  const cents = parseInt(centsStr, 10)
  if (!Number.isInteger(naira) || !Number.isInteger(cents)) {
    throw new Error(`Invalid Naira amount: "${input}"`)
  }
  return naira * 100 + (naira < 0 ? -cents : cents)
}

/** Convert kobo to naira for display purposes only (truncates, never rounds). */
export function koboToNairaDisplay(kobo: number): number {
  return Math.trunc(kobo / 100)
}

/** Sum an array of kobo values — integer addition, never float. */
export function sumKobo(values: number[]): number {
  return values.reduce((acc, v) => acc + v, 0)
}
