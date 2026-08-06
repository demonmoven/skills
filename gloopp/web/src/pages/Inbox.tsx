import { useMemo, useState } from 'react'
import { api } from '../api/client'
import type { HumanExceptionItem, QuestMeta, QuestType } from '../api/types'
import HumanExceptionCard from '../components/HumanExceptionCard'
import Icon from '../components/Icon'
import { relativeTime, typeLabel } from '../components/util'
import { emitStuckDiagnosisPath, humanExceptionSourceKind } from '../features/quest/stuckDiagnosisMetrics'
import useDragToDismiss from '../hooks/useDragToDismiss'
import { automationCandidateMeta, isAutomationCandidate, splitInboxWorkQueues } from './inboxHelpers'
import { useTranslation } from 'react-i18next'

type Props = {
  items: QuestMeta[]
  humanExceptions?: HumanExceptionItem[]
  loading?: boolean
  searchQuery?: string
  onOpen: (qid: string) => void
  onError: (msg: string) => void
  onChanged: () => void
  onRemoveItem: (id: string) => void
}

type InboxEditDraft = {
  query: string
  type: QuestType
  workDir: string
}

type InboxAcceptResponse = {
  ok: boolean
  qid: string
  quest?: QuestMeta
}

export default function Inbox({ items, humanExceptions = [], loading, searchQuery = '', onOpen, onError, onChanged, onRemoveItem }: Props) {
  const { t } = useTranslation()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editDraft, setEditDraft] = useState<InboxEditDraft>({
    query: '',
    type: 'execute',
    workDir: '',
  })
  const [saving, setSaving] = useState<string | null>(null)
  const [rejectingId, setRejectingId] = useState<string | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const { sheetRef: rejectSheetRef, sheetStyle: rejectSheetStyle, scrimStyle: rejectScrimStyle } = useDragToDismiss({
    onDismiss: () => setRejectingId(null),
    disabled: !rejectingId,
  })
  const filtered = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return items
    return items.filter((item) => {
      const meta = automationCandidateMeta(item)
      const creator = isAutomationCandidate(item) ? meta.source : item.created_by || t('inbox.pendingDelegation.manualCreator')
      return [
        item.query,
        item.id,
        typeLabel(item.type),
        creator,
        meta.triageMode,
        meta.holdReason,
      ].some((value) => value?.toLowerCase().includes(q))
    })
  }, [items, searchQuery])

  async function accept(id: string) {
    if (saving) return
    setSaving(id)
    try {
      const res = await api.post<InboxAcceptResponse>('/api/inbox/' + id + '/accept')
      onRemoveItem(id)
      onChanged()
      if (res.quest?.id) {
        onOpen(res.quest.id)
      }
    } catch (e) {
      onError(e instanceof Error ? e.message : t('inbox.error.acceptFail'))
    } finally {
      setSaving(null)
    }
  }

  function openReject(id: string) {
    setRejectingId(id)
    setRejectReason('')
  }

  async function confirmReject() {
    if (!rejectingId || saving) return
    const id = rejectingId
    setSaving(id)
    try {
      await api.post('/api/inbox/' + id + '/reject', { reason: rejectReason.trim() || 'rejected from dashboard' })
      setRejectingId(null)
      setRejectReason('')
      onRemoveItem(id)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('inbox.error.rejectFail'))
    } finally {
      setSaving(null)
    }
  }

  function startEdit(item: QuestMeta) {
    setEditingId(item.id)
    setEditDraft({
      query: item.query || '',
      type: item.type || 'execute',
      workDir: item.base_working_dir || '',
    })
  }

  async function saveEdit(id: string) {
    setSaving(id)
    try {
      await api.post('/api/inbox/' + id, {
        query: editDraft.query,
        type: editDraft.type,
        work_dir: editDraft.workDir,
      })
      setEditingId(null)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('inbox.error.saveFail'))
    } finally {
      setSaving(null)
    }
  }

  const filteredExceptions = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return humanExceptions
    return humanExceptions.filter((item) => [
      item.quest_id,
      item.reason,
      item.recommended_action,
      item.source_status,
    ].some((value) => value?.toLowerCase().includes(q)))
  }, [humanExceptions, searchQuery])
  const workQueues = useMemo(() => splitInboxWorkQueues(filtered, filteredExceptions), [filtered, filteredExceptions])

  function openHumanException(item: HumanExceptionItem) {
    const kind = humanExceptionSourceKind(item.source_status)
    if (kind) {
      emitStuckDiagnosisPath({
        entry_surface: 'inbox',
        quest_id: item.quest_id,
        exception_kind: kind,
        click_count: 1,
        resolved_surface: 'quest_detail',
        has_recommended_action: Boolean(item.recommended_action),
        has_evidence_jump: Boolean(item.evidence_summary || item.audit_ref),
      })
    }
    onOpen(item.quest_id)
  }

  if ((items.length === 0 && humanExceptions.length === 0) || (workQueues.automationCandidates.length === 0 && workQueues.pendingDelegations.length === 0 && filteredExceptions.length === 0))
    return (
      <div className="inbox-empty">
        {loading ? (
          <>
            <Icon name="spinner" className="spin" />
            <h2>{t('inbox.empty.loading')}</h2>
          </>
        ) : (
          <>
            <Icon name="inbox" />
            {items.length === 0 && humanExceptions.length === 0 ? (
              <>
                <h2>{t('inbox.empty.title')}</h2>
                <p>{t('inbox.empty.desc')}</p>
              </>
            ) : (
              <>
                <h2>{t('inbox.empty.noMatchTitle')}</h2>
                <p>{t('inbox.empty.noMatchDesc')}</p>
              </>
            )}
          </>
        )}
      </div>
    )

  return (
    <>
    {filteredExceptions.length > 0 && (
      <section className="human-exception-list" aria-label={t('aria.humanExceptionQueue')}>
        <div className="inbox-lane-head">
          <div>
            <h2>{t('inbox.humanException.title')}</h2>
            <p>{t('inbox.humanException.desc')}</p>
          </div>
          <span className="mono">{filteredExceptions.length}</span>
        </div>
        {filteredExceptions.map((item) => (
          <HumanExceptionCard item={item} onOpen={openHumanException} key={item.id} />
        ))}
      </section>
    )}
    <div className="inbox-list">
      {workQueues.automationCandidates.length > 0 && (
        <div className="inbox-lane-head automation">
          <div>
            <h2>{t('inbox.automationCandidate.title')}</h2>
            <p>{t('inbox.automationCandidate.desc')}</p>
          </div>
          <span className="mono">{workQueues.automationCandidates.length}</span>
        </div>
      )}
      {workQueues.automationCandidates.map((item) => {
        const editing = editingId === item.id
        const candidateMeta = automationCandidateMeta(item)
        return (
          <article className={'inbox-card ' + (editing ? 'editing ' : '')} key={item.id}>
            <div className="inbox-icon">
              <Icon name={item.type === 'design' ? 'shield' : 'zap'} />
            </div>
            <div className="inbox-body">
              {editing ? (
                <div className="inbox-main editing">
                  <div className="quest-card-top">
                    <strong>
                      {candidateMeta.source}
                    </strong>
                    <span className="chip source plain">
                      {candidateMeta.triageMode}
                    </span>
                    <span className="mono push">{relativeTime(item.created_at_ms)}</span>
                  </div>
                  <textarea
                    className="inbox-edit"
                    rows={5}
                    autoFocus
                    value={editDraft.query}
                    onChange={(e) => setEditDraft((prev) => ({ ...prev, query: e.target.value }))}
                    onKeyDown={(e) => {
                      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                        e.preventDefault()
                        saveEdit(item.id)
                      }
                      // 阻止回车冒泡到父元素
                      if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
                        e.stopPropagation()
                      }
                    }}
                  />
                  <div className="inbox-edit-grid">
                    <div className="field">
                      <span>{t('inbox.edit.taskType')}</span>
                      <div className="segmented wide">
                        <button
                          type="button"
                          className={editDraft.type === 'execute' ? 'active' : ''}
                          onClick={() => setEditDraft((prev) => ({ ...prev, type: 'execute' }))}
                        >
                          {t('quest.type.execute')}
                        </button>
                        <button
                          type="button"
                          className={editDraft.type === 'design' ? 'active' : ''}
                          onClick={() => setEditDraft((prev) => ({ ...prev, type: 'design' }))}
                        >
                          {t('quest.type.design')}
                        </button>
                      </div>
                    </div>
                    <label className="field inbox-workdir-field">
                      <span>{t('inbox.edit.workDir')}</span>
                      <input
                        className="mono"
                        value={editDraft.workDir}
                        onChange={(e) => setEditDraft((prev) => ({ ...prev, workDir: e.target.value }))}
                        placeholder={t('inbox.edit.workDirPlaceholder')}
                      />
                    </label>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  className="inbox-main"
                  onClick={() => onOpen(item.id)}
                >
                  <div className="quest-card-top">
                    <strong>
                      {candidateMeta.source}
                    </strong>
                    <span className="chip source plain">
                      {candidateMeta.triageMode}
                    </span>
                    <span className="mono push">{relativeTime(item.created_at_ms)}</span>
                  </div>
                  <p>{item.query}</p>
                  <div className="candidate-policy-row">
                    <span className="chip mono tiny">{candidateMeta.triageMode}</span>
                    <span>{candidateMeta.holdReason}</span>
                  </div>
                </button>
              )}
              <div className="inbox-source">
                <span>{t('inbox.automationCandidate.source')}</span>
                <span className="mono">{t('inbox.automationCandidate.from', { source: candidateMeta.source })}</span>
              </div>
              <div className="row-actions inbox-actions">
                {editing ? (
                  <>
                    <button
                      className="button primary"
                      onClick={() => saveEdit(item.id)}
                      disabled={saving === item.id}
                    >
                      <Icon name="save" />
                      {saving === item.id ? t('inbox.action.saving') : t('inbox.action.saveChanges')}
                    </button>
                    <button
                      className="button ghost"
                      onClick={() => setEditingId(null)}
                      disabled={saving === item.id}
                    >
                      {t('inbox.action.cancel')}
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      className="button primary"
                      onClick={() => accept(item.id)}
                      disabled={saving === item.id}
                    >
                      {saving === item.id ? (
                        <Icon name="spinner" className="spin" />
                      ) : (
                        <Icon name="check" />
                      )}
                      {t('inbox.action.accept')}
                    </button>
                    <button
                      className="button"
                      onClick={() => openReject(item.id)}
                      disabled={saving === item.id}
                    >
                      {saving === item.id ? (
                        <Icon name="spinner" className="spin" />
                      ) : (
                        <Icon name="x" />
                      )}
                      {t('inbox.action.reject')}
                    </button>
                    <button
                      className="button square"
                      onClick={() => startEdit(item)}
                      aria-label={t('inbox.action.edit')}
                    >
                      <Icon name="pencil" />
                    </button>

                  </>
                )}
              </div>
            </div>
          </article>
        )
      })}
      {workQueues.pendingDelegations.length > 0 && (
        <div className="inbox-lane-head pending">
          <div>
            <h2>{t('inbox.pendingDelegation.title')}</h2>
            <p>{t('inbox.pendingDelegation.desc')}</p>
          </div>
          <span className="mono">{workQueues.pendingDelegations.length}</span>
        </div>
      )}
      {workQueues.pendingDelegations.map((item) => {
        const editing = editingId === item.id
        return (
          <article className={'inbox-card ' + (editing ? 'editing ' : '')} key={item.id}>
            <div className="inbox-icon">
              <Icon name={item.type === 'design' ? 'shield' : 'zap'} />
            </div>
            <div className="inbox-body">
              {editing ? (
                <div className="inbox-main editing">
                  <div className="quest-card-top">
                    <strong>{item.created_by || t('inbox.pendingDelegation.manualCreator')}</strong>
                    <span className="chip source plain">{typeLabel(item.type)}</span>
                    <span className="mono push">{relativeTime(item.created_at_ms)}</span>
                  </div>
                  <textarea
                    className="inbox-edit"
                    rows={5}
                    autoFocus
                    value={editDraft.query}
                    onChange={(e) => setEditDraft((prev) => ({ ...prev, query: e.target.value }))}
                    onKeyDown={(e) => {
                      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                        e.preventDefault()
                        saveEdit(item.id)
                      }
                      if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
                        e.stopPropagation()
                      }
                    }}
                  />
                  <div className="inbox-edit-grid">
                    <div className="field">
                      <span>{t('inbox.edit.taskType')}</span>
                      <div className="segmented wide">
                        <button
                          type="button"
                          className={editDraft.type === 'execute' ? 'active' : ''}
                          onClick={() => setEditDraft((prev) => ({ ...prev, type: 'execute' }))}
                        >
                          {t('quest.type.execute')}
                        </button>
                        <button
                          type="button"
                          className={editDraft.type === 'design' ? 'active' : ''}
                          onClick={() => setEditDraft((prev) => ({ ...prev, type: 'design' }))}
                        >
                          {t('quest.type.design')}
                        </button>
                      </div>
                    </div>
                    <label className="field inbox-workdir-field">
                      <span>{t('inbox.edit.workDir')}</span>
                      <input
                        className="mono"
                        value={editDraft.workDir}
                        onChange={(e) => setEditDraft((prev) => ({ ...prev, workDir: e.target.value }))}
                        placeholder={t('inbox.edit.workDirPlaceholder')}
                      />
                    </label>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  className="inbox-main"
                  onClick={() => onOpen(item.id)}
                >
                  <div className="quest-card-top">
                    <strong>{item.created_by || t('inbox.pendingDelegation.manualCreator')}</strong>
                    <span className="chip source plain">{typeLabel(item.type)}</span>
                    <span className="mono push">{relativeTime(item.created_at_ms)}</span>
                  </div>
                  <p>{item.query}</p>
                </button>
              )}
              <div className="inbox-source">
                <span>{t('inbox.pendingDelegation.source')}</span>
                <span className="mono">{t('inbox.pendingDelegation.from', { source: item.created_by || t('inbox.pendingDelegation.manualCreator') })}</span>
              </div>
              <div className="row-actions inbox-actions">
                {editing ? (
                  <>
                    <button className="button primary" onClick={() => saveEdit(item.id)} disabled={saving === item.id}>
                      <Icon name="save" />
                      {saving === item.id ? t('inbox.action.saving') : t('inbox.action.saveChanges')}
                    </button>
                    <button className="button ghost" onClick={() => setEditingId(null)} disabled={saving === item.id}>
                      {t('inbox.action.cancel')}
                    </button>
                  </>
                ) : (
                  <>
                    <button className="button primary" onClick={() => accept(item.id)} disabled={saving === item.id}>
                      {saving === item.id ? <Icon name="spinner" className="spin" /> : <Icon name="check" />}
                      {t('inbox.action.accept')}
                    </button>
                    <button className="button" onClick={() => openReject(item.id)} disabled={saving === item.id}>
                      {saving === item.id ? <Icon name="spinner" className="spin" /> : <Icon name="x" />}
                      {t('inbox.action.reject')}
                    </button>
                    <button className="button square" onClick={() => startEdit(item)} aria-label={t('inbox.action.edit')}>
                      <Icon name="pencil" />
                    </button>
                  </>
                )}
              </div>
            </div>
          </article>
        )
      })}
    </div>
    {rejectingId && (
      <div className="sheet-scrim" style={rejectScrimStyle} role="dialog" aria-modal="true" aria-label={t('inbox.reject.dialogLabel')} onClick={() => setRejectingId(null)}>
        <div className="sheet-card" ref={rejectSheetRef} style={rejectSheetStyle} onClick={(e) => e.stopPropagation()}>
          <div className="sheet-head">
            <span className="sheet-icon">
              <Icon name="x" />
            </span>
            <div>
              <h2>{t('inbox.reject.title')}</h2>
              <p>{t('inbox.reject.desc')}</p>
            </div>
            <button className="icon-button" onClick={() => setRejectingId(null)} aria-label={t('inbox.reject.close')}>
              <Icon name="x" />
            </button>
          </div>
          <div className="sheet-body">
            <label className="field">
              <span>{t('inbox.reject.reasonLabel')}</span>
              <textarea
                rows={3}
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder={t('inbox.reject.reasonPlaceholder')}
                autoFocus
              />
            </label>
          </div>
          <div className="design-sheet-foot">
            <button className="button" onClick={() => setRejectingId(null)} disabled={!!saving}>{t('inbox.action.cancel')}</button>
            <button className="button danger" onClick={confirmReject} disabled={!!saving}>
              {saving === rejectingId ? <Icon name="spinner" className="spin" /> : <Icon name="x" />}
              {t('inbox.action.confirmReject')}
            </button>
          </div>
        </div>
      </div>
    )}
    </>
  )
}
