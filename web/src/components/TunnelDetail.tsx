import { useEffect, useState } from 'preact/hooks';
import { IconButton } from '@zovaris/sephiro/icon-button';
import { Button } from '@zovaris/sephiro/button';
import { EmptyState } from '@zovaris/sephiro/empty-state';
import { RunMatrix, type RunState } from '@zovaris/sephiro/run-matrix';
import { Chart } from '@zovaris/sephiro/chart';
import {
  ArrowsOutSimple,
  Check,
  Copy,
  MouseSimple,
  PencilSimple,
  QrCode as QrIcon,
  Trash,
} from '@phosphor-icons/react';
import { statusApi, pipApi } from '@/lib/api';
import { now, useClock } from '@/lib/stores/clock';
import { historyFor } from '@/lib/stores/history';
import { providerLabel } from '@/lib/stores/providers';
import { tunnelsConnected } from '@/lib/stores/tunnels';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { formatDuration } from '@/lib/utils/duration';
import { supportsDocumentPip, openDocumentPip } from '@/lib/utils/pip';
import { isRunningState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { LogsPanel } from './LogsPanel';
import { QrDialog } from './QrDialog';
import { StatusBadge } from './StatusPill';

function runState(state: string): RunState {
  if (state === 'online') return 'passed';
  if (['error', 'timeout'].includes(state)) return 'failed';
  if (['stopped', 'offline', 'need_installing'].includes(state)) return 'skipped';
  return 'running';
}

export function TunnelDetail({
  tunnel,
  onEdit,
  onDelete,
}: {
  tunnel: Tunnel | null;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}) {
  useEffect(() => useClock(), []);

  const [copied, setCopied] = useState(false);
  const [qrOpen, setQrOpen] = useState(false);
  const [nativePip, setNativePip] = useState(false);

  useEffect(() => {
    statusApi
      .get()
      .then((status) => setNativePip(status.nativePip === true))
      .catch(() => setNativePip(false));
  }, []);

  const pipAvailable = nativePip || supportsDocumentPip();

  async function openPip() {
    if (!tunnel) return;
    try {
      if (nativePip) await pipApi.open(tunnel.id);
      else await openDocumentPip(tunnel.id);
    } catch {
      toast.error(t('pip_failed'));
    }
  }

  async function copy() {
    if (!tunnel?.publicUrl) return;
    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t('copy_failed'));
      return;
    }
    toast.success(t('overview_copied'));
    setCopied(true);
    setTimeout(() => setCopied(false), 3000);
  }

  const running = tunnel ? isRunningState(tunnel.state) : false;
  const stale = running && !tunnelsConnected.value;
  const uptime =
    tunnel?.startedAt && running && !stale ? formatDuration(now.value - tunnel.startedAt) : '';
  const remaining = tunnel?.expiresAt && running ? tunnel.expiresAt - now.value : 0;

  const history = tunnel ? historyFor(tunnel.id).slice(-24) : [];
  const runs = history.map((sample) => runState(sample.state));
  const categories = history.map((sample) =>
    new Date(sample.at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }),
  );

  return (
    <section class="sph-card flex min-h-0 flex-col" aria-label={t('detail_title')}>
      <div class="flex shrink-0 items-center justify-between gap-2 border-b border-border px-3.5 py-2.5">
        <h2 class="m-0 truncate text-sm font-semibold text-text">{t('detail_title')}</h2>
        {tunnel && (
          <div class="flex shrink-0 items-center gap-0.5">
            {pipAvailable && (
              <IconButton label={t('pip_open')} size="sm" icon={<ArrowsOutSimple size={15} />} onClick={openPip} />
            )}
            <IconButton
              label={t('edit')}
              size="sm"
              icon={<PencilSimple size={15} />}
              disabled={running}
              onClick={() => onEdit(tunnel.id)}
            />
            <IconButton
              label={t('delete')}
              size="sm"
              icon={<Trash size={15} className="text-text-error" />}
              disabled={running}
              onClick={() => onDelete(tunnel)}
            />
          </div>
        )}
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto p-3.5">
        {!tunnel ? (
          <EmptyState
            compact
            icon={<MouseSimple size={20} />}
            title={t('detail_empty')}
          />
        ) : (
          <div class="flex flex-col gap-3">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <h3 class="m-0 truncate text-base font-semibold text-text" title={tunnel.name}>
                  {tunnel.name}
                </h3>
                <p class="m-0 mt-0.5 text-2xs text-text-muted">
                  {providerLabel(tunnel.provider)} ·{' '}
                  <span class="font-mono">localhost:{tunnel.port}</span>
                </p>
              </div>
              <StatusBadge state={tunnel.state} />
            </div>

            {tunnel.publicUrl && (
              <div class="sph-well flex items-center gap-1 py-1 pr-1 pl-2.5">
                <span class="min-w-0 flex-1 truncate font-mono text-2xs text-url-text" title={tunnel.publicUrl}>
                  {tunnel.publicUrl}
                </span>
                <IconButton
                  label={copied ? t('link_copied') : t('copy_link')}
                  size="sm"
                  icon={copied ? <Check size={14} className="text-status-running" /> : <Copy size={14} />}
                  onClick={copy}
                />
                <IconButton
                  label={t('qr_for_phones')}
                  size="sm"
                  icon={<QrIcon size={14} />}
                  onClick={() => setQrOpen(true)}
                />
              </div>
            )}

            <dl class="m-0 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-2xs">
              <dt class="text-text-muted">{t('status_label')}</dt>
              <dd class="m-0 text-text">{t(tunnel.state)}</dd>

              {stale && (
                <>
                  <dt class="text-text-muted">{t('connection_label')}</dt>
                  <dd class="m-0 text-text-error">{t('connection_lost')}</dd>
                </>
              )}

              {uptime && (
                <>
                  <dt class="text-text-muted">{t('detail_uptime')}</dt>
                  <dd class="m-0 font-mono tabular-nums">{uptime}</dd>
                </>
              )}

              {remaining > 0 && (
                <>
                  <dt class="text-text-muted">{t('detail_expires')}</dt>
                  <dd class="m-0 font-mono tabular-nums">{formatDuration(remaining)}</dd>
                </>
              )}

              {running && (
                <>
                  <dt class="text-text-muted">{t('detail_sessions')}</dt>
                  <dd class="m-0 font-mono tabular-nums">{tunnel.activeSessions ?? 0}</dd>
                  <dt class="text-text-muted">{t('detail_visitors')}</dt>
                  <dd class="m-0 font-mono tabular-nums">{tunnel.visitors ?? 0}</dd>
                </>
              )}
            </dl>

            {runs.length >= 6 && (
              <div class="border-t border-border-light pt-3">
                <RunMatrix
                  label={t('overview_uptime')}
                  compact
                  rows={[
                    {
                      id: tunnel.id,
                      label: t('detail_uptime'),
                      detail: uptime,
                      runs,
                    },
                  ]}
                />
              </div>
            )}

            {history.length >= 8 && (
              <div class="border-t border-border-light pt-3">
                <Chart
                  categories={categories}
                  series={[
                    {
                      name: t('detail_sessions'),
                      values: history.map((sample) => sample.sessions),
                      tone: 'accent',
                    },
                    {
                      name: t('detail_visitors'),
                      values: history.map((sample) => sample.visitors),
                      tone: 'success',
                    },
                  ]}
                  variant="area"
                  smooth
                  height={128}
                  ticks={3}
                  label={t('detail_sessions')}
                  legend={false}
                />
              </div>
            )}

            <div class="border-t border-border-light pt-3">
              <LogsPanel tunnelId={tunnel.id} state={tunnel.state} />
            </div>
          </div>
        )}
      </div>

      <QrDialog open={qrOpen} url={tunnel?.publicUrl ?? ''} onClose={() => setQrOpen(false)} />
    </section>
  );
}
