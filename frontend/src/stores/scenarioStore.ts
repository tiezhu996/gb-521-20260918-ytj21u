import { create } from 'zustand';
import * as scenarioApi from '../api/scenarios';
import type { CreateScenarioInput, FanScenario, ScenarioStatus } from '../types/scenario';

interface ScenarioState {
  scenarios: FanScenario[];
  loading: boolean;
  load(): Promise<void>;
  create(input: CreateScenarioInput): Promise<void>;
  transition(id: number, status: ScenarioStatus, version: number, reason?: string): Promise<void>;
}

export const useScenarioStore = create<ScenarioState>((set) => ({
  scenarios: [], loading: false,
  async load() {
    set({ loading: true });
    try { set({ scenarios: (await scenarioApi.listScenarios()).items }); } finally { set({ loading: false }); }
  },
  async create(input) {
    await scenarioApi.createScenario(input);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
  },
  async transition(id, status, version, reason = '') {
    await scenarioApi.transitionScenario(id, status, version, reason);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
  },
}));
