import { create } from 'zustand';
import * as edgeApi from '../api/edges';
import type { AirwayEdge, CreateEdgeInput, UpdateEdgeInput } from '../types/edge';

interface EdgeState {
  edges: AirwayEdge[];
  loading: boolean;
  load(): Promise<void>;
  create(input: CreateEdgeInput): Promise<void>;
  update(id: number, input: UpdateEdgeInput): Promise<void>;
}

export const useEdgeStore = create<EdgeState>((set) => ({
  edges: [], loading: false,
  async load() {
    set({ loading: true });
    try { set({ edges: (await edgeApi.listEdges()).items }); } finally { set({ loading: false }); }
  },
  async create(input) {
    await edgeApi.createEdge(input);
    set({ edges: (await edgeApi.listEdges()).items });
  },
  async update(id, input) {
    await edgeApi.updateEdge(id, input);
    set({ edges: (await edgeApi.listEdges()).items });
  },
}));
