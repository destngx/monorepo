import { request } from './client';
import type {
  AICheck,
  CaptureLink,
  AuditEntry,
  CreatedInvite,
  CreatedUploadToken,
  DocumentDetail,
  DocumentKind,
  EngineSettingsInput,
  EngineSettingsView,
  Filed,
  FileInput,
  Finding,
  FindingInput,
  GroupFileInput,
  InboxGroup,
  InboxItem,
  ItemDetail,
  Metric,
  Observation,
  ObservationInput,
  Page,
  Quad,
  RecordStatus,
  SearchResult,
  Invite,
  InviteInfo,
  Me,
  Member,
  Person,
  PersonInput,
  Provider,
  Role,
  SetupStatus,
  Tenant,
  User,
  UploadResponse,
  UploadToken,
  Visit,
  VisitFilter,
  VisitInput,
} from './types';

type Fetch = typeof fetch;

export interface SetupInput {
  email: string;
  displayName: string;
  password: string;
  tenantName: string;
  setupCode: string;
}

export interface SignUpInput {
  email: string;
  displayName: string;
  password: string;
}

/** Account routes, outside any tenant. */
export const auth = {
  setupStatus: (fetch?: Fetch) => request<SetupStatus>('/setup', { fetch }),
  setup: (body: SetupInput) => request<Me>('/setup', { method: 'POST', body }),
  login: (email: string, password: string) => request<Me>('/auth/login', { method: 'POST', body: { email, password } }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  me: (fetch?: Fetch) => request<Me>('/auth/me', { fetch }),
  changePassword: (current: string, next: string) =>
    request<Me>('/auth/password', { method: 'POST', body: { current, next } }),
  createTenant: (name: string) => request<Tenant>('/tenants', { method: 'POST', body: { name } }),
  invite: (token: string, fetch?: Fetch) => request<InviteInfo>(`/invites/${encodeURIComponent(token)}`, { fetch }),
  acceptInvite: (token: string) => request<Tenant>(`/invites/${encodeURIComponent(token)}/accept`, { method: 'POST' }),
  signUp: (token: string, body: SignUpInput) =>
    request<Me>(`/invites/${encodeURIComponent(token)}/signup`, { method: 'POST', body }),
};

/** Engine settings and admins; the engine lets only engine admins in (HW-OPS-9). */
export const engine = {
  settings: (fetch?: Fetch) => request<EngineSettingsView>('/engine/settings', { fetch }),
  save: (body: EngineSettingsInput) => request<EngineSettingsView>('/engine/settings', { method: 'PUT', body }),
  checkAI: () => request<AICheck>('/engine/settings/check-ai', { method: 'POST' }),
  users: (fetch?: Fetch) => request<User[]>('/engine/users', { fetch }),
  setAdmin: (userId: string, admin: boolean) =>
    request<User>(`/engine/users/${encodeURIComponent(userId)}/admin`, { method: 'PUT', body: { admin } }),
};

/** Routes of one tenant. */
export function tenantApi(tenantId: string) {
  const t = `/tenants/${tenantId}`;
  return {
    rename: (name: string) => request<void>(t, { method: 'PATCH', body: { name } }),
    members: (fetch?: Fetch) => request<Member[]>(`${t}/members`, { fetch }),
    setRole: (userId: string, role: Role) =>
      request<void>(`${t}/members/${userId}`, { method: 'PATCH', body: { role } }),
    removeMember: (userId: string) => request<void>(`${t}/members/${userId}`, { method: 'DELETE' }),
    invites: (fetch?: Fetch) => request<Invite[]>(`${t}/invites`, { fetch }),
    createInvite: (role: Role) => request<CreatedInvite>(`${t}/invites`, { method: 'POST', body: { role } }),
    revokeInvite: (id: string) => request<void>(`${t}/invites/${id}`, { method: 'DELETE' }),
    uploadTokens: (fetch?: Fetch) => request<UploadToken[]>(`${t}/upload-tokens`, { fetch }),
    createUploadToken: (name: string) =>
      request<CreatedUploadToken>(`${t}/upload-tokens`, { method: 'POST', body: { name } }),
    revokeUploadToken: (id: string) => request<void>(`${t}/upload-tokens/${id}`, { method: 'DELETE' }),
    /** A QR link that opens the phone camera page for this inbox (HW-IMP-9). */
    capture: (origin: string) => request<CaptureLink>(`${t}/capture`, { method: 'POST', body: { origin } }),

    people: (fetch?: Fetch) => request<Person[]>(`${t}/people`, { fetch }),
    person: (id: string, fetch?: Fetch) => request<Person>(`${t}/people/${id}`, { fetch }),
    createPerson: (body: PersonInput) => request<Person>(`${t}/people`, { method: 'POST', body }),
    updatePerson: (id: string, body: PersonInput) => request<Person>(`${t}/people/${id}`, { method: 'PATCH', body }),
    deletePerson: (id: string) => request<void>(`${t}/people/${id}`, { method: 'DELETE' }),
    visits: (personId: string, filter: VisitFilter = {}, fetch?: Fetch) =>
      request<Visit[]>(`${t}/people/${personId}/visits`, { query: { ...filter }, fetch }),
    providers: (fetch?: Fetch) => request<Provider[]>(`${t}/providers`, { fetch }),
    createVisit: (body: VisitInput) => request<Visit>(`${t}/visits`, { method: 'POST', body }),
    visit: (id: string, fetch?: Fetch) => request<Visit>(`${t}/visits/${id}`, { fetch }),
    updateVisit: (id: string, body: VisitInput) => request<Visit>(`${t}/visits/${id}`, { method: 'PATCH', body }),
    deleteVisit: (id: string) => request<void>(`${t}/visits/${id}`, { method: 'DELETE' }),
    deleteDocument: (id: string) => request<void>(`${t}/documents/${id}`, { method: 'DELETE' }),

    inbox: (fetch?: Fetch) => request<InboxItem[]>(`${t}/inbox`, { fetch }),
    upload: (files: File[], options: { oneItem?: boolean; note?: string; personId?: string } = {}) => {
      const form = new FormData();
      for (const file of files) form.append('files', file, file.name);
      if (options.oneItem) form.set('oneItem', 'true');
      if (options.note) form.set('note', options.note);
      if (options.personId) form.set('personId', options.personId);
      return request<UploadResponse>(`${t}/inbox`, { method: 'POST', body: form });
    },
    file: (itemId: string, body: FileInput) => request<Filed>(`${t}/inbox/${itemId}/file`, { method: 'POST', body }),
    discard: (itemId: string) => request<void>(`${t}/inbox/${itemId}/discard`, { method: 'POST' }),
    item: (itemId: string, fetch?: Fetch) => request<ItemDetail>(`${t}/inbox/${itemId}`, { fetch }),
    sendReading: (itemId: string) => request<void>(`${t}/inbox/${itemId}/reading/send`, { method: 'POST' }),
    skipReading: (itemId: string) => request<void>(`${t}/inbox/${itemId}/reading/skip`, { method: 'POST' }),
    retryReading: (itemId: string) => request<void>(`${t}/inbox/${itemId}/reading/retry`, { method: 'POST' }),
    setPageBoxes: (itemId: string, pageId: string, boxes: Quad[]) =>
      request<Page>(`${t}/inbox/${itemId}/pages/${pageId}/boxes`, { method: 'PUT', body: { boxes } }),
    groups: (fetch?: Fetch) => request<InboxGroup[]>(`${t}/inbox/groups`, { fetch }),
    fileGroup: (body: GroupFileInput) => request<Visit>(`${t}/inbox/groups/file`, { method: 'POST', body }),

    document: (id: string, fetch?: Fetch) => request<DocumentDetail>(`${t}/documents/${id}`, { fetch }),
    updateDocument: (id: string, body: { kind: DocumentKind; title: string }) =>
      request<DocumentDetail>(`${t}/documents/${id}`, { method: 'PATCH', body }),
    addObservation: (documentId: string, body: ObservationInput) =>
      request<Observation>(`${t}/documents/${documentId}/observations`, { method: 'POST', body }),
    addFinding: (documentId: string, body: FindingInput) =>
      request<Finding>(`${t}/documents/${documentId}/findings`, { method: 'POST', body }),
    updateObservation: (id: string, body: ObservationInput & { status?: RecordStatus }) =>
      request<Observation>(`${t}/observations/${id}`, { method: 'PATCH', body }),
    linkMetric: (id: string, code: string) =>
      request<Observation>(`${t}/observations/${id}/metric`, { method: 'PUT', body: { code } }),
    deleteObservation: (id: string) => request<void>(`${t}/observations/${id}`, { method: 'DELETE' }),
    observationHistory: (id: string) => request<AuditEntry[]>(`${t}/observations/${id}/history`),
    updateFinding: (id: string, body: FindingInput & { status?: RecordStatus }) =>
      request<Finding>(`${t}/findings/${id}`, { method: 'PATCH', body }),
    deleteFinding: (id: string) => request<void>(`${t}/findings/${id}`, { method: 'DELETE' }),
    findingHistory: (id: string) => request<AuditEntry[]>(`${t}/findings/${id}/history`),
    search: (q: string, fetch?: Fetch) => request<SearchResult>(`${t}/search`, { query: { q }, fetch }),
    metrics: (fetch?: Fetch) => request<Metric[]>(`${t}/metrics`, { fetch }),
  };
}

export type TenantApi = ReturnType<typeof tenantApi>;
