import Icon from '../components/Icon'
import { fmtBlockedReason, fmtTime } from '../components/util'
import useDragToDismiss from '../hooks/useDragToDismiss'
import type { QuestMeta, Verdict } from '../api/types'
import { useTranslation } from 'react-i18next'
import {
  warningLabel,
  type ApplyWarningState,
  type BackupDetailState,
  type BlockedActionState,
} from './questDetailHelpers'

export type ReviewAction = {
  verdict: Verdict
  apply?: boolean
  label: string
  tone: 'primary' | 'danger' | 'default'
}

type ReviewSheetProps = {
  action: ReviewAction | null
  comment: string
  onCommentChange: (v: string) => void
  error: string
  busy: boolean
  onConfirm: () => void
  onClose: () => void
}

export function ReviewSheet({ action, comment, onCommentChange, error, busy, onConfirm, onClose }: ReviewSheetProps) {
  const { t } = useTranslation()
  const { sheetRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose, disabled: !action })
  if (!action) return null
  let descText = ''
  if (action.verdict === 'request_changes') descText = t('reviewSheet.desc.requestChanges')
  else if (action.verdict === 'pass' && action.apply) descText = t('reviewSheet.desc.passApply')
  else if (action.verdict === 'pass') descText = t('reviewSheet.desc.passOnly')
  else descText = t('reviewSheet.desc.reject')
  const fieldLabel = action.verdict === 'request_changes' ? t('reviewSheet.field.rework') : t('reviewSheet.field.supplement')
  const placeholder = action.verdict === 'request_changes' ? t('reviewSheet.placeholder.rework') : t('reviewSheet.placeholder.optional')
  const confirmLabel = action.apply ? t('reviewSheet.confirm.passApply') : t('reviewSheet.confirm.pass', { label: action.label })
  return (
    <div className="sheet-scrim" style={scrimStyle} role="dialog" aria-modal="true" aria-label={action.label} onClick={onClose}>
      <div ref={sheetRef} style={sheetStyle} className={'sheet-card review-dialog review-dialog-' + action.verdict} onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <span className="sheet-icon">
            <Icon name={action.verdict === 'reject' ? 'x' : action.verdict === 'pass' ? 'check' : 'rotate-ccw'} />
          </span>
          <div>
            <h2>{action.label}</h2>
            <p>{descText}</p>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body review-dialog-body">
          {error && (
            <div className="alert-box danger">
              {error}
            </div>
          )}
          <label className="field">
            <span>{fieldLabel}</span>
            <textarea
              rows={3}
              value={comment}
              onChange={(e) => onCommentChange(e.target.value)}
              placeholder={placeholder}
              autoFocus
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  onConfirm()
                }
              }}
            />
          </label>
        </div>
        <div className="design-sheet-foot review-dialog-foot">
          <button className="button" onClick={onClose} disabled={busy}>{t('common.action.cancel')}</button>
          <button className={'button ' + (action.tone === 'danger' ? 'danger' : action.tone === 'primary' ? 'primary' : '')} onClick={onConfirm} disabled={busy}>
            {busy ? <Icon name="spinner" className="spin" /> : <Icon name={action.verdict === 'reject' ? 'x' : action.verdict === 'pass' ? 'check' : 'rotate-ccw'} />}
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

type BlockedSheetProps = {
  action: BlockedActionState | null
  addTurns: number
  addMinutes: number
  quest: QuestMeta | null
  busy: boolean
  onConfirm: () => void
  onClose: () => void
}

export function BlockedSheet({ action, addTurns, addMinutes, quest, busy, onConfirm, onClose }: BlockedSheetProps) {
  const { t } = useTranslation()
  const { sheetRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose, disabled: !action })
  if (!action) return null
  return (
    <div className="sheet-scrim" style={scrimStyle} role="dialog" aria-modal="true" aria-label={action.label} onClick={onClose}>
      <div ref={sheetRef} style={sheetStyle} className={'sheet-card review-dialog blocked-confirm-dialog ' + (action.tone === 'danger' ? 'review-dialog-reject' : '')} onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <span className={'sheet-icon ' + (action.tone === 'danger' ? 'warning' : '')}>
            <Icon name={action.tone === 'danger' ? 'triangle-alert' : action.tone === 'primary' ? 'play' : 'shield'} />
          </span>
          <div>
            <h2>{action.label}</h2>
            <p>{action.description}</p>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body review-dialog-body">
          <div className="blocked-confirm-summary">
            <div>
              <span>{t('blockedSheet.extraTurns')}</span>
              <strong className="mono">{addTurns}</strong>
            </div>
            <div>
              <span>{t('blockedSheet.extraMinutes')}</span>
              <strong className="mono">{t('blockedSheet.minutes', { count: addMinutes })}</strong>
            </div>
          </div>
          {quest?.blocked_reason && (
            <div className="alert-box">{fmtBlockedReason(quest)}</div>
          )}
        </div>
        <div className="design-sheet-foot review-dialog-foot">
          <button className="button" onClick={onClose} disabled={busy}>{t('common.action.cancel')}</button>
          <button className={'button ' + (action.tone === 'danger' ? 'danger' : action.tone === 'primary' ? 'primary' : '')} onClick={onConfirm} disabled={busy}>
            {busy ? <Icon name="spinner" className="spin" /> : <Icon name={action.tone === 'danger' ? 'triangle-alert' : 'check'} />}
            {t('blockedSheet.confirm', { label: action.label })}
          </button>
        </div>
      </div>
    </div>
  )
}

type ApplyWarningSheetProps = {
  warning: ApplyWarningState | null
  busy: boolean
  onConfirm: () => void
  onClose: () => void
}

export function ApplyWarningSheet({ warning, busy, onConfirm, onClose }: ApplyWarningSheetProps) {
  const { t } = useTranslation()
  const { sheetRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose, disabled: !warning })
  if (!warning) return null
  const title = warning.fromReview ? t('applyWarning.title.fromReview') : t('applyWarning.title.plain')
  return (
    <div className="sheet-scrim" style={scrimStyle} role="dialog" aria-modal="true" aria-label={t('applyWarning.aria')} onClick={onClose}>
      <div ref={sheetRef} style={sheetStyle} className="sheet-card review-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <span className="sheet-icon warning">
            <Icon name="shield-alert" />
          </span>
          <div>
            <h2>{title}</h2>
            <p>{warning.message}</p>
            {warning.fromReview && (
              <p className="faint mt-1" style={{ fontSize: 12 }}>
                {t('applyWarning.fromReviewHint')}
              </p>
            )}
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body review-dialog-body">
          <div className="warning-list">
            {warning.warnings.map((warning, index) => (
              <div className={'warning-item severity-' + warning.severity} key={`${warning.category}-${warning.path || index}`}>
                <strong>{warningLabel(warning)}</strong>
              </div>
            ))}
          </div>
        </div>
        <div className="design-sheet-foot">
          <button className="button" onClick={onClose} disabled={busy}>{t('common.action.cancel')}</button>
          <button className="button danger" onClick={onConfirm} disabled={busy}>
            {busy ? <Icon name="spinner" className="spin" /> : <Icon name="shield-alert" />}
            {t('applyWarning.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}

type BackupDetailSheetProps = {
  detail: BackupDetailState
  onClose: () => void
}

export function BackupDetailSheet({ detail, onClose }: BackupDetailSheetProps) {
  const { t } = useTranslation()
  const { sheetRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose, disabled: !detail.item })
  if (!detail.loading && !detail.item) return null
  const backupLabels: Array<[string, string]> = [
    [t('backupDetail.mode'), detail.item?.mode || ''],
    [t('backupDetail.baseDir'), detail.item?.base_dir || ''],
    [t('backupDetail.baseBranch'), detail.item?.base_branch || ''],
    ['Base Commit', detail.item?.base_commit || ''],
    ['Apply Commit', detail.item?.apply_commit || ''],
    ['Work Commit', detail.item?.work_commit || ''],
    [t('backupDetail.filesChanged'), detail.item?.files_changed != null ? String(detail.item.files_changed) : ''],
    ['Bundle', detail.item?.bundle_path || ''],
    ['Patch', detail.item?.patch_path || ''],
    [t('backupDetail.createdAt'), detail.item?.created_at_ms ? fmtTime(detail.item.created_at_ms) : ''],
  ]
  return (
    <div className="sheet-scrim" style={scrimStyle} role="dialog" aria-modal="true" aria-label={t('backupDetail.aria')} onClick={onClose}>
      <div ref={sheetRef} style={sheetStyle} className="sheet-card review-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <span className="sheet-icon">
            <Icon name="save" />
          </span>
          <div>
            <h2>{t('backupDetail.title')}</h2>
            <p>{detail.item?.id || t('backupDetail.loading')}</p>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body review-dialog-body">
          {detail.loading ? (
            <div className="empty-page small">{t('backupDetail.loading')}</div>
          ) : detail.item ? (
            <div className="backup-detail-grid">
              {backupLabels.map(([label, value]) => (
                <div className="backup-detail-row" key={label}>
                  <span>{label}</span>
                  <code>{value || '-'}</code>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
