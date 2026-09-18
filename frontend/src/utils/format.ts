export const formatDateTime = (value?: string) => value
  ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short', hour12: false }).format(new Date(value))
  : '未完成';

export const formatNumber = (value: number, digits = 2) => new Intl.NumberFormat('zh-CN', { maximumFractionDigits: digits }).format(value);

export const roleLabel = { engineer: '通风工程师', reviewer: '安全复核员', admin: '系统管理员' } as const;
