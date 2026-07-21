import { routes } from './routes';
import type { ClaimDetail, ClaimList, Health, RecallResponse } from './types';

// The principal is a seam: today it rides on a header, later an OIDC/SSO
// middleware replaces the source without any handler changing. It is settable
// so the console can act as a different principal without a rebuild.
let currentPrincipal = 'local';

export function setPrincipal(principal: string): void {
  currentPrincipal = principal;
}

export function getPrincipal(): string {
  return currentPrincipal;
}

export class ApiError extends Error {
  readonly status: number;
  readonly body: string;

  constructor(status: number, statusText: string, body: string) {
    super(`API ${status} ${statusText}`.trim());
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
  }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      Accept: 'application/json',
      'X-CRED-Principal': currentPrincipal,
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res.status, res.statusText, body);
  }

  return (await res.json()) as T;
}

export type StatusFilter = 'live' | 'expired' | 'all';

export interface ClaimsParams {
  status?: StatusFilter;
  limit?: number;
  offset?: number;
}

export function getHealth(): Promise<Health> {
  return request<Health>(routes.health());
}

export function getClaims(params: ClaimsParams = {}): Promise<ClaimList> {
  const query = new URLSearchParams();
  if (params.status && params.status !== 'all') {
    query.set('status', params.status);
  }
  if (params.limit !== undefined) query.set('limit', String(params.limit));
  if (params.offset !== undefined) query.set('offset', String(params.offset));

  const qs = query.toString();
  return request<ClaimList>(qs ? `${routes.claims()}?${qs}` : routes.claims());
}

export function getClaim(id: string): Promise<ClaimDetail> {
  return request<ClaimDetail>(routes.claim(id));
}

export interface RecallParams {
  q: string;
  limit?: number;
  depth?: number;
  budget?: number;
}

export function getRecall(params: RecallParams): Promise<RecallResponse> {
  const query = new URLSearchParams();
  query.set('q', params.q);
  if (params.limit !== undefined) query.set('limit', String(params.limit));
  if (params.depth !== undefined) query.set('depth', String(params.depth));
  if (params.budget !== undefined) query.set('budget', String(params.budget));

  return request<RecallResponse>(`${routes.recall()}?${query.toString()}`);
}
