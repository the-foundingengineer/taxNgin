/**
 * usePayment.ts
 * ──────────────
 * Payment session hook — initiates a Remita RRR or Paystack checkout
 * for settling a tax liability from a filed return.
 */

import { useState } from 'react'
import { paymentApi } from '@/lib/api/payment'

export type PaymentProvider = 'remita' | 'paystack'
export type PaymentStatus = 'idle' | 'initiating' | 'redirecting' | 'success' | 'error'

export interface PaymentSession {
    provider: PaymentProvider
    rrr?: string          // Remita Retrieval Reference
    checkoutUrl?: string  // Paystack or Remita checkout URL
    amount: number
}

export interface UsePaymentReturn {
    status: PaymentStatus
    session: PaymentSession | null
    error: string | null
    initiatePayment: (amount: number, provider: PaymentProvider) => Promise<void>
    markPaid: () => void
}

export function usePayment(): UsePaymentReturn {
    const [status, setStatus] = useState<PaymentStatus>('idle')
    const [session, setSession] = useState<PaymentSession | null>(null)
    const [error, setError] = useState<string | null>(null)

    async function initiatePayment(amount: number, provider: PaymentProvider) {
        setStatus('initiating')
        setError(null)
        const orderId = localStorage.getItem('tc_efile_ref') ?? `TX-${Date.now()}`

        try {
            if (provider === 'remita') {
                const data = await paymentApi.remitaInit(amount, orderId)
                setSession({ provider: 'remita', rrr: data.rrr, amount })
                setStatus('redirecting')
            } else {
                // Paystack is for service subscription upgrades, not tax payments.
                // Tax payment via card/USSD is a future addition once LIRS confirms
                // Paystack as an accepted channel.
                setSession({ provider: 'paystack', amount })
                setStatus('redirecting')
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Payment service unavailable. Please try again.')
            setStatus('error')
        }
    }

    function markPaid() {
        localStorage.setItem('tc_payment_status', 'paid')
        localStorage.setItem('tc_payment_amount', String(session?.amount ?? 0))
        setStatus('success')
    }

    return { status, session, error, initiatePayment, markPaid }
}
