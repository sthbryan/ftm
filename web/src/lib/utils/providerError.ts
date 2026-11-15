const PATTERNS: Array<[RegExp, string]> = [
  [/executable file not found|no such file or directory|not installed|command not found/i, 'error_hint_binary'],
  [/connection refused|econnrefused|failed to connect to localhost|dial tcp .*: connect/i, 'error_hint_port'],
  [/429|rate.?limit|too many requests|quota/i, 'error_hint_rate_limit'],
  [/no such host|dns|network is unreachable|i\/o timeout|tls handshake/i, 'error_hint_network'],
  [/unauthorized|forbidden|401|403|invalid token|authentication/i, 'error_hint_auth'],
  [/address already in use|port .* in use/i, 'error_hint_port_taken'],
];

export function providerErrorHint(message: string | undefined): string | null {
  if (!message) return null;
  for (const [pattern, key] of PATTERNS) {
    if (pattern.test(message)) return key;
  }
  return null;
}
