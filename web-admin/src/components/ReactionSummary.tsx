import { useState, useRef, useLayoutEffect, useCallback, useId } from 'react';
import { createPortal } from 'react-dom';
import type { AdminReactionCount } from '../api/client';

/** Sum of all reaction counts (inactive types included). */
export function reactionTotal(reactions?: AdminReactionCount[]): number {
  return (reactions ?? []).reduce((sum, r) => sum + r.count, 0);
}

/**
 * Display label for the total. An em dash when reactions were not loaded
 * (undefined — e.g. the admin API omitted them because aggregation failed),
 * which distinguishes "unknown" from a genuine zero. A loaded array (even an
 * empty one) formats as its numeric total.
 */
export function reactionTotalLabel(reactions?: AdminReactionCount[]): string {
  return reactions === undefined ? '—' : reactionTotal(reactions).toLocaleString();
}

interface BreakdownProps {
  reactions?: AdminReactionCount[];
}

/**
 * Inline emoji+count list. Renders every type the backend returns, in order
 * (active types incl. 0-count, plus inactive types). Inactive types are greyed
 * out (their count is still shown; the "（無効）" note lives in the title only).
 * Chips are inline-block so they sit on the surrounding text baseline.
 */
export function ReactionBreakdown({ reactions }: BreakdownProps) {
  const entries = reactions ?? [];
  if (entries.length === 0) {
    return <span className="text-primary-400">—</span>;
  }
  return (
    <span>
      {entries.map((r, i) => (
        <span
          key={r.id}
          title={r.is_active ? r.label : `${r.label}（無効）`}
          className={[
            'inline-block whitespace-nowrap',
            // Gap between chips only — never after the last one (no trailing
            // margin before a closing paren). Done in JS rather than a `last:`
            // Tailwind variant, which v4 fails to extract next to `${`.
            i < entries.length - 1 ? 'mr-3' : '',
            r.is_active ? '' : 'text-primary-400 opacity-60',
          ]
            .filter(Boolean)
            .join(' ')}
        >
          {r.emoji} {r.count.toLocaleString()}
        </span>
      ))}
    </span>
  );
}

/**
 * Desktop list cell: shows the total; on hover/focus reveals a breakdown popover
 * rendered through a portal with fixed positioning, so the table's
 * overflow-x-auto container cannot clip it. Closes on scroll/resize/Escape.
 */
export function ReactionTotalPopover({ reactions }: BreakdownProps) {
  const total = reactionTotal(reactions);
  const hasEntries = (reactions ?? []).length > 0;
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const tipId = useId();

  const updatePos = useCallback(() => {
    const el = triggerRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    setPos({ top: rect.bottom + 4, left: rect.left });
  }, []);

  useLayoutEffect(() => {
    if (!open) return;
    updatePos();
    const close = () => setOpen(false);
    window.addEventListener('scroll', close, true);
    window.addEventListener('resize', close);
    return () => {
      window.removeEventListener('scroll', close, true);
      window.removeEventListener('resize', close);
    };
  }, [open, updatePos]);

  const show = () => {
    if (hasEntries) setOpen(true);
  };
  const hide = () => setOpen(false);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-label={reactions === undefined ? 'Reactions: unavailable' : `Reactions: ${total}`}
        aria-describedby={open ? tipId : undefined}
        className="tabular-nums cursor-default rounded focus:outline-none focus:ring-2 focus:ring-primary-500"
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
        onKeyDown={(e) => {
          if (e.key === 'Escape') setOpen(false);
        }}
      >
        {reactionTotalLabel(reactions)}
      </button>
      {open &&
        createPortal(
          <div
            id={tipId}
            role="tooltip"
            style={{ position: 'fixed', top: pos.top, left: pos.left, zIndex: 50 }}
            className="rounded-md border border-primary-200 bg-white px-3 py-2 text-sm text-primary-700 shadow-lg"
          >
            <ReactionBreakdown reactions={reactions} />
          </div>,
          document.body,
        )}
    </>
  );
}
