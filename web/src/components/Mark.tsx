/**
 * The app mark: the faceted d20 on the orange ground, the same drawing as
 * favicon.svg and the desktop icons. Its colours are the brand's, not the
 * theme's, so it reads the same in Tinta dark and light.
 */
export function Mark({ class: className = '' }: { class?: string }) {
  return (
    <svg class={className} viewBox="0 0 40 40" role="img" aria-label="Foundry Tunnel Manager">
      <rect width="40" height="40" rx="9" fill="#ac4f06" />
      <g transform="translate(8 8)">
        <path d="M3.34 7 L12 2 L20.66 7 L20.66 17 L12 22 L3.34 17 Z" fill="#e3c05c" />
        <g
          fill="none"
          stroke="#ac4f06"
          stroke-width="1.15"
          stroke-linejoin="round"
          stroke-linecap="round"
        >
          <path d="M12 5.82 L17.352 15.09 L6.648 15.09 Z" />
          <path d="M12 2 L12 5.82 M3.34 7 L12 5.82 M20.66 7 L12 5.82 M3.34 7 L6.648 15.09 M20.66 7 L17.352 15.09 M3.34 17 L6.648 15.09 M20.66 17 L17.352 15.09 M12 22 L6.648 15.09 M12 22 L17.352 15.09" />
        </g>
      </g>
    </svg>
  );
}
