import { routes } from './routes';
import type {
  AuthResponse,
  ClaimDetail,
  ClaimList,
  CreateInviteResponse,
  Health,
  Invite,
  LoginRequest,
  OrgUsageResponse,
  RecallResponse,
  RegisterRequest,
  TeamMember,
  UsageResponse,
} from './types';

// X-CRED-Principal remains the identity source for the header/bearer-token
// path (automation, testing, MCP) -- it is not used for the browser session
// path added in this feature, where authenticate() resolves the principal
// from a verified session cookie and a client-supplied header cannot
// override it. Settable so a script can act as a different principal
// without a rebuild.
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

  // logout returns 204 with no body; res.json() would throw on an empty
  // response.
  if (res.status === 204) {
    return undefined as T;
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

export function getUsage(): Promise<UsageResponse> {
  return request<UsageResponse>(routes.usage());
}

export interface OrgUsageParams {
  scopes?: number;
}

export function getUsageOrg(params: OrgUsageParams = {}): Promise<OrgUsageResponse> {
  const query = new URLSearchParams();
  if (params.scopes !== undefined) query.set('scopes', String(params.scopes));

  const qs = query.toString();
  return request<OrgUsageResponse>(
    qs ? `${routes.usageOrg()}?${qs}` : routes.usageOrg(),
  );
}

function postJSON<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export function register(params: RegisterRequest): Promise<AuthResponse> {
  return postJSON<AuthResponse>(routes.register(), params);
}

export function login(params: LoginRequest): Promise<AuthResponse> {
  return postJSON<AuthResponse>(routes.login(), params);
}

export function logout(): Promise<void> {
  return request<void>(routes.logout(), { method: 'POST' });
}

export function getTeamMembers(): Promise<TeamMember[]> {
  return request<TeamMember[]>(routes.teamMembers());
}

export function getInvites(): Promise<Invite[]> {
  return request<Invite[]>(routes.teamInvites());
}

export interface CreateInviteParams {
  email: string;
  role: string;
}

export function createInvite(
  params: CreateInviteParams,
): Promise<CreateInviteResponse> {
  return postJSON<CreateInviteResponse>(routes.teamInvites(), params);
}

export function revokeInvite(id: string): Promise<void> {
  return request<void>(routes.teamInvite(id), { method: 'DELETE' });
}
