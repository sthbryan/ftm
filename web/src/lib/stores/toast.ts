import { springToast } from '@zovaris/sephiro/toast';

export type ToastType = 'success' | 'error' | 'info' | 'warning' | 'alert';

function options(duration?: number) {
  return duration ? { duration } : undefined;
}

/**
 * Sephiro's spring toasts are the only toast surface in the app, so the tone
 * vocabulary stays the one the backend already emits.
 */
export const toast = {
  show(message: string, type: ToastType = 'info', duration?: number) {
    switch (type) {
      case 'success':
        return springToast.success(message, options(duration));
      case 'error':
        return springToast.error(message, options(duration));
      case 'warning':
      case 'alert':
        return springToast.warning(message, options(duration));
      default:
        return springToast.info(message, options(duration));
    }
  },
  success: (message: string, duration?: number) =>
    springToast.success(message, options(duration)),
  error: (message: string, duration?: number) =>
    springToast.error(message, options(duration)),
  info: (message: string, duration?: number) =>
    springToast.info(message, options(duration)),
  warning: (message: string, duration?: number) =>
    springToast.warning(message, options(duration)),
};
