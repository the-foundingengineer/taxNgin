// Onboarding schema types — mirrors the Go onboarding.SchemaNode struct.

export interface SchemaCondition {
    condition: '==' | '!=' | 'contains' | 'not_contains'
    field: string
    value: unknown
}

export interface SchemaOption {
    id: string
    label: string
    icon?: string
    sub?: string
}

export interface SchemaNode {
    id: string
    screen: string
    code: string
    type: 'text' | 'date' | 'currency' | 'boolean' | 'single_select' | 'multi_select' | 'file' | 'number' | 'sentiment_picker'
    label: string
    hint?: string
    required?: boolean
    group?: string
    required_if?: SchemaCondition
    show_if?: SchemaCondition
    options?: SchemaOption[]
}

/** Flat key-value bag holding all user answers keyed by question `code`. */
export type TaxpayerProfile = Record<string, unknown>
