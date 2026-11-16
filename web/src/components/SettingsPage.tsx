import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { SegmentedControl } from '@zovaris/sephiro/segmented-control';
import { HoldButton } from '@zovaris/sephiro/hold-button';
import { Card } from '@zovaris/sephiro/card';
import type { Renderable } from '@zovaris/sephiro';
import { Spinner } from '@zovaris/sephiro/spinner';
import { Toggle } from '@zovaris/sephiro/toggle';
import { IconAction } from '@/components/IconAction';
import {
  ArrowClockwise,
  Bell,
  BellSlash,
  CircleHalf,
  DownloadSimple,
  GlobeHemisphereWest,
  Power,
  SpeakerHigh,
  SpeakerSlash,
  Trash,
} from '@phosphor-icons/react';
import { statusApi } from '@/lib/api';
import { loadSettings, settings, settingsLoaded, updateSettings } from '@/lib/stores/settings';
import {
  notificationsEnabled,
  requestNotificationPermission,
  soundEnabled,
} from '@/lib/stores/notification';
import { availableLanguages, LANGUAGE_AUTO, setLanguage, t } from '@/lib/stores/i18n';
import { clearAllTunnels, installProgress, tunnels } from '@/lib/stores/tunnels';
import { toast } from '@/lib/stores/toast';
import {
  canUpdate,
  checkingProviders,
  installedProviders,
  loadInstalledProviders,
  updateProvider,
  type InstalledProvider,
} from '@/lib/stores/installed';
import { unlockSound } from '@/lib/sound';
import { ThemeGrid } from './ThemeGrid';

function Row({
  icon,
  title,
  description,
  children,
  last = false,
  wide = false,
}: {
  icon?: preact.ComponentChildren;
  title: string;
  description: string;
  children: preact.ComponentChildren;
  last?: boolean;
  /** Puts a wide control (segmented) on its own line on narrow screens. */
  wide?: boolean;
}) {
  return (
    <div
      class={`grid min-h-16 items-center gap-x-4 gap-y-3 px-5 py-3 ${
        icon ? 'grid-cols-[auto_1fr_auto]' : 'grid-cols-[1fr_auto]'
      } ${last ? '' : 'border-b border-border-light'}`}
    >
      {icon && (
        <span class="flex shrink-0 text-text-muted" aria-hidden="true">
          {icon}
        </span>
      )}
      <div class="min-w-0">
        <p class="m-0 text-sm font-medium text-text">{title}</p>
        <p class="m-0 text-xs text-text-muted">{description}</p>
      </div>
      {wide ? (
        <div class={icon ? 'max-sm:col-span-3 max-sm:justify-self-end' : 'max-sm:col-span-2'}>
          {children}
        </div>
      ) : (
        children
      )}
    </div>
  );
}

/**
 * A settings group: its name and note on the left at desktop width, the rows
 * on one surface to the right. On a phone the name sits above the surface.
 */
function Section({
  title,
  description,
  action,
  children,
}: {
  title: string;
  description?: string;
  action?: preact.ComponentChildren;
  children: preact.ComponentChildren;
}) {
  return (
    <section class="grid gap-3 border-t border-border-light py-8 first:border-t-0 first:pt-0 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-12">
      <header class="flex items-start justify-between gap-3 lg:flex-col lg:justify-start">
        <div class="min-w-0">
          <h2 class="m-0 text-sm font-semibold text-text">{title}</h2>
          {description && <p class="m-0 mt-1 text-xs text-text-muted">{description}</p>}
        </div>
        {action}
      </header>
      <Card className="ftm-settings-card">{children as Renderable}</Card>
    </section>
  );
}

export function SettingsPage() {
  const [saving, setSaving] = useState(false);
  const [version, setVersion] = useState('');

  useEffect(() => {
    void loadSettings().catch(() => undefined);
    void loadInstalledProviders();
    statusApi
      .get()
      .then((status) => setVersion(status.version))
      .catch(() => undefined);
  }, []);

  const languageOptions = [
    { value: LANGUAGE_AUTO, label: t('lang_auto_native') },
    ...availableLanguages.value
      .filter((code) => code !== LANGUAGE_AUTO)
      .map((code) => ({ value: code, label: t(`lang_${code}`) })),
  ];

  async function toggleNotifications() {
    setSaving(true);
    try {
      if (notificationsEnabled.value) {
        await updateSettings({ notifications_enabled: 'rejected' });
        return;
      }
      await requestNotificationPermission();
      await loadSettings();
    } finally {
      setSaving(false);
    }
  }

  async function toggleSound() {
    setSaving(true);
    try {
      const next = !soundEnabled.value;
      await updateSettings({ notification_sound: next });
      if (next) unlockSound();
    } finally {
      setSaving(false);
    }
  }

  async function toggleAutostart() {
    setSaving(true);
    try {
      await updateSettings({ autostart_enabled: !settings.value.autostart_enabled });
    } catch (error) {
      const status = (error as { response?: Response }).response?.status;
      toast.error(status === 409 ? t('autostart_blocked') : t('autostart_failed'));
    } finally {
      setSaving(false);
    }
  }

  async function changeLanguage(value: string) {
    if (settings.value.language === value || saving) return;
    setSaving(true);
    try {
      await updateSettings({ language: value });
      await setLanguage(value);
    } finally {
      setSaving(false);
    }
  }

  async function resetConnections() {
    try {
      await clearAllTunnels();
      toast.success(t('connections_cleared'));
    } catch (error) {
      toast.error(t('reset_failed', { 0: (error as Error).message }));
    }
  }

  const count = tunnels.value.length;

  return (
    <div class="flex w-full flex-col">
      <div class="mb-8 flex items-center justify-between gap-3">
        <h1 class="m-0 text-2xl font-semibold text-text">{t('web_settings_title')}</h1>
        {saving && <Spinner size="sm" label={t('loading')} />}
      </div>

      <div>
        {!settingsLoaded.value ? (
          <div class="flex justify-center py-10">
            <Spinner size="md" label={t('loading')} />
          </div>
        ) : (
          <div class="flex flex-col">
            <Section title={t('preferences_section')}>
              <Row
                icon={notificationsEnabled.value ? <Bell size={20} /> : <BellSlash size={20} />}
                title={t('enable_notifications_web')}
                description={t('settings_notifications_desc')}
              >
                <Toggle
                  checked={notificationsEnabled.value}
                  disabled={saving}
                  label={t('enable_notifications_web')}
                  onCheckedChange={toggleNotifications}
                />
              </Row>

              <Row
                icon={soundEnabled.value ? <SpeakerHigh size={20} /> : <SpeakerSlash size={20} />}
                title={t('sound_effects')}
                description={t('settings_sound_desc')}
              >
                <Toggle
                  checked={soundEnabled.value}
                  disabled={saving}
                  label={t('sound_effects')}
                  onCheckedChange={toggleSound}
                />
              </Row>

              {settings.value.autostart_supported && (
                <Row
                  icon={<Power size={20} />}
                  title={t('autostart_login')}
                  description={t('settings_autostart_desc')}
                >
                  <Toggle
                    checked={settings.value.autostart_enabled}
                    disabled={saving}
                    label={t('autostart_login')}
                    onCheckedChange={toggleAutostart}
                  />
                </Row>
              )}

              <Row
                icon={<GlobeHemisphereWest size={20} />}
                title={t('language_section')}
                description={t('settings_language_desc')}
                last
                wide
              >
                <SegmentedControl
                  ariaLabel={t('language_section')}
                  size="sm"
                  options={languageOptions}
                  value={settings.value.language}
                  onValueChange={changeLanguage}
                />
              </Row>
            </Section>

            <Section title={t('appearance_section')}>
              <Row
                icon={<CircleHalf size={20} />}
                title={t('theme')}
                description={t('match_system')}
                last
                wide
              >
                <ThemeGrid />
              </Row>
            </Section>

            {installedProviders.value && installedProviders.value.length > 0 && (
              <InstalledProviders providers={installedProviders.value} />
            )}

            <Section title={t('danger_section')}>
              <Row
                icon={<Trash size={20} className="text-text-error" />}
                title={t('reset_connections')}
                description={t('reset_connections_desc')}
                last
              >
                <HoldButton
                  variant="danger"
                  size="sm"
                  disabled={count === 0}
                  holdLabel={t('keep_holding')}
                  confirmedLabel={t('deleted_all')}
                  hint={
                    count === 1
                      ? t('confirm_reset_body_one')
                      : t('confirm_reset_body_other', { 0: count })
                  }
                  onConfirm={() => void resetConnections()}
                >
                  {t('hold_to_delete')}
                </HoldButton>
              </Row>
            </Section>

            {version && (
              <footer class="pt-2 text-end font-mono text-2xs text-text-muted">v{version}</footer>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function providerDescription(provider: InstalledProvider): string {
  const parts: string[] = [];
  if (provider.installed) {
    // ssh-based providers report no version; say nothing rather than "?".
    if (provider.version) parts.push(t('provider_version', { 0: provider.version }));
    if (provider.hasUpdate && provider.latest)
      parts.push(t('provider_latest', { 0: provider.latest }));
  } else {
    parts.push(t('provider_not_installed'));
  }
  if (provider.source !== 'none') parts.push(t(`providers_source_${provider.source}`));
  const text = parts.join(' · ');
  return text.charAt(0).toLocaleUpperCase() + text.slice(1);
}

function InstalledProviders({ providers }: { providers: InstalledProvider[] }) {
  const progress = installProgress.value;

  return (
    <Section
      title={t('installed_providers')}
      description={t('installed_providers_desc')}
      action={
        <Button
          variant="quiet"
          size="sm"
          className="shrink-0 whitespace-nowrap lg:-ms-3"
          loading={checkingProviders.value}
          onClick={() => void loadInstalledProviders(true)}
        >
          <ArrowClockwise size={16} />
          {t('providers_recheck')}
        </Button>
      }
    >
      {providers.map((provider, index) => {
        const percent = progress[provider.id]?.percent;
        const external = provider.source === 'path';
        return (
          <Row
            key={provider.id}
            title={provider.name}
            description={
              external && provider.hasUpdate
                ? `${providerDescription(provider)} · ${t('providers_external_hint', { 0: provider.name, 1: provider.path })}`
                : providerDescription(provider)
            }
            last={index === providers.length - 1}
          >
            {provider.updating ? (
              <Button variant="secondary" size="sm" loading disabled>
                {t('provider_updating')}
                {typeof percent === 'number' ? ` ${Math.round(percent)}%` : ''}
              </Button>
            ) : canUpdate(provider) ? (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => void updateProvider(provider.id)}
              >
                <DownloadSimple size={16} />
                {provider.source === 'none' ? t('provider_install') : t('provider_update')}
              </Button>
            ) : (
              <span />
            )}
          </Row>
        );
      })}
    </Section>
  );
}
