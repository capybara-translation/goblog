import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ReactionBreakdown, ReactionTotalPopover, reactionTotal } from './ReactionSummary';
import type { AdminReactionCount } from '../api/client';

const data: AdminReactionCount[] = [
  { id: 1, emoji: '👍', label: 'いいね', count: 3, is_active: true },
  { id: 2, emoji: '❤️', label: '好き', count: 0, is_active: true },
  { id: 5, emoji: '🤔', label: 'なるほど', count: 2, is_active: false },
];

describe('reactionTotal', () => {
  it('sums all counts including inactive types', () => {
    expect(reactionTotal(data)).toBe(5);
  });
  it('returns 0 for undefined', () => {
    expect(reactionTotal(undefined)).toBe(0);
  });
  it('returns 0 for empty array', () => {
    expect(reactionTotal([])).toBe(0);
  });
});

describe('ReactionBreakdown', () => {
  it('renders every type including 0-count actives', () => {
    render(<ReactionBreakdown reactions={data} />);
    expect(screen.getByText(/❤️/)).toBeInTheDocument();
  });
  it('marks inactive types with （無効）', () => {
    render(<ReactionBreakdown reactions={data} />);
    expect(screen.getAllByText('（無効）').length).toBeGreaterThan(0);
  });
  it('renders an em dash for empty input', () => {
    render(<ReactionBreakdown reactions={[]} />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });
});

describe('ReactionTotalPopover', () => {
  it('renders the total as a labeled button', () => {
    render(<ReactionTotalPopover reactions={data} />);
    expect(screen.getByRole('button', { name: /Reactions: 5/i })).toBeInTheDocument();
  });
  it('renders zero total when reactions is empty', () => {
    render(<ReactionTotalPopover reactions={[]} />);
    expect(screen.getByRole('button', { name: /Reactions: 0/i })).toBeInTheDocument();
  });
});
