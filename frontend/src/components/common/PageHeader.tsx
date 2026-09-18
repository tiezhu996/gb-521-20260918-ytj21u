import type { ReactNode } from 'react';

export function PageHeader({ eyebrow, title, meta, actions }: { eyebrow: string; title: string; meta?: ReactNode; actions?: ReactNode }) {
  return (
    <header className="page-header">
      <div>
        <span className="page-eyebrow">{eyebrow}</span>
        <h1>{title}</h1>
        {meta && <div className="page-meta">{meta}</div>}
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </header>
  );
}
