import { useEffect, useRef, useState } from 'preact/hooks';
import { AnimatedNumber } from '@zovaris/sephiro/animated-number';
import { Button } from '@zovaris/sephiro/button';
import { HoldButton } from '@zovaris/sephiro/hold-button';
import { Progress } from '@zovaris/sephiro/progress';
import { ArrowLeft, Check, Copy, DoorOpen, Plugs, QrCode, User } from '@phosphor-icons/react';
import { IconAction } from '@/components/IconAction';
import { dragRegion } from '@/lib/desktop';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { startTunnel, stopTunnel, tunnels } from '@/lib/stores/tunnels';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { formatClock, formatDuration } from '@/lib/utils/duration';
import { providerErrorHint } from '@/lib/utils/providerError';
import { isRunningState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { queryParam } from '@/router';
import { DecodeText } from './DecodeText';
import { QrImage } from './QrImage';
import { StatusMarkDot } from './StatusPill';

/** Seats drawn before the rest collapse into the count. */
const MAX_SEATS = 6;

type Closed = { elapsed: number; peak: number };

/**
 * The pop-out: one table at a glance while the GM runs the game in another
 * window. The top strip moves the window. The middle says who is in and for
 * how long, or shows the QR. Closing takes a deliberate hold, and once closed
 * the panel keeps a one-line summary until the table opens again.
 */
export function MiniPanel() {
  useEffect(() => useClock(), []);

  const id = queryParam('id');
  const tunnel = tunnels.value.find((candidate) => candidate.id === id) ?? null;

  return (
    <div class="ftm-mini">
      <div class="ftm-mini__bar" {...dragRegion()}>
        {tunnel && (
          <span class="ftm-mini__state">
            <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} size="sm" />
            <span>{t(tunnel.state)}</span>
          </span>
        )}
      </div>
      {tunnel ? (
        <Panel tunnel={tunnel} />
      ) : (
        <div class="ftm-mini__empty">
          <Plugs size={22} />
          <p class="m-0">{t('pip_gone')}</p>
        </div>
      )}
    </div>
  );
}

function Panel({ tunnel }: { tunnel: Tunnel }) {
  const [copied, setCopied] = useState(false);
  const [qr, setQr] = useState(false);
  const [closed, setClosed] = useState<Closed | null>(null);
  const peak = useRef(0);
  const lastElapsed = useRef(0);

  const running = isRunningState(tunnel.state);
  const online = tunnel.state === 'online';
  const stopping = tunnel.state === 'stopping';
  const players = online ? (tunnel.activeSessions ?? 0) : 0;
  const elapsed = tunnel.startedAt && running ? now.value - tunnel.startedAt : 0;
  const remaining = tunnel.expiresAt && online ? tunnel.expiresAt - now.value : 0;
  const errorHint = providerErrorHint(tunnel.errorMessage);

  if (players > peak.current) peak.current = players;
  if (elapsed > 0) lastElapsed.current = elapsed;

  // Remember the session the moment it ends, for the summary.
  const wasRunning = useRef(running);
  useEffect(() => {
    if (wasRunning.current && !running && !tunnel.errorMessage) {
      setClosed({ elapsed: lastElapsed.current, peak: peak.current });
      setQr(false);
    }
    if (running) {
      setClosed(null);
      if (!wasRunning.current) peak.current = 0;
    }
    wasRunning.current = running;
  }, [running]);

  async function copy() {
    if (!tunnel.publicUrl) return;
    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t('copy_failed'));
      return;
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  async function act(action: () => Promise<unknown>) {
    try {
      await action();
    } catch (error) {
      toast.error((error as Error).message);
    }
  }

  return (
    <div class="ftm-mini__body">
      <header class="ftm-mini__head">
        <h1 class="ftm-mini__name" title={tunnel.name}>
          {tunnel.name}
        </h1>
        <p class="ftm-mini__meta">
          {providerLabel(tunnel.provider)} · <span class="font-mono">:{tunnel.port}</span>
        </p>
      </header>

      <div class="ftm-mini__stage">
        {online && qr && tunnel.publicUrl ? (
          <div key="qr" class="ftm-mini__qr ftm-mini__swap">
            <span class="ftm-qr">
              <QrImage value={tunnel.publicUrl} size={176} fluid alt={t('qr_for_phones')} />
            </span>
          </div>
        ) : online ? (
          <div key="live" class="ftm-mini__live ftm-mini__swap">
            <div class="ftm-mini__figures">
              <div>
                <AnimatedNumber value={players} className="ftm-mini__count" />
                <p class="ftm-mini__caption">
                  {players === 1 ? t('pip_player') : t('pip_players')}
                </p>
              </div>
              <div class="text-end">
                <span class="ftm-mini__clock">{formatClock(elapsed)}</span>
                <p class="ftm-mini__caption">
                  {remaining > 0
                    ? t('card_expires', { 0: formatDuration(remaining) })
                    : t('session_time')}
                </p>
              </div>
            </div>
            <div class="ftm-mini__seats" aria-hidden="true">
              {Array.from({ length: Math.max(MAX_SEATS, 0) }, (_, index) => (
                <span key={index} class="ftm-mini__seat" data-taken={index < players ? '' : undefined}>
                  <User size={12} weight={index < players ? 'fill' : 'regular'} />
                </span>
              ))}
              {players > MAX_SEATS && <span class="ftm-mini__more">+{players - MAX_SEATS}</span>}
            </div>
          </div>
        ) : stopping ? (
          <div key="closing" class="ftm-mini__note ftm-mini__swap">
            <p class="m-0 text-sm text-text">{t('pip_closing')}</p>
          </div>
        ) : running ? (
          <div key="opening" class="ftm-mini__note ftm-mini__swap">
            <p class="m-0 text-sm text-text">{t('pip_opening')}</p>
            <Progress label={t(tunnel.state)} size="sm" />
          </div>
        ) : closed ? (
          <div key="closed" class="ftm-mini__note ftm-mini__swap">
            <p class="m-0 font-serif text-lg text-text">{t('pip_closed')}</p>
            <p class="m-0 text-2xs text-text-muted">
              {t('pip_closed_summary', { time: formatClock(closed.elapsed) })}
              {closed.peak > 0 &&
                ` · ${closed.peak === 1 ? t('pip_closed_peak_one') : t('pip_closed_peak', { count: closed.peak })}`}
            </p>
          </div>
        ) : (
          <div key="idle" class="ftm-mini__note ftm-mini__swap">
            {tunnel.errorMessage ? (
              <p class="m-0 text-2xs break-words text-text-error" title={tunnel.errorMessage}>
                {errorHint ? t(errorHint) : tunnel.errorMessage}
              </p>
            ) : (
              <p class="m-0 text-2xs text-text-muted">{t('pip_idle')}</p>
            )}
          </div>
        )}
      </div>

      {(online || stopping) && tunnel.publicUrl && (
        <div class="ftm-mini__link">
          <Button
            variant="secondary"
            size="sm"
            className="ftm-mini__url"
            disabled={stopping}
            aria-label={t('session_link')}
            onClick={copy}
          >
            <DecodeText
              text={tunnel.publicUrl.replace(/^https?:\/\//, '')}
              closing={stopping}
              class="min-w-0 flex-1 truncate text-start font-mono text-2xs text-url-text"
            />
            {copied ? (
              <Check size={14} className="shrink-0 text-status-running" />
            ) : (
              <Copy size={14} className="shrink-0 text-text-muted" />
            )}
          </Button>
          <IconAction
            label={qr ? t('pip_hide_qr') : t('pip_show_qr')}
            size="sm"
            variant="outline"
            disabled={!online}
            aria-pressed={qr}
            icon={qr ? <ArrowLeft size={14} /> : <QrCode size={14} />}
            onClick={() => setQr((open) => !open)}
          />
        </div>
      )}

      <div class="ftm-mini__action">
        {running ? (
          <HoldButton
            variant="secondary"
            className="w-full"
            disabled={stopping || tunnel.state === 'installing'}
            holdLabel={t('pip_holding')}
            confirmedLabel={t('stopping')}
            onConfirm={() => void act(() => stopTunnel(tunnel.id))}
          >
            {t('pip_hold_stop')}
          </HoldButton>
        ) : (
          <Button variant="primary" className="w-full" onClick={() => void act(() => startTunnel(tunnel.id))}>
            <DoorOpen size={16} />
            {closed ? t('pip_open_again') : t('open_table')}
          </Button>
        )}
      </div>
    </div>
  );
}
