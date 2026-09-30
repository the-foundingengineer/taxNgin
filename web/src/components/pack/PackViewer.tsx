import { useState } from 'react'
import { Copy, Check, AlertCircle, Upload } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { cn } from '@/lib/utils'
import type { Pack, PackField } from '@/types/api'

interface PackViewerProps {
  pack: Pack
}

export function PackViewer({ pack }: PackViewerProps) {
  const blockers = pack.Preconditions?.filter(p => !p.Met && p.BlocksPack) ?? []

  return (
    <div className="space-y-4">
      {blockers.length > 0 && (
        <Alert variant="warning">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Complete these before filing</AlertTitle>
          <AlertDescription>
            <ul className="mt-2 space-y-1">
              {blockers.map(p => (
                <li key={p.ID} className="flex items-start gap-2">
                  <span className="font-mono text-xs bg-amber-200 text-amber-800 rounded px-1 mt-0.5">{p.ID}</span>
                  <span>{p.Label}: {p.Detail}</span>
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {pack.Pages?.map((page) => (
        <Card key={page.ID}>
          <CardHeader>
            <CardTitle className="text-base">{page.Title}</CardTitle>
            {page.Note && <p className="text-sm text-neutral-500 mt-1">{page.Note}</p>}
          </CardHeader>
          <CardContent className="pt-2 space-y-1">
            {page.Rows?.flatMap((row, ri) =>
              row.Fields?.map((field, fi) => (
                <PackFieldRow key={`${ri}-${fi}`} field={field} />
              )) ?? [],
            )}
          </CardContent>
        </Card>
      ))}

      {pack.UnmappedIncome?.length > 0 && (
        <Alert variant="info">
          <AlertTitle>Income not mapped to portal fields</AlertTitle>
          <AlertDescription>
            The following income types don't have a dedicated portal field. Mention them to a tax practitioner.
            <ul className="mt-2 space-y-1">
              {pack.UnmappedIncome.map(u => (
                <li key={u.Category}>{u.Category} — {u.Note}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}

function PackFieldRow({ field }: { field: PackField }) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(field.Value).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className={cn(
      'flex items-start gap-3 py-3 border-b border-neutral-100 last:border-0',
      field.Missing && 'opacity-50',
    )}>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-neutral-500 uppercase tracking-wider mb-0.5">
          {field.PortalLabel}
        </p>
        {field.Missing ? (
          <p className="text-sm text-neutral-400 italic">Not available</p>
        ) : (
          <p className="text-base font-medium text-neutral-900 break-all">{field.Value}</p>
        )}
        {field.Note && <p className="text-xs text-neutral-400 mt-0.5">{field.Note}</p>}
        {field.Upload && (
          <p className="text-xs text-brand-500 mt-1 flex items-center gap-1">
            <Upload className="h-3 w-3" /> {field.Upload}
          </p>
        )}
      </div>
      {!field.Missing && (
        <button
          onClick={copy}
          className={cn(
            'flex-shrink-0 flex items-center justify-center h-10 w-10 rounded-lg transition-colors',
            copied ? 'bg-success-light text-success' : 'bg-neutral-100 text-neutral-500 hover:bg-brand-50 hover:text-brand-500',
          )}
          aria-label={`Copy ${field.PortalLabel}`}
        >
          {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
        </button>
      )}
    </div>
  )
}

