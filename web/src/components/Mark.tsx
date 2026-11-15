export function Mark({ class: className = '' }: { class?: string }) {
  return (
    <svg
      class={className}
      viewBox="0 0 48 48"
      role="img"
      aria-label="Foundry Tunnel Manager"
      fill="none"
    >
      <rect width="48" height="48" rx="12" fill="var(--color-primary)" />
      <path
        d="M24 9.5 36.5 17v14L24 38.5 11.5 31V17L24 9.5Z"
        stroke="var(--color-btn-text)"
        stroke-width="1.6"
        stroke-linejoin="round"
        opacity="0.55"
      />
      <path
        d="M24 9.5 36.5 17 24 24.5 11.5 17 24 9.5Z"
        fill="var(--color-btn-text)"
        opacity="0.18"
      />
      <circle cx="24" cy="24" r="2.4" fill="var(--color-btn-text)" />
      <circle cx="24" cy="9.5" r="1.6" fill="var(--color-btn-text)" />
      <circle cx="36.5" cy="31" r="1.6" fill="var(--color-btn-text)" />
      <circle cx="11.5" cy="31" r="1.6" fill="var(--color-btn-text)" />
    </svg>
  );
}
