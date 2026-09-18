export type NodeType = 'intake' | 'exhaust' | 'workface' | 'junction';
export type NodeStatus = 'active' | 'inactive' | 'blocked';

export interface VentilationNode {
  id: number;
  code: string;
  node_type: NodeType;
  elevation_m: number;
  required_airflow_m3s: number;
  pressure_pa: number;
  status: NodeStatus;
  created_at: string;
  updated_at: string;
}

export type CreateNodeInput = Omit<VentilationNode, 'id' | 'created_at' | 'updated_at'>;
export type UpdateNodeInput = Omit<CreateNodeInput, 'code'>;

export interface NetworkIssue {
  code: string;
  entity_type: string;
  entity_id: number;
  message: string;
  severity: 'warning' | 'critical';
}

export interface NetworkValidation {
  valid: boolean;
  node_count: number;
  edge_count: number;
  intake_count: number;
  exhaust_count: number;
  issues: NetworkIssue[];
}
