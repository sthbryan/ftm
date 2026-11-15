import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { SegmentedControl } from '@zovaris/sephiro/segmented-control';
import { HoldButton } from '@zovaris/sephiro/hold-button';
import {
  Bell,
  BellSlash,
  CircleHalf,
  DownloadSimple,
  GlobeHemisphereWest,
  Package,
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
import { clearAllTunnels, tunnels } from '@/lib/stores/tunnels';
import { toast } from '@/lib/stores/toast';
import {
  installedProviders,
  loadInstalledProviders,
  updateProvider,
  type InstalledProvider,
} from '@/lib/stores/installed';
import { unlockSound } from '@/lib/sound';
import { ThemeGrid } from './ThemeGrid';

function Switch({
  checked,
  disabled,
  label,
  onChange,
}: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (next: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      class="sph-toggle"
      data-state={checked ? 'checked' : 'unchecked'}
      onClick={() => onChange(!checked)}
    >
      <span class="sph-toggle__track">
        <span class="sph-toggle__thumb" />
      </span>
    </button>
  );
}

function Row({
  icon,
  title,
  description,
  children,
  last = false,
  wide = false,
}: {
  icon: preact.ComponentChildren;
  title: string;
  description: string;
  children: preact.ComponentChildren;
  last?: boolean;
  /** Puts a wide control (segmented) on its own line on narrow screens. */
  wide?: boolean;
}) {
  return (
    <div
      class={`grid grid-cols-[auto_1fr_auto] items-center gap-3 px-5 py-3.5 ${
        last ? '' : 'border-b border-border-light'
      }`}
    >
      <span class="flex size-9 shrink-0 items-center justify-center rounded-[var(--sph-radius-sm)] bg-secondary-btn text-text-muted">
        {icon}
      </span>
      <div class="min-w-0">
        <p class="m-0 text-sm font-medium text-text">{title}</p>
        <p class="m-0 truncate text-2xs text-text-muted">{description}</p>
      </div>
      {wide ? <div class="max-sm:col-span-3 max-sm:justify-self-end">{children}</div> : children}
    </div>
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
    <div class="mx-auto flex h-full min-h-0 w-full max-w-3xl flex-col">
      <div class="mb-3 flex shrink-0 items-center justify-between gap-3">
        <h1 class="m-0 text-xl font-semibold tracking-tight text-text">{t('web_settings_title')}</h1>
        {saving && <span class="sph-spinner sph-spinner--sm" />}
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto pb-4">
        {!settingsLoaded.value ? (
          <div class="flex justify-center py-10">
            <span class="sph-spinner sph-spinner--md" />
          </div>
        ) : (
          <div class="flex flex-col gap-3">
            <section class="sph-card">
              <header class="px-5 pt-4 pb-2">
                <h2 class="m-0 text-sm font-semibold text-text">{t('preferences_section')}</h2>
              </header>

              <div class="border-t border-border-light">
                <Row
                  icon={notificationsEnabled.value ? <Bell size={17} /> : <BellSlash size={17} />}
                  title={t('enable_notifications_web')}
                  description={t('settings_notifications_desc')}
                >
                  <Switch
                    checked={notificationsEnabled.value}
                    disabled={saving}
                    label={t('enable_notifications_web')}
                    onChange={toggleNotifications}
                  />
                </Row>

                <Row
                  icon={soundEnabled.value ? <SpeakerHigh size={17} /> : <SpeakerSlash size={17} />}
                  title={t('sound_effects')}
                  description={t('settings_sound_desc')}
                >
                  <Switch
                    checked={soundEnabled.value}
                    disabled={saving}
                    label={t('sound_effects')}
                    onChange={toggleSound}
                  />
                </Row>

                {settings.value.autostart_supported && (
                  <Row
                    icon={<Power size={17} />}
                    title={t('autostart_login')}
                    description={t('settings_autostart_desc')}
                  >
                    <Switch
                      checked={settings.value.autostart_enabled}
                      disabled={saving}
                      label={t('autostart_login')}
                      onChange={toggleAutostart}
                    />
                  </Row>
                )}

                <Row
                  icon={<GlobeHemisphereWest size={17} />}
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
              </div>
            </section>

            <section class="sph-card">
              <header class="px-5 pt-4 pb-2">
                <h2 class="m-0 text-sm font-semibold text-text">{t('appearance_section')}</h2>
              </header>
              <div class="border-t border-border-light">
                <Row
                  icon={<CircleHalf size={17} />}
                  title={t('theme')}
                  description={t('match_system')}
                  last
                  wide
                >
                  <ThemeGrid />
                </Row>
              </div>
            </section>

            {installedProviders.value && installedProviders.value.length > 0 && (
              <InstalledProviders providers={installedProviders.value} />
            )}

            <section class="sph-card">
              <header class="px-5 pt-4 pb-2">
                <h2 class="m-0 text-sm font-semibold text-text">{t('danger_section')}</h2>
              </header>

              <div class="grid grid-cols-[auto_1fr_auto] items-center gap-3 border-t border-border-light px-5 py-3.5">
                <span class="flex size-9 shrink-0 items-center justify-center rounded-[var(--sph-radius-sm)] bg-status-error/15 text-text-error">
                  <Trash size={17} />
                </span>
                <div class="min-w-0">
                  <p class="m-0 text-sm font-medium text-text">{t('reset_connections')}</p>
                  <p class="m-0 text-2xs text-text-muted">{t('reset_connections_desc')}</p>
                </div>
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
              </div>
            </section>

            {version && (
              <footer class="px-1 text-end font-mono text-2xs text-text-muted">v{version}</footer>
            )}
          </div>
        )}
      </div>

    </div>
  );
}

function InstalledProviders({ providers }: { providers: InstalledProvider[] }) {
  async function update(provider: InstalledProvider) {
    try {
      await updateProvider(provider.id);
    } catch (error) {
      toast.error(t('provider_update_failed', { 0: provider.name, 1: (error as Error).message }));
    }
  }

  return (
    <section class="sph-card">
      <header class="px-5 pt-4 pb-2">
        <h2 class="m-0 text-sm font-semibold text-text">{t('installed_providers')}</h2>
        <p class="m-0 mt-0.5 text-2xs text-text-muted">{t('installed_providers_desc')}</p>
      </header>
      <div class="border-t border-border-light">
        {providers.map((provider, index) => (
          <Row
            key={provider.id}
            icon={<Package size={17} />}
            title={provider.name}
            description={
              provider.installed
                ? [
                    t('provider_version', { 0: provider.version || '?' }),
                    provider.hasUpdate && provider.latest
                      ? t('provider_latest', { 0: provider.latest })
                      : '',
                  ]
                    .filter(Boolean)
                    .join(' · ')
                : t('provider_not_installed')
            }
            last={index === providers.length - 1}
          >
            {provider.hasUpdate && provider.updatable ? (
              <Button
                variant="secondary"
                size="sm"
                loading={provider.updating}
                onClick={() => void update(provider)}
              >
                <DownloadSimple size={14} />
                {provider.updating ? t('provider_updating') : t('provider_update')}
              </Button>
            ) : (
              <span />
            )}
          </Row>
        ))}
      </div>
    </section>
  );
}
