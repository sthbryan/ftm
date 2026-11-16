import { signal } from '@preact/signals';

export const now = signal(Date.now());

let timer: ReturnType<typeof setInterval> | null = null;
let subscribers = 0;

/** One shared ticker; the UI unsubscribes when its last timer view unmounts. */
export function useClock(): () => void {
  subscribers += 1;
  if (!timer) {
    timer = setInterval(() => {
      now.value = Date.now();
    }, 1000);
  }

  return () => {
    subscribers -= 1;
    if (subscribers <= 0 && timer) {
      clearInterval(timer);
      timer = null;
      subscribers = 0;
    }
  };
}
