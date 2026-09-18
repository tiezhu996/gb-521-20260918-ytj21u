import type { VentilationNode } from './node';

export type DoorState = 'open' | 'closed' | 'regulating';

export interface AirwayEdge {
  id: number;
  code: string;
  from_node_id: number;
  to_node_id: number;
  resistance_ns2m8: number;
  area_m2: number;
  max_velocity_ms: number;
  door_state: DoorState;
  enabled: boolean;
  critical_path: boolean;
  version: number;
  from_node?: VentilationNode;
  to_node?: VentilationNode;
  created_at: string;
  updated_at: string;
}

export type CreateEdgeInput = Omit<AirwayEdge, 'id' | 'version' | 'from_node' | 'to_node' | 'created_at' | 'updated_at'>;
export type UpdateEdgeInput = Pick<AirwayEdge, 'resistance_ns2m8' | 'area_m2' | 'max_velocity_ms' | 'door_state' | 'enabled' | 'critical_path' | 'version'>;
