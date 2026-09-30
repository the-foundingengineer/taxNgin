import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { returnsApi } from '@/lib/api/returns'
import type { ReturnProjection } from '@/types/api'

export function returnKey(id: string) {
  return ['return', id] as const
}

export function useReturn(id: string) {
  return useQuery({
    queryKey: returnKey(id),
    queryFn: () => returnsApi.get(id),
    enabled: id !== '',
    staleTime: 0,
  })
}

/** Update the cached projection after any mutation that returns ReturnProjection. */
export function useSetReturn() {
  const qc = useQueryClient()
  return (id: string, proj: ReturnProjection) => {
    qc.setQueryData(returnKey(id), proj)
  }
}

export function useAssess(returnId: string) {
  const setReturn = useSetReturn()
  return useMutation({
    mutationFn: () => returnsApi.assess(returnId),
    onSuccess: (proj) => setReturn(returnId, proj),
  })
}

export function useSign(returnId: string) {
  const setReturn = useSetReturn()
  return useMutation({
    mutationFn: () => returnsApi.sign(returnId),
    onSuccess: (proj) => setReturn(returnId, proj),
  })
}
