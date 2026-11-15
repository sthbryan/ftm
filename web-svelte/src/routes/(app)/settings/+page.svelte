<script lang="ts">
  import { onMount } from "svelte";
  import { useSettings } from "$lib/stores/settings.svelte";
  import { useNotifications } from "$lib/stores/notification.svelte";
  import { useTheme } from "$lib/stores/theme.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { t, useI18n, LANGUAGE_AUTO } from "$lib/stores/i18n.svelte";
  import { statusApi } from "$lib/api/endpoints/status";
  import type { HTTPError } from "ky";
  import {
    Bell,
    BellOff,
    Volume2,
    VolumeX,
    Languages,
    Power,
    PowerOff,
    Trash2,
  } from "lucide-svelte";
  import ToggleTrack from "$lib/components/ToggleTrack.svelte";
  import Button from "$lib/components/Button.svelte";
  import DeleteModal from "$lib/components/DeleteModal.svelte";
  import { rovingRadioKeydown } from "$lib/utils/roving";
  import ThemeSelector from "$lib/components/ThemeSelector.svelte";
  import { themeFamilies } from "$lib/data/themes";
  import { cn } from "$lib/utils/cn";

  const settingsStore = useSettings();
  const notifications = useNotifications();
  const theme = useTheme();
  const tunnels = useTunnels();
  const toast = useToast();
  const i18n = useI18n();

  let saving = $state(false);
  let version = $state("");
  let confirmingReset = $state(false);
  let resetting = $state(false);

  const connectionCount = $derived(tunnels.tunnels.length);
  const resetBody = $derived(
    connectionCount === 1
      ? t("confirm_reset_body_one")
      : t("confirm_reset_body_other", { 0: connectionCount }),
  );

  const languageOptions = $derived([
    { id: LANGUAGE_AUTO, label: t("lang_auto"), native: t("lang_auto_native") },
    ...i18n.available.filter((l) => l !== LANGUAGE_AUTO).map((l) => ({
      id: l,
      label: t(`lang_${l}`),
      native: t(`lang_${l}`),
    })),
  ]);

  onMount(async () => {
    settingsStore.load();
    tunnels.sync();
    await i18n.init();
    statusApi
      .get()
      .then((s) => (version = s.version))
      .catch(() => {});
  });

  async function confirmReset() {
    resetting = true;
    try {
      await tunnels.clearAll();
      confirmingReset = false;
      toast.success(t("connections_cleared"));
    } catch (err) {
      toast.error(t("reset_failed", { 0: (err as Error).message }));
    } finally {
      resetting = false;
    }
  }

  async function toggleNotifications() {
    saving = true;
    try {
      if (settingsStore.settings.notifications_enabled === "granted") {
        await settingsStore.update({ notifications_enabled: "rejected" });
        return;
      }
      await notifications.requestPermission();
      await settingsStore.load();
    } finally {
      saving = false;
    }
  }

  async function toggleSound() {
    saving = true;
    try {
      await settingsStore.update({
        notification_sound: !settingsStore.settings.notification_sound,
      });
    } finally {
      saving = false;
    }
  }

  async function toggleAutostart() {
    saving = true;
    try {
      await settingsStore.update({
        autostart_enabled: !settingsStore.settings.autostart_enabled,
      });
    } catch (err) {
      const status = (err as HTTPError).response?.status;
      toast.error(status === 409 ? t("autostart_blocked") : t("autostart_failed"));
    } finally {
      saving = false;
    }
  }

  async function changeLanguage(lang: string) {
    if (settingsStore.settings.language === lang || saving) return;
    saving = true;
    try {
      await settingsStore.update({ language: lang });
      await i18n.setLanguage(lang);
    } finally {
      saving = false;
    }
  }

  const notifActive = $derived(
    settingsStore.settings.notifications_enabled === "granted",
  );
  const soundActive = $derived(!!settingsStore.settings.notification_sound);
  const autostartActive = $derived(!!settingsStore.settings.autostart_enabled);
  const autostartSupported = $derived(
    !!settingsStore.settings.autostart_supported,
  );
</script>

<div class="mx-auto flex w-full max-w-app min-h-0 flex-1 flex-col">
  <div class="mb-4 flex items-center justify-between gap-3 px-1">
    <h1 class="font-serif text-xl font-bold tracking-tight text-text-heading sm:text-2xl">
      {t("web_settings_title")}
    </h1>
    {#if saving}
      <div
        class="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent"
        aria-hidden="true"
      ></div>
    {/if}
  </div>

  <div class="min-h-0 flex-1 overflow-y-auto pb-2">
    {#if !settingsStore.loaded || !i18n.ready}
      <div class="flex justify-center py-8">
        <div
          class="h-7 w-7 animate-spin rounded-full border-2 border-primary border-t-transparent"
        ></div>
      </div>
    {:else}
      <section
        class="relative sph-card"
      >
        <div class="panel-pattern" aria-hidden="true"></div>
        <header class="relative z-10 px-5 pt-5 pb-3">
          <h2
            class="font-serif text-base font-semibold tracking-tight text-text-heading"
          >
            {t("preferences_section")}
          </h2>
        </header>

        <button
          type="button"
          role="switch"
          aria-checked={notifActive}
          disabled={saving}
          onclick={toggleNotifications}
          class="relative z-10 grid w-full cursor-pointer grid-cols-[auto_1fr_auto] items-center gap-4 border-t border-border-light px-5 py-3.5 text-left transition-colors hover:bg-hover disabled:cursor-not-allowed"
        >
          <span
            class={cn(
              "flex h-9 w-9 shrink-0 items-center justify-center rounded-control transition-colors",
              notifActive ? "bg-primary text-btn-text" : "bg-secondary text-text-muted",
            )}
          >
            {#if notifActive}
              <Bell size={17} />
            {:else}
              <BellOff size={17} />
            {/if}
          </span>
          <span class="min-w-0">
            <span class="block text-sm font-medium text-text-heading">
              {t("enable_notifications_web")}
            </span>
            <span class="block truncate text-xs text-text-muted">
              {t("settings_notifications_desc")}
            </span>
          </span>
          <ToggleTrack checked={notifActive} disabled={saving} />
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={soundActive}
          disabled={saving}
          onclick={toggleSound}
          class="relative z-10 grid w-full cursor-pointer grid-cols-[auto_1fr_auto] items-center gap-4 border-t border-border-light px-5 py-3.5 text-left transition-colors hover:bg-hover disabled:cursor-not-allowed"
        >
          <span
            class={cn(
              "flex h-9 w-9 shrink-0 items-center justify-center rounded-control transition-colors",
              soundActive ? "bg-primary text-btn-text" : "bg-secondary text-text-muted",
            )}
          >
            {#if soundActive}
              <Volume2 size={17} />
            {:else}
              <VolumeX size={17} />
            {/if}
          </span>
          <span class="min-w-0">
            <span class="block text-sm font-medium text-text-heading">
              {t("sound_effects")}
            </span>
            <span class="block truncate text-xs text-text-muted">
              {t("settings_sound_desc")}
            </span>
          </span>
          <ToggleTrack checked={soundActive} disabled={saving} />
        </button>

        {#if autostartSupported}
          <button
            type="button"
            role="switch"
            aria-checked={autostartActive}
            disabled={saving}
            onclick={toggleAutostart}
            class="relative z-10 grid w-full cursor-pointer grid-cols-[auto_1fr_auto] items-center gap-4 border-t border-border-light px-5 py-3.5 text-left transition-colors hover:bg-hover disabled:cursor-not-allowed"
          >
            <span
              class={cn(
                "flex h-9 w-9 shrink-0 items-center justify-center rounded-control transition-colors",
                autostartActive ? "bg-primary text-btn-text" : "bg-secondary text-text-muted",
              )}
            >
              {#if autostartActive}
                <Power size={17} />
              {:else}
                <PowerOff size={17} />
              {/if}
            </span>
            <span class="min-w-0">
              <span class="block text-sm font-medium text-text-heading">
                {t("autostart_login")}
              </span>
              <span class="block truncate text-xs text-text-muted">
                {t("settings_autostart_desc")}
              </span>
            </span>
            <ToggleTrack checked={autostartActive} disabled={saving} />
          </button>
        {/if}

        <div
          class="relative z-10 grid grid-cols-[auto_1fr_auto] items-center gap-4 border-t border-border-light px-5 py-3.5"
        >
              <div
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-secondary text-text-muted"
              >
                <Languages size={17} />
              </div>
              <div class="min-w-0">
                <p class="m-0 text-sm font-medium text-text-heading">
                  {t("language_section")}
                </p>
                <p class="m-0 truncate text-xs text-text-muted">
                  {t("settings_language_desc")}
                </p>
              </div>
            <div
              role="radiogroup"
              aria-label={t("language_section")}
              tabindex={-1}
              onkeydown={rovingRadioKeydown}
              class="flex rounded-control border border-border bg-input-bg p-0.5 max-sm:col-span-2 max-sm:col-start-2"
            >
              {#each languageOptions as opt (opt.id)}
                {@const selected = settingsStore.settings.language === opt.id}
                <button
                  type="button"
                  role="radio"
                  aria-checked={selected}
                  tabindex={selected ? 0 : -1}
                  disabled={saving}
                  onclick={() => changeLanguage(opt.id)}
                  class={cn(
                    "cursor-pointer rounded-sm px-2.5 py-1 text-xs transition-colors",
                    selected
                      ? "bg-primary font-medium text-btn-text"
                      : "text-text-muted hover:text-text",
                    saving && "cursor-not-allowed opacity-60",
                  )}
                >
                  {opt.native}
                </button>
              {/each}
            </div>
        </div>
      </section>

      <section
        class="relative mt-4 sph-card"
      >
        <div class="panel-pattern" aria-hidden="true"></div>
        <header class="relative z-10 px-5 pt-5 pb-3">
          <h2
            class="font-serif text-base font-semibold tracking-tight text-text-heading"
          >
            {t("appearance_section")}
          </h2>
        </header>

        <div class="relative z-10">
          <ThemeSelector families={themeFamilies} />
        </div>
      </section>

      <section
        class="relative mt-4 sph-card border-status-error/40"
      >
        <div class="panel-pattern" aria-hidden="true"></div>
        <header class="relative z-10 px-5 pt-5 pb-3">
          <h2
            class="font-serif text-base font-semibold tracking-tight text-text-heading"
          >
            {t("danger_section")}
          </h2>
        </header>

        <div
          class="relative z-10 grid grid-cols-[auto_1fr_auto] items-center gap-4 border-t border-border-light px-5 py-3.5"
        >
          <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-status-error/15 text-text-error"
          >
            <Trash2 size={17} />
          </div>
          <div class="min-w-0">
            <p class="m-0 text-sm font-medium text-text-heading">
              {t("reset_connections")}
            </p>
            <p class="m-0 text-xs text-text-muted">
              {t("reset_connections_desc")}
            </p>
          </div>
          <Button
            variant="error"
            disabled={connectionCount === 0 || resetting}
            onclick={() => (confirmingReset = true)}
            class="max-sm:col-span-2 max-sm:col-start-2 max-sm:justify-self-start"
          >
            {t("delete")}
          </Button>
        </div>
      </section>

      {#if version}
        <footer class="mt-4 px-1 text-right font-mono text-2xs text-text-muted">
          v{version}
        </footer>
      {/if}
    {/if}
  </div>
</div>

<DeleteModal
  show={confirmingReset}
  title={t("confirm_reset_title")}
  body={resetBody}
  onConfirm={confirmReset}
  onCancel={() => (confirmingReset = false)}
/>