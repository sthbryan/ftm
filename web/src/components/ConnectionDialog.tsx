import { useEffect, useState } from 'preact/hooks';
import { Dialog } from '@zovaris/sephiro/dialog';
import { Button } from '@zovaris/sephiro/button';
import { Field } from '@zovaris/sephiro/field';
import { Input } from '@zovaris/sephiro/input';
import { Plus } from '@phosphor-icons/react';
import { createTunnel, getTunnel, updateTunnel } from '@/lib/stores/tunnels';
import { detectPort, providers } from '@/lib/stores/providers';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';

export function ConnectionDialog({
  open,
  tunnelId,
  onClose,
}: {
  open: boolean;
  tunnelId: string | null;
  onClose: () => void;
}) {
  const isEdit = Boolean(tunnelId);

  const [name, setName] = useState('');
  const [provider, setProvider] = useState('cloudflared');
  const [port, setPort] = useState(30000);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;

    const list = providers.value;
    const fallback = list[0]?.id ?? 'cloudflared';

    if (tunnelId) {
      const tunnel = getTunnel(tunnelId);
      if (tunnel) {
        setName(tunnel.name || '');
        setProvider(tunnel.provider || fallback);
        setPort(tunnel.port || 30000);
      }
      return;
    }

    setName('');
    setProvider(fallback);
    void detectPort().then(setPort);
  }, [open, tunnelId]);

  async function submit(event: Event) {
    event.preventDefault();
    if (submitting) return;

    setSubmitting(true);

    try {
      if (tunnelId) {
        await updateTunnel(tunnelId, { name, provider, localPort: port });
        toast.success(t('connection_updated', { name }));
      } else {
        await createTunnel({ name, provider, localPort: port });
        toast.success(t('connection_created', { name }));
      }
      onClose();
    } catch (error) {
      toast.error(
        t(isEdit ? 'connection_update_failed' : 'connection_create_failed', {
          error: (error as Error).message,
        }),
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={isEdit ? t('edit_connection') : t('new_connection')}
      footer={
        <>
          {isEdit && (
            <Button variant="secondary" type="button" onClick={onClose}>
              {t('cancel')}
            </Button>
          )}
          <Button variant="primary" type="submit" form="connection-form" disabled={submitting}>
            {!isEdit && <Plus size={14} weight="bold" />}
            {isEdit ? t('save') : t('create_connection')}
          </Button>
        </>
      }
    >
      <form id="connection-form" onSubmit={submit} class="flex flex-col gap-3">
        <Field
          label={isEdit ? t('connection_name_label') : t('tunnel_name')}
          htmlFor="conn-name"
          required
        >
          <Input
            id="conn-name"
            value={name}
            required
            autoComplete="off"
            placeholder={isEdit ? t('name_placeholder') : t('tunnel_name_hint')}
            onInput={(event) => setName((event.target as HTMLInputElement).value)}
          />
        </Field>

        <div class="grid grid-cols-2 gap-3">
          <Field label={isEdit ? t('port') : t('local_port')} htmlFor="conn-port" required>
            <Input
              id="conn-port"
              type="number"
              min={1}
              max={65535}
              required
              value={String(port)}
              onInput={(event) => setPort(Number((event.target as HTMLInputElement).value))}
            />
          </Field>

          <Field label={isEdit ? t('provider_label') : t('select_provider')} htmlFor="conn-provider">
            <select
              id="conn-provider"
              class="sph-input"
              value={provider}
              onChange={(event) => setProvider((event.target as HTMLSelectElement).value)}
            >
              {providers.value.map((option) => (
                <option value={option.id}>{option.name}</option>
              ))}
            </select>
          </Field>
        </div>

        {!isEdit && (
          <div class="rounded-[var(--sph-radius-sm)] border border-border-light bg-bg/40 p-3 text-2xs leading-relaxed text-text-muted">
            <p class="m-0 mb-1.5 font-semibold text-text">{t('form_tips_title')}</p>
            <ul class="m-0 grid list-none gap-1 p-0">
              <li>{t('form_tips_port')}</li>
              <li>{t('form_tips_provider')}</li>
              <li>{t('form_tips_name')}</li>
            </ul>
          </div>
        )}
      </form>
    </Dialog>
  );
}
