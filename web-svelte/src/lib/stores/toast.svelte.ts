import type { Toast, ToastType } from '$lib/types';

let toasts: Toast[] = $state([]);

export function useToast() {
  return {
    get toasts() { return toasts; },
    
    show(message: string, type: ToastType = 'info', duration: number = 3000) {
      const id = Date.now() + Math.random();
      const toast: Toast = { id, message, type };
      toasts = [...toasts, toast];
      
      setTimeout(() => {
        toasts = toasts.filter(t => t.id !== id);
      }, duration);
    },
    
    success(message: string, duration?: number) {
      this.show(message, 'success', duration);
    },
    
    error(message: string, duration?: number) {
      this.show(message, 'error', duration);
    },
    
    info(message: string, duration?: number) {
      this.show(message, 'info', duration);
    },
    
    remove(id: number) {
      toasts = toasts.filter(t => t.id !== id);
    }
  };
}
