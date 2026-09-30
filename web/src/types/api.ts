// API types mirror the Go backend structs.
//
// WIRE FORMAT NOTE: The Go Projection struct has no json tags, so field names
// are PascalCase on the wire (ReturnID, YearOfAssessment, etc.).
// Request bodies use snake_case because the handler anonymous request structs
// DO have json tags. This asymmetry is by design in the Go code.

export type ReturnState =
  | 'DRAFT'
  | 'VALIDATED'
  | 'SIGNED'
  | 'PACK_ISSUED'
  | 'SELF_ATTESTED'
  | 'ACK_UPLOADED'
  | 'ASSESSED'
  | 'PAID'
  | 'CLEARED'
  | 'OBJECTED'
  | 'SUPERSEDED'
  | 'ABANDONED'

export type IncomeCategory =
  | 'EMPLOYMENT'
  | 'BUSINESS'
  | 'PROFESSIONAL'
  | 'RENT_RECEIVED'
  | 'DIVIDEND'
  | 'INTEREST'
  | 'ROYALTY'
  | 'PENSION_RECEIVED'
  | 'GRATUITY'
  | 'DIGITAL_ASSET_GAIN'
  | 'CAPITAL_GAIN'
  | 'OTHER'
  | 'FOREIGN'

export type ReliefCode =
  | 'RENT_RELIEF'
  | 'PENSION'
  | 'NHF'
  | 'NHIS'
  | 'LIFE_ASSURANCE'
  | 'MEDICAL'
  | 'HOUSING_LOAN_INTEREST'
  | 'BUSINESS_EXPENSES'

export type CreditCode = 'PAYE_REMITTED' | 'FOREIGN_TAX_CREDIT' | 'WHT_SUFFERED'

export type BillingTier = 'FREE' | 'STANDARD' | 'PREMIUM'

// ---- Income ----

export interface IncomeLine {
  ID: string
  Category: IncomeCategory
  AmountKobo: number       // int64 — integer, never float
  SourceRef: string
  Label: string
  DeclaredBy: 'TAXPAYER' | 'EMPLOYER' | 'SYSTEM' | 'STAFF'
  EvidenceIDs: string[] | null
}

// ---- Reliefs ----

export interface ReliefLine {
  Code: ReliefCode
  AmountKobo: number
  EvidenceIDs: string[] | null
}

// ---- Payroll proposals ----

export interface PayrollProposal {
  RecordID: string
  EmployerID: string
  GrossKobo: number
  PayeKobo: number
  Status: 'PROPOSED' | 'ACCEPTED' | 'REJECTED'
}

// ---- Assessment ----

export interface AssessmentSnapshot {
  FactsHash: string
  RuleSetVersion: string
  RuleSetDigest: string
  RuleSetPublished: boolean
  ChargeableIncomeKobo: number
  TaxPayableKobo: number
  TotalCreditsKobo: number
  BalanceKobo: number        // positive = owed, negative = refund
  ComputedAt: string         // ISO 8601
}

// ---- Return projection (PascalCase — no json tags on Go struct) ----

export interface ReturnProjection {
  ReturnID: string
  TaxpayerID: string
  YearOfAssessment: number
  BasisPeriodStart: string
  BasisPeriodEnd: string
  Jurisdiction: string
  State: ReturnState
  Version: number
  Income: Record<string, IncomeLine>
  Reliefs: Record<string, ReliefLine>
  Bases: Record<string, number>
  ProposedPayroll: Record<string, PayrollProposal>
  Assessment: AssessmentSnapshot | null
  AssessmentStale: boolean
  RuleSetVersion: string
  RuleSetPublished: boolean
  PortalMapVersion: string
  BlockingErrors: string[]
  Warnings: string[]
  DeclaredByTaxpayer: boolean
  SignedAt: string | null
  PackIssuedAt: string | null
  AttestedAt: string | null
  AckUploadedAt: string | null
  AssessmentServedAt: string | null
  ObjectionDeadline: string | null
  PaidAt: string | null
  AccommodationType: string
  OwnershipType: string
  LandlordName: string
  LandlordAddress: string
}

// ---- Pack (PascalCase — no json tags on Go pack structs) ----

export interface PackField {
  PortalLabel: string
  Value: string
  Note: string
  Upload: string   // non-empty = "Bring: <document>"
  Missing: boolean
}

export interface PackRow {
  Fields: PackField[]
}

export interface PackPage {
  ID: string
  Title: string
  Note: string
  Rows: PackRow[]
}

export interface PreconditionResult {
  ID: string
  Label: string
  Detail: string
  BlocksPack: boolean
  BlocksTCC: boolean
  Met: boolean
}

export interface UnmappedIncomeItem {
  Category: string
  Amount: number
  Note: string
}

export interface Pack {
  ReturnID: string
  YearOfAssessment: number
  PortalMapVersion: string
  Pages: PackPage[]
  UnmappedIncome: UnmappedIncomeItem[]
  Preconditions: PreconditionResult[]
}

// ---- Billing ----

export interface Entitlements {
  MaxYearsOfAssessment: number
  EmployerPreFill: boolean
  TCCTracking: boolean
  MultiYearCatchUp: boolean
  PrioritySupport: boolean
}

export interface EntitlementsResponse {
  tier: BillingTier
  entitlements: Entitlements
}

// ---- API error (from Go writeError) ----

export interface ApiErrorBody {
  error: string
}

// ---- Request shapes (snake_case — handler anonymous structs have json tags) ----

export interface DeclareIncomeRequest {
  id?: string
  category: IncomeCategory
  amount_kobo: number
  source_ref: string
  label: string
}

export interface ClaimReliefRequest {
  code: ReliefCode
  amount_kobo: number
  has_evidence: boolean
}

export interface ValidateRequest {
  nin: string
  tax_id: string
  tax_id_verified: boolean
  nil_declaration: boolean
  duplicate_check_ok: boolean
}

export interface AccommodationRequest {
  type: string
  ownership_type: string
  landlord_name: string
  landlord_address: string
}
