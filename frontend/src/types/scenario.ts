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
