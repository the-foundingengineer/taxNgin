import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { returnsApi } from '@/lib/api/returns'

const CURRENT_YEAR = new Date().getFullYear()
const YEARS = [CURRENT_YEAR, CURRENT_YEAR - 1, CURRENT_YEAR - 2]

export function NewReturnPage() {
  const navigate = useNavigate()
  const [year, setYear] = useState(CURRENT_YEAR)
  const [loading, setLoading] = useState(false)

  async function start() {
    setLoading(true)
    try {
      const returnId = await returnsApi.open(year, 'NG')
      navigate(`/returns/${returnId}/file?step=1`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-neutral-900">New return</h1>
          <p className="text-sm text-neutral-500 mt-1">
            Which tax year are you filing for?
          </p>
        </div>

        <div className="space-y-3">
          {YEARS.map(y => (
            <button
              key={y}
              onClick={() => setYear(y)}
              className={`w-full text-left rounded-xl p-4 border-2 transition-all ${
                year === y
                  ? 'border-brand-400 bg-brand-50'
                  : 'border-neutral-200 bg-white hover:border-brand-200'
              }`}
            >
              <div className="font-semibold text-neutral-900">
                {y} year of assessment
              </div>
              <div className="text-sm text-neutral-500 mt-0.5">
                Income earned 1 Jan {y} – 31 Dec {y}
              </div>
            </button>
          ))}
        </div>

        <Card>
          <CardContent className="py-4">
            <p className="text-sm text-neutral-500">
              <strong>Jurisdiction:</strong> Lagos State (LIRS) — the filing authority is
              determined by where you were resident during the tax year.
            </p>
          </CardContent>
        </Card>

        <Button className="w-full" onClick={start} disabled={loading}>
          {loading ? 'Opening return…' : `Start ${year} return`}
        </Button>
      </div>
    </AppShell>
  )
}
