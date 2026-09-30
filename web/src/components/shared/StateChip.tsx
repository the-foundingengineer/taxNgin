import { Badge } from '@/components/ui/badge'
import type { ReturnState } from '@/types/api'

const STATE_LABELS: Record<ReturnState, string> = {
  DRAFT:          'Draft',
  VALIDATED:      'Validated',
  SIGNED:         'Signed',
  PACK_ISSUED:    'Pack ready',
  SELF_ATTESTED:  'Filed',
  ACK_UPLOADED:   'Confirmed',
  ASSESSED:       'Assessed',
  PAID:           'Paid',
  CLEARED:        'Cleared',
  OBJECTED:       'Objected',
  SUPERSEDED:     'Superseded',
  ABANDONED:      'Abandoned',
}

type BadgeVariant = 'default' | 'secondary' | 'success' | 'warning' | 'danger' | 'outline'

const STATE_VARIANT: Record<ReturnState, BadgeVariant> = {
  DRAFT:          'secondary',
  VALIDATED:      'default',
  SIGNED:         'default',
  PACK_ISSUED:    'success',
  SELF_ATTESTED:  'success',
  ACK_UPLOADED:   'success',
  ASSESSED:       'warning',
  PAID:           'success',
  CLEARED:        'success',
  OBJECTED:       'warning',
  SUPERSEDED:     'outline',
  ABANDONED:      'outline',
}

export function StateChip({ state }: { state: ReturnState }) {
  return (
    <Badge variant={STATE_VARIANT[state]}>
      {STATE_LABELS[state] ?? state}
    </Badge>
  )
}
