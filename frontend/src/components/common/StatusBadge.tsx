import { AlertTriangle, Archive, CheckCircle2, CircleDot, Clock3, FilePenLine, LoaderCircle, XCircle } from 'lucide-react';
import type { ScenarioStatus } from '../../types/scenario';
import type { SimulationStatus } from '../../types/simulation';

type Status = ScenarioStatus | SimulationStatus | 'active' | 'inactive' | 'blocked' | 'confirmed';

const statusMap: Record<Status, { label: string; tone: string; Icon: typeof CircleDot }> = {
  draft: { label: '草稿', tone: 'neutral', Icon: FilePenLine },
  pending_review: { label: '待复核', tone: 'warning', Icon: Clock3 },
  approved: { label: '已批准', tone: 'success', Icon: CheckCircle2 },
  archived: { label: '已归档', tone: 'neutral', Icon: Archive },
  queued: { label: '排队中', tone: 'neutral', Icon: Clock3 },
  running: { label: '计算中', tone: 'info', Icon: LoaderCircle },
  converged: { label: '已收敛', tone: 'success', Icon: CheckCircle2 },
  not_converged: { label: '未收敛', tone: 'warning', Icon: AlertTriangle },
  invalid_input: { label: '输入无效', tone: 'danger', Icon: XCircle },
  failed: { label: '执行失败', tone: 'danger', Icon: XCircle },
  active: { label: '启用', tone: 'success', Icon: CheckCircle2 },
  inactive: { label: '停用', tone: 'neutral', Icon: CircleDot },
  blocked: { label: '阻断', tone: 'danger', Icon: XCircle },
  confirmed: { label: '已人工确认', tone: 'success', Icon: CheckCircle2 },
};

export function StatusBadge({ status }: { status: Status }) {
  const config = statusMap[status] ?? { label: status, tone: 'neutral', Icon: CircleDot };
  return <span className={`status-badge status-${config.tone}`}><config.Icon size={14} aria-hidden="true" />{config.label}</span>;
}
