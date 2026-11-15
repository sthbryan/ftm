export function sanitizeLogs(text: string): string {
  const ansiPattern = /\x1B\[[0-9;]*[a-zA-Z]/g;
  return text.replace(ansiPattern, "").trim();
}

export function formatLogs(logs: string): string {
  return logs.split("\n").map(sanitizeLogs).join("\n");
}
