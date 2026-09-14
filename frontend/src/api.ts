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
  warmth_levels: WarmthLevelOption[]
  limits: {
    name: number
    description: number
  }
}

export type NewItem = {
  name: string
  description: string
  category: string
  main_color: string
  extra_colors: string[]
  seasons: string[]
  warmth_level: number
  waterproof: boolean
}

export type Item = NewItem & {
  id: number
  status: string
}

export type ApiErrorCode = 'unauthorized' | 'bad_request' | 'invalid_item' | 'wardrobe_full' | 'internal' | 'network'

const serverErrorCodes: ApiErrorCode[] = ['unauthorized', 'bad_request', 'invalid_item', 'wardrobe_full', 'internal']

export class ApiError extends Error {
  readonly code: ApiErrorCode

  constructor(code: ApiErrorCode) {
    super(code)
    this.code = code
  }
}

export function fetchOptions(): Promise<Options> {
  return request<Options>('GET', '/api/options')
}

export function createItem(item: NewItem): Promise<Item> {
  return request<Item>('POST', '/api/items', item)
}

async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<T> {
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
  return (await response.json()) as T
}
