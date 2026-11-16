import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { FieldMessage } from '@zovaris/sephiro/field-message';
import { HoldButton } from '@zovaris/sephiro/hold-button';
import { Input } from '@zovaris/sephiro/input';
import { NumberInput } from '@zovaris/sephiro/number-input';
import { RadioGroup } from '@zovaris/sephiro/radio-group';
import { Spinner } from '@zovaris/sephiro/spinner';
import { ArrowLeft, Check, Minus, Sparkle } from '@phosphor-icons/react';
import { IconAction } from '@/components/IconAction';
import { busyPorts, checkFoundry } from '@/lib/stores/foundry';
import { t } from '@/lib/stores/i18n';
import { detectPort, providers } from '@/lib/stores/providers';
import { selectedPreset } from '@/lib/stores/ritual';
import { toast } from '@/lib/stores/toast';
import {
  createTunnel,
  deleteTunnel,
  getTunnel,
  tunnels,
  tunnelsLoading,
  updateTunnel,
} from '@/lib/stores/tunnels';
import { navigate, type ConnectionRoute } from '@/router';
import { ProviderIcon } from './ritual/ProviderIcon';

const DEFAULT_PORT = 30000;

/**
 * What each provider allows, for the comparison next to the sentence. The API
 * only names providers; these facts change with the providers themselves, so
 * they live beside their descriptions.
 */
const PROVIDER_FACTS: Record<string, { limit: 'none' | '60'; download: boolean; https: boolean }> = {
  cloudflared: { limit: 'none', download: true, https: true },
  tunnelmole: { limit: 'none', download: true, https: true },
  localhostrun: { limit: 'none', download: false, https: true },
  serveo: { limit: 'none', download: false, https: true },
  pinggy: { limit: '60', download: true, https: true },
  bore: { limit: 'none', download: true, https: false },
};

function leave() {
  navigate('/');
}

/**
 * New and edit connection, one page for both. The mode comes from the route,
 * so there is nothing to reset on the way out and no other form to flash.
 */
export function ConnectionPage({ route }: { route: ConnectionRoute }) {
  const editId = route.mode === 'edit' ? route.id : null;
  const existing = editId ? tunnels.value.find((tunnel) => tunnel.id === editId) : undefined;

  if (editId && !existing) {
    if (tunnelsLoading.value) {
      return (
        <div class="flex justify-center py-24">
          <Spinner size="md" label={t('loading')} />
        </div>
      );
    }
    return <Missing />;
  }

  return <ConnectionForm key={editId ?? 'new'} editId={editId} />;
}

function Missing() {
  useEffect(() => navigate('/', { replace: true }), []);
  return null;
}

function ConnectionForm({ editId }: { editId: string | null }) {
  const isEdit = editId !== null;
  const initial = editId ? getTunnel(editId) : undefined;

  const [name, setName] = useState(initial?.name ?? '');
  const [provider, setProvider] = useState(initial?.provider ?? 'cloudflared');
  const [port, setPort] = useState<number | null>(initial?.port ?? DEFAULT_PORT);
  const [suggested, setSuggested] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    void checkFoundry();
    void detectPort().then((found) => {
      setSuggested(found);
      if (!isEdit) setPort(found);
    });
  }, [isEdit]);

  useEffect(() => {
    const list = providers.value;
    if (list.length > 0 && !list.some((option) => option.id === provider)) setProvider(list[0].id);
  }, [providers.value]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !event.defaultPrevented) leave();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const foundryOnSuggested =
    suggested !== null && (busyPorts.value ?? []).includes(suggested) && suggested !== port;
  const portValid = port !== null && port >= 1 && port <= 65535;

  async function submit(event: Event) {
    event.preventDefault();
    if (submitting || !portValid || !name.trim()) return;
    setSubmitting(true);

    const data = { name: name.trim(), provider, localPort: port as number };
    try {
      if (editId) {
        await updateTunnel(editId, data);
        toast.success(t('connection_updated', { name: data.name }));
        selectedPreset.value = editId;
      } else {
        const created = await createTunnel(data);
        toast.success(t('connection_created', { name: data.name }));
        selectedPreset.value = created.id;
      }
      leave();
    } catch (error) {
      toast.error(
        t(isEdit ? 'connection_update_failed' : 'connection_create_failed', {
          error: (error as Error).message,
        }),
      );
      setSubmitting(false);
    }
  }

  async function remove() {
    if (!editId) return;
    try {
      await deleteTunnel(editId);
      toast.success(t('connection_deleted', { name: initial?.name ?? '' }));
      leave();
    } catch (error) {
      toast.error(t('connection_delete_failed', { error: (error as Error).message }));
    }
  }

  const providerName = providers.value.find((option) => option.id === provider)?.name ?? provider;

  const providerOptions = providers.value.map((option) => {
    const facts = PROVIDER_FACTS[option.id];
    return {
      value: option.id,
      label: (
        <span class="ftm-prov">
          <span class="ftm-prov__name">
            <ProviderIcon provider={option.id} />
            <span>{option.name}</span>
          </span>
          {facts && (
            <>
              <span class="ftm-prov__fact">{t(`provider_limit_${facts.limit}`)}</span>
              <span class="ftm-prov__fact">
                {facts.download ? t('provider_download_once') : t('provider_download_none')}
              </span>
              <span class="ftm-prov__fact" data-good={facts.https ? '' : undefined}>
                {facts.https ? (
                  <Check size={14} aria-label={t('provider_https_yes')} />
                ) : (
                  <Minus size={14} aria-label={t('provider_https_no')} />
                )}
              </span>
            </>
          )}
        </span>
      ),
      description: t(`provider_note_${option.id}`),
    };
  });

  return (
    <section class="ftm-page" aria-labelledby="connection-title">
      <form onSubmit={submit} class="ftm-compose">
        <div class="ftm-compose__text">
          <header class="flex items-center gap-2">
            <IconAction
              label={t('go_back')}
              side="bottom"
              icon={<ArrowLeft size={18} />}
              onClick={leave}
            />
            <h1 id="connection-title" class="m-0 text-sm font-medium text-text-muted">
              {isEdit ? t('edit_connection') : t('new_connection')}
            </h1>
          </header>

          <p class="ftm-sentence">
            {t('sentence_open')}{' '}
            <Input
              id="conn-name"
              className="ftm-slot ftm-slot--name"
              value={name}
              required
              autoFocus={!isEdit}
              autoComplete="off"
              aria-label={t('connection_name_label')}
              placeholder={t('tunnel_name_hint')}
              onInput={(event) => setName((event.target as HTMLInputElement).value)}
            />{' '}
            {t('sentence_port')}{' '}
            <NumberInput
              id="conn-port"
              className="ftm-slot ftm-slot--port"
              value={port}
              min={1}
              max={65535}
              step={1}
              format={{ useGrouping: false }}
              invalid={!portValid}
              label={t('local_port')}
              onValueChange={setPort}
            />{' '}
            {t('sentence_using')} <strong class="ftm-sentence__provider">{providerName}</strong>
          </p>

          <div class="ftm-compose__hints">
            {!portValid && <FieldMessage variant="error">{t('port_invalid')}</FieldMessage>}
            {foundryOnSuggested && (
              <Button variant="secondary" size="sm" onClick={() => setPort(suggested)}>
                <Sparkle size={16} />
                {t('port_use_detected', { port: suggested ?? '' })}
              </Button>
            )}
            <p class="m-0">{t('form_tips_name')}</p>
            <p class="m-0">{t('form_tips_port')}</p>
          </div>

          <div class="ftm-compose__actions">
            <Button
              variant="primary"
              size="lg"
              type="submit"
              loading={submitting}
              disabled={!portValid || !name.trim()}
            >
              {isEdit ? t('save') : t('create_connection')}
            </Button>
            <Button variant="quiet" size="lg" onClick={leave}>
              {t('cancel')}
            </Button>
            {isEdit && (
              <HoldButton
                variant="danger"
                size="md"
                className="ms-auto"
                holdLabel={t('keep_holding')}
                confirmedLabel={t('connection_deleted_short')}
                hint={t('confirm_delete_body', { 0: initial?.name ?? '' })}
                onConfirm={() => void remove()}
              >
                {t('delete')}
              </HoldButton>
            )}
          </div>
        </div>

        <div class="ftm-compose__providers">
          <div class="ftm-prov ftm-prov--head" aria-hidden="true">
            <span>{t('provider_label')}</span>
            <span>{t('provider_col_limit')}</span>
            <span>{t('provider_col_download')}</span>
            <span>{t('provider_col_https')}</span>
          </div>
          <RadioGroup
            name="provider"
            label={t('provider_label')}
            description={t('form_tips_provider')}
            value={provider}
            options={providerOptions}
            onValueChange={setProvider}
            className="ftm-provider-table"
          />
        </div>
      </form>
    </section>
  );
}
