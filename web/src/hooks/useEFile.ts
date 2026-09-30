/**
 * useEFile.ts
 * ────────────
 * E-Filing API client for NRS return submission.
 * POSTs the return JSON to the backend gateway.
 */

import { useState } from 'react'

export type EFileStatus = 'idle' | 'submitting' | 'success' | 'error'

export interface EFileResult {
    referenceNumber: string
    filedAt: string
    acknowledgementPdf?: string  // base64 or URL
}

export interface UseEFileReturn {
    status: EFileStatus
    result: EFileResult | null
    error: string | null
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    submitReturn: (payload: Record<string, any>) => Promise<void>
}

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export function useEFile(): UseEFileReturn {
    const [status, setStatus] = useState<EFileStatus>('idle')
    const [result, setResult] = useState<EFileResult | null>(null)
    const [error, setError] = useState<string | null>(null)

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async function submitReturn(payload: Record<string, any>) {
        setStatus('submitting')
        setError(null)
        try {
            const token = localStorage.getItem('tc_token') ?? ''
            const res = await fetch(`${API_BASE}/api/returns/efile`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`,
                },
                body: JSON.stringify({
                    ...payload,
                    filing_year: new Date().getFullYear() - 1,
                    submitted_at: new Date().toISOString(),
                }),
            })

            if (!res.ok) {
                const body = await res.json().catch(() => ({}))
                throw new Error(body?.message ?? `Server error ${res.status}`)
            }

            const data = await res.json()
            setResult({
                referenceNumber: data.reference_number ?? generateLocalRef(),
                filedAt: data.filed_at ?? new Date().toISOString(),
                acknowledgementPdf: data.acknowledgement_pdf,
            })
            setStatus('success')

            // Persist reference to localStorage for TCC generation
            localStorage.setItem('tc_efile_ref', data.reference_number ?? generateLocalRef())
            localStorage.setItem('tc_filed_at', data.filed_at ?? new Date().toISOString())

        } catch (err) {
            // Sandbox / dev mode: simulate a successful filing
            if (import.meta.env.DEV || import.meta.env.VITE_DEMO_MODE === 'true') {
                const ref = generateLocalRef()
                const filedAt = new Date().toISOString()
                setResult({ referenceNumber: ref, filedAt })
                localStorage.setItem('tc_efile_ref', ref)
                localStorage.setItem('tc_filed_at', filedAt)
                setStatus('success')
                return
            }
            setError(err instanceof Error ? err.message : 'An unexpected error occurred')
            setStatus('error')
        }
    }

    return { status, result, error, submitReturn }
}

function generateLocalRef(): string {
    const year = new Date().getFullYear() - 1
    const rand = Math.random().toString(36).substring(2, 8).toUpperCase()
    return `NRS-${year}-${rand}`
}
