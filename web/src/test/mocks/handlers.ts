import { http, HttpResponse } from 'msw'
import type { ReturnProjection, Pack } from '@/types/api'

const BASE = '/v1'

const mockProjection: ReturnProjection = {
  ReturnID: 'ret-test',
  TaxpayerID: 'acct-test',
  YearOfAssessment: 2026,
  BasisPeriodStart: '2026-01-01T00:00:00Z',
  BasisPeriodEnd:   '2026-12-31T00:00:00Z',
  Jurisdiction: 'NG',
  State: 'DRAFT',
  Version: 1,
  Income: {},
  Reliefs: {},
  Bases: {},
  ProposedPayroll: {},
  Assessment: null,
  AssessmentStale: false,
  RuleSetVersion: '2026-1.0',
  RuleSetPublished: false,
  PortalMapVersion: '',
  BlockingErrors: [],
  Warnings: [],
  DeclaredByTaxpayer: true,
  SignedAt: null,
  PackIssuedAt: null,
  AttestedAt: null,
  AckUploadedAt: null,
  AssessmentServedAt: null,
  ObjectionDeadline: null,
  PaidAt: null,
  AccommodationType: '',
  OwnershipType: '',
  LandlordName: '',
  LandlordAddress: '',
}

export const handlers = [
  // Auth
  http.post(`${BASE}/accounts`, () => HttpResponse.json({ account_id: 'acct-test' }, { status: 201 })),
  http.post(`${BASE}/sessions`, () => HttpResponse.json({ token: 'sess-test-token' })),
  http.delete(`${BASE}/sessions`, () => new HttpResponse(null, { status: 204 })),

  // Returns
  http.get(`${BASE}/returns`, () =>
    HttpResponse.json({ return_ids: ['ret-test'] }),
  ),
  http.post(`${BASE}/returns`, () =>
    HttpResponse.json({ return_id: 'ret-test' }, { status: 201 }),
  ),
  http.get(`${BASE}/returns/ret-test`, () =>
    HttpResponse.json(mockProjection),
  ),
  http.post(`${BASE}/returns/ret-test/income`, async ({ request }) => {
    const body = await request.json() as { category: string; amount_kobo: number }
    const updated: ReturnProjection = {
      ...mockProjection,
      Version: 2,
      Income: {
        'inc-1': {
          ID: 'inc-1',
          Category: body.category as ReturnProjection['Income'][string]['Category'],
          AmountKobo: body.amount_kobo,
          SourceRef: '',
          Label: '',
          DeclaredBy: 'TAXPAYER',
          EvidenceIDs: null,
        },
      },
    }
    return HttpResponse.json(updated, { status: 201 })
  }),
  http.post(`${BASE}/returns/ret-test/assess`, () =>
    HttpResponse.json({
      ...mockProjection,
      State: 'DRAFT',
      Version: 3,
      Assessment: {
        FactsHash: 'abc',
        RuleSetVersion: '2026-1.0',
        RuleSetDigest: 'def',
        RuleSetPublished: false,
        ChargeableIncomeKobo: 492_000_000,
        TaxPayableKobo: 86_400_000,
        TotalCreditsKobo: 0,
        BalanceKobo: 86_400_000,
        ComputedAt: '2027-01-15T09:00:00Z',
      },
      AssessmentStale: false,
    }),
  ),
  http.post(`${BASE}/returns/ret-test/validate`, () =>
    HttpResponse.json({ ...mockProjection, State: 'VALIDATED', Version: 4, BlockingErrors: [], Warnings: [] }),
  ),
  http.post(`${BASE}/returns/ret-test/sign`, () =>
    HttpResponse.json({ ...mockProjection, State: 'SIGNED', Version: 5, SignedAt: '2027-01-15T10:00:00Z' }),
  ),
  http.post(`${BASE}/returns/ret-test/pack`, () => {
    const pack: Pack = {
      ReturnID: 'ret-test',
      YearOfAssessment: 2026,
      PortalMapVersion: '1.0',
      Pages: [{
        ID: 'P1',
        Title: 'Personal data',
        Note: '',
        Rows: [{
          Fields: [{
            PortalLabel: 'First name',
            Value: 'Test',
            Note: '',
            Upload: '',
            Missing: false,
          }],
        }],
      }],
      UnmappedIncome: [],
      Preconditions: [],
    }
    return HttpResponse.json(pack, { status: 201 })
  }),
  http.get(`${BASE}/returns/ret-test/pack`, () =>
    HttpResponse.json({
      ReturnID: 'ret-test',
      YearOfAssessment: 2026,
      PortalMapVersion: '1.0',
      Pages: [],
      UnmappedIncome: [],
      Preconditions: [],
    } satisfies Pack),
  ),

  // Billing
  http.get(`${BASE}/entitlements`, () =>
    HttpResponse.json({ tier: 'FREE', entitlements: { MaxYearsOfAssessment: 1, EmployerPreFill: false, TCCTracking: false, MultiYearCatchUp: false, PrioritySupport: false } }),
  ),
]
