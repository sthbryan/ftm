import { playersAt } from '@/lib/ritual';
import type { Tunnel } from '@/lib/types';
import { foundryWorlds } from './foundry';
import { tunnels } from './tunnels';

/** Players at a connection's table right now; see playersAt. */
export function playersOf(tunnel: Tunnel): number {
  return playersAt(tunnel, tunnels.value, foundryWorlds.value);
}
