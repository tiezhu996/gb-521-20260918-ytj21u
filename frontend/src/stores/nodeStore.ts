import { create } from 'zustand';
import * as nodeApi from '../api/nodes';
import type { CreateNodeInput, NetworkValidation, UpdateNodeInput, VentilationNode } from '../types/node';

interface NodeState {
  nodes: VentilationNode[];
  validation: NetworkValidation | null;
  loading: boolean;
  load(): Promise<void>;
  create(input: CreateNodeInput): Promise<void>;
  update(id: number, input: UpdateNodeInput): Promise<void>;
  validate(): Promise<void>;
}

export const useNodeStore = create<NodeState>((set) => ({
  nodes: [], validation: null, loading: false,
  async load() {
    set({ loading: true });
    try { set({ nodes: (await nodeApi.listNodes()).items }); } finally { set({ loading: false }); }
  },
  async create(input) {
    await nodeApi.createNode(input);
    set({ nodes: (await nodeApi.listNodes()).items });
  },
  async update(id, input) {
    await nodeApi.updateNode(id, input);
    set({ nodes: (await nodeApi.listNodes()).items });
  },
  async validate() { set({ validation: await nodeApi.validateNetwork() }); },
}));
