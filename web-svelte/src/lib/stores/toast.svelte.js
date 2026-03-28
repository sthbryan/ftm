let toasts = $state([]);

export function useToast() {
  return {
    get toasts() { return toasts; },
    
    show(message, type = 'info', duration = 3000) {
      const id = Date.now() + Math.random();
      const toast = { id, message, type, duration };
      toasts = [...toasts, toast];
      
      setTimeout(() => {
        toasts = toasts.filter(t => t.id !== id);
      }, duration);
    },
    
    success(message, duration) {
      this.show(message, 'success', duration);
    },
    
    error(message, duration) {
      this.show(message, 'error', duration);
    },
    
    info(message, duration) {
      this.show(message, 'info', duration);
    },
    
    remove(id) {
      toasts = toasts.filter(t => t.id !== id);
    }
  };
}
