import { api } from './client'
import type {
  ReturnProjection,
  Pack,
  DeclareIncomeRequest,
  ClaimReliefRequest,
  ValidateRequest,
  AccommodationRequest,
} from '@/types/api'

const RETURNS_KEY = 'tx_return_ids'

// Local storage helpers for return ID list (until GET /v1/returns is the primary source)
export function saveReturnId(id: string): void {
  const ids = loadReturnIds()
  if (!ids.includes(id)) {
    localStorage.setItem(RETURNS_KEY, JSON.stringify([...ids, id]))
  }
}

export function loadReturnIds(): string[] {
  try {
    return JSON.parse(localStorage.getItem(RETURNS_KEY) ?? '[]') as string[]
  } catch {
    return []
  }
}

export const returnsApi = {
  list: () =>
    api.get<{ return_ids: string[] }>('/returns'),

  open: async (yearOfAssessment: number, jurisdiction: string): Promise<string> => {
    const res = await api.post<{ return_id: string }>('/returns', {
      year_of_assessment: yearOfAssessment,
      jurisdiction,
    })
    saveReturnId(res.return_id)
    return res.return_id
  },

  get: (id: string) =>
    api.get<ReturnProjection>(`/returns/${id}`),

  // Income
  declareIncome: (id: string, req: DeclareIncomeRequest) =>
    api.post<ReturnProjection>(`/returns/${id}/income`, req),

  removeIncome: (id: string, incomeId: string) =>
    api.delete<ReturnProjection>(`/returns/${id}/income/${incomeId}`),

  // Reliefs
  claimRelief: (id: string, req: ClaimReliefRequest) =>
    api.post<ReturnProjection>(`/returns/${id}/reliefs`, req),

  removeRelief: (id: string, code: string) =>
    api.delete<ReturnProjection>(`/returns/${id}/reliefs/${code}`),

  // Accommodation
  declareAccommodation: (id: string, req: AccommodationRequest) =>
    api.post<ReturnProjection>(`/returns/${id}/accommodation`, req),

  // Payroll proposals
  acceptPayroll: (id: string, recordId: string, opts: { income_id?: string; category?: string; label?: string }) =>
    api.post<ReturnProjection>(`/returns/${id}/payroll/${recordId}/accept`, opts),

  rejectPayroll: (id: string, recordId: string) =>
    api.post<ReturnProjection>(`/returns/${id}/payroll/${recordId}/reject`, {}),

  // CSV ingestion — sends raw bytes with text/csv content type
  ingestCSV: (id: string, file: File) =>
    api.postRaw<{ proposals_created: number }>(`/returns/${id}/ingest/csv`, file, 'text/csv'),

  // Computation
  assess: (id: string) =>
    api.post<ReturnProjection>(`/returns/${id}/assess`),

  // Validation
  validate: (id: string, req: ValidateRequest) =>
    api.post<ReturnProjection>(`/returns/${id}/validate`, req),

  // Signing
  sign: (id: string) =>
    api.post<ReturnProjection>(`/returns/${id}/sign`),

  // Pack
  issuePack: (id: string) =>
    api.post<Pack>(`/returns/${id}/pack`),

  getPack: (id: string) =>
    api.get<Pack>(`/returns/${id}/pack`),

  // Post-filing
  attest: (id: string) =>
    api.post<ReturnProjection>(`/returns/${id}/attest`),

  uploadAck: (id: string, reference: string) =>
    api.post<ReturnProjection>(`/returns/${id}/ack`, { reference }),
}
