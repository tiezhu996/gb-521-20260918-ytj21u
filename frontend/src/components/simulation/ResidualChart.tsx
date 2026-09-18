import { formatNumber } from '../../utils/format';

export function ResidualChart({ values }: { values: number[] }) {
  if (!values.length) return <div className="chart-empty">暂无残差序列</div>;
  const width = 720;
  const height = 180;
  const max = Math.max(...values, 0.001);
  const points = values.map((value, index) => {
    const x = 16 + (index / Math.max(values.length - 1, 1)) * (width - 32);
    const y = height - 18 - (value / max) * (height - 36);
    return `${x},${y}`;
  }).join(' ');
  return (
    <figure className="residual-chart">
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`残差曲线，共 ${values.length} 轮，最终残差 ${formatNumber(values.at(-1) ?? 0, 6)}`}>
        <line x1="16" y1={height - 18} x2={width - 16} y2={height - 18} className="chart-axis" />
        <line x1="16" y1="18" x2="16" y2={height - 18} className="chart-axis" />
        <polyline points={points} fill="none" className="chart-line" />
      </svg>
      <figcaption>最大残差从 {formatNumber(values[0], 4)} 降至 {formatNumber(values.at(-1) ?? 0, 6)}</figcaption>
    </figure>
  );
}
