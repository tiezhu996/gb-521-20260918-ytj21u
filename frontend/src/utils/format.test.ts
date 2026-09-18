import { describe, expect, it } from 'vitest';
import { formatNumber, roleLabel } from './format';

describe('format utilities', () => {
  it('formats engineering values with a stable precision ceiling', () => {
    expect(formatNumber(12.34567, 3)).toBe('12.346');
  });

  it('keeps the backend role vocabulary mapped to operator labels', () => {
    expect(roleLabel.engineer).toBe('通风工程师');
    expect(roleLabel.reviewer).toBe('安全复核员');
    expect(roleLabel.admin).toBe('系统管理员');
  });
});
