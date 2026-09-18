import { requestPage } from './client';
import type { AuditEvent } from '../types/audit';

export interface AuditFilters {
  actor?: string;
  action?: string;
  entity_type?: string;
  status?: string;
  from?: string;
  to?: string;
}

export const listAudits = (filters: AuditFilters = {}) => {
  const query = new URLSearchParams({ page_size: '100' });
  Object.entries(filters).forEach(([key, value]) => value && query.set(key, value));
  return requestPage<AuditEvent>(`/api/v1/audits?${query}`);
};
