import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Select, Table, message } from 'antd';
import { Play, RefreshCw } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader } from '../components/common/PageHeader';
import { RiskEvidenceTable } from '../components/common/RiskEvidenceTable';
import { StatusBadge } from '../components/common/StatusBadge';
import { ResidualChart } from '../components/simulation/ResidualChart';
import { useAuth } from '../hooks/useAuth';
import { useSimulationPolling } from '../hooks/useSimulationPolling';
import { useScenarioStore } from '../stores/scenarioStore';
import { useSimulationStore } from '../stores/simulationStore';
import type { SimulationRun } from '../types/simulation';
import { reportError } from '../utils/errors';
import { formatDateTime, formatNumber } from '../utils/format';

export function SimulationsPage() {
  const { hasRole } = useAuth();
  const { scenarios, load: loadScenarios } = useScenarioStore();
  const { runs, selected, loading, load, select, start } = useSimulationStore();
  const [scenarioId, setScenarioId] = useState<number>();
  const [starting, setStarting] = useState(false);
  useSimulationPolling(true, 8000);
  useEffect(() => { loadScenarios().catch((error) => reportError(error, '方案列表加载失败')); }, [loadScenarios]);
  const approved = useMemo(() => scenarios.filter((item) => item.scenario_status === 'approved'), [scenarios]);
  useEffect(() => { if (!scenarioId && approved[0]) setScenarioId(approved[0].id); }, [approved, scenarioId]);
  const launch = async () => {
    if (!scenarioId) return;
    setStarting(true);
    try { const run = await start(scenarioId); message.success(`推演 #${run.id} 已完成并保存结果快照`); } catch (error) { reportError(error, '推演未能启动'); } finally { setStarting(false); }
  };
  const columns: ColumnsType<SimulationRun> = [
    { title: '运行', dataIndex: 'id', width: 90, render: (value) => <strong>#{value}</strong> },
    { title: '方案', width: 220, render: (_, row) => row.scenario?.name ?? `方案 #${row.scenario_id}` },
    { title: '状态', dataIndex: 'run_status', width: 135, render: (value) => <StatusBadge status={value} /> },
    { title: '迭代', dataIndex: 'iteration_count', width: 90 },
    { title: '最终残差', dataIndex: 'residual', width: 130, render: (value) => <code>{formatNumber(value, 6)}</code> },
    { title: '风险', width: 95, render: (_, row) => row.risk_flags_json?.length ?? 0 },
    { title: '开始时间', dataIndex: 'started_at', width: 180, render: formatDateTime },
    { title: '', width: 90, render: (_, row) => <Button size="small" onClick={() => select(row.id).catch((error) => reportError(error))}>查看</Button> },
  ];
  return (
    <div className="page">
      <PageHeader eyebrow="确定性求解器 / air-balance-v1" title="离线推演工作台" meta={<><span>{runs.length} 条历史运行</span><span>{approved.length} 个已批准方案</span><span>历史结果只读保留</span></>} actions={<Button icon={<RefreshCw size={17} />} onClick={() => load().catch(reportError)}>刷新结果</Button>} />
      <Alert className="section-alert" type="warning" showIcon message="推演结果是离线决策证据，不是可直接执行的安全指令" />
      <section className="run-launcher" aria-labelledby="launch-heading"><div><span className="section-index">01</span><h2 id="launch-heading">选择已批准方案</h2></div><Select aria-label="已批准方案" value={scenarioId} onChange={setScenarioId} options={approved.map((item) => ({ value: item.id, label: `${item.name} · v${item.version}` }))} placeholder="当前没有可运行方案" /><Button type="primary" icon={<Play size={17} />} disabled={!scenarioId || !hasRole('engineer', 'admin')} loading={starting} onClick={() => void launch()}>开始离线推演</Button></section>
      <section className="workspace-section"><div className="section-heading"><div><span className="section-index">02</span><h2>运行历史</h2></div></div><Table rowKey="id" columns={columns} dataSource={runs} loading={loading} size="small" pagination={{ pageSize: 8 }} scroll={{ x: 980 }} onRow={(row) => ({ onClick: () => select(row.id).catch(reportError) })} rowClassName={(row) => selected?.id === row.id ? 'selected-row' : ''} /></section>
      {selected && <section className="workspace-section result-detail" aria-labelledby="result-heading"><div className="section-heading"><div><span className="section-index">03</span><h2 id="result-heading">运行 #{selected.id} 计算证据</h2></div><StatusBadge status={selected.run_status} /></div><div className="result-metrics"><div><span>迭代轮次</span><strong>{selected.iteration_count}</strong></div><div><span>最终残差</span><strong>{formatNumber(selected.residual, 6)}</strong></div><div><span>风险证据</span><strong>{selected.risk_flags_json?.length ?? 0}</strong></div><div><span>算法版本</span><strong>{selected.algorithm_version}</strong></div></div><ResidualChart values={selected.residuals_json ?? []} /><h3>联锁规则证据</h3><RiskEvidenceTable risks={selected.risk_flags_json ?? []} /></section>}
    </div>
  );
}
