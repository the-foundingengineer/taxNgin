import { AlertCircle, AlertTriangle } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

// Human-readable descriptions for validation codes from internal/validate/validate.go
const ERROR_DESCRIPTIONS: Record<string, string> = {
  E001: 'Your Tax ID must be present and verified before filing.',
  E002: 'Your NIN (National Identity Number) is required to file in Lagos.',
  E003: 'The year of assessment does not match the rule set for this jurisdiction.',
  E004: 'The relevant tax authority could not be determined for your jurisdiction.',
  E005: 'You need at least one income line, or must declare a nil return.',
  E006: 'Negative amounts are not allowed — please check your income or relief figures.',
  E007: 'One of your claimed reliefs is not in the approved list for this year.',
  E008: 'The rule set for this tax year has not yet been published. Filing is temporarily blocked.',
  E009: 'You must sign the return declaration before it can be submitted.',
  E010: 'A return for this year already exists. Duplicate filings are blocked.',
}

const WARNING_DESCRIPTIONS: Record<string, string> = {
  W101: 'You claimed rent relief without uploading supporting evidence.',
  W102: 'Your declared employment income differs from your employer\'s payroll record.',
  W103: 'Your pension claim exceeds 8% of your employment income.',
  W104: 'You claimed rent relief, but your accommodation is recorded as owner-occupied.',
  W105: 'Your income this year is more than 50% lower than last year.',
  W106: 'You have fewer than 3 years of filings — this may block a TCC application.',
  W107: 'You declared foreign income but did not claim a foreign tax credit.',
  W108: 'Exchange activity was imported but digital asset gains are declared as zero.',
  W109: 'This return was filed after the March 31 deadline. A late-filing penalty applies.',
  W110: 'You declared business income but claimed no business expenses.',
}

interface ValidationPanelProps {
  blockingCodes: string[]
  warningCodes: string[]
}

export function ValidationPanel({ blockingCodes, warningCodes }: ValidationPanelProps) {
  if (blockingCodes.length === 0 && warningCodes.length === 0) {
    return (
      <Alert variant="success">
        <AlertTitle>Validation passed</AlertTitle>
        <AlertDescription>
          No issues found. You can proceed to sign your return.
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-3">
      {blockingCodes.map((code) => (
        <Alert key={code} variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>{code} — Blocking issue</AlertTitle>
          <AlertDescription>
            {ERROR_DESCRIPTIONS[code] ?? `Validation error (${code}). Please contact support.`}
          </AlertDescription>
        </Alert>
      ))}
      {warningCodes.map((code) => (
        <Alert key={code} variant="warning">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>{code} — Advisory</AlertTitle>
          <AlertDescription>
            {WARNING_DESCRIPTIONS[code] ?? `Advisory notice (${code}).`}
          </AlertDescription>
        </Alert>
      ))}
    </div>
  )
}
