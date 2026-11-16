import { useEffect, useRef, useState } from 'preact/hooks';

/**
 * Keeps items that just left a list around for `ms`, flagged as leaving, so
 * they can animate out instead of disappearing in one frame. New and current
 * items come back in their list order; leaving ones stay where they were.
 */
export function useLeaving<T>(items: T[], key: (item: T) => string, ms: number) {
  const previous = useRef<T[]>(items);
  const [leaving, setLeaving] = useState<Map<string, { item: T; index: number }>>(new Map());

  useEffect(() => {
    const now = new Set(items.map(key));
    const gone = previous.current
      .map((item, index) => ({ item, index }))
      .filter(({ item }) => !now.has(key(item)));
    previous.current = items;
    if (gone.length === 0) return;

    setLeaving((current) => {
      const next = new Map(current);
      for (const entry of gone) next.set(key(entry.item), entry);
      return next;
    });
    const timer = setTimeout(() => {
      setLeaving((current) => {
        const next = new Map(current);
        for (const entry of gone) next.delete(key(entry.item));
        return next;
      });
    }, ms);
    return () => clearTimeout(timer);
  }, [items.map(key).join('|')]);

  const merged: { item: T; leaving: boolean }[] = items.map((item) => ({ item, leaving: false }));
  for (const [id, entry] of leaving) {
    if (items.some((item) => key(item) === id)) continue;
    merged.splice(Math.min(entry.index, merged.length), 0, { item: entry.item, leaving: true });
  }
  return merged;
}
