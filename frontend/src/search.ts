import type { Item } from './api'

export function searchWords(query: string): string[] {
  return normalize(query)
    .split(/\s+/)
    .filter((word) => word !== '')
}

export function matchesSearch(item: Item, categoryLabel: string, words: string[]): boolean {
  const text = normalize(`${item.name} ${item.description} ${categoryLabel}`)
  return words.every((word) => text.includes(word))
}

function normalize(text: string): string {
  return text.toLowerCase().replaceAll('ё', 'е')
}
