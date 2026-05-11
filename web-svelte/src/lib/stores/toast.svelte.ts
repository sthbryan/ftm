import { toast as sonnerToast } from 'svelte-sonner';

export type ToastType = 'success' | 'error' | 'info' | 'warning' | 'alert';

export const toast = {
  show: (message: string, type: ToastType = "info", duration?: number) => {
    const options = duration ? { duration } : {};
    switch (type) {
      case 'success': return sonnerToast.success(message, options);
      case 'error': return sonnerToast.error(message, options);
      case 'warning': return sonnerToast.warning(message, options);
      default: return sonnerToast(message, options);
    }
  },
  success: (msg: string, duration?: number) => sonnerToast.success(msg, duration ? { duration } : {}),
  error: (msg: string, duration?: number) => sonnerToast.error(msg, duration ? { duration } : {}),
  info: (msg: string, duration?: number) => sonnerToast.info(msg, duration ? { duration } : {}),
  warning: (msg: string, duration?: number) => sonnerToast.warning(msg, duration ? { duration } : {}),
  remove: (id?: string | number) => id && sonnerToast.dismiss(id),
};

export function useToast() { return toast; }
