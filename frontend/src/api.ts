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
}

export type Item = ItemFields & {
  id: number
  status: string
}

export const availableStatus = 'available'

export type ApiErrorCode =
  | 'unauthorized'
  | 'bad_request'
  | 'invalid_item'
  | 'not_found'
  | 'wardrobe_full'
  | 'internal'
  | 'network'

const serverErrorCodes: ApiErrorCode[] = ['unauthorized', 'bad_request', 'invalid_item', 'not_found', 'wardrobe_full', 'internal']

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
