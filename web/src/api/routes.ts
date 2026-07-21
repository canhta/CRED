// Every API URL lives here, once. A mistyped route is a runtime 404, not a tsc
// error; the cost of that is what centralizing the table keeps small.

export const API_BASE = '/api';

export const routes = {
  health: () => `${API_BASE}/health`,
  claims: () => `${API_BASE}/claims`,
  claim: (id: string) => `${API_BASE}/claims/${encodeURIComponent(id)}`,
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
  usageOrg: () => `${API_BASE}/usage/org`,
  register: () => `${API_BASE}/auth/register`,
  login: () => `${API_BASE}/auth/login`,
  logout: () => `${API_BASE}/auth/logout`,
  teamMembers: () => `${API_BASE}/team/members`,
  teamInvites: () => `${API_BASE}/team/invites`,
  teamInvite: (id: string) => `${API_BASE}/team/invites/${encodeURIComponent(id)}`,
} as const;
