import { Table, Typography } from 'antd';
import { AlertOctagon, AlertTriangle, Info } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import type { RiskEvidence } from '../../types/simulation';
import { formatNumber } from '../../utils/format';

const levelConfig = {
  critical: { label: '严重', Icon: AlertOctagon, className: 'risk-critical' },
  warning: { label: '警告', Icon: AlertTriangle, className: 'risk-warning' },
  info: { label: '提示', Icon: Info, className: 'risk-info' },
};

export function RiskEvidenceTable({ risks, loading = false }: { risks: RiskEvidence[]; loading?: boolean }) {
  const columns: ColumnsType<RiskEvidence> = [
    {
      title: '等级', dataIndex: 'level', width: 105,
      render: (level: RiskEvidence['level']) => {
        const config = levelConfig[level];
        return <span className={`risk-level ${config.className}`}><config.Icon size={15} />{config.label}</span>;
      },
    },
    { title: '规则', dataIndex: 'rule_code', width: 168, render: (value: string) => <code>{value}</code> },
    { title: '对象', width: 150, render: (_, row) => `${row.entity_type} #${row.entity_id || '-'}` },
    { title: '证据', width: 150, render: (_, row) => <strong>{formatNumber(row.evidence, 3)} {row.unit}</strong> },
    { title: '阈值', width: 150, render: (_, row) => `${formatNumber(row.threshold, 3)} ${row.unit}` },
    { title: '判定说明', dataIndex: 'description', width: 260 },
  ];
  return (
    <Table<RiskEvidence>
      rowKey={(row) => `${row.rule_code}-${row.entity_type}-${row.entity_id}`}
      columns={columns}
      dataSource={risks}
      loading={loading}
      size="small"
      pagination={false}
      scroll={{ x: 900 }}
      locale={{ emptyText: <Typography.Text type="secondary">当前结果未触发联锁风险规则</Typography.Text> }}
    />
  );
}
