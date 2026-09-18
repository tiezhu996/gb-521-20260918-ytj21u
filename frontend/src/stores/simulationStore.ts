import { create } from 'zustand';
import * as simulationApi from '../api/simulations';
import type { SimulationRun } from '../types/simulation';

interface SimulationState {
  runs: SimulationRun[];
  selected: SimulationRun | null;
  loading: boolean;
  load(): Promise<void>;
  select(id: number): Promise<void>;
  start(scenarioId: number): Promise<SimulationRun>;
  confirm(id: number, note: string): Promise<void>;
}

export const useSimulationStore = create<SimulationState>((set) => ({
  runs: [], selected: null, loading: false,
  async load() {
    set({ loading: true });
    try { set({ runs: (await simulationApi.listSimulations()).items }); } finally { set({ loading: false }); }
  },
  async select(id) { set({ selected: await simulationApi.getSimulation(id) }); },
  async start(scenarioId) {
    const run = await simulationApi.startSimulation(scenarioId);
    set({ selected: run, runs: (await simulationApi.listSimulations()).items });
    return run;
  },
  async confirm(id, note) {
    const selected = await simulationApi.confirmRisks(id, note);
    set({ selected, runs: (await simulationApi.listSimulations()).items });
  },
}));
