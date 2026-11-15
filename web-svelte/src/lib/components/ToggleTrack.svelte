<script lang="ts">
  let {
    checked = false,
    disabled = false,
  }: {
    checked?: boolean;
    disabled?: boolean;
  } = $props();
</script>

<span
  aria-hidden="true"
  class="sph-toggle"
  data-size="sm"
  data-state={checked ? "checked" : "unchecked"}
  data-disabled={disabled ? "true" : undefined}
>
  <span class="sph-toggle__track">
    <span class="sph-toggle__thumb"></span>
  </span>
</span>
