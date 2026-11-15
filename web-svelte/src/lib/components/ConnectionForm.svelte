<script lang="ts">
  import { Plus } from "lucide-svelte";
  import { onMount } from "svelte";
  import { useProviders, detectPort } from "$lib/stores/providers.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { t } from "$lib/stores/i18n.svelte";
  import Button from "./Button.svelte";
  import Dropdown from "./Dropdown.svelte";
  import type { DropdownOption } from "$lib/types";

  type Props = {
    mode?: "create" | "edit";
    tunnelId?: string | null;
    onCancel?: () => void;
    onSaved?: () => void;
  };

  let { mode = "create", tunnelId = null, onCancel, onSaved }: Props = $props();

  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();

  let formData = $state({
    name: "",
    provider: "cloudflared",
    localPort: 30000,
  });
  let loadedEditId = $state("");
  let submitting = $state(false);

  const isEdit = $derived(mode === "edit" && !!tunnelId);

  const providerOptions: DropdownOption[] = $derived(
    providerStore.providers.map((p) => ({
      label: p.name,
      hint: p.id === "cloudflared" ? t("provider_recommended") : undefined,
      value: p.id,
    })),
  );

  const selectedProvider = $derived(
    providerOptions.find((p) => p.value === formData.provider),
  );

  const defaultProvider = $derived(
    providerStore.providers[0]?.id ?? "cloudflared",
  );

  onMount(async () => {
    if (!isEdit) {
      const port = await detectPort();
      formData.localPort = port;
      formData.provider = defaultProvider;
    }
  });

  $effect(() => {
    if (!isEdit || !tunnelId) return;
    if (loadedEditId === tunnelId) return;

    const tunnel = store.getById(tunnelId);
    if (!tunnel) return;

    loadedEditId = tunnelId;
    formData = {
      name: tunnel.name || "",
      provider: tunnel.provider || defaultProvider,
      localPort: tunnel.port || 30000,
    };

    if (!tunnel.port) {
      detectPort().then((port) => {
        if (loadedEditId === tunnelId) {
          formData = { ...formData, localPort: port };
        }
      });
    }
  });

  function selectProvider(option: DropdownOption) {
    if (option.value) formData.provider = option.value;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (submitting) return;
    submitting = true;

    const name = formData.name;

    try {
      if (isEdit && tunnelId) {
        await store.update(tunnelId, {
          name: formData.name,
          provider: formData.provider,
          localPort: formData.localPort,
        });
        toast.success(t("connection_updated", { name }));
        onSaved?.();
      } else {
        await store.create({ ...formData });
        const detectedPort = await detectPort({ forceRefresh: true });
        formData = {
          name: "",
          provider: defaultProvider,
          localPort: detectedPort,
        };
        toast.success(t("connection_created", { name }));
        onSaved?.();
      }
    } catch (err) {
      const key = isEdit ? "connection_update_failed" : "connection_create_failed";
      toast.error(t(key, { error: (err as Error).message }));
    } finally {
      submitting = false;
    }
  }
</script>

<form onsubmit={handleSubmit} class="contents">
  <div class="sph-dialog__header">
    <div class="sph-dialog__heading">
      <h2 class="sph-dialog__title">
        {isEdit ? t("edit_connection") : t("new_connection")}
      </h2>
    </div>
    {#if onCancel}
      <button
        type="button"
        class="sph-dialog__close"
        aria-label={t("cancel")}
        onclick={onCancel}
      >
        ×
      </button>
    {/if}
  </div>

  <div class="sph-dialog__content">
    {#if providerStore.error}
      <p
        class="m-0 mb-3 rounded-[var(--sph-radius-sm)] border border-status-error/40 bg-status-error/10 px-2.5 py-1.5 text-2xs text-text-error"
      >
        {providerStore.error}
      </p>
    {/if}

    <div class="sph-field">
      <label class="sph-field__label" for="conn-name">
        {isEdit ? t("connection_name_label") : t("tunnel_name")}
      </label>
      <div class="sph-field__control">
        <input
          type="text"
          id="conn-name"
          bind:value={formData.name}
          placeholder={isEdit ? t("name_placeholder") : t("tunnel_name_hint")}
          required
          autocomplete="off"
          class="sph-input"
        />
      </div>
    </div>

    <div class="mt-3 grid grid-cols-2 gap-3">
      <div class="sph-field">
        <label class="sph-field__label" for="conn-port">
          {isEdit ? t("port") : t("local_port")}
        </label>
        <div class="sph-field__control">
          <input
            type="number"
            id="conn-port"
            bind:value={formData.localPort}
            min="1"
            max="65535"
            required
            class="sph-input"
          />
        </div>
      </div>
      <div class="sph-field">
        <label class="sph-field__label" for="conn-provider">
          {isEdit ? t("provider_label") : t("select_provider")}
        </label>
        <div class="sph-field__control">
          <Dropdown
            id="conn-provider"
            class="w-full"
            options={providerOptions}
            onSelect={selectProvider}
            align="left"
            ariaLabel={t("select_provider")}
            label={selectedProvider?.label || t("select")}
          />
        </div>
      </div>
    </div>

    {#if !isEdit}
      <div
        class="mt-4 rounded-[var(--sph-radius-sm)] border border-border-light bg-bg/40 p-3 text-2xs leading-relaxed text-text-muted"
      >
        <p class="m-0 mb-1.5 font-semibold text-text-heading">
          {t("form_tips_title")}
        </p>
        <ul class="m-0 grid list-none gap-1 p-0">
          <li class="flex gap-2">
            <span aria-hidden="true">·</span>
            <span>{t("form_tips_port")}</span>
          </li>
          <li class="flex gap-2">
            <span aria-hidden="true">·</span>
            <span>{t("form_tips_provider")}</span>
          </li>
          <li class="flex gap-2">
            <span aria-hidden="true">·</span>
            <span>{t("form_tips_name")}</span>
          </li>
        </ul>
      </div>
    {/if}
  </div>

  <div class="sph-dialog__footer">
    {#if isEdit}
      <Button variant="default" type="button" onclick={onCancel}>
        {t("cancel")}
      </Button>
      <Button variant="primary" type="submit" disabled={submitting}>
        {t("save")}
      </Button>
    {:else}
      <Button variant="primary" type="submit" icon={Plus} disabled={submitting}>
        {t("create_connection")}
      </Button>
    {/if}
  </div>
</form>
