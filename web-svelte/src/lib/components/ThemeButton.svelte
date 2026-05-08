<script lang="ts">
  interface Props {
    id: string;
    color: string;
    selected: boolean;
    label: string;
    onclick?: () => void;
  }

  let { id, color, selected, label, onclick }: Props = $props();
</script>

<button
  type="button"
  onclick={onclick}
  class="group relative cursor-pointer w-10 h-10 rounded-full transition-all duration-200 hover:scale-110 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-card"
  style={selected ? `box-shadow: 0 0 0 3px var(--color-bg), 0 0 0 5px ${color};` : ''}
  aria-label={label}
  title={label}
>
  <div 
    class="w-full h-full rounded-full"
    style="background: {color}; box-shadow: 0 2px 8px {color}50;"
  ></div>
</button>
