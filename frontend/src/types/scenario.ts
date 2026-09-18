export type ScenarioStatus = 'draft' | 'pending_review' | 'approved' | 'archived';

export interface FanCurvePoint {
  flow_m3s: number;
  pressure_pa: number;
}

export interface FanScenario {
  id: number;
  name: string;
  description: string;
  fan_curve_json: FanCurvePoint[];
  operating_mode: 'normal' | 'reduced' | 'emergency_test';
  scenario_status: ScenarioStatus;
  solver_tolerance: number;
  max_iterations: number;
  version: number;
  created_by: number;
  approved_by?: number;
  reject_reason: string;
  // 批准时绑定的通风网络版本号与内容指纹；仅已批准方案有值。
  network_revision?: number;
  network_snapshot_hash?: string;
  // 网络变化导致批准被系统撤销时的原因，重新批准后清空。
  invalidation_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateScenarioInput {
  name: string;
  description: string;
  fan_curve: FanCurvePoint[];
  operating_mode: FanScenario['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
}
