import { setCSRFToken } from '$lib/api/client';
import { tenantApi, type TenantApi } from '$lib/api/endpoints';
import type { Me, Membership, Role } from '$lib/api/types';

const TENANT_KEY = 'hw.tenant';

function readTenant(): string | null {
  try {
    return localStorage.getItem(TENANT_KEY);
  } catch {
    return null;
  }
}

function saveTenant(id: string): void {
  try {
    localStorage.setItem(TENANT_KEY, id);
  } catch {
    // Storage unavailable: the choice lasts for this page only.
  }
}

/**
 * The signed-in user and the active tenant (HW-TEN-5). The active tenant is
 * kept per browser; the engine checks the membership on every request.
 */
class Session {
  me = $state<Me | null>(null);
  tenantId = $state<string | null>(null);

  readonly membership: Membership | null = $derived(
    this.me?.memberships.find((m) => m.tenant.id === this.tenantId) ?? null,
  );
  readonly role: Role | null = $derived(this.membership?.role ?? null);
  readonly canEdit = $derived(this.role === 'owner' || this.role === 'editor');
  readonly canManage = $derived(this.role === 'owner');

  /** The active tenant's routes; only valid once signed in with a tenant. */
  get api(): TenantApi {
    if (!this.tenantId) throw new Error('No tenant selected.');
    return tenantApi(this.tenantId);
  }

  set(me: Me, preferTenant?: string): void {
    this.me = me;
    setCSRFToken(me.csrfToken);
    const ids = me.memberships.map((m) => m.tenant.id);
    const wanted = [preferTenant, this.tenantId, readTenant()].find((id) => id && ids.includes(id));
    this.tenantId = wanted ?? ids[0] ?? null;
    if (this.tenantId) saveTenant(this.tenantId);
  }

  selectTenant(id: string): void {
    if (!this.me?.memberships.some((m) => m.tenant.id === id)) return;
    this.tenantId = id;
    saveTenant(id);
  }

  clear(): void {
    this.me = null;
    this.tenantId = null;
    setCSRFToken('');
  }
}

export const session = new Session();
