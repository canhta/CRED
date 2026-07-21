import { useQuery } from '@tanstack/react-query';
import { getClaim, getClaims, getHealth } from './client';
import type { ClaimsParams } from './client';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
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
