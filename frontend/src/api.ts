import { getInitData } from './telegram'

export type Option = {
  value: string
  label: string
}

export type CategoryOption = Option & {
  has_warmth: boolean
}

export type WarmthLevelOption = {
  value: number
  label: string
}

export type Options = {
  categories: CategoryOption[]
  colors: Option[]
  seasons: Option[]
  current_season: string
  warmth_levels: WarmthLevelOption[]
  statuses: Option[]
  limits: {
    name: number
    description: number
    photo_bytes: number
    photo_types: string[]
    min_city_query: number
  }
}

export type ItemFields = {
  name: string
  description: string
  category: string
  main_color: string
  extra_colors: string[]
  seasons: string[]
  warmth_level: number
  waterproof: boolean
  photo_key: string
}

export type Item = ItemFields & {
  id: number
  status: string
  photo_url: string
}

// Пустое поле - не распозналось.
export type Suggestion = {
  name: string
  category: string
  main_color: string
  extra_colors: string[]
  seasons: string[]
  warmth_level: number
  waterproof: boolean
}

export const availableStatus = 'available'
export const dirtyStatus = 'dirty'
export const archivedStatus = 'archived'

export type City = {
  name: string
  region: string
  latitude: number
  longitude: number
  timezone: string
}

export type Settings = {
  morning_enabled: boolean
  send_at: string
  city: City | null
}

export type Outfit = {
  items: Item[]
  notes: string[]
}

export type Recommendation = {
  city: City
  weather: {
    temperature: string
    details: string[]
  }
  outfits: Outfit[]
  notes: string[]
}

export type ApiErrorCode =
  | 'unauthorized'
  | 'bad_request'
  | 'invalid_item'
  | 'not_found'
  | 'wardrobe_full'
  | 'invalid_location'
  | 'location_not_set'
  | 'invalid_settings'
  | 'weather_unavailable'
  | 'photo_too_large'
  | 'photo_type_unsupported'
  | 'photo_not_uploaded'
  | 'photo_upload_failed'
  | 'recognition_unavailable'
  | 'recognition_limit'
  | 'internal'
  | 'network'

const serverErrorCodes: ApiErrorCode[] = [
  'unauthorized',
  'bad_request',
  'invalid_item',
  'not_found',
  'wardrobe_full',
  'invalid_location',
  'location_not_set',
  'invalid_settings',
  'weather_unavailable',
  'photo_too_large',
  'photo_type_unsupported',
  'photo_not_uploaded',
  'recognition_unavailable',
  'recognition_limit',
  'internal',
]

export class ApiError extends Error {
  readonly code: ApiErrorCode

  constructor(code: ApiErrorCode) {
    super(code)
    this.code = code
  }
}

export function labelOf(options: { value: string | number; label: string }[], value: string | number): string {
  return options.find((option) => option.value === value)?.label ?? String(value)
}

export function fetchOptions(): Promise<Options> {
  return request<Options>('GET', '/api/options')
}

export async function fetchItems(): Promise<Item[]> {
  const body = await request<{ items: Item[] }>('GET', '/api/items')
  return body.items
}

export function createItem(fields: ItemFields): Promise<Item> {
  return request<Item>('POST', '/api/items', fields)
}

export function updateItem(itemId: number, fields: ItemFields): Promise<Item> {
  return request<Item>('PUT', `/api/items/${itemId}`, fields)
}

export function changeItemStatus(itemId: number, status: string): Promise<void> {
  return request<void>('PUT', `/api/items/${itemId}/status`, { status })
}

export function deleteItems(itemIds: number[]): Promise<void> {
  return request<void>('POST', '/api/items/delete', { ids: itemIds })
}

export type UploadedPhoto = {
  key: string
  url: string
}

export async function uploadPhoto(file: File): Promise<UploadedPhoto> {
  const upload = await request<{ key: string; upload_url: string; view_url: string }>('POST', '/api/photos', {
    content_type: file.type,
    size: file.size,
  })

  let response: Response
  try {
    response = await fetch(upload.upload_url, { method: 'PUT', body: file, headers: { 'Content-Type': file.type } })
  } catch {
    throw new ApiError('network')
  }
  if (!response.ok) {
    throw new ApiError('photo_upload_failed')
  }
  return { key: upload.key, url: upload.view_url }
}

export function recognizePhoto(photoKey: string): Promise<Suggestion> {
  return request<Suggestion>('POST', '/api/photos/recognize', { photo_key: photoKey })
}

export function fetchRecommendation(): Promise<Recommendation> {
  return request<Recommendation>('GET', '/api/recommendation')
}

export async function fetchTodayOutfit(): Promise<number[]> {
  const body = await request<{ item_ids: number[] }>('GET', '/api/outfits/today')
  return body.item_ids
}

export function wearToday(itemIds: number[]): Promise<void> {
  return request<void>('PUT', '/api/outfits/today', { item_ids: itemIds })
}

export type Candidate = {
  item_id: number
  notes: string[]
}

// replace = null - что добавить в образ, иначе - чем заменить эту вещь.
export async function fetchCandidates(outfit: number[], replace: number | null): Promise<Candidate[]> {
  const query = new URLSearchParams({ items: outfit.join(',') })
  if (replace !== null) {
    query.set('replace', String(replace))
  }
  const body = await request<{ candidates: Candidate[] }>('GET', `/api/outfits/candidates?${query}`)
  return body.candidates
}

export async function reviewOutfit(itemIds: number[]): Promise<string[]> {
  const query = new URLSearchParams({ items: itemIds.join(',') })
  const body = await request<{ notes: string[] }>('GET', `/api/outfits/review?${query}`)
  return body.notes
}

export async function searchCities(query: string): Promise<City[]> {
  const body = await request<{ cities: City[] }>('GET', `/api/cities?${new URLSearchParams({ query })}`)
  return body.cities
}

export function saveCity(city: City): Promise<void> {
  return request<void>('PUT', '/api/city', city)
}

export function fetchSettings(): Promise<Settings> {
  return request<Settings>('GET', '/api/settings')
}

export function saveSettings(settings: { morning_enabled: boolean; send_at: string }): Promise<void> {
  return request<void>('PUT', '/api/settings', settings)
}

async function request<T>(method: 'GET' | 'POST' | 'PUT', path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { Authorization: `tma ${getInitData()}` }
  const init: RequestInit = { method, headers }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  let response: Response
  try {
    response = await fetch(path, init)
  } catch {
    throw new ApiError('network')
  }

  if (!response.ok) {
    const payload: { error?: string } | null = await response.json().catch(() => null)
    throw new ApiError(serverErrorCodes.find((code) => code === payload?.error) ?? 'internal')
  }
  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}
