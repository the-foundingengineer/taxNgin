import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MoneyDisplay } from '../shared/MoneyDisplay'

describe('MoneyDisplay', () => {
  it('renders ₦ formatted amount from kobo', () => {
    render(<MoneyDisplay kobo={480_000_000} />)
    expect(screen.getByText('₦4,800,000')).toBeInTheDocument()
  })

  it('never renders raw kobo integer', () => {
    render(<MoneyDisplay kobo={480_000_000} />)
    expect(screen.queryByText('480000000')).not.toBeInTheDocument()
  })

  it('shows + prefix when signed and positive', () => {
    render(<MoneyDisplay kobo={100_000_000} signed />)
    expect(screen.getByText('+₦1,000,000')).toBeInTheDocument()
  })

  it('handles negative (refund) amount', () => {
    render(<MoneyDisplay kobo={-50_000_000} />)
    expect(screen.getByText('-₦500,000')).toBeInTheDocument()
  })

  it('uses mono font class when mono prop is set', () => {
    const { container } = render(<MoneyDisplay kobo={100} mono />)
    expect(container.firstChild).toHaveClass('font-mono')
  })
})
