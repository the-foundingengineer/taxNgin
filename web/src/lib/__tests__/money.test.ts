import { describe, it, expect } from 'vitest'
import { formatNaira, parseNairaToKobo, sumKobo } from '../money'

describe('formatNaira', () => {
  it('formats whole naira amounts', () => {
    expect(formatNaira(480_000_000)).toBe('₦4,800,000')
  })

  it('formats kobo remainder', () => {
    expect(formatNaira(123_456_789)).toBe('₦1,234,567.89')
  })

  it('formats zero', () => {
    expect(formatNaira(0)).toBe('₦0')
  })

  it('formats negative amounts', () => {
    expect(formatNaira(-100_000_000)).toBe('-₦1,000,000')
  })

  it('formats single kobo', () => {
    expect(formatNaira(1)).toBe('₦0.01')
  })

  it('never introduces float rounding errors on large amounts', () => {
    // 900,000,000 kobo = ₦9,000,000 exactly — no floating point issues
    const result = formatNaira(900_000_000)
    expect(result).toBe('₦9,000,000')
    expect(result).not.toContain('.')  // exact, no kobo remainder
  })
})

describe('parseNairaToKobo', () => {
  it('parses plain naira', () => {
    expect(parseNairaToKobo('4800000')).toBe(480_000_000)
  })

  it('parses naira with commas', () => {
    expect(parseNairaToKobo('4,800,000')).toBe(480_000_000)
  })

  it('parses naira with kobo', () => {
    expect(parseNairaToKobo('1,234.56')).toBe(123_456)
  })

  it('parses ₦ prefix', () => {
    expect(parseNairaToKobo('₦500,000')).toBe(50_000_000)
  })

  it('returns 0 for empty string', () => {
    expect(parseNairaToKobo('')).toBe(0)
  })

  it('result is an integer (never float)', () => {
    const result = parseNairaToKobo('4,800,000.50')
    expect(Number.isInteger(result)).toBe(true)
    expect(result).toBe(480_000_050)
  })

  it('round-trips with formatNaira', () => {
    const kobo = 960_000_000  // ₦9,600,000
    const formatted = formatNaira(kobo)
    const parsed = parseNairaToKobo(formatted)
    expect(parsed).toBe(kobo)
  })
})

describe('sumKobo', () => {
  it('sums integer values without float error', () => {
    const values = [100_000_000, 200_000_000, 300_000_000]
    expect(sumKobo(values)).toBe(600_000_000)
  })

  it('returns 0 for empty array', () => {
    expect(sumKobo([])).toBe(0)
  })
})
