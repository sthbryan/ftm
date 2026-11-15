<script lang="ts">
  import { Plus } from "lucide-svelte";
  import Button from "./Button.svelte";
  import Mark from "./Mark.svelte";
  import { t } from "$lib/stores/i18n.svelte";

  let { onCreate }: { onCreate: () => void } = $props();

  const steps = [
    ["welcome_step_provider_title", "welcome_step_provider_body"],
    ["welcome_step_port_title", "welcome_step_port_body"],
    ["welcome_step_share_title", "welcome_step_share_body"],
  ];
</script>

<main class="flex min-h-0 flex-1 items-center justify-center py-6">
  <section class="ftm-enter sph-card w-full max-w-[34rem]">
    <div class="sph-empty-state">
      <span class="sph-empty-state__icon">
        <Mark class="h-6 w-6" />
      </span>

      <h2 class="sph-empty-state__title">{t("welcome_title")}</h2>
      <p class="sph-empty-state__description">{t("welcome_lead")}</p>

      <ol
        class="mx-auto mt-2 grid w-full max-w-[26rem] list-none gap-3 p-0 text-start"
      >
        {#each steps as [title, body], index (title)}
          <li class="flex items-start gap-2.5">
            <span
              class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-primary font-mono text-2xs font-semibold text-btn-text"
              aria-hidden="true"
            >
              {index + 1}
            </span>
            <div class="min-w-0">
              <p class="m-0 text-xs font-medium text-text-heading">
                {t(title)}
              </p>
              <p class="m-0 text-2xs leading-relaxed text-text-muted">
                {t(body)}
              </p>
            </div>
          </li>
        {/each}
      </ol>

      <div class="sph-empty-state__action flex flex-col items-center gap-2">
        <Button variant="primary" size="lg" icon={Plus} onclick={onCreate}>
          {t("create_first")}
        </Button>
        <p class="m-0 text-2xs text-text-muted">{t("list_hint_new")}</p>
      </div>
    </div>
  </section>
</main>
