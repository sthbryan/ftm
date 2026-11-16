/** Eight fill steps per cell, bottom-up, in braille dots. */
const STEPS = ['⠀', '⡀', '⣀', '⣄', '⣤', '⣦', '⣶', '⣷', '⣿'];

/**
 * A download bar drawn in braille: each character fills in eight steps, so a
 * thirty-cell bar moves in 240 increments and never looks stuck on a slow
 * connection. The filled part is the accent, the rest a dotted track.
 */
export function BrailleBar({ percent, cells = 30, label }: { percent: number; cells?: number; label: string }) {
  const clamped = Math.max(0, Math.min(100, percent));
  const steps = Math.round((clamped / 100) * cells * 8);
  const full = Math.floor(steps / 8);
  const partial = steps % 8;

  return (
    <span
      class="ftm-braille"
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(clamped)}
    >
      <span class="ftm-braille__done" aria-hidden="true">
        {STEPS[8].repeat(full)}
        {full < cells && partial > 0 ? STEPS[partial] : ''}
      </span>
      <span class="ftm-braille__rest" aria-hidden="true">
        {'⣀'.repeat(Math.max(0, cells - full - (partial > 0 ? 1 : 0)))}
      </span>
    </span>
  );
}
