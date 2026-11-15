import { useEffect } from 'preact/hooks';
import { MetricCard } from '@zovaris/sephiro/metric-card';
import { Badge } from '@zovaris/sephiro/badge';
import { now, useClock } from '@/lib/stores/clock';
import { fleetSamples } from '@/lib/stores/history';
import { tunnels, tunnelsConnected, installProgress } from '@/lib/stores/tunnels';
import { t } from '@/lib/stores/i18n';
import { formatDuration } from '@/lib/utils/duration';

const ACTIVE = ['online', 'starting', 'connecting'];

export function StatStrip() {
  useEffect(() => useClock(), []);

  const list = tunnels.value;
  const online = list.filter((tunnel) => tunnel.state === 'online');
  const active = list.filter((tunnel) => ACTIVE.includes(tunnel.state));
  const failing = list.filter((tunnel) => ['error', 'timeout'].includes(tunnel.state));
  const sessions = active.reduce((sum, tunnel) => sum + (tunnel.activeSessions ?? 0), 0);
  const players = active.reduce((sum, tunnel) => sum + (tunnel.visitors ?? 0), 0);

  const nextExpiry = online
    .filter((tunnel) => (tunnel.expiresAt ?? 0) > now.value)
    .sort((a, b) => (a.expiresAt ?? 0) - (b.expiresAt ?? 0))[0];

  const installing = list.find((tunnel) => tunnel.state === 'installing');

  const fleet = fleetSamples.value;
  const sessionTrend = fleet.map((sample) => sample.sessions);
  const onlineTrend = fleet.map((sample) => sample.online);

  const expiryMinutes = nextExpiry ? Math.max(0, (nextExpiry.expiresAt ?? 0) - now.value) / 60000 : 0;

  return (
    <section class="grid shrink-0 gap-3 sm:grid-cols-2 xl:grid-cols-3" aria-label={t('overview_online')}>
      <MetricCard
        label={t('overview_online')}
        value={`${online.length}/${list.length || 0}`}
        tone={online.length > 0 ? 'success' : 'muted'}
        trend={onlineTrend.length > 1 ? onlineTrend : undefined}
        trendLabel={t('overview_online')}
        trendVariant="area"
        footer={
          failing.length > 0 ? (
            <Badge variant="danger" size="sm">
              {failing.length} {t('overview_failing')}
            </Badge>
          ) : undefined
        }
      />

      <MetricCard
        label={t('overview_players')}
        value={String(players)}
        tone="accent"
        trend={sessionTrend.length > 1 ? sessionTrend : undefined}
        trendLabel={t('overview_players')}
        trendVariant="line"
        footer={
          active.length > 0 ? (
            <span class="text-2xs text-text-muted">
              {sessions} {t('detail_sessions')}
            </span>
          ) : undefined
        }
      />

      <MetricCard
        label={nextExpiry ? t('overview_next_expiry', { name: nextExpiry.name }) : t('overview_idle')}
        value={nextExpiry ? formatDuration((nextExpiry.expiresAt ?? 0) - now.value) : '—'}
        tone={expiryMinutes > 0 && expiryMinutes < 5 ? 'danger' : 'muted'}
        footer={
          installing ? (
            <div class="w-full">
              <div class="h-1 w-full overflow-hidden rounded-full bg-border-light">
                <div
                  class="h-full rounded-full bg-status-installing transition-[width] duration-300"
                  style={{ width: `${Math.trunc((installProgress.value[installing.provider]?.percent ?? 0) * 100) / 100}%` }}
                />
              </div>
            </div>
          ) : (
            !tunnelsConnected.value && (
              <span class="text-2xs text-text-muted">{t('overview_reconnecting')}</span>
            )
          )
        }
      />
    </section>
  );
}
