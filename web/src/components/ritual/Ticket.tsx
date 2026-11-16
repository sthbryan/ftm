import type { ComponentChildren } from 'preact';
import { Surface } from '@zovaris/sephiro/surface';

/**
 * A table as a ticket: the body says which world and how it opens, the stub
 * (behind a perforation with two notches) holds what a player needs: the port
 * while it waits, the code printing while it opens, the QR once it is open.
 * The same object carries the GM through ready, opening and invite, so each
 * moment changes what is printed on it rather than replacing the screen.
 */
export function Ticket({
  label,
  stub,
  children,
  state,
}: {
  label: string;
  stub: ComponentChildren;
  children: ComponentChildren;
  state?: 'ready' | 'opening' | 'open' | 'failed';
}) {
  return (
    <Surface padding="none" className="ftm-ticket" role="group" aria-label={label} data-state={state}>
      <div class="ftm-ticket__main">{children}</div>
      <div class="ftm-ticket__stub">{stub}</div>
    </Surface>
  );
}
