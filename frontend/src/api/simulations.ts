import { request, requestPage } from './client';
import type { SimulationRun } from '../types/simulation';

export const listSimulations = () => requestPage<SimulationRun>('/api/v1/simulations?page_size=100');
export const startSimulation = (scenario_id: number) => request<SimulationRun>('/api/v1/simulations', { method: 'POST', body: JSON.stringify({ scenario_id }) });
export const confirmRisks = (id: number, note: string) => request<SimulationRun>(`/api/v1/simulations/${id}/confirm-risks`, { method: 'POST', body: JSON.stringify({ note }) });
export const getSimulation = (id: number) => request<SimulationRun>(`/api/v1/simulations/${id}`);
