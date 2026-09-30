/**
 * ProfileTransfer
 * ---------------
 * Handles export (download JSON) and import (drag-and-drop / file picker)
 * of the tc_profile. Equivalent of TurboTax's "Year-over-year data transfer".
 */

import { useState, useRef, useCallback } from 'react'
import { Upload, X, CheckCircle2, AlertCircle } from 'lucide-react'
import { useOnboardingStore } from '@/stores/onboardingStore'

interface ProfileImportModalProps {
    onClose: () => void
}

type ImportStatus = 'idle' | 'dragging' | 'parsing' | 'success' | 'error'

export function downloadProfile() {
    const raw = localStorage.getItem('tc_profile')
    if (!raw) return
    const profile = JSON.parse(raw)
    const year = new Date().getFullYear()
    const firstName = profile.first_name ?? 'profile'
    const blob = new Blob([JSON.stringify(profile, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `taxcheck_${firstName}_${year}.json`
    a.click()
    URL.revokeObjectURL(url)
}

export function ProfileImportModal({ onClose }: ProfileImportModalProps) {
    const [status, setStatus] = useState<ImportStatus>('idle')
    const [errorMsg, setErrorMsg] = useState('')
    const [importedName, setImportedName] = useState('')
    const inputRef = useRef<HTMLInputElement>(null)
    const setAnswer = useOnboardingStore(s => s.setAnswer)

    const processFile = useCallback((file: File) => {
        if (!file.name.endsWith('.json')) {
            setStatus('error')
            setErrorMsg('Only .json profile files exported from Tax Check are supported.')
            return
        }
        setStatus('parsing')
        const reader = new FileReader()
        reader.onload = (e) => {
            try {
                const profile = JSON.parse(e.target?.result as string)
                if (typeof profile !== 'object' || !profile) throw new Error('Invalid profile format.')

                // Hydrate the zustand store and localStorage
                Object.entries(profile).forEach(([k, v]) => setAnswer(k, v))
                localStorage.setItem('tc_profile', JSON.stringify(profile))

                setImportedName(profile.first_name ?? 'your')
                setStatus('success')
            } catch (err) {
                setStatus('error')
                setErrorMsg(err instanceof Error ? err.message : 'Could not read profile file.')
            }
        }
        reader.readAsText(file)
    }, [setAnswer])

    const onDrop = useCallback((e: React.DragEvent) => {
        e.preventDefault()
        setStatus('idle')
        const file = e.dataTransfer.files[0]
        if (file) processFile(file)
    }, [processFile])

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="w-full max-w-md bg-white rounded-2xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-5 border-b border-neutral-100">
                    <div>
                        <h2 className="font-bold text-[17px] text-neutral-900">Import last year's profile</h2>
                        <p className="text-sm text-neutral-500">Pre-fill your return from a previous Tax Check export</p>
                    </div>
                    <button onClick={onClose} className="w-8 h-8 rounded-full flex items-center justify-center text-neutral-400 hover:bg-neutral-100">
                        <X className="w-4 h-4" />
                    </button>
                </div>

                {/* Body */}
                <div className="p-6 space-y-4">
                    {status === 'success' ? (
                        <div className="text-center py-6 space-y-4">
                            <div className="w-16 h-16 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center mx-auto">
                                <CheckCircle2 className="w-8 h-8" />
                            </div>
                            <div>
                                <h3 className="font-bold text-lg text-neutral-900">{importedName}'s profile loaded!</h3>
                                <p className="text-sm text-neutral-500 mt-1">All fields have been pre-filled. Review and update anything that changed this year.</p>
                            </div>
                            <button onClick={onClose} className="w-full h-11 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-xl transition-colors">
                                Start reviewing
                            </button>
                        </div>
                    ) : (
                        <>
                            {/* Drop zone */}
                            <div
                                onDragOver={e => { e.preventDefault(); setStatus('dragging') }}
                                onDragLeave={() => setStatus('idle')}
                                onDrop={onDrop}
                                onClick={() => inputRef.current?.click()}
                                className={`border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-all ${status === 'dragging'
                                    ? 'border-blue-500 bg-blue-50'
                                    : status === 'error'
                                        ? 'border-rose-300 bg-rose-50'
                                        : 'border-neutral-300 hover:border-blue-400 hover:bg-neutral-50'
                                    }`}
                            >
                                <input ref={inputRef} type="file" accept=".json" className="hidden" onChange={e => e.target.files?.[0] && processFile(e.target.files[0])} />

                                {status === 'error' ? (
                                    <>
                                        <AlertCircle className="w-10 h-10 text-rose-400 mx-auto mb-3" />
                                        <p className="font-semibold text-rose-700">{errorMsg}</p>
                                        <p className="text-sm text-rose-500 mt-1">Try again with a valid file.</p>
                                    </>
                                ) : (
                                    <>
                                        <Upload className={`w-10 h-10 mx-auto mb-3 ${status === 'dragging' ? 'text-blue-500' : 'text-neutral-400'}`} />
                                        <p className="font-semibold text-neutral-700">Drop your profile JSON here</p>
                                        <p className="text-sm text-neutral-400 mt-1">or click to browse for <code className="text-xs bg-neutral-100 px-1 py-0.5 rounded">taxcheck_*.json</code></p>
                                    </>
                                )}
                            </div>

                            <div className="flex items-center gap-3 text-xs text-neutral-400">
                                <div className="flex-1 h-px bg-neutral-200" />
                                <span>Your data never leaves your device</span>
                                <div className="flex-1 h-px bg-neutral-200" />
                            </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}
