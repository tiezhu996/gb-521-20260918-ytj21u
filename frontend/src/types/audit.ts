export interface AuditEvent {
  id: number;
  request_id: string;
  actor_id: number;
  actor_email: string;
  action: string;
  entity_type: string;
  entity_id: number;
  before_state: string;
  after_state: string;
  metadata: string;
  created_at: string;
}
