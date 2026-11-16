import { useEffect } from 'preact/hooks';
import { AnimatedNumber } from '@zovaris/sephiro/animated-number';
import { Crown, User } from '@phosphor-icons/react';
import { now, useClock } from '@/lib/stores/clock';
import { worldOn, worldTitle } from '@/lib/stores/foundry';
import { t } from '@/lib/stores/i18n';
import { playersOf } from '@/lib/stores/players';
import { formatClock } from '@/lib/utils/duration';
import type { Tunnel } from '@/lib/types';

/** Seats drawn around the table: at least this many, more when they fill. */
const MIN_SEATS = 6;
/** Past this many, the last seat holds "+N" for everyone who doesn't fit. */
const MAX_SEATS = 12;

/**
 * The session at a glance: a round table with the world and the session clock
 * in the middle, the GM's seat at the bottom and a seat around it for every
 * player who could join. A seat fills (and sparks) when a player connects, so
 * the GM sees who is in without reading anything.
 */
export function SessionTable({ tunnel }: { tunnel: Tunnel }) {
  useEffect(() => useClock(), []);

  const players = playersOf(tunnel);
  const seats = Math.min(MAX_SEATS, Math.max(MIN_SEATS, players + 1));
  // When the table is full, the last seat says how many more are in.
  const overflow = players > MAX_SEATS ? players - (MAX_SEATS - 1) : 0;
  const elapsed = tunnel.startedAt ? now.value - tunnel.startedAt : 0;
  const status = worldOn(tunnel.port);
  const world = status?.active && status.world ? worldTitle(status.world) : '';
  // Foundry's world, when it isn't simply the table's own name.
  const otherWorld = world && world.toLowerCase() !== tunnel.name.toLowerCase() ? world : '';

  return (
    <section class="ftm-table" aria-label={t('table_label', { name: tunnel.name })}>
      <div class="ftm-table__ring" data-crowded={seats > 8 ? '' : undefined}>
        {Array.from({ length: seats }, (_, index) => {
          // Seats start at the GM's chair (bottom) and go round clockwise.
          const angle = 90 + ((index + 1) * 360) / (seats + 1);
          const taken = index < players;
          const more = overflow > 0 && index === seats - 1;
          return (
            <span
              key={index}
              class="ftm-table__seat"
              data-taken={taken ? '' : undefined}
              data-more={more ? '' : undefined}
              style={{ '--angle': `${angle}deg` }}
              aria-hidden="true"
            >
              {more ? `+${overflow}` : <User size={16} weight={taken ? 'fill' : 'regular'} />}
            </span>
          );
        })}
        <span class="ftm-table__seat ftm-table__gm" style={{ '--angle': '90deg' }} title={t('table_gm')}>
          <Crown size={16} weight="fill" aria-hidden="true" />
        </span>

        <div class="ftm-table__top">
          <p class="ftm-table__world">{tunnel.name}</p>
          {otherWorld && <p class="ftm-table__count">{t('table_world', { world: otherWorld })}</p>}
          <p class="ftm-table__clock">{formatClock(elapsed)}</p>
          <p class="ftm-table__count">
            {players === 0 ? (
              t('session_waiting')
            ) : (
              <>
                <AnimatedNumber value={players} className="ftm-count" />{' '}
                {players === 1 ? t('pip_player') : t('pip_players')}
              </>
            )}
          </p>
        </div>
      </div>
    </section>
  );
}
