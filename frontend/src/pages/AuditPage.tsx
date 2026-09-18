import { useEffect, useState } from 'react';
import { Button, Input, Select, Table } from 'antd';
import { Filter, RefreshCw } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { listAudits, type AuditFilters } from '../api/audits';
import { PageHeader } from '../components/common/PageHeader';
import { useScenarioStore } from '../stores/scenarioStore';
import { useSimulationStore } from '../stores/simulationStore';
import type { AuditEvent } from '../types/audit';
import { reportError } from '../utils/errors';
import { formatDateTime } from '../utils/format';

export function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [filters, setFilters] = useState<AuditFilters>({});
  const [loading, setLoading] = useState(false);
  const scenarios = useScenarioStore((state) => state.scenarios);
  const loadScenarios = useScenarioStore((state) => state.load);
  const runs = useSimulationStore((state) => state.runs);
  const loadRuns = useSimulationStore((state) => state.load);
  const load = async (next = filters) => {
    setLoading(true);
    try { setEvents((await listAudits(next)).items); } catch (error) { reportError(error, '审计记录加载失败'); } finally { setLoading(false); }
  };
  useEffect(() => { Promise.all([load(), loadScenarios(), loadRuns()]).catch(reportError); }, []);
  const columns: ColumnsType<AuditEvent> = [
    { title: '时间', dataIndex: 'created_at', width: 180, render: formatDateTime },
    { title: '操作者', dataIndex: 'actor_email', width: 220 },
    { title: '动作', dataIndex: 'action', width: 230, render: (value) => <code>{value}</code> },
    { title: '对象', width: 170, render: (_, row) => `${row.entity_type} #${row.entity_id}` },
    { title: '请求 ID', dataIndex: 'request_id', width: 190, render: (value) => <code title={value}>{value.length > 18 ? `${value.slice(0, 18)}…` : value}</code> },
    { title: '变更摘要', width: 240, render: (_, row) => <span className="audit-diff">{row.before_state === '{}' ? '创建' : '状态或字段变更'} → {row.after_state.length} 字节证据</span> },
  ];
  const apply = () => void load(filters);
  const reset = () => { setFilters({}); void load({}); };
  return (
    <div className="page">
      <PageHeader eyebrow="不可变事件 / 请求追踪" title="审计追踪" meta={<><span>{events.length} 条当前结果</span><span>{scenarios.length} 个方案</span><span>{runs.length} 次推演可关联</span></>} actions={<Button icon={<RefreshCw size={17} />} onClick={() => void load()}>刷新</Button>} />
      <section className="audit-filters" aria-label="审计筛选器">
        <Input aria-label="操作者邮箱" placeholder="操作者邮箱" value={filters.actor} onChange={(event) => setFilters({ ...filters, actor: event.target.value })} />
        <Select aria-label="对象类型" allowClear placeholder="对象类型" value={filters.entity_type} onChange={(value) => setFilters({ ...filters, entity_type: value })} options={[{ value: 'ventilation_node', label: '通风节点' }, { value: 'airway_edge', label: '巷道边' }, { value: 'fan_scenario', label: '风机方案' }, { value: 'simulation_run', label: '推演记录' }]} />
        <Select aria-label="方案状态" allowClear placeholder="方案状态" value={filters.status} onChange={(value) => setFilters({ ...filters, status: value })} options={[{ value: 'draft', label: '草稿' }, { value: 'pending_review', label: '待复核' }, { value: 'approved', label: '已批准' }, { value: 'archived', label: '已归档' }]} />
        <Input aria-label="动作关键字" placeholder="动作关键字" value={filters.action} onChange={(event) => setFilters({ ...filters, action: event.target.value })} />
        <Button type="primary" icon={<Filter size={17} />} onClick={apply}>应用筛选</Button><Button onClick={reset}>清除</Button>
      </section>
      <section className="workspace-section"><div className="section-heading"><div><span className="section-index">01</span><h2>按时间倒序的不可变事件</h2></div></div><Table rowKey="id" columns={columns} dataSource={events} loading={loading} size="small" pagination={{ pageSize: 12 }} scroll={{ x: 1050 }} /></section>
    </div>
  );
}
