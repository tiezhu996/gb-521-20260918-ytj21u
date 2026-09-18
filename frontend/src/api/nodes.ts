import { request, requestPage } from './client';
import type { CreateNodeInput, NetworkValidation, UpdateNodeInput, VentilationNode } from '../types/node';

export const listNodes = () => requestPage<VentilationNode>('/api/v1/nodes?page_size=100');
export const createNode = (input: CreateNodeInput) => request<VentilationNode>('/api/v1/nodes', { method: 'POST', body: JSON.stringify(input) });
export const updateNode = (id: number, input: UpdateNodeInput) => request<VentilationNode>(`/api/v1/nodes/${id}`, { method: 'PUT', body: JSON.stringify(input) });
export const validateNetwork = () => request<NetworkValidation>('/api/v1/network/validate');
