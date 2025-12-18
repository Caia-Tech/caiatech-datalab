import React, { useEffect, useMemo, useState } from 'react'
import {
  apiBase,
  approveProposal,
  createConversation,
  createDataset,
  createDatasetItem,
  createProposal,
  deleteConversation,
  deleteDataset,
  deleteDatasetItem,
  exportUrl,
  getConversation,
  getDataset,
  getDatasetItem,
  listDatasetConversations,
  listDatasetItems,
  listDatasets,
  listProposalsAdmin,
  rejectProposal,
  updateConversation,
  updateDataset,
  updateDatasetItem,
  type Conversation,
  type ConversationStatus,
  type Dataset,
  type DatasetItem,
  type Message,
  type Proposal,
  type ProposalStatus,
  type Role,
  type Split
} from './api'

type Route =
  | { name: 'datasets' }
  | { name: 'dataset'; datasetId: number }
  | { name: 'submit' }
  | { name: 'admin' }
  | { name: 'export' }

function parseRoute(): Route {
  const raw = (window.location.hash || '').replace(/^#\/?/, '')
  if (!raw) return { name: 'datasets' }
  const parts = raw.split('/').filter(Boolean)
  if (parts[0] === 'dataset' && parts[1]) {
    const id = Number(parts[1])
    if (Number.isFinite(id) && id > 0) return { name: 'dataset', datasetId: id }
  }
  if (parts[0] === 'submit') return { name: 'submit' }
  if (parts[0] === 'admin') return { name: 'admin' }
  if (parts[0] === 'export') return { name: 'export' }
  return { name: 'datasets' }
}

function navLink(href: string, label: string, active: boolean) {
  return (
    <a className={active ? 'active' : ''} href={href}>
      {label}
    </a>
  )
}

export default function App() {
  const [route, setRoute] = useState<Route>(() => parseRoute())

  useEffect(() => {
    const onHash = () => setRoute(parseRoute())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  const active = route.name

  return (
    <div className="container">
      <div className="header">
        <div className="brand">
          <div className="brandTitle">
            <span>caiatech</span>-datalab
          </div>
          <div className="brandSub">datasets · items · conversations · exports</div>
        </div>
        <div className="nav">
          {navLink('#/datasets', 'Datasets', active === 'datasets' || active === 'dataset')}
          {navLink('#/submit', 'Submit', active === 'submit')}
          {navLink('#/admin', 'Admin', active === 'admin')}
          {navLink('#/export', 'Export', active === 'export')}
        </div>
      </div>

      <div className="banner">
        API: <b>{apiBase() || '(via /api proxy)'}</b>
      </div>

      {route.name === 'datasets' && <Datasets />}
      {route.name === 'dataset' && <DatasetDetail datasetId={route.datasetId} />}
      {route.name === 'submit' && <Submit />}
      {route.name === 'admin' && <Admin />}
      {route.name === 'export' && <Export />}
    </div>
  )
}

function useAdminToken() {
  const [adminToken, setAdminToken] = useState(() => localStorage.getItem('datalab_admin_token') ?? '')
  useEffect(() => {
    localStorage.setItem('datalab_admin_token', adminToken)
  }, [adminToken])
  return { adminToken, setAdminToken }
}

function Datasets() {
  const { adminToken, setAdminToken } = useAdminToken()

  const [q, setQ] = useState('')
  const [items, setItems] = useState<Dataset[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newKind, setNewKind] = useState<'items' | 'conversations'>('items')

  const canAdmin = adminToken.trim().length > 0

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const res = await listDatasets({ q: q.trim() || undefined, limit: 200, offset: 0 })
      setItems(res.items)
    } catch (e: any) {
      setError(e?.message ?? 'failed')
    } finally {
      setLoading(false)
    }
  }

  async function onCreate() {
    if (!canAdmin) return
    try {
      const d = await createDataset({ name: newName, description: newDesc, kind: newKind }, adminToken)
      setNewName('')
      setNewDesc('')
      setNewKind('items')
      await load()
      window.location.hash = `#/dataset/${d.id}`
    } catch (e: any) {
      setError(e?.message ?? 'failed to create')
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="panel">
      <div className="row">
        <div style={{ flex: 1, minWidth: 260 }}>
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search datasets…" />
        </div>
        <button className="secondary" onClick={load}>
          {loading ? 'Loading…' : 'Search'}
        </button>
      </div>

      <div style={{ marginTop: 10 }}>
        <div className="pairLabel">admin token (for CRUD)</div>
        <input value={adminToken} onChange={(e) => setAdminToken(e.target.value)} placeholder="DATALAB_ADMIN_TOKEN" />
      </div>

      {canAdmin && (
        <div className="card" style={{ marginTop: 12 }}>
          <div className="pairLabel">create dataset</div>
          <div className="row">
            <div style={{ flex: 1, minWidth: 220 }}>
              <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="name (unique)" />
            </div>
            <div style={{ width: 220 }}>
              <input value={newKind} onChange={(e) => setNewKind(e.target.value as any)} list="datasetkinds" placeholder="kind" />
              <datalist id="datasetkinds">
                <option value="items" />
                <option value="conversations" />
              </datalist>
            </div>
            <div style={{ flex: 2, minWidth: 260 }}>
              <input value={newDesc} onChange={(e) => setNewDesc(e.target.value)} placeholder="description (optional)" />
            </div>
            <button onClick={onCreate} disabled={!newName.trim()}>
              Create
            </button>
          </div>
        </div>
      )}

      {error && <div className="banner">Error: {error}</div>}

      <div style={{ marginTop: 14 }}>
        <small>Showing {items.length} datasets.</small>
      </div>

      <div style={{ marginTop: 12, display: 'grid', gap: 10 }}>
        {items.map((d) => (
          <a
            key={d.id}
            href={`#/dataset/${d.id}`}
            className="card"
            style={{ display: 'block' }}
          >
            <div className="row" style={{ justifyContent: 'space-between' }}>
              <div>
                <div style={{ fontWeight: 700 }}>{d.name}</div>
                <small>{d.description || '—'}</small>
                <div style={{ height: 6 }} />
                <small style={{ color: 'var(--muted)' }}>kind: {d.kind}</small>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ color: 'var(--brand)', fontWeight: 700 }}>{d.item_count}</div>
                <small>items</small>
                <div style={{ height: 6 }} />
                <div style={{ color: 'var(--muted)', fontWeight: 700 }}>{d.conversation_count}</div>
                <small>conversations</small>
              </div>
            </div>
          </a>
        ))}
      </div>
    </div>
  )
}

function DatasetDetail({ datasetId }: { datasetId: number }) {
  const { adminToken, setAdminToken } = useAdminToken()
  const canAdmin = adminToken.trim().length > 0

  const [dataset, setDataset] = useState<Dataset | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [view, setView] = useState<'items' | 'conversations'>('items')

  const [editDatasetName, setEditDatasetName] = useState('')
  const [editDatasetDesc, setEditDatasetDesc] = useState('')
  const [editDatasetKind, setEditDatasetKind] = useState('items')

  // Items
  const [itemQ, setItemQ] = useState('')
  const [items, setItems] = useState<DatasetItem[]>([])
  const [loadingItems, setLoadingItems] = useState(false)
  const [selectedItemId, setSelectedItemId] = useState<number | null>(null)
  const [selectedItem, setSelectedItem] = useState<DatasetItem | null>(null)
  const [selectedItemErr, setSelectedItemErr] = useState<string | null>(null)

  // Conversations (legacy)
  const [convQ, setConvQ] = useState('')
  const [split, setSplit] = useState<Split>('train')
  const [status, setStatus] = useState<ConversationStatus>('approved')
  const [convos, setConvos] = useState<Conversation[]>([])
  const [loadingConvos, setLoadingConvos] = useState(false)

  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selected, setSelected] = useState<Conversation | null>(null)
  const [selectedErr, setSelectedErr] = useState<string | null>(null)

  async function loadDataset() {
    setError(null)
    try {
      const d = await getDataset(datasetId)
      setDataset(d)
      setEditDatasetName(d.name)
      setEditDatasetDesc(d.description)
      setEditDatasetKind(d.kind || 'items')
      setView(d.kind === 'conversations' ? 'conversations' : 'items')
    } catch (e: any) {
      setError(e?.message ?? 'failed to load dataset')
    }
  }

  async function loadItems() {
    setLoadingItems(true)
    setError(null)
    try {
      const res = await listDatasetItems({
        datasetId,
        q: itemQ.trim() || undefined,
        limit: 200,
        offset: 0
      })
      setItems(res.items)
    } catch (e: any) {
      setError(e?.message ?? 'failed to list items')
    } finally {
      setLoadingItems(false)
    }
  }

  async function loadItem(id: number) {
    setSelectedItemId(id)
    setSelectedItem(null)
    setSelectedItemErr(null)
    try {
      const it = await getDatasetItem(id)
      setSelectedItem(it)
    } catch (e: any) {
      setSelectedItemErr(e?.message ?? 'failed to load item')
    }
  }

  async function loadConversations() {
    setLoadingConvos(true)
    try {
      const res = await listDatasetConversations({
        datasetId,
        q: convQ.trim() || undefined,
        split,
        status,
        limit: 200,
        offset: 0
      })
      setConvos(res.items)
    } catch (e: any) {
      setError(e?.message ?? 'failed to list conversations')
    } finally {
      setLoadingConvos(false)
    }
  }

  async function loadConversation(id: number) {
    setSelectedId(id)
    setSelected(null)
    setSelectedErr(null)
    try {
      const c = await getConversation(id)
      setSelected(c)
    } catch (e: any) {
      setSelectedErr(e?.message ?? 'failed to load conversation')
    }
  }

  async function onSaveDataset() {
    if (!dataset || !canAdmin) return
    try {
      const updated = await updateDataset(dataset.id, { name: editDatasetName, description: editDatasetDesc, kind: editDatasetKind }, adminToken)
      setDataset(updated)
    } catch (e: any) {
      setError(e?.message ?? 'failed to update dataset')
    }
  }

  async function onDeleteDataset() {
    if (!dataset || !canAdmin) return
    const ok = window.confirm(`Delete dataset "${dataset.name}" and all its items/conversations? This cannot be undone.`)
    if (!ok) return
    try {
      await deleteDataset(dataset.id, adminToken)
      window.location.hash = '#/datasets'
    } catch (e: any) {
      setError(e?.message ?? 'failed to delete dataset')
    }
  }

  async function onCreateConversation() {
    if (!canAdmin) return
    try {
      const created = await createConversation(
        {
          dataset_id: datasetId,
          split,
          status: "draft",
          tags: [],
          source: dataset?.name ?? '',
          notes: '',
          messages: [
            { role: 'user', content: '' },
            { role: 'assistant', content: '' }
          ]
        },
        adminToken
      )
      await loadConversations()
      await loadConversation(created.id)
    } catch (e: any) {
      setError(e?.message ?? 'failed to create conversation')
    }
  }

  async function onCreateItem() {
    if (!canAdmin) return
    try {
      const created = await createDatasetItem(datasetId, { data: {}, source_ref: '' }, adminToken)
      await loadItems()
      await loadItem(created.id)
    } catch (e: any) {
      setError(e?.message ?? 'failed to create item')
    }
  }

  useEffect(() => {
    void loadDataset()
    setSelected(null)
    setSelectedId(null)
    setSelectedErr(null)
    setSelectedItem(null)
    setSelectedItemId(null)
    setSelectedItemErr(null)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [datasetId])

  useEffect(() => {
    if (view === 'items') void loadItems()
    if (view === 'conversations') void loadConversations()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view, datasetId])

  function previewAny(v: any): string {
    try {
      if (typeof v === 'string') return v
      return JSON.stringify(v)
    } catch {
      return String(v)
    }
  }

  return (
    <div className="panel">
      <div className="row" style={{ justifyContent: 'space-between' }}>
	        <div>
	          <div style={{ fontWeight: 800, fontSize: 18 }}>{dataset?.name ?? `dataset #${datasetId}`}</div>
	          <small>{dataset?.description || '—'}</small>
	          <div style={{ height: 6 }} />
	          <small style={{ color: 'var(--muted)' }}>kind: {dataset?.kind ?? '—'}</small>
	        </div>
	        <div style={{ textAlign: 'right' }}>
	          <small>
	            items: <b>{dataset?.item_count ?? '—'}</b> · conversations: <b>{dataset?.conversation_count ?? '—'}</b>
	          </small>
	        </div>
	      </div>

      <div style={{ marginTop: 10 }}>
        <div className="pairLabel">admin token (for CRUD)</div>
        <input value={adminToken} onChange={(e) => setAdminToken(e.target.value)} placeholder="DATALAB_ADMIN_TOKEN" />
      </div>

  	      {canAdmin && dataset && (
  	        <div className="card" style={{ marginTop: 12 }}>
  	          <div className="pairLabel">dataset settings</div>
  	          <div className="row">
  	            <div style={{ flex: 1, minWidth: 240 }}>
  	              <input value={editDatasetName} onChange={(e) => setEditDatasetName(e.target.value)} />
  	            </div>
	            <div style={{ width: 220 }}>
	              <input value={editDatasetKind} onChange={(e) => setEditDatasetKind(e.target.value)} list="datasetkinds2" placeholder="kind" />
	              <datalist id="datasetkinds2">
	                <option value="items" />
	                <option value="conversations" />
	              </datalist>
	            </div>
  	            <div style={{ flex: 2, minWidth: 260 }}>
  	              <input value={editDatasetDesc} onChange={(e) => setEditDatasetDesc(e.target.value)} placeholder="description" />
  	            </div>
  	            <button onClick={onSaveDataset}>Save</button>
  	            <button className="danger" onClick={onDeleteDataset}>
  	              Delete
            </button>
          </div>
        </div>
      )}

      <div className="row" style={{ marginTop: 12, gap: 10 }}>
        <button className={view === 'items' ? '' : 'secondary'} onClick={() => setView('items')}>
          Items
        </button>
        <button className={view === 'conversations' ? '' : 'secondary'} onClick={() => setView('conversations')}>
          Conversations
        </button>
      </div>

      <div className="row" style={{ marginTop: 12 }}>
        {view === 'items' ? (
          <>
            <div style={{ flex: 1, minWidth: 240 }}>
              <input value={itemQ} onChange={(e) => setItemQ(e.target.value)} placeholder="Search items (JSON / source_ref)..." />
            </div>
            <button className="secondary" onClick={loadItems}>
              {loadingItems ? 'Loading…' : 'Refresh'}
            </button>
            {canAdmin && <button onClick={onCreateItem}>+ Item</button>}
          </>
        ) : (
          <>
            <div style={{ flex: 1, minWidth: 240 }}>
              <input value={convQ} onChange={(e) => setConvQ(e.target.value)} placeholder="Search message content…" />
            </div>
            <div style={{ width: 140 }}>
              <input value={split} onChange={(e) => setSplit(e.target.value as Split)} list="splits" />
              <datalist id="splits">
                <option value="train" />
                <option value="valid" />
                <option value="test" />
              </datalist>
            </div>
            <div style={{ width: 160 }}>
              <input value={status} onChange={(e) => setStatus(e.target.value as any)} list="statuses" />
              <datalist id="statuses">
                <option value="approved" />
                <option value="pending" />
                <option value="draft" />
                <option value="rejected" />
                <option value="archived" />
              </datalist>
            </div>
            <button className="secondary" onClick={loadConversations}>
              {loadingConvos ? 'Loading…' : 'Refresh'}
            </button>
            {canAdmin && <button onClick={onCreateConversation}>+ Conversation</button>}
          </>
        )}
      </div>

      {error && <div className="banner">Error: {error}</div>}

	      <div className="grid" style={{ marginTop: 12 }}>
	        <div className="card">
	          <div className="pairLabel">{view === 'items' ? 'items' : 'conversations'}</div>
	          <small>Showing {view === 'items' ? items.length : convos.length}.</small>
	          <div style={{ height: 10 }} />
	          <div style={{ display: 'grid', gap: 8 }}>
	            {view === 'items'
	              ? items.map((it) => (
	                  <a
	                    key={it.id}
	                    href="#"
	                    onClick={(e) => {
	                      e.preventDefault()
	                      void loadItem(it.id)
	                    }}
	                    style={{
	                      padding: '10px 12px',
	                      borderRadius: 12,
	                      border: '1px solid var(--border)',
	                      background: selectedItemId === it.id ? 'rgba(98, 179, 255, 0.12)' : 'rgba(255, 255, 255, 0.03)'
	                    }}
	                  >
	                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}>
	                      <small>#{it.id}</small>
	                      <small>{new Date(it.created_at).toLocaleString()}</small>
	                    </div>
	                    <div style={{ height: 6 }} />
	                    <small style={{ color: 'var(--muted)' }}>{(it.source_ref || '').slice(0, 120) || '—'}</small>
	                    <div style={{ height: 6 }} />
	                    <small style={{ color: 'var(--muted)' }}>{previewAny(it.data).slice(0, 140) || '—'}</small>
	                  </a>
	                ))
	              : convos.map((c) => (
	                  <a
	                    key={c.id}
	                    href="#"
	                    onClick={(e) => {
	                      e.preventDefault()
	                      void loadConversation(c.id)
	                    }}
	                    style={{
	                      padding: '10px 12px',
	                      borderRadius: 12,
	                      border: '1px solid var(--border)',
	                      background: selectedId === c.id ? 'rgba(98, 179, 255, 0.12)' : 'rgba(255, 255, 255, 0.03)'
	                    }}
	                  >
	                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}>
	                      <small>
	                        #{c.id} · {c.split} · {c.status}
	                      </small>
	                      <small>{new Date(c.created_at).toLocaleString()}</small>
	                    </div>
	                    <div style={{ height: 6 }} />
	                    <small style={{ color: 'var(--muted)' }}>{(c.preview_user || '').slice(0, 120) || '—'}</small>
	                  </a>
	                ))}
	          </div>
	        </div>

	        <div className="card">
	          <div className="pairLabel">editor</div>
	          {view === 'items' ? (
	            <>
	              {selectedItemErr && <div className="banner">Error: {selectedItemErr}</div>}
	              {!selectedItem && !selectedItemErr && <small>Select an item.</small>}
	              {selectedItem && (
	                <DatasetItemEditor
	                  item={selectedItem}
	                  adminToken={adminToken}
	                  onSaved={async (id) => {
	                    await loadItems()
	                    await loadItem(id)
	                  }}
	                  onDeleted={async () => {
	                    setSelectedItem(null)
	                    setSelectedItemId(null)
	                    await loadItems()
	                  }}
	                />
	              )}
	            </>
	          ) : (
	            <>
	              {selectedErr && <div className="banner">Error: {selectedErr}</div>}
	              {!selected && !selectedErr && <small>Select a conversation.</small>}
	              {selected && (
	                <ConversationEditor
	                  conversation={selected}
	                  adminToken={adminToken}
	                  onSaved={async (id) => {
	                    await loadConversations()
	                    await loadConversation(id)
	                  }}
	                  onDeleted={async () => {
	                    setSelected(null)
	                    setSelectedId(null)
	                    await loadConversations()
	                  }}
	                />
	              )}
	            </>
	          )}
	        </div>
	      </div>
    </div>
  )
}

function ConversationEditor({
  conversation,
  adminToken,
  onSaved,
  onDeleted
}: {
  conversation: Conversation
  adminToken: string
  onSaved: (id: number) => Promise<void>
  onDeleted: () => Promise<void>
}) {
  const canAdmin = adminToken.trim().length > 0

  const [split, setSplit] = useState<Split>(conversation.split)
  const [status, setStatus] = useState<ConversationStatus>(conversation.status)
  const [tags, setTags] = useState((conversation.tags || []).join(', '))
  const [source, setSource] = useState(conversation.source || '')
  const [notes, setNotes] = useState(conversation.notes || '')
  const [messages, setMessages] = useState<Message[]>(conversation.messages ?? [])

  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    setSplit(conversation.split)
    setStatus(conversation.status)
    setTags((conversation.tags || []).join(', '))
    setSource(conversation.source || '')
    setNotes(conversation.notes || '')
    setMessages(conversation.messages ?? [])
    setErr(null)
  }, [conversation.id])

  function setMsgRole(i: number, role: Role) {
    setMessages((prev) => prev.map((m, idx) => (idx === i ? { ...m, role } : m)))
  }

  function setMsgContent(i: number, content: string) {
    setMessages((prev) => prev.map((m, idx) => (idx === i ? { ...m, content } : m)))
  }

  function addMsg(role: Role) {
    setMessages((prev) => [...prev, { role, content: '' }])
  }

  function removeMsg(i: number) {
    setMessages((prev) => prev.filter((_, idx) => idx !== i))
  }

  async function onSave() {
    if (!canAdmin) return
    setSaving(true)
    setErr(null)
    try {
      const tagList = tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean)

      const updated = await updateConversation(
        conversation.id,
        {
          dataset_id: conversation.dataset_id,
          split,
          status: "draft",
          tags: tagList,
          source,
          notes,
          messages
        },
        adminToken
      )
      await onSaved(updated.id)
    } catch (e: any) {
      setErr(e?.message ?? 'failed to save')
    } finally {
      setSaving(false)
    }
  }

  async function onDelete() {
    if (!canAdmin) return
    const ok = window.confirm(`Delete conversation #${conversation.id}?`)
    if (!ok) return
    try {
      await deleteConversation(conversation.id, adminToken)
      await onDeleted()
    } catch (e: any) {
      setErr(e?.message ?? 'failed to delete')
    }
  }

  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <small>
        id: <b>#{conversation.id}</b> · dataset_id: <b>{conversation.dataset_id}</b>
      </small>

      {!canAdmin && <div className="banner">Read-only. Set admin token to edit.</div>}
      {err && <div className="banner">Error: {err}</div>}

      <div className="row">
        <div style={{ width: 140 }}>
          <div className="pairLabel">split</div>
          <input value={split} onChange={(e) => setSplit(e.target.value as Split)} list={`splits-${conversation.id}`} disabled={!canAdmin} />
          <datalist id={`splits-${conversation.id}`}>
            <option value="train" />
            <option value="valid" />
            <option value="test" />
          </datalist>
        </div>
        <div style={{ width: 170 }}>
          <div className="pairLabel">status</div>
          <input value={status} onChange={(e) => setStatus(e.target.value as any)} list={`statuses-${conversation.id}`} disabled={!canAdmin} />
          <datalist id={`statuses-${conversation.id}`}>
            <option value="approved" />
            <option value="pending" />
            <option value="draft" />
            <option value="rejected" />
            <option value="archived" />
          </datalist>
        </div>
        <div style={{ flex: 1, minWidth: 220 }}>
          <div className="pairLabel">tags</div>
          <input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="comma-separated" disabled={!canAdmin} />
        </div>
      </div>

      <div className="row">
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="pairLabel">source</div>
          <input value={source} onChange={(e) => setSource(e.target.value)} disabled={!canAdmin} />
        </div>
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="pairLabel">notes</div>
          <input value={notes} onChange={(e) => setNotes(e.target.value)} disabled={!canAdmin} />
        </div>
      </div>

      <div>
        <div className="pairLabel">messages</div>
        <div style={{ display: 'grid', gap: 10 }}>
          {messages.map((m, i) => (
            <div key={i} className="card">
              <div className="row" style={{ justifyContent: 'space-between' }}>
                <div className="row">
                  <input value={m.role} onChange={(e) => setMsgRole(i, e.target.value as Role)} list={`roles-${conversation.id}-${i}`} disabled={!canAdmin} style={{ width: 140 }} />
                  <datalist id={`roles-${conversation.id}-${i}`}>
                    <option value="system" />
                    <option value="user" />
                    <option value="assistant" />
                  </datalist>
                  <small>#{i}</small>
                </div>
                <button className="danger" onClick={() => removeMsg(i)} disabled={!canAdmin || messages.length <= 1}>
                  Remove
                </button>
              </div>
              <div style={{ height: 10 }} />
              <textarea value={m.content} onChange={(e) => setMsgContent(i, e.target.value)} disabled={!canAdmin} />
            </div>
          ))}
        </div>

        <div className="row" style={{ marginTop: 10 }}>
          <button className="secondary" onClick={() => addMsg('user')} disabled={!canAdmin}>
            + user
          </button>
          <button className="secondary" onClick={() => addMsg('assistant')} disabled={!canAdmin}>
            + assistant
          </button>
          <button className="secondary" onClick={() => addMsg('system')} disabled={!canAdmin}>
            + system
          </button>
        </div>
      </div>

      <div className="row">
        <button onClick={onSave} disabled={!canAdmin || saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button className="danger" onClick={onDelete} disabled={!canAdmin}>
          Delete
        </button>
      </div>
    </div>
  )
}

function DatasetItemEditor({
  item,
  adminToken,
  onSaved,
  onDeleted
}: {
  item: DatasetItem
  adminToken: string
  onSaved: (id: number) => Promise<void>
  onDeleted: () => Promise<void>
}) {
  const canAdmin = adminToken.trim().length > 0

  const [sourceRef, setSourceRef] = useState(item.source_ref || '')
  const [dataText, setDataText] = useState(() => {
    try {
      return JSON.stringify(item.data, null, 2)
    } catch {
      return String(item.data ?? '')
    }
  })

  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    setSourceRef(item.source_ref || '')
    try {
      setDataText(JSON.stringify(item.data, null, 2))
    } catch {
      setDataText(String(item.data ?? ''))
    }
    setErr(null)
  }, [item.id])

  async function onSave() {
    if (!canAdmin) return
    setSaving(true)
    setErr(null)
    try {
      const parsed = JSON.parse(dataText)
      const updated = await updateDatasetItem(item.id, { data: parsed, source_ref: sourceRef }, adminToken)
      await onSaved(updated.id)
    } catch (e: any) {
      setErr(e?.message ?? 'failed to save item')
    } finally {
      setSaving(false)
    }
  }

  async function onDelete() {
    if (!canAdmin) return
    const ok = window.confirm(`Delete item #${item.id}?`)
    if (!ok) return
    try {
      await deleteDatasetItem(item.id, adminToken)
      await onDeleted()
    } catch (e: any) {
      setErr(e?.message ?? 'failed to delete item')
    }
  }

  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <small>
        id: <b>#{item.id}</b> · dataset_id: <b>{item.dataset_id}</b>
      </small>

      {!canAdmin && <div className="banner">Read-only. Set admin token to edit.</div>}
      {err && <div className="banner">Error: {err}</div>}

      <div className="row">
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="pairLabel">source_ref</div>
          <input value={sourceRef} onChange={(e) => setSourceRef(e.target.value)} disabled={!canAdmin} placeholder="file:line or URL…" />
        </div>
      </div>

      <div>
        <div className="pairLabel">data (JSON)</div>
        <textarea
          value={dataText}
          onChange={(e) => setDataText(e.target.value)}
          disabled={!canAdmin}
          style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
        />
      </div>

      <div className="row">
        <button onClick={onSave} disabled={!canAdmin || saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button className="danger" onClick={onDelete} disabled={!canAdmin}>
          Delete
        </button>
      </div>
    </div>
  )
}

function Submit() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [datasetId, setDatasetId] = useState<number>(0)

  const [split, setSplit] = useState<Split>('train')
  const [source, setSource] = useState('')
  const [notes, setNotes] = useState('')
  const [tags, setTags] = useState('')

  const [draft, setDraft] = useState<Message[]>([
    { role: 'user', content: '' },
    { role: 'assistant', content: '' }
  ])

  const [statusText, setStatusText] = useState<string | null>(null)

  useEffect(() => {
    void (async () => {
      const res = await listDatasets({ limit: 200, offset: 0 })
      setDatasets(res.items)
      if (res.items[0]) setDatasetId(res.items[0].id)
    })()
  }, [])

  function setRole(i: number, role: Role) {
    setDraft((prev) => prev.map((m, idx) => (idx === i ? { ...m, role } : m)))
  }

  function setContent(i: number, content: string) {
    setDraft((prev) => prev.map((m, idx) => (idx === i ? { ...m, content } : m)))
  }

  function addMessage(role: Role) {
    setDraft((prev) => [...prev, { role, content: '' }])
  }

  function removeMessage(i: number) {
    setDraft((prev) => prev.filter((_, idx) => idx !== i))
  }

  const canSubmit = useMemo(() => datasetId > 0 && draft.length >= 2 && draft.every((m) => m.content.trim().length > 0), [draft, datasetId])

  async function onSubmit() {
    setStatusText(null)
    try {
      const tagList = tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean)

      await createProposal({
        dataset_id: datasetId,
        split,
        source,
        notes,
        tags: tagList,
        messages: draft
      })

      setDraft([
        { role: 'user', content: '' },
        { role: 'assistant', content: '' }
      ])
      setStatusText('Submitted! (Pending admin review)')
    } catch (e: any) {
      setStatusText('Error: ' + (e?.message ?? 'failed'))
    }
  }

  return (
    <div className="panel">
      <div className="row">
        <div style={{ width: 260 }}>
          <div className="pairLabel">dataset</div>
          <select
            value={datasetId}
            onChange={(e) => setDatasetId(Number(e.target.value))}
            style={{ width: '100%', padding: '10px 12px', borderRadius: 12, border: '1px solid var(--border)', background: 'rgba(0,0,0,0.25)', color: 'var(--text)' }}
          >
            {datasets.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
        <div style={{ width: 140 }}>
          <div className="pairLabel">split</div>
          <input value={split} onChange={(e) => setSplit(e.target.value as Split)} list="splits-submit" />
          <datalist id="splits-submit">
            <option value="train" />
            <option value="valid" />
            <option value="test" />
          </datalist>
        </div>
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="pairLabel">source (optional)</div>
          <input value={source} onChange={(e) => setSource(e.target.value)} placeholder="e.g. user-submission" />
        </div>
      </div>

      <div className="row" style={{ marginTop: 10 }}>
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="pairLabel">tags (optional)</div>
          <input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="comma-separated" />
        </div>
        <div style={{ flex: 2, minWidth: 240 }}>
          <div className="pairLabel">notes (optional)</div>
          <input value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="constraints / rationale" />
        </div>
      </div>

      <div style={{ marginTop: 14 }}>
        <div className="pairLabel">messages</div>
        <div style={{ display: 'grid', gap: 10 }}>
          {draft.map((m, i) => (
            <div key={i} className="card">
              <div className="row" style={{ justifyContent: 'space-between' }}>
                <div className="row">
                  <input value={m.role} onChange={(e) => setRole(i, e.target.value as Role)} list={`roles-submit-${i}`} style={{ width: 150 }} />
                  <datalist id={`roles-submit-${i}`}>
                    <option value="system" />
                    <option value="user" />
                    <option value="assistant" />
                  </datalist>
                  <small>message {i + 1}</small>
                </div>
                <button className="danger" onClick={() => removeMessage(i)} disabled={draft.length <= 2}>
                  Remove
                </button>
              </div>
              <div style={{ height: 10 }} />
              <textarea value={m.content} onChange={(e) => setContent(i, e.target.value)} placeholder="Content…" />
            </div>
          ))}
        </div>

        <div className="row" style={{ marginTop: 10 }}>
          <button className="secondary" onClick={() => addMessage('user')}>
            + user
          </button>
          <button className="secondary" onClick={() => addMessage('assistant')}>
            + assistant
          </button>
          <button className="secondary" onClick={() => addMessage('system')}>
            + system
          </button>
        </div>
      </div>

      <div className="row" style={{ marginTop: 14 }}>
        <button onClick={onSubmit} disabled={!canSubmit}>
          Submit for review
        </button>
        <small>Approved conversations become exportable.</small>
      </div>

      {statusText && <div className="banner">{statusText}</div>}
    </div>
  )
}

function Admin() {
  const { adminToken, setAdminToken } = useAdminToken()

  const [status, setStatus] = useState<ProposalStatus>('pending')
  const [items, setItems] = useState<Proposal[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const canLoad = useMemo(() => adminToken.trim().length > 0, [adminToken])

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const res = await listProposalsAdmin(status, adminToken)
      setItems(res.items)
    } catch (e: any) {
      setError(e?.message ?? 'failed')
    } finally {
      setLoading(false)
    }
  }

  async function onApprove(id: number) {
    await approveProposal(id, adminToken)
    await load()
  }

  async function onReject(id: number) {
    await rejectProposal(id, adminToken)
    await load()
  }

  return (
    <div className="panel">
      <div className="row">
        <div style={{ flex: 1, minWidth: 260 }}>
          <div className="pairLabel">admin token</div>
          <input value={adminToken} onChange={(e) => setAdminToken(e.target.value)} placeholder="DATALAB_ADMIN_TOKEN" />
        </div>
        <div style={{ width: 220 }}>
          <div className="pairLabel">proposal status</div>
          <input value={status} onChange={(e) => setStatus(e.target.value as ProposalStatus)} list="proposal-statuses" />
          <datalist id="proposal-statuses">
            <option value="pending" />
            <option value="approved" />
            <option value="rejected" />
          </datalist>
        </div>
        <button className="secondary" onClick={load} disabled={!canLoad}>
          {loading ? 'Loading…' : 'Load'}
        </button>
      </div>

      {error && <div className="banner">Error: {error}</div>}

      <div style={{ marginTop: 14 }}>
        <small>Showing {items.length} proposals.</small>
      </div>

      <div style={{ marginTop: 12, display: 'grid', gap: 12 }}>
        {items.map((p) => (
          <div key={p.id} className="card">
            <div className="row" style={{ justifyContent: 'space-between' }}>
              <small>
                #{p.id} · {p.status} · {new Date(p.created_at).toLocaleString()}
              </small>
              {p.status === 'pending' && (
                <div className="row">
                  <button onClick={() => void onApprove(p.id)}>Approve</button>
                  <button className="danger" onClick={() => void onReject(p.id)}>
                    Reject
                  </button>
                </div>
              )}
            </div>

            <div style={{ height: 10 }} />
            <div className="pairLabel">payload</div>
            <pre className="pairText">{JSON.stringify(p.payload, null, 2)}</pre>
          </div>
        ))}
      </div>
    </div>
  )
}

function Export() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [datasetId, setDatasetId] = useState<number | 'all'>('all')

  const [type, setType] = useState<'pairs' | 'conversations' | 'items' | 'items_with_meta'>('pairs')
  const [split, setSplit] = useState<'train' | 'valid' | 'test' | 'all'>('train')
  const [status, setStatus] = useState<ConversationStatus>('approved')

  const [includeSystem, setIncludeSystem] = useState(false)
  const [context, setContext] = useState<'none' | 'window' | 'full'>('window')
  const [contextTurns, setContextTurns] = useState(6)
  const [roleStyle, setRoleStyle] = useState<'labels' | 'plain'>('labels')
  const [maxExamples, setMaxExamples] = useState<number>(0)

  useEffect(() => {
    void (async () => {
      const res = await listDatasets({ limit: 200, offset: 0 })
      setDatasets(res.items)
    })()
  }, [])

  const url = exportUrl({
    type,
    dataset_id: datasetId === 'all' ? undefined : datasetId,
    split,
    status,
    include_system: includeSystem,
    context,
    context_turns: contextTurns,
    role_style: roleStyle,
    max_examples: maxExamples
  })

  return (
    <div className="panel">
      <div className="row">
        <div style={{ width: 300 }}>
          <div className="pairLabel">dataset</div>
          <select
            value={datasetId}
            onChange={(e) => setDatasetId(e.target.value === 'all' ? 'all' : Number(e.target.value))}
            style={{ width: '100%', padding: '10px 12px', borderRadius: 12, border: '1px solid var(--border)', background: 'rgba(0,0,0,0.25)', color: 'var(--text)' }}
          >
            <option value="all">all datasets</option>
            {datasets.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
        <div style={{ width: 220 }}>
          <div className="pairLabel">type</div>
          <input value={type} onChange={(e) => setType(e.target.value as any)} list="exporttypes" />
          <datalist id="exporttypes">
            <option value="pairs" />
            <option value="conversations" />
            <option value="items" />
            <option value="items_with_meta" />
          </datalist>
        </div>
        <div style={{ width: 160 }}>
          <div className="pairLabel">split</div>
          <input value={split} onChange={(e) => setSplit(e.target.value as any)} list="exportsplits" />
          <datalist id="exportsplits">
            <option value="train" />
            <option value="valid" />
            <option value="test" />
            <option value="all" />
          </datalist>
        </div>
        <div style={{ width: 180 }}>
          <div className="pairLabel">status</div>
          <input value={status} onChange={(e) => setStatus(e.target.value as any)} list="exportstatus" />
          <datalist id="exportstatus">
            <option value="approved" />
            <option value="pending" />
            <option value="draft" />
            <option value="rejected" />
            <option value="archived" />
          </datalist>
        </div>
      </div>

      {type === 'pairs' && (
        <div style={{ marginTop: 12 }}>
          <div className="row">
            <label style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <input type="checkbox" checked={includeSystem} onChange={(e) => setIncludeSystem(e.target.checked)} />
              <small>include system messages</small>
            </label>
            <div style={{ width: 200 }}>
              <div className="pairLabel">context</div>
              <input value={context} onChange={(e) => setContext(e.target.value as any)} list="contexts" />
              <datalist id="contexts">
                <option value="none" />
                <option value="window" />
                <option value="full" />
              </datalist>
            </div>
            <div style={{ width: 180 }}>
              <div className="pairLabel">context_turns</div>
              <input value={contextTurns} onChange={(e) => setContextTurns(Number(e.target.value) || 0)} placeholder="6" />
            </div>
            <div style={{ width: 200 }}>
              <div className="pairLabel">role_style</div>
              <input value={roleStyle} onChange={(e) => setRoleStyle(e.target.value as any)} list="rolestyles" />
              <datalist id="rolestyles">
                <option value="labels" />
                <option value="plain" />
              </datalist>
            </div>
          </div>
        </div>
      )}

      <div style={{ marginTop: 12 }}>
        <div className="pairLabel">max_examples (0 = unlimited)</div>
        <input value={maxExamples} onChange={(e) => setMaxExamples(Number(e.target.value) || 0)} placeholder="0" />
      </div>

      <div className="banner" style={{ wordBreak: 'break-all' }}>
        <div>
          Export URL:
          <div style={{ height: 6 }} />
          <a href={url}>{url}</a>
        </div>
      </div>

      <div className="row" style={{ marginTop: 12 }}>
        <a href={url}>
          <button>Download</button>
        </a>
        <small>Exports are streamed from Postgres on demand.</small>
      </div>
    </div>
  )
}
