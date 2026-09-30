import { api } from './client'
import type { BillingTier, EntitlementsResponse } from '@/types/api'

export interface UpgradeResponse {
  authorization_url: string
  reference: string
}

export const billingApi = {
  getEntitlements: () =>
    api.get<EntitlementsResponse>('/entitlements'),

  initiateUpgrade: (tier: BillingTier, callbackUrl: string) =>
    api.post<UpgradeResponse>('/billing/upgrade', {
      tier,
      callback_url: callbackUrl,
    }),
}
