<script lang="ts">
  import { onMount } from "svelte";
  import { useSettings } from "$lib/stores/settings.svelte";
  import { useNotifications } from "$lib/stores/notification.svelte";
  import { useTheme } from "$lib/stores/theme.svelte";
  import {
    i18n,
    translate,
    availableLanguages,
    currentLanguage,
  } from "$lib/i18n";
  import {
    Bell,
    BellOff,
    Volume2,
    VolumeX,
    ChevronLeft,
    Globe,
  } from "lucide-svelte";
  import SettingsSection from "$lib/components/SettingsSection.svelte";
  import SettingRow from "$lib/components/SettingRow.svelte";
  import ThemeSelector from "$lib/components/ThemeSelector.svelte";
  import { themeGroups } from "$lib/data/themes";
  import clsx from "clsx";

  const settingsStore = useSettings();
  const notifications = useNotifications();
  const theme = useTheme();

  let saving = $state(false);
  let t = $derived($translate);

  onMount(async () => {
    theme.init();
    settingsStore.load();
    await i18n.init();
  });

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

  async function changeLanguage(lang: string) {
    saving = true;
    try {
      await i18n.setLanguage(lang);
      await settingsStore.update({ language: lang });
    } finally {
      saving = false;
    }
  }
</script>

<div class="max-w-4xl mx-auto py-8">
  <div class="flex items-center gap-4 mb-8">
    <a
      href="/"
      class="p-2 rounded-lg hover:bg-secondary transition-colors"
      aria-label={t('go_back')}
    >
      <ChevronLeft size={20} />
    </a>
    <h1 class="text-2xl font-semibold">{t("web_settings_title")}</h1>
    {#if saving}
      <div
        class="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin ml-auto"
      ></div>
    {/if}
  </div>

  {#if !settingsStore.loaded || $i18n.loading}
    <div class="flex justify-center py-12">
      <div
        class="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"
      ></div>
    </div>
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <SettingsSection title={t("notifications_section")}>
        {#snippet children()}
          <div class="space-y-3">
            <SettingRow
              icon={BellOff}
              iconActive={Bell}
              active={settingsStore.settings.notifications_enabled ===
                "granted"}
              label={t("enable_notifications_web")}
              disabled={saving}
              onchange={toggleNotifications}
            />
            <SettingRow
              icon={VolumeX}
              iconActive={Volume2}
              active={settingsStore.settings.notification_sound}
              label={t("sound_effects")}
              disabled={saving}
              onchange={toggleSound}
            />
          </div>
        {/snippet}
      </SettingsSection>

      <SettingsSection title={t("appearance_section")}>
        {#snippet children()}
          <ThemeSelector groups={themeGroups} />
        {/snippet}
      </SettingsSection>

      <SettingsSection title={t("language_section")}>
        {#snippet children()}
          <div class="flex gap-2">
            {#each $availableLanguages as lang}
              <button
                class={clsx(
                  "px-4 py-2 rounded-lg border transition-colors cursor-pointer",
                  $currentLanguage === lang
                    ? "border-primary bg-primary/10 text-primary"
                    : "border-border hover:border-primary/50",
                )}
                onclick={() => changeLanguage(lang)}
                disabled={saving}
              >
                <span class="flex items-center gap-2">
                  <Globe size={16} />
                  {t(`lang_${lang}`)}
                </span>
              </button>
            {/each}
          </div>
        {/snippet}
      </SettingsSection>
    </div>
  {/if}
</div>
