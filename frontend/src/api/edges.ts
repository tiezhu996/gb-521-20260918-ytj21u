import { request, requestPage } from './client';
import type { AirwayEdge, CreateEdgeInput, UpdateEdgeInput } from '../types/edge';

export const listEdges = () => requestPage<AirwayEdge>('/api/v1/edges?page_size=100');
export const createEdge = (input: CreateEdgeInput) => request<AirwayEdge>('/api/v1/edges', { method: 'POST', body: JSON.stringify(input) });
export const updateEdge = (id: number, input: UpdateEdgeInput) => request<AirwayEdge>(`/api/v1/edges/${id}`, { method: 'PUT', body: JSON.stringify(input) });
