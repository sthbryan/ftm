import { describe, it, expect } from 'vitest';
import { extractErrorMessage } from './client';

describe('extractErrorMessage', () => {
  it('keeps a plain text body', () => {
    expect(
      extractErrorMessage('tunnelmole requires Rosetta 2 to run on Apple Silicon\n')
    ).toBe('tunnelmole requires Rosetta 2 to run on Apple Silicon');
  });

  it('unwraps an error field', () => {
    expect(extractErrorMessage('{"error":"could not connect to bore.pub:7835"}')).toBe(
      'could not connect to bore.pub:7835'
    );
  });

  it('unwraps a message field', () => {
    expect(extractErrorMessage('{"message":"tunnel is already running"}')).toBe(
      'tunnel is already running'
    );
  });

  it('falls back to the raw body for json without a message', () => {
    expect(extractErrorMessage('{"code":500}')).toBe('{"code":500}');
  });

  it('reads an already parsed body object', () => {
    expect(extractErrorMessage({ error: 'tunnel tunnel-1 is not running' })).toBe(
      'tunnel tunnel-1 is not running'
    );
  });

  it('returns nothing for an empty body', () => {
    expect(extractErrorMessage('   ')).toBe('');
  });

  it('returns nothing when there is no body at all', () => {
    expect(extractErrorMessage(undefined)).toBe('');
  });
});
