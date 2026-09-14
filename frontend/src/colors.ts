const swatches: Record<string, string> = {
  red: '#d93025',
  burgundy: '#800020',
  blue: '#1a56db',
  light_blue: '#8ecdf7',
  navy: '#1f2a44',
  green: '#2e7d32',
  khaki: '#a89f68',
  yellow: '#f2c200',
  black: '#111111',
  white: '#ffffff',
  gray: '#9e9e9e',
  brown: '#6d4c41',
  beige: '#e8d9b5',
  orange: '#f57c00',
  purple: '#7b1fa2',
  pink: '#f48fb1',
}

export function swatch(color: string): string {
  return swatches[color] ?? '#bdbdbd'
}
