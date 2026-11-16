import { MARK_DIE, MARK_EDGES, MARK_FACE } from '@/lib/mark';

/**
 * The app mark: the faceted d20 on the accent ground. It follows the theme, so
 * it repaints with Tinta dark and light, and the favicon is drawn from the same
 * colours (lib/mark.ts).
 */
export function Mark({ class: className = '' }: { class?: string }) {
  return (
    <svg class={className} viewBox="0 0 40 40" aria-hidden="true">
      <rect width="40" height="40" rx="9" style={{ fill: 'var(--sph-accent)' }} />
      <g transform="translate(8 8)">
        <path d={MARK_DIE} style={{ fill: 'var(--sph-on-accent)' }} />
        <g
          fill="none"
          style={{ stroke: 'var(--sph-accent)' }}
          stroke-width="1.15"
          stroke-linejoin="round"
          stroke-linecap="round"
        >
          <path d={MARK_FACE} />
          <path d={MARK_EDGES} />
        </g>
      </g>
    </svg>
  );
}
