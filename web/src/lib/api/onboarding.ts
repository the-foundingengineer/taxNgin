import { api } from './client'
import type { SchemaNode } from '@/types/onboarding'

export function fetchOnboardingSchema(): Promise<SchemaNode[]> {
    return api.get<SchemaNode[]>('/onboarding/schema')
}
