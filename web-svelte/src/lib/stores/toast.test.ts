import { describe, it, expect, vi, beforeEach } from 'vitest';
import { toast } from './toast.svelte';

vi.mock('svelte-sonner', () => {
  const fn = vi.fn((message: string, opts?: object) => ({ message, opts }));
  return {
    toast: Object.assign(fn, {
      success: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
      dismiss: vi.fn(),
    }),
  };
});

import { toast as sonnerToast } from 'svelte-sonner';

describe('toast store', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('success()', () => {
    it('calls sonner toast.success with message', () => {
      toast.success('Test message');
      expect(sonnerToast.success).toHaveBeenCalledWith('Test message', {});
    });

    it('passes duration option when provided', () => {
      toast.success('Test', 5000);
      expect(sonnerToast.success).toHaveBeenCalledWith('Test', { duration: 5000 });
    });
  });

  describe('error()', () => {
    it('calls sonner toast.error with message', () => {
      toast.error('Error message');
      expect(sonnerToast.error).toHaveBeenCalledWith('Error message', {});
    });

    it('passes duration option when provided', () => {
      toast.error('Error', 3000);
      expect(sonnerToast.error).toHaveBeenCalledWith('Error', { duration: 3000 });
    });
  });

  describe('info()', () => {
    it('calls sonner toast.info with message', () => {
      toast.info('Info message');
      expect(sonnerToast.info).toHaveBeenCalledWith('Info message', {});
    });

    it('passes duration option when provided', () => {
      toast.info('Info', 2000);
      expect(sonnerToast.info).toHaveBeenCalledWith('Info', { duration: 2000 });
    });
  });

  describe('warning()', () => {
    it('calls sonner toast.warning with message', () => {
      toast.warning('Warning message');
      expect(sonnerToast.warning).toHaveBeenCalledWith('Warning message', {});
    });

    it('passes duration option when provided', () => {
      toast.warning('Warn', 4000);
      expect(sonnerToast.warning).toHaveBeenCalledWith('Warn', { duration: 4000 });
    });
  });

  describe('show()', () => {
    it('maps success type to sonner.success', () => {
      toast.show('Message', 'success');
      expect(sonnerToast.success).toHaveBeenCalled();
      expect(sonnerToast.success).toHaveBeenCalledWith('Message', {});
    });

    it('maps error type to sonner.error', () => {
      toast.show('Message', 'error');
      expect(sonnerToast.error).toHaveBeenCalled();
      expect(sonnerToast.error).toHaveBeenCalledWith('Message', {});
    });

    it('maps warning type to sonner.warning', () => {
      toast.show('Message', 'warning');
      expect(sonnerToast.warning).toHaveBeenCalled();
      expect(sonnerToast.warning).toHaveBeenCalledWith('Message', {});
    });

    it('maps info type to sonner (default)', () => {
      toast.show('Message', 'info');
      expect(sonnerToast).toHaveBeenCalled();
      expect(sonnerToast).toHaveBeenCalledWith('Message', {});
    });

    it('passes duration option to sonner', () => {
      toast.show('Message', 'success', 6000);
      expect(sonnerToast.success).toHaveBeenCalledWith('Message', { duration: 6000 });
    });
  });

  describe('remove()', () => {
    it('calls sonner dismiss with id', () => {
      toast.remove(123);
      expect(sonnerToast.dismiss).toHaveBeenCalledWith(123);
    });

    it('calls sonner dismiss with string id', () => {
      toast.remove('abc-123');
      expect(sonnerToast.dismiss).toHaveBeenCalledWith('abc-123');
    });

    it('does not call dismiss when id is undefined', () => {
      toast.remove(undefined);
      expect(sonnerToast.dismiss).not.toHaveBeenCalled();
    });
  });
});