import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { ValidationPanel } from '../validation/ValidationPanel'

describe('ValidationPanel', () => {
  it('shows human-readable text for E001, not the raw code as the message', () => {
    render(<ValidationPanel blockingCodes={['E001']} warningCodes={[]} />)
    expect(screen.getByText(/Tax ID must be present and verified/)).toBeInTheDocument()
  })

  it('labels E001 as a blocking issue', () => {
    render(<ValidationPanel blockingCodes={['E001']} warningCodes={[]} />)
    expect(screen.getByText(/Blocking issue/)).toBeInTheDocument()
  })

  it('shows human-readable text for W101', () => {
    render(<ValidationPanel blockingCodes={[]} warningCodes={['W101']} />)
    expect(screen.getByText(/without uploading supporting evidence/)).toBeInTheDocument()
  })

  it('shows success message when no issues', () => {
    render(<ValidationPanel blockingCodes={[]} warningCodes={[]} />)
    expect(screen.getByText(/Validation passed/)).toBeInTheDocument()
  })

  it('renders all codes in order', () => {
    render(<ValidationPanel blockingCodes={['E001', 'E002']} warningCodes={['W101']} />)
    const issues = screen.getAllByRole('alert')
    expect(issues.length).toBeGreaterThanOrEqual(3)
  })
})
