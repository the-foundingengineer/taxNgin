import { create } from 'zustand'
import type { SchemaNode, TaxpayerProfile, SchemaCondition } from '@/types/onboarding'

// ---------------------------------------------------------------------------
// Checkpoint config — injected between major screen transitions
// ---------------------------------------------------------------------------

export interface CheckpointConfig {
    icon: string
    title: string
    subtitle: string
    /** Key used to build personalised "what's coming" pills */
    section: 'income' | 'deductions' | string
}

/** Map: afterScreen → checkpoint to show when that screen is completed */
const CHECKPOINTS: Record<string, CheckpointConfig> = {
    '01': {
        icon: 'verified_user',
        title: 'Identity confirmed!',
        subtitle: "You're verified. Now let's discover every naira you're owed.",
        section: 'income',
    },
    '07': {
        icon: 'payments',
        title: 'Employment income captured!',
        subtitle: "Based on your profile we've prepared the exact deductions that apply to you.",
        section: 'deductions',
    },
}

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

interface OnboardingStore {
    schema: SchemaNode[]
    profile: TaxpayerProfile
    screens: string[]
    currentScreenIndex: number
    /** 0-based index within the current screen's visible questions */
    currentQuestionIndex: number
    direction: 'forward' | 'back'
    /** Non-null while an interstitial checkpoint is showing */
    showingCheckpoint: CheckpointConfig | null

    setSchema: (nodes: SchemaNode[]) => void
    setAnswer: (code: string, value: unknown) => void
    goNextScreen: () => void
    goPrevScreen: () => void
    /** Dismiss the current checkpoint and truly advance to the next screen */
    dismissCheckpoint: () => void
    getVisibleQuestions: () => SchemaNode[]
    canAdvance: () => boolean
    questionProgress: () => { current: number; total: number }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function evalCondition(cond: SchemaCondition, profile: TaxpayerProfile): boolean {
    const fieldVal = profile[cond.field]
    switch (cond.condition) {
        case '==': return fieldVal === cond.value
        case '!=': return fieldVal !== cond.value
        case 'contains': return Array.isArray(fieldVal) && fieldVal.includes(cond.value)
        case 'not_contains': return !Array.isArray(fieldVal) || !fieldVal.includes(cond.value)
        default: return true
    }
}

function computeActiveScreens(schema: SchemaNode[], profile: TaxpayerProfile): string[] {
    const active = new Set<string>()
    for (const q of schema) {
        if (!q.show_if || evalCondition(q.show_if, profile)) {
            active.add(q.screen)
        }
    }
    return Array.from(active).sort()
}

function getScreenQuestions(
    schema: SchemaNode[],
    screenId: string,
    profile: TaxpayerProfile,
): SchemaNode[] {
    return schema
        .filter((q) => q.screen === screenId)
        .filter((q) => !q.show_if || evalCondition(q.show_if, profile))
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useOnboardingStore = create<OnboardingStore>((set, get) => ({
    schema: [],
    profile: {},
    screens: [],
    currentScreenIndex: 0,
    currentQuestionIndex: 0,
    direction: 'forward',
    showingCheckpoint: null,

    setSchema: (nodes) => {
        const screens = computeActiveScreens(nodes, get().profile)
        set({ schema: nodes, screens })
    },

    setAnswer: (code, value) => {
        set((state) => {
            const newProfile = { ...state.profile, [code]: value }
            const newScreens = computeActiveScreens(state.schema, newProfile)

            const currentScreenId = state.screens[state.currentScreenIndex]
            let newScreenIndex = newScreens.indexOf(currentScreenId)
            if (newScreenIndex === -1) {
                newScreenIndex = Math.max(0, Math.min(state.currentScreenIndex, newScreens.length - 1))
            }

            // Clamp question index if visible questions shrink due to branching
            const newScreenId = newScreens[newScreenIndex]
            const newScreenQCount = newScreenId
                ? getScreenQuestions(state.schema, newScreenId, newProfile).length
                : 0
            const newQuestionIndex = Math.min(
                state.currentQuestionIndex,
                Math.max(0, newScreenQCount - 1),
            )

            return {
                profile: newProfile,
                screens: newScreens,
                currentScreenIndex: newScreenIndex,
                currentQuestionIndex: newQuestionIndex,
            }
        })
    },

    goNextScreen: () =>
        set((state) => {
            const screenId = state.screens[state.currentScreenIndex]
            const screenQs = getScreenQuestions(state.schema, screenId, state.profile)

            // Advance within screen first
            if (state.currentQuestionIndex < screenQs.length - 1) {
                return {
                    currentQuestionIndex: state.currentQuestionIndex + 1,
                    direction: 'forward',
                }
            }

            // Last question on screen — check for checkpoint before advancing
            const checkpoint = CHECKPOINTS[screenId] ?? null
            if (checkpoint) {
                return { showingCheckpoint: checkpoint }
            }

            return {
                currentScreenIndex: Math.min(state.currentScreenIndex + 1, state.screens.length),
                currentQuestionIndex: 0,
                direction: 'forward',
                showingCheckpoint: null,
            }
        }),

    dismissCheckpoint: () =>
        set((state) => ({
            showingCheckpoint: null,
            currentScreenIndex: Math.min(state.currentScreenIndex + 1, state.screens.length),
            currentQuestionIndex: 0,
            direction: 'forward',
        })),

    goPrevScreen: () =>
        set((state) => {
            // Back within screen
            if (state.currentQuestionIndex > 0) {
                return {
                    currentQuestionIndex: state.currentQuestionIndex - 1,
                    direction: 'back',
                    showingCheckpoint: null,
                }
            }

            // Back to previous screen's last question
            if (state.currentScreenIndex > 0) {
                const prevScreenIndex = state.currentScreenIndex - 1
                const prevScreenId = state.screens[prevScreenIndex]
                const prevQs = getScreenQuestions(state.schema, prevScreenId, state.profile)
                return {
                    currentScreenIndex: prevScreenIndex,
                    currentQuestionIndex: Math.max(0, prevQs.length - 1),
                    direction: 'back',
                    showingCheckpoint: null,
                }
            }

            return { showingCheckpoint: null }
        }),

    getVisibleQuestions: () => {
        const { schema, screens, currentScreenIndex, currentQuestionIndex, profile } = get()
        const screenId = screens[currentScreenIndex]
        const screenQs = getScreenQuestions(schema, screenId, profile)
        const q = screenQs[currentQuestionIndex]
        return q ? [q] : []
    },

    canAdvance: () => {
        const { profile } = get()
        const questions = get().getVisibleQuestions()
        return questions.every((q) => {
            const isRequired = q.required || (q.required_if && evalCondition(q.required_if, profile))
            if (!isRequired) return true
            const val = profile[q.code]
            if (val === undefined || val === null || val === '') return false
            if (Array.isArray(val) && val.length === 0) return false
            return true
        })
    },

    questionProgress: () => {
        const { schema, screens, currentScreenIndex, currentQuestionIndex, profile } = get()
        const screenId = screens[currentScreenIndex]
        const total = getScreenQuestions(schema, screenId, profile).length
        return { current: currentQuestionIndex + 1, total }
    },
}))
