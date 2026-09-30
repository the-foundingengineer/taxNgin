import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { WizardShell } from '@/components/wizard/WizardShell'
import { PackViewer } from '@/components/pack/PackViewer'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { returnsApi } from '@/lib/api/returns'
import { returnKey } from '@/hooks/useReturn'
import type { ReturnProjection, Pack } from '@/types/api'

interface Step7PackProps {
  returnId: string
  proj: ReturnProjection
}

export function Step7Pack({ returnId, proj }: Step7PackProps) {
  const qc = useQueryClient()
  const [ackRef, setAckRef] = useState('')

  const packIssued = proj.PackIssuedAt !== null
  const filed = proj.AttestedAt !== null
  const ackUploaded = proj.AckUploadedAt !== null

  const { data: pack, isLoading: packLoading } = useQuery<Pack>({
    queryKey: ['pack', returnId],
    queryFn: () => returnsApi.getPack(returnId),
    enabled: packIssued,
  })

  const issuePack = useMutation({
    mutationFn: () => returnsApi.issuePack(returnId),
    onSuccess: (p) => {
      qc.setQueryData(returnKey(returnId), (prev: ReturnProjection | undefined) =>
        prev ? { ...prev, PackIssuedAt: new Date().toISOString(), State: 'PACK_ISSUED' } : prev,
      )
      qc.setQueryData(['pack', returnId], p)
    },
  })

  const attest = useMutation({
    mutationFn: () => returnsApi.attest(returnId),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  const uploadAck = useMutation({
    mutationFn: () => returnsApi.uploadAck(returnId, ackRef),
    onSuccess: (p) => qc.setQueryData(returnKey(returnId), p),
  })

  if (ackUploaded) {
    return (
      <WizardShell step={7} returnId={returnId} hideNext>
        <div className="space-y-4">
          <Card>
            <CardContent className="py-12 flex flex-col items-center gap-4 text-center">
              <div className="h-16 w-16 rounded-full bg-success-light flex items-center justify-center">
                <CheckCircle2 className="h-8 w-8 text-success" />
              </div>
              <h2 className="text-2xl font-bold text-neutral-900">Return filed!</h2>
              <p className="text-neutral-500 text-sm max-w-xs">
                Your {proj.YearOfAssessment} LIRS personal income tax return is complete.
                Keep your acknowledgement receipt safe.
              </p>
            </CardContent>
          </Card>
        </div>
      </WizardShell>
    )
  }

  return (
    <WizardShell step={7} returnId={returnId} hideNext>
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">Filing pack</h2>
          <p className="text-neutral-500 text-sm mt-1">
            Use these instructions to file on the LIRS eTax portal.
          </p>
        </div>

        {/* Phase A — Generate */}
        {!packIssued && (
          <Card>
            <CardContent className="py-8 flex flex-col items-center gap-4">
              <p className="text-sm text-neutral-500 text-center max-w-xs">
                Generate your personalised filing guide. Each field shows exactly what to type on the portal.
              </p>
              <Button onClick={() => issuePack.mutate()} disabled={issuePack.isPending}>
                {issuePack.isPending ? <><LoadingSpinner className="h-4 w-4 mr-2" />Generating…</> : 'Generate filing pack'}
              </Button>
            </CardContent>
          </Card>
        )}

        {/* Phase B — Review pack */}
        {packIssued && packLoading && (
          <Card>
            <CardContent className="py-8 flex items-center justify-center gap-3">
              <LoadingSpinner />
              <span className="text-sm text-neutral-500">Loading pack…</span>
            </CardContent>
          </Card>
        )}

        {pack && <PackViewer pack={pack} />}

        {/* Phase C — Attest and ACK */}
        {packIssued && !filed && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Step 2 — File on eTax portal</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-neutral-500 mb-4">
                Go to <strong>etax.lirs.net</strong>, log in, and enter the values above. When done, come back here.
              </p>
              <Button className="w-full" onClick={() => attest.mutate()} disabled={attest.isPending}>
                {attest.isPending ? 'Recording…' : 'I have filed on the LIRS eTax portal'}
              </Button>
            </CardContent>
          </Card>
        )}

        {filed && !ackUploaded && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Step 3 — Upload acknowledgement</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-neutral-500">
                Enter the acknowledgement reference from the LIRS portal receipt.
              </p>
              <div className="space-y-2">
                <Label>Acknowledgement reference</Label>
                <Input
                  value={ackRef}
                  onChange={(e) => setAckRef(e.target.value)}
                  placeholder="e.g. LIRS-2026-ABC123"
                />
              </div>
              <Button
                className="w-full"
                onClick={() => uploadAck.mutate()}
                disabled={!ackRef || uploadAck.isPending}
              >
                {uploadAck.isPending ? 'Saving…' : 'Save acknowledgement'}
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </WizardShell>
  )
}
