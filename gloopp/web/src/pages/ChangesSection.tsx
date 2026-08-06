import type React from 'react'
import Icon from '../components/Icon'
import { fmtTime } from '../components/util'
import type { QuestMeta } from '../api/types'
import { formatArtifactSize, type DiffFile, type KnowledgeArtifactPreview } from './questDetailHelpers'
import { useTranslation } from 'react-i18next'

export type ChangesSectionProps = {
  quest: QuestMeta
  loading: boolean
  diff: string | null
  diffSource: string | null
  diffBackupID: string | null
  onJumpToArtifacts: () => void
  onJumpToTrace: () => void
  onJumpToConclusion: () => void
  onJumpToArtifact: (id: string) => void
  onCopy: (value: string) => Promise<boolean> | boolean | undefined
  parsedFiles: DiffFile[]
  knowledgeArtifacts: KnowledgeArtifactPreview[]
}

export default function ChangesSection(props: ChangesSectionProps) {
  const { t } = useTranslation()
  const { quest, loading, diffSource, diffBackupID, onJumpToArtifacts, onJumpToTrace, onJumpToConclusion, onJumpToArtifact, onCopy, parsedFiles, knowledgeArtifacts } = props
  const effectType: QuestMeta['effect_type'] | undefined = quest.effect_type
  const isWorkspace = effectType === 'workspace_diff' || (!effectType && (quest.workspace_diff_pending || !!quest.diff_changed_files))
  const isReadonly = quest.workspace_mode === 'readonly'
  const isContextStore = effectType === 'context_store'
  const isExternal = effectType === 'external_side_effect'

  let tone: 'workspace' | 'readonly' | 'context' | 'external' | 'default' = 'default'
  let title = t('changes.title.default')
  let iconName = 'file-diff'
  let subtitle: React.ReactNode = null

  if (isReadonly) {
    tone = 'readonly'; title = t('changes.title.readonly'); iconName = 'shield'
  } else if (isContextStore) {
    tone = 'context'; title = t('changes.title.context'); iconName = 'database'
    subtitle = knowledgeArtifacts.length > 0 ? t('changes.subtitle.contextWith', { count: knowledgeArtifacts.length }) : t('changes.subtitle.contextDefault')
  } else if (isExternal) {
    tone = 'external'; title = t('changes.title.external'); iconName = 'external-link'
    subtitle = t('changes.subtitle.external')
  } else if (isWorkspace) {
    tone = 'workspace'; title = t('changes.title.workspace'); iconName = 'file-diff'
    subtitle = quest.workspace_mode ? (
      <>{t('changes.subtitle.modePrefix')}<span className="mono">{quest.workspace_mode}</span>{diffSource ? <>{t('changes.subtitle.sourcePrefix')}<span className="mono">{diffSource}</span></> : null}</>
    ) : null
  }

  return (
    <section id="quest-changes" className="panel changes-panel mt-3">
      <div className="panel-title">
        <h2>
          <span className={`changes-badge ${tone}`}>
            <Icon name={iconName} size={12} />
            {title}
          </span>
        </h2>
        {subtitle && <span className="changes-subtitle">{subtitle}</span>}
        {isWorkspace && quest.diff_changed_files != null && <span className="mono">{t('changes.fileCount', { count: quest.diff_changed_files })}</span>}
        {isWorkspace && quest.diff_additions != null && quest.diff_deletions != null && (
          <span className="mono diff-summary">
            <span className="diff-add">+{quest.diff_additions}</span>
            <span className="diff-del">-{quest.diff_deletions}</span>
          </span>
        )}
        {diffBackupID && <span className="mono faint">{t('changes.baseline', { id: diffBackupID.slice(0, 8) })}</span>}
      </div>

      {loading && (
        <div className="exec-trace-empty">
          <Icon name="spinner" className="spin" />
          <span>{t('changes.loading')}</span>
        </div>
      )}

      {!loading && isWorkspace && (
        parsedFiles.length > 0 ? (
          <div className="diff-files">
            {parsedFiles.map((file, i) => {
              const slug = encodeURIComponent(file.name.replace(/[^a-zA-Z0-9._-]/g, '_')) || String(i)
              return (
                <div className="diff-file-card" key={i} id={`diff-file-${slug}`} data-diff-file={file.name} style={{ scrollMarginTop: 72 }}>
                  <header className="diff-file-head">
                    <Icon name="file-text" size={14} />
                    <strong title={file.name}>{file.name}</strong>
                    <span className="diff-file-actions">
                      <button type="button" className="link-button micro" onClick={() => { onCopy(file.name) }} title={t('changes.copyPath')}>
                        <Icon name="copy" size={11} /> {t('changes.path')}
                      </button>
                      <span className="mono diff-file-stat">
                        {file.additions > 0 && <span className="diff-add">+{file.additions}</span>}
                        {file.deletions > 0 && <span className="diff-del">-{file.deletions}</span>}
                      </span>
                    </span>
                  </header>
                  <pre className="diff-view structured">{file.lines.map((line, j) => (
                    <span key={j} className={line.startsWith('+') ? 'line-add' : line.startsWith('-') ? 'line-del' : line.startsWith('@@') ? 'line-hunk' : ''}>{line + '\n'}</span>
                  ))}</pre>
                </div>
              )
            })}
          </div>
        ) : (
          <div className="changes-empty">
            <Icon name="circle-check" size={16} />
            <span>{t('changes.emptyWorkspace')}{quest.workspace_diff_pending ? t('changes.regenerateHint') : ''}</span>
          </div>
        )
      )}

      {!loading && isReadonly && (
        <div className="changes-explain">
          <Icon name="shield" size={16} />
          <div>
            <strong>{t('changes.readonlyTitle')}</strong>
            <span>{t('changes.readonlyBody1')}<em>{t('changes.readonlyBody2')}</em>{t('changes.readonlyBody3')}<br />{t('changes.readonlyBody4')}<button type="button" className="link-button micro" onClick={onJumpToArtifacts}>{t('changes.artifactsBtn')}</button>{t('changes.readonlyBody5')}<button type="button" className="link-button micro" onClick={onJumpToTrace}>{t('changes.traceBtn')}</button>{t('changes.readonlyBody6')}</span>
          </div>
        </div>
      )}

      {!loading && isContextStore && (
        <div className="changes-grid">
          {knowledgeArtifacts.length > 0 ? (
            knowledgeArtifacts.map((k) => {
              const add = k.diff?.added?.length ?? 0
              const chg = k.diff?.changed?.length ?? 0
              const rmv = k.diff?.removed?.length ?? 0
              return (
                <div className="changes-card" key={k.artifact.id}>
                  <header>
                    <Icon name="file-text" size={14} />
                    <strong title={k.artifact.name}>{k.artifact.name}</strong>
                    <span className="mono">{formatArtifactSize(k.artifact.size)}</span>
                  </header>
                  {(k.meta || add || chg || rmv) ? (
                    <ul className="changes-stat-list">
                      {k.meta?.file_count != null && <li><em>{t('changes.stat.file')}</em><b>{k.meta.file_count}</b></li>}
                      {k.meta?.dimension_count != null && <li><em>{t('changes.stat.dimension')}</em><b>{k.meta.dimension_count}</b></li>}
                      {k.meta?.total_size_bytes != null && <li><em>{t('changes.stat.size')}</em><b>{formatArtifactSize(k.meta.total_size_bytes)}</b></li>}
                      {(add || chg || rmv) && (
                        <li>
                          <em>{t('changes.stat.diff')}</em>
                          <b>
                            {add > 0 && <span className="diff-add">+{add} </span>}
                            {chg > 0 && <span className="diff-chg">~{chg} </span>}
                            {rmv > 0 && <span className="diff-del">-{rmv}</span>}
                          </b>
                        </li>
                      )}
                      {k.meta?.exported_at_ms && <li><em>{t('changes.stat.export')}</em><b>{fmtTime(k.meta.exported_at_ms)}</b></li>}
                    </ul>
                  ) : (
                    <p className="changes-muted">{t('changes.noMetadata')}</p>
                  )}
                  <footer>
                    <button type="button" className="link-button tiny" onClick={() => onJumpToArtifact(k.artifact.id)}>
                      <Icon name="file-text" size={11} /> {t('changes.viewArtifact')}
                    </button>
                    <a className="link-button tiny" href="/knowledge">
                      <Icon name="database" size={11} /> {t('changes.openKnowledge')}
                    </a>
                  </footer>
                </div>
              )
            })
          ) : (
            <div className="changes-card changes-explain">
              <Icon name="database" size={16} />
              <div>
                <strong>{t('changes.noKnowledgeArtifact')}</strong>
                <span>{t('changes.knowledgeHint1')}<button type="button" className="link-button micro" onClick={() => window.location.assign('/knowledge')}>{t('changes.knowledgeBtn')}</button>{t('changes.knowledgeHint2')}<button type="button" className="link-button micro" onClick={onJumpToTrace}>{t('changes.traceBtn')}</button>{t('changes.knowledgeHint3')}<code>context_store</code>/<code>knowledge</code>{t('changes.knowledgeHint4')}</span>
              </div>
            </div>
          )}
        </div>
      )}

      {!loading && isExternal && (
        <div className="changes-explain">
          <Icon name="external-link" size={16} />
          <div>
            <strong>{t('changes.externalTitle')}</strong>
            <span>{t('changes.externalHint1')}<button type="button" className="link-button micro" onClick={onJumpToTrace}>{t('changes.traceBtn')}</button>{t('changes.externalHint2')}<button type="button" className="link-button micro" onClick={onJumpToConclusion}>{t('changes.conclusionBtn')}</button>{t('changes.externalHint3')}</span>
          </div>
        </div>
      )}

      {!loading && !isWorkspace && !isReadonly && !isContextStore && !isExternal && (
        <div className="changes-empty">
          <Icon name="info" size={16} />
          <span>{t('changes.unknown')}</span>
        </div>
      )}
    </section>
  )
}
