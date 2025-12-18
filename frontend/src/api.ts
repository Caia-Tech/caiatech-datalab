export type Split = 'train' | 'valid' | 'test'
export type ConversationStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'archived'
export type ProposalStatus = 'pending' | 'approved' | 'rejected'
export type Role = 'system' | 'user' | 'assistant'

export type Message = {
  role: Role
  content: string
  name?: string
  meta?: any
}

export type Dataset = {
  id: number
  name: string
  description: string
  kind: string
  item_count: number
  conversation_count: number
  created_at: string
  updated_at: string
}

export type DatasetItem = {
  id: number
  dataset_id: number
  data: any
  source_ref: string
  created_at: string
  updated_at: string
}

export type Conversation = {
  id: number
  dataset_id: number
  split: Split
  status: ConversationStatus
  tags: string[]
  source: string
  notes: string
  created_at: string
  updated_at: string
  message_count?: number
  preview_user?: string
  preview_assistant?: string
  messages?: Message[]
}

export type Proposal = {
  id: number
  payload: any
  status: ProposalStatus
  created_at: string
  decided_at: string | null
}

// Prefer relative API calls (so the UI works when served from another machine).
// For local dev you can set VITE_API_BASE_URL=http://localhost:8080
const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''

export function apiBase(): string {
  return (API_BASE || '').replace(/\/$/, '')
}

function apiUrl(path: string): string {
  const base = apiBase()
  if (!base) return path
  if (!path.startsWith('/')) return base + '/' + path
  return base + path
}

function toURL(path: string): URL {
  const base = apiUrl(path)
  return new URL(base, base.startsWith('http') ? undefined : window.location.origin)
}

// Datasets
export async function listDatasets(params: { q?: string; limit?: number; offset?: number }): Promise<{ items: Dataset[]; limit: number; offset: number }> {
  const url = toURL('/api/v1/datasets')
  if (params.q) url.searchParams.set('q', params.q)
  if (params.limit != null) url.searchParams.set('limit', String(params.limit))
  if (params.offset != null) url.searchParams.set('offset', String(params.offset))

  const res = await fetch(url.toString())
  if (!res.ok) throw new Error('failed to list datasets')
  return res.json()
}

export async function getDataset(id: number): Promise<Dataset> {
  const res = await fetch(apiUrl(`/api/v1/datasets/${id}`))
  if (!res.ok) throw new Error('failed to get dataset')
  return res.json()
}

export async function createDataset(body: { name: string; description?: string; kind?: string }, adminToken: string): Promise<Dataset> {
  const res = await fetch(apiUrl('/api/v1/datasets'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify({ name: body.name, description: body.description ?? '', kind: body.kind ?? '' })
  })
  if (!res.ok) throw new Error('failed to create dataset')
  return res.json()
}

export async function updateDataset(id: number, body: { name?: string; description?: string; kind?: string }, adminToken: string): Promise<Dataset> {
  const res = await fetch(apiUrl(`/api/v1/datasets/${id}`), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify({ name: body.name ?? '', description: body.description ?? '', kind: body.kind ?? '' })
  })
  if (!res.ok) throw new Error('failed to update dataset')
  return res.json()
}

export async function deleteDataset(id: number, adminToken: string): Promise<void> {
  const res = await fetch(apiUrl(`/api/v1/datasets/${id}`), {
    method: 'DELETE',
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to delete dataset')
}

// Dataset Items
export async function listDatasetItems(params: {
  datasetId: number
  q?: string
  limit?: number
  offset?: number
}): Promise<{ items: DatasetItem[]; limit: number; offset: number }> {
  const url = toURL(`/api/v1/datasets/${params.datasetId}/items`)
  if (params.q) url.searchParams.set('q', params.q)
  if (params.limit != null) url.searchParams.set('limit', String(params.limit))
  if (params.offset != null) url.searchParams.set('offset', String(params.offset))
  const res = await fetch(url.toString())
  if (!res.ok) throw new Error('failed to list items')
  return res.json()
}

export async function getDatasetItem(id: number): Promise<DatasetItem> {
  const res = await fetch(apiUrl(`/api/v1/items/${id}`))
  if (!res.ok) throw new Error('failed to get item')
  return res.json()
}

export async function createDatasetItem(datasetId: number, body: { data: any; source_ref?: string }, adminToken: string): Promise<DatasetItem> {
  const res = await fetch(apiUrl(`/api/v1/datasets/${datasetId}/items`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify({ data: body.data, source_ref: body.source_ref ?? '' })
  })
  if (!res.ok) throw new Error('failed to create item')
  return res.json()
}

export async function updateDatasetItem(id: number, body: { data?: any; source_ref?: string }, adminToken: string): Promise<DatasetItem> {
  const res = await fetch(apiUrl(`/api/v1/items/${id}`), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify({ data: body.data, source_ref: body.source_ref })
  })
  if (!res.ok) throw new Error('failed to update item')
  return res.json()
}

export async function deleteDatasetItem(id: number, adminToken: string): Promise<void> {
  const res = await fetch(apiUrl(`/api/v1/items/${id}`), {
    method: 'DELETE',
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to delete item')
}

// Conversations
export async function listDatasetConversations(params: {
  datasetId: number
  q?: string
  split?: Split
  status?: ConversationStatus
  limit?: number
  offset?: number
}): Promise<{ items: Conversation[]; limit: number; offset: number }> {
  const url = toURL(`/api/v1/datasets/${params.datasetId}/conversations`)
  if (params.q) url.searchParams.set('q', params.q)
  if (params.split) url.searchParams.set('split', params.split)
  if (params.status) url.searchParams.set('status', params.status)
  if (params.limit != null) url.searchParams.set('limit', String(params.limit))
  if (params.offset != null) url.searchParams.set('offset', String(params.offset))

  const res = await fetch(url.toString())
  if (!res.ok) throw new Error('failed to list conversations')
  return res.json()
}

export async function getConversation(id: number): Promise<Conversation> {
  const res = await fetch(apiUrl(`/api/v1/conversations/${id}`))
  if (!res.ok) throw new Error('failed to get conversation')
  return res.json()
}

export async function createConversation(body: {
  dataset_id: number
  split: Split
  status: ConversationStatus
  tags?: string[]
  source?: string
  notes?: string
  messages: Message[]
}, adminToken: string): Promise<Conversation> {
  const res = await fetch(apiUrl('/api/v1/conversations'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify(body)
  })
  if (!res.ok) throw new Error('failed to create conversation')
  return res.json()
}

export async function updateConversation(id: number, body: {
  dataset_id: number
  split: Split
  status: ConversationStatus
  tags?: string[]
  source?: string
  notes?: string
  messages: Message[]
}, adminToken: string): Promise<Conversation> {
  const res = await fetch(apiUrl(`/api/v1/conversations/${id}`), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': adminToken },
    body: JSON.stringify(body)
  })
  if (!res.ok) throw new Error('failed to update conversation')
  return res.json()
}

export async function deleteConversation(id: number, adminToken: string): Promise<void> {
  const res = await fetch(apiUrl(`/api/v1/conversations/${id}`), {
    method: 'DELETE',
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to delete conversation')
}

// Proposals
export async function createProposal(body: {
  dataset_id: number
  split?: Split
  tags?: string[]
  source?: string
  notes?: string
  messages?: Message[]
  user?: string
  assistant?: string
  system?: string
}): Promise<Proposal> {
  const res = await fetch(apiUrl('/api/v1/proposals'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  if (!res.ok) throw new Error('failed to create proposal')
  return res.json()
}

export async function listProposalsAdmin(status: ProposalStatus, adminToken: string): Promise<{ items: Proposal[] }> {
  const url = toURL('/api/v1/proposals')
  url.searchParams.set('status', status)

  const res = await fetch(url.toString(), {
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to list proposals')
  return res.json()
}

export async function approveProposal(id: number, adminToken: string): Promise<Conversation> {
  const res = await fetch(apiUrl(`/api/v1/proposals/${id}/approve`), {
    method: 'POST',
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to approve')
  return res.json()
}

export async function rejectProposal(id: number, adminToken: string): Promise<void> {
  const res = await fetch(apiUrl(`/api/v1/proposals/${id}/reject`), {
    method: 'POST',
    headers: { 'X-Admin-Token': adminToken }
  })
  if (!res.ok) throw new Error('failed to reject')
}

// Export
export function exportUrl(params: {
  type?: 'pairs' | 'conversations' | 'items' | 'items_with_meta'
  dataset_id?: number
  split?: Split | 'all'
  status?: ConversationStatus
  include_system?: boolean
  context?: 'none' | 'window' | 'full'
  context_turns?: number
  role_style?: 'labels' | 'plain'
  max_examples?: number
}): string {
  const url = toURL('/api/v1/export.jsonl')

  if (params.type) url.searchParams.set('type', params.type)
  if (params.dataset_id != null) url.searchParams.set('dataset_id', String(params.dataset_id))
  if (params.split) url.searchParams.set('split', params.split)
  if (params.status) url.searchParams.set('status', params.status)
  if (params.include_system != null) url.searchParams.set('include_system', params.include_system ? '1' : '0')
  if (params.context) url.searchParams.set('context', params.context)
  if (params.context_turns != null) url.searchParams.set('context_turns', String(params.context_turns))
  if (params.role_style) url.searchParams.set('role_style', params.role_style)
  if (params.max_examples != null) url.searchParams.set('max_examples', String(params.max_examples))

  return url.toString()
}
