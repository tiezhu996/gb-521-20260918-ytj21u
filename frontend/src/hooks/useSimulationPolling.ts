import { useEffect } from 'react';
import { useSimulationStore } from '../stores/simulationStore';

export function useSimulationPolling(active = true, intervalMs = 5000) {
  const load = useSimulationStore((state) => state.load);
  useEffect(() => {
    if (!active) return undefined;
    void load();
    const timer = window.setInterval(() => void load(), intervalMs);
    return () => window.clearInterval(timer);
  }, [active, intervalMs, load]);
}
