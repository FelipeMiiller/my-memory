import { describe, it, expect } from 'vitest';
import { cn } from '@lib/utils';

describe('cn() helper', () => {
  it('merges simple class strings', () => {
    expect(cn('foo', 'bar')).toBe('foo bar');
  });

  it('deduplicates conflicting tailwind classes (last wins)', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4');
  });

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'extra')).toBe('base extra');
  });

  it('handles undefined / null / false', () => {
    expect(cn('a', undefined, null, false, 'b')).toBe('a b');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
