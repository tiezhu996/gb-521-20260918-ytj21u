import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Modal, Select, Switch, Table, Tabs, message } from 'antd';
import { Edit3, GitPullRequestArrow, Network, Plus, RefreshCw } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader } from '../components/common/PageHeader';
import { StatusBadge } from '../components/common/StatusBadge';
import { TopologyView } from '../components/network/TopologyView';
import { useAuth } from '../hooks/useAuth';
import { useEdgeStore } from '../stores/edgeStore';
import { useNodeStore } from '../stores/nodeStore';
import type { CreateEdgeInput, AirwayEdge, UpdateEdgeInput } from '../types/edge';
import type { CreateNodeInput, UpdateNodeInput, VentilationNode } from '../types/node';
import { reportError } from '../utils/errors';
import { formatNumber } from '../utils/format';

export function NetworkPage() {
  const { hasRole } = useAuth();
  const { nodes, validation, loading: nodesLoading, load: loadNodes, create: createNode, update: updateNode, validate } = useNodeStore();
  const { edges, loading: edgesLoading, load: loadEdges, create: createEdge, update: updateEdge } = useEdgeStore();
  const [nodeOpen, setNodeOpen] = useState(false);
  const [edgeOpen, setEdgeOpen] = useState(false);
  const [editingNode, setEditingNode] = useState<VentilationNode | null>(null);
  const [editingEdge, setEditingEdge] = useState<AirwayEdge | null>(null);
  const [saving, setSaving] = useState(false);
  const [nodeForm] = Form.useForm<CreateNodeInput>();
  const [edgeForm] = Form.useForm<CreateEdgeInput>();
  const canEdit = hasRole('engineer', 'admin');

  const refresh = async () => {
    try { await Promise.all([loadNodes(), loadEdges(), validate()]); } catch (error) { reportError(error, '网络数据加载失败'); }
  };
  useEffect(() => { void refresh(); }, []);

  const nodeColumns: ColumnsType<VentilationNode> = [
    { title: '节点编码', dataIndex: 'code', width: 130, render: (value) => <strong>{value}</strong> },
    { title: '类型', dataIndex: 'node_type', width: 110, render: (value) => ({ intake: '进风口', exhaust: '回风口', workface: '工作面', junction: '网络交点' } as Record<string, string>)[value] },
    { title: '标高', dataIndex: 'elevation_m', width: 110, render: (value) => `${formatNumber(value)} m` },
    { title: '需风量', dataIndex: 'required_airflow_m3s', width: 125, render: (value) => `${formatNumber(value)} m³/s` },
    { title: '边界压力', dataIndex: 'pressure_pa', width: 125, render: (value) => `${formatNumber(value)} Pa` },
    { title: '状态', dataIndex: 'status', width: 110, render: (value) => <StatusBadge status={value} /> },
    ...(canEdit ? [{ title: '操作', width: 96, render: (_: unknown, row: VentilationNode) => <Button size="small" icon={<Edit3 size={14} />} onClick={() => openNodeEditor(row)}>编辑</Button> }] : []),
  ];
  const edgeColumns: ColumnsType<AirwayEdge> = [
    { title: '巷道编码', dataIndex: 'code', width: 130, render: (value) => <strong>{value}</strong> },
    { title: '方向', width: 190, render: (_, row) => `${row.from_node?.code ?? row.from_node_id} → ${row.to_node?.code ?? row.to_node_id}` },
    { title: '阻力', dataIndex: 'resistance_ns2m8', width: 115, render: (value) => formatNumber(value, 3) },
    { title: '面积', dataIndex: 'area_m2', width: 110, render: (value) => `${formatNumber(value)} m²` },
    { title: '风速上限', dataIndex: 'max_velocity_ms', width: 120, render: (value) => `${formatNumber(value)} m/s` },
    { title: '风门', dataIndex: 'door_state', width: 110, render: (value) => ({ open: '开启', closed: '关闭', regulating: '调节' } as Record<string, string>)[value] },
    { title: '关键路径', dataIndex: 'critical_path', width: 100, render: (value) => value ? '是' : '否' },
    ...(canEdit ? [{ title: '操作', width: 96, render: (_: unknown, row: AirwayEdge) => <Button size="small" icon={<Edit3 size={14} />} onClick={() => openEdgeEditor(row)}>编辑</Button> }] : []),
  ];
  const activeOptions = useMemo(() => nodes.filter((node) => node.status === 'active').map((node) => ({ value: node.id, label: `${node.code} · ${node.node_type}` })), [nodes]);

  const openNodeCreator = () => {
    setEditingNode(null);
    nodeForm.resetFields();
    nodeForm.setFieldsValue({ node_type: 'junction', status: 'active', elevation_m: 0, required_airflow_m3s: 0, pressure_pa: 0 });
    setNodeOpen(true);
  };
  const openNodeEditor = (node: VentilationNode) => {
    setEditingNode(node);
    nodeForm.setFieldsValue(node);
    setNodeOpen(true);
  };
  const openEdgeCreator = () => {
    setEditingEdge(null);
    edgeForm.resetFields();
    edgeForm.setFieldsValue({ resistance_ns2m8: 2, area_m2: 6, max_velocity_ms: 8, door_state: 'open', enabled: true, critical_path: false });
    setEdgeOpen(true);
  };
  const openEdgeEditor = (edge: AirwayEdge) => {
    setEditingEdge(edge);
    edgeForm.setFieldsValue(edge);
    setEdgeOpen(true);
  };
  const closeNodeEditor = () => { if (!saving) { setNodeOpen(false); setEditingNode(null); } };
  const closeEdgeEditor = () => { if (!saving) { setEdgeOpen(false); setEditingEdge(null); } };

  const submitNode = async (input: CreateNodeInput) => {
    setSaving(true);
    try {
      if (editingNode) {
        const { node_type, elevation_m, required_airflow_m3s, pressure_pa, status } = input;
        await updateNode(editingNode.id, { node_type, elevation_m, required_airflow_m3s, pressure_pa, status } satisfies UpdateNodeInput);
      } else {
        await createNode(input);
      }
      await validate();
      message.success(editingNode ? '通风节点已更新并写入审计记录' : '通风节点已创建并写入审计记录');
      setNodeOpen(false); setEditingNode(null); nodeForm.resetFields();
    } catch (error) { reportError(error, editingNode ? '节点更新失败' : '节点创建失败'); } finally { setSaving(false); }
  };
  const submitEdge = async (input: CreateEdgeInput) => {
    setSaving(true);
    try {
      if (editingEdge) {
        const { resistance_ns2m8, area_m2, max_velocity_ms, door_state, enabled, critical_path } = input;
        await updateEdge(editingEdge.id, { resistance_ns2m8, area_m2, max_velocity_ms, door_state, enabled, critical_path, version: editingEdge.version } satisfies UpdateEdgeInput);
      } else {
        await createEdge(input);
      }
      await validate();
      message.success(editingEdge ? '巷道参数已更新并重新校验网络' : '巷道连接已创建并重新校验网络');
      setEdgeOpen(false); setEditingEdge(null); edgeForm.resetFields();
    } catch (error) { reportError(error, editingEdge ? '巷道更新失败' : '巷道创建失败'); } finally { setSaving(false); }
  };

  return (
    <div className="page">
      <PageHeader eyebrow="网络模型 / 当前版本" title="通风网络编辑器" meta={<><span>{nodes.length} 个节点</span><span>{edges.length} 条巷道</span><span>{validation?.valid ? '拓扑校验通过' : `${validation?.issues.length ?? 0} 项拓扑问题`}</span></>} actions={<><Button icon={<RefreshCw size={17} />} onClick={() => void refresh()}>刷新</Button>{canEdit && <Button icon={<Plus size={17} />} onClick={openNodeCreator}>新增节点</Button>}{canEdit && <Button type="primary" icon={<GitPullRequestArrow size={17} />} onClick={openEdgeCreator}>新增巷道</Button>}</>} />
      {validation && !validation.valid && <Alert className="section-alert" type="error" showIcon message="网络校验未通过" description={validation.issues.map((issue) => issue.message).join('；')} />}
      {validation?.valid && <Alert className="section-alert" type="success" showIcon message="有向网络边界与工作面可达性校验通过" />}
      <section className="workspace-section topology-section" aria-labelledby="topology-heading">
        <div className="section-heading"><div><span className="section-index">01</span><h2 id="topology-heading">只读拓扑</h2></div><span>实线为启用风路，红线标识关键路径</span></div>
        <TopologyView nodes={nodes} edges={edges} />
      </section>
      <section className="workspace-section" aria-labelledby="network-data-heading">
        <div className="section-heading"><div><span className="section-index">02</span><h2 id="network-data-heading">模型数据</h2></div></div>
        <Tabs items={[
          { key: 'nodes', label: `节点 ${nodes.length}`, children: <Table rowKey="id" columns={nodeColumns} dataSource={nodes} loading={nodesLoading} size="small" pagination={{ pageSize: 8 }} scroll={{ x: 800 }} /> },
          { key: 'edges', label: `巷道 ${edges.length}`, children: <Table rowKey="id" columns={edgeColumns} dataSource={edges} loading={edgesLoading} size="small" pagination={{ pageSize: 8 }} scroll={{ x: 950 }} /> },
        ]} />
      </section>
      <Modal title={editingNode ? `编辑节点 ${editingNode.code}` : '创建通风节点'} open={nodeOpen} onCancel={closeNodeEditor} footer={null} destroyOnClose maskClosable={!saving} closable={!saving}>
        <Form form={nodeForm} layout="vertical" onFinish={submitNode}>
          <Form.Item name="code" label="节点编码" rules={[{ required: true, min: 2, message: '请输入至少 2 个字符的节点编码' }]}><Input placeholder="例如 JCT-18" disabled={Boolean(editingNode)} /></Form.Item>
          <div className="form-grid"><Form.Item name="node_type" label="节点类型" rules={[{ required: true }]}><Select options={[{ value: 'intake', label: '进风口' }, { value: 'exhaust', label: '回风口' }, { value: 'workface', label: '工作面' }, { value: 'junction', label: '网络交点' }]} /></Form.Item><Form.Item name="status" label="状态" rules={[{ required: true }]}><Select options={[{ value: 'active', label: '启用' }, { value: 'inactive', label: '停用' }, { value: 'blocked', label: '阻断' }]} /></Form.Item></div>
          <div className="form-grid"><Form.Item name="elevation_m" label="标高（m）" rules={[{ required: true }]}><InputNumber min={-2000} max={9000} /></Form.Item><Form.Item name="pressure_pa" label="边界压力（Pa）" rules={[{ required: true }]}><InputNumber min={-100000} max={100000} /></Form.Item></div>
          <Form.Item name="required_airflow_m3s" label="工作面最低需风量（m³/s）" rules={[{ required: true }]}><InputNumber min={0} max={1000} /></Form.Item>
          <div className="modal-actions"><Button onClick={closeNodeEditor} disabled={saving}>返回网络</Button><Button type="primary" htmlType="submit" loading={saving}>{editingNode ? '保存节点' : '创建节点'}</Button></div>
        </Form>
      </Modal>
      <Modal title={editingEdge ? `编辑巷道 ${editingEdge.code}` : '创建巷道连接'} open={edgeOpen} onCancel={closeEdgeEditor} footer={null} destroyOnClose maskClosable={!saving} closable={!saving}>
        <Form form={edgeForm} layout="vertical" onFinish={submitEdge}>
          <Form.Item name="code" label="巷道编码" rules={[{ required: true, min: 2, message: '请输入至少 2 个字符的巷道编码' }]}><Input placeholder="例如 AW-205" disabled={Boolean(editingEdge)} /></Form.Item>
          <div className="form-grid"><Form.Item name="from_node_id" label="起点" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={activeOptions} disabled={Boolean(editingEdge)} /></Form.Item><Form.Item name="to_node_id" label="终点" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={activeOptions} disabled={Boolean(editingEdge)} /></Form.Item></div>
          <div className="form-grid"><Form.Item name="resistance_ns2m8" label="阻力（N·s²/m⁸）" rules={[{ required: true }]}><InputNumber min={0.001} /></Form.Item><Form.Item name="area_m2" label="面积（m²）" rules={[{ required: true }]}><InputNumber min={0.1} /></Form.Item></div>
          <div className="form-grid"><Form.Item name="max_velocity_ms" label="风速上限（m/s）" rules={[{ required: true }]}><InputNumber min={0.1} /></Form.Item><Form.Item name="door_state" label="风门状态" rules={[{ required: true }]}><Select options={[{ value: 'open', label: '开启' }, { value: 'regulating', label: '调节' }, { value: 'closed', label: '关闭' }]} /></Form.Item></div>
          <div className="form-grid switches"><Form.Item name="enabled" label="启用巷道" valuePropName="checked"><Switch /></Form.Item><Form.Item name="critical_path" label="关键路径" valuePropName="checked"><Switch /></Form.Item></div>
          <div className="modal-actions"><Button onClick={closeEdgeEditor} disabled={saving}>返回网络</Button><Button type="primary" htmlType="submit" loading={saving}>{editingEdge ? '保存巷道' : '创建巷道'}</Button></div>
        </Form>
      </Modal>
    </div>
  );
}
