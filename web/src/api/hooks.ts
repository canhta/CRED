import { useQuery } from '@tanstack/react-query';
import { getClaim, getClaims, getHealth, getRecall } from './client';
import type { ClaimsParams, RecallParams } from './client';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
};

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: getHealth,
  });
}

export function useClaims(params: ClaimsParams = {}) {
  return useQuery({
    queryKey: queryKeys.claims(params),
    queryFn: () => getClaims(params),
  });
}

export function useClaim(id: string) {
  return useQuery({
    queryKey: queryKeys.claim(id),
    queryFn: () => getClaim(id),
    enabled: id.length > 0,
  });
}

// Recall fires on submit, not on every keystroke: the caller gates it with
// `enabled` so an empty or unsubmitted query never hits the retrieval engine.
export function useRecall(params: RecallParams, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.recall(params),
    queryFn: () => getRecall(params),
    enabled,
  });
}
