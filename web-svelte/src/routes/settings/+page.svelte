<script lang="ts">
  import { onMount } from 'svelte';
  import { useSettings } from '$lib/stores/settings.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import { Bell, BellOff, Volume2, VolumeX, ChevronLeft } from 'lucide-svelte';
  import SettingsSection from '$lib/components/SettingsSection.svelte';
  import SettingRow from '$lib/components/SettingRow.svelte';
  import ThemeSelector from '$lib/components/ThemeSelector.svelte';
  import { themeGroups } from '$lib/data/themes';

  const settingsStore = useSettings();
  const theme = useTheme();

  let saving = $state(false);

  onMount(() => {
    theme.init();
    settingsStore.load();
  });

  async function toggleNotifications() {
    saving = true;
    try {
      await settingsStore.update({ notifications_enabled: !settingsStore.settings.notifications_enabled });
    } finally {
      saving = false;
    }
  }

  async function toggleSound() {
    saving = true;
    try {
      await settingsStore.update({ notification_sound: !settingsStore.settings.notification_sound });
    } finally {
      saving = false;
    }
  }
</script>

<div class="max-w-4xl mx-auto py-8">
  <div class="flex items-center gap-4 mb-8">
    <a href="/" class="p-2 rounded-lg hover:bg-secondary transition-colors" aria-label="Go back">
      <ChevronLeft size={20} />
    </a>
    <h1 class="text-2xl font-semibold">Settings</h1>
    {#if saving}
      <div class="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin ml-auto"></div>
    {/if}
  </div>

  {#if !settingsStore.loaded}
    <div class="flex justify-center py-12">
      <div class="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <SettingsSection title="Notifications">
        {#snippet children()}
          <div class="space-y-3">
            <SettingRow
              icon={BellOff}
              iconActive={Bell}
              active={settingsStore.settings.notifications_enabled}
              label="Enable Notifications"
              disabled={saving}
              onchange={toggleNotifications}
            />
            <SettingRow
              icon={VolumeX}
              iconActive={Volume2}
              active={settingsStore.settings.notification_sound}
              label="Sound Effects"
              disabled={saving}
              onchange={toggleSound}
            />
          </div>
        {/snippet}
      </SettingsSection>

      <SettingsSection title="Appearance">
        {#snippet children()}
          <ThemeSelector groups={themeGroups} />
        {/snippet}
      </SettingsSection>
    </div>
  {/if}
</div>
