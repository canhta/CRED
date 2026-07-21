import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getClaim,
  getClaims,
  getHealth,
  getRecall,
  getUsage,
  getUsageOrg,
  login,
  logout,
  register,
} from './client';
import type { ClaimsParams, OrgUsageParams, RecallParams } from './client';
import type { LoginRequest, RegisterRequest } from './types';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
  usage: ['usage'] as const,
  usageOrg: (params: OrgUsageParams) => ['usage', 'org', params] as const,
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

export function useUsage() {
  return useQuery({
    queryKey: queryKeys.usage,
    queryFn: getUsage,
  });
}

// The org-wide report is admin-only; the caller passes enabled so a
// member's browser never issues a request that would 403 -- the same
// enabled-gating shape useRecall already uses for its own precondition.
export function useUsageOrg(params: OrgUsageParams, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.usageOrg(params),
    queryFn: () => getUsageOrg(params),
    enabled,
  });
}

export function useRegister() {
  return useMutation({ mutationFn: (data: RegisterRequest) => register(data) });
}

export function useLogin() {
  return useMutation({ mutationFn: (data: LoginRequest) => login(data) });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.health });
    },
  });
}
