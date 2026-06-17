import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReactionBreakdown, ReactionTotalPopover, reactionTotal, reactionTotalLabel } from './ReactionSummary';
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
  it('greys out inactive types without a visible 無効 marker', () => {
    render(<ReactionBreakdown reactions={data} />);
    // The visible "（無効）" text marker was removed; grey-out is the only visual cue.
    expect(screen.queryByText('（無効）')).not.toBeInTheDocument();
    // The inactive type (🤔 / なるほど) chip is muted; its note lives in the title.
    expect(screen.getByTitle('なるほど（無効）')).toHaveClass('opacity-60');
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
  it('shows an em dash (not 0) and no popover when reactions are unavailable', async () => {
    const user = userEvent.setup();
    render(<ReactionTotalPopover reactions={undefined} />);
    const btn = screen.getByRole('button', { name: 'Reactions: unavailable' });
    expect(btn).toHaveTextContent('—');
    // Unavailable totals have nothing to expand, so hovering opens no tooltip.
    await user.hover(btn);
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });
});

describe('reactionTotalLabel', () => {
  it('shows the numeric total for a loaded array', () => {
    expect(reactionTotalLabel(data)).toBe('5');
  });
  it('shows 0 for a genuinely empty (loaded) array', () => {
    expect(reactionTotalLabel([])).toBe('0');
  });
  it('shows an em dash when reactions were not loaded (undefined)', () => {
    expect(reactionTotalLabel(undefined)).toBe('—');
  });
});
