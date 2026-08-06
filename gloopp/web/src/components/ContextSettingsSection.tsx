import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { ContextDim, ContextDimInfo, KnowledgeExportMeta, KnowledgeExportPreview } from '../api/types'
import { useContextRefreshStream } from '../hooks/useContextRefreshStream'
import i18n from '../i18n'
import Icon from './Icon'
import MarkdownRenderer from './MarkdownRenderer'

type Props = {
  onError: (msg: string) => void
  onOpenQuest: (qid: string) => void
  embedded?: boolean
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

export function formatTime(ms: number): string {
  const d = new Date(ms)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffHours = diffMs / (1000 * 60 * 60)
  if (diffHours < 1) return i18n.t('context.time.minutesAgo', { count: Math.round(diffMs / (1000 * 60)) })
  if (diffHours < 24) return i18n.t('context.time.hoursAgo', { count: Math.round(diffHours) })
  const diffDays = diffHours / 24
  if (diffDays < 30) return i18n.t('context.time.daysAgo', { count: Math.round(diffDays) })
  return d.toLocaleDateString()
}

function stripLeadingMarkdownH1(source: string): string {
  return source.replace(/^\s*#\s+[^\n]+\n+/, '').trimStart()
}

function yamlQuote(value: string): string {
  return '"' + value.replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"'
}

function okfTimestamp(ms: number): string {
  if (!ms) return new Date().toISOString()
  return new Date(ms).toISOString()
}

function safeKnowledgeFilename(name: string): string {
  const cleaned = name.trim().replace(/[^a-zA-Z0-9_-]+/g, '_').replace(/^[._-]+|[._-]+$/g, '')
  return cleaned || 'untitled'
}

function renderOKFDocument(meta: ContextDimInfo, body: string): string {
  const tags = ['gloop', 'context', meta.name].filter(Boolean).map(yamlQuote).join(', ')
  const frontmatter = [
    '---',
    `type: ${yamlQuote('gloop_context_dimension')}`,
    `title: ${yamlQuote(meta.title || meta.name)}`,
    meta.description ? `description: ${yamlQuote(meta.description)}` : '',
    `resource: ${yamlQuote('gloop://context/dims/' + meta.name)}`,
    `tags: [${tags}]`,
    `timestamp: ${yamlQuote(okfTimestamp(meta.updated_at_ms))}`,
    '---',
  ].filter(Boolean).join('\n')
  return frontmatter + '\n\n' + stripLeadingMarkdownH1(body).trim() + '\n'
}

export function ContextWorkbench({ onError, onOpenQuest, embedded = false }: Props) {
  const { t } = useTranslation()
  const [dims, setDims] = useState<ContextDimInfo[]>([])
  const [dimsLoading, setDimsLoading] = useState(true)
  const [selectedDim, setSelectedDim] = useState<ContextDim | null>(null)
  const [dimLoading, setDimLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [lastContextRefreshQid, setLastContextRefreshQid] = useState('')
  const [contextSummary, setContextSummary] = useState('')
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [exportPreview, setExportPreview] = useState<KnowledgeExportPreview | null>(null)
  const [lastExport, setLastExport] = useState<KnowledgeExportMeta | null>(null)
  const [exportPreviewLoading, setExportPreviewLoading] = useState(false)
  const [importing, setImporting] = useState(false)
  const [importName, setImportName] = useState('')
  const [importContent, setImportContent] = useState('')
  const [importFile, setImportFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const fetchContextDims = useCallback(async () => {
    setDimsLoading(true)
    try {
      const res = await api.get<{ items: ContextDimInfo[] }>('/api/context/dims')
      setDims(res.items || [])
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.loadDimsFail'))
    } finally {
      setDimsLoading(false)
    }
  }, [onError])

  const fetchContextSummary = useCallback(async () => {
    setSummaryLoading(true)
    try {
      const res = await api.get<{ summary: string }>('/api/context/summary')
      setContextSummary(res.summary || '')
    } catch {
      setContextSummary('')
    } finally {
      setSummaryLoading(false)
    }
  }, [])

  const loadDim = useCallback(async (name: string) => {
    setDimLoading(true)
    try {
      const res = await api.get<{ dim: ContextDim }>('/api/context/dims/' + encodeURIComponent(name))
      setSelectedDim(res.dim)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.loadDimFail'))
    } finally {
      setDimLoading(false)
    }
  }, [onError])

  const fetchExportPreview = useCallback(async () => {
    setExportPreviewLoading(true)
    try {
      const res = await api.get<{ preview: KnowledgeExportPreview; last?: KnowledgeExportMeta | null }>('/api/context/export/preview')
      setExportPreview(res.preview || null)
      setLastExport(res.last || null)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.loadExportPreviewFail'))
    } finally {
      setExportPreviewLoading(false)
    }
  }, [onError])

  useEffect(() => {
    fetchContextDims()
    fetchContextSummary()
    fetchExportPreview()
  }, [fetchContextDims, fetchContextSummary, fetchExportPreview])

  useContextRefreshStream({
    enabled: true,
    refreshContext: () => {
      fetchContextDims()
      fetchContextSummary()
      fetchExportPreview()
      if (selectedDim?.name) {
        loadDim(selectedDim.name)
      }
    },
  })

  useEffect(() => {
    if (dimsLoading || dimLoading || dims.length === 0) return
    if (selectedDim && dims.some((item) => item.name === selectedDim.name)) return
    loadDim(dims[0].name)
  }, [dims, dimsLoading, dimLoading, selectedDim, loadDim])

  const viewDim = useCallback(async (name: string) => {
    if (selectedDim?.name === name) {
      return
    }
    await loadDim(name)
  }, [selectedDim, loadDim])

  const refreshContext = useCallback(async () => {
    setRefreshing(true)
    setLastContextRefreshQid('')
    try {
      const res = await api.post<{ qid?: string }>('/api/context/refresh')
      setLastContextRefreshQid(res.qid || '')
      setTimeout(() => {
        fetchContextDims()
        fetchContextSummary()
        fetchExportPreview()
      }, 2000)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.refreshFail'))
    } finally {
      setRefreshing(false)
    }
  }, [fetchContextDims, fetchContextSummary, fetchExportPreview, onError])

  const exportDim = useCallback(async (name: string) => {
    try {
      const download = await api.download('/api/context/dims/' + encodeURIComponent(name) + '/export')
      const a = document.createElement('a')
      a.href = download.url
      a.download = download.filename || 'context_' + name + '.md'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(download.url), 1000)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.exportFail'))
    }
  }, [onError])

  const exportAll = useCallback(async () => {
    setExporting(true)
    try {
      const download = await api.download('/api/context/export?format=okf')
      const a = document.createElement('a')
      a.href = download.url
      a.download = download.filename || 'gloop_knowledge_bundle.zip'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(download.url), 1000)
      await fetchExportPreview()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.exportFail'))
    } finally {
      setExporting(false)
    }
  }, [fetchExportPreview, onError])

  const openImport = useCallback(() => {
    setImportName('')
    setImportContent('')
    setImportFile(null)
    setImporting(true)
    setTimeout(() => fileInputRef.current?.click(), 50)
  }, [])

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return
      setImportFile(file)
      let name = file.name.replace(/\.md$/i, '')
      name = name.replace(/^context_/, '')
      name = name.replace(/[^a-zA-Z0-9_-]/g, '_')
      setImportName(name)
      const reader = new FileReader()
      reader.onload = () => {
        setImportContent(String(reader.result || ''))
      }
      reader.onerror = () => {
        onError(t('context.error.importReadFail'))
        setImporting(false)
      }
      reader.readAsText(file)
      e.target.value = ''
    },
    [onError],
  )

  const confirmImport = useCallback(async () => {
    if (!importName.trim() || !importContent.trim()) {
      onError(t('context.error.importEmptyFail'))
      return
    }
    try {
      await api.put('/api/context/dims/' + encodeURIComponent(importName.trim()), {
        content: importContent,
      })
      setImporting(false)
      fetchContextDims()
      fetchExportPreview()
      if (selectedDim?.name === importName.trim()) {
        setSelectedDim(null)
      }
    } catch (e) {
      onError(e instanceof Error ? e.message : t('context.error.importFail'))
    }
  }, [importName, importContent, fetchContextDims, fetchExportPreview, selectedDim, onError])

  const cancelImport = useCallback(() => {
    setImporting(false)
    setImportName('')
    setImportContent('')
    setImportFile(null)
  }, [])

  const contextTotalBytes = dims.reduce((sum, item) => sum + (item.size_bytes || 0), 0)
  const latestContextUpdated = dims.reduce((latest, item) => Math.max(latest, item.updated_at_ms || 0), 0)
  const selectedDimInfo = selectedDim ? dims.find((item) => item.name === selectedDim.name) : null
  const previewFiles = exportPreview?.files || []
  const exportDiff = exportPreview?.diff || null
  const selectedDimPreviewPath = selectedDimInfo ? 'dimensions/' + safeKnowledgeFilename(selectedDimInfo.name) + '.md' : ''
  const selectedDimPreviewFile = selectedDimPreviewPath
    ? previewFiles.find((file) => file.path === selectedDimPreviewPath)
    : null
  const selectedDimPreviewBody = selectedDim && selectedDimInfo ? renderOKFDocument(selectedDimInfo, selectedDim.body || '') : ''
  const hasExportChanges = Boolean(
    exportDiff &&
      (exportDiff.added.length > 0 ||
        exportDiff.changed.length > 0 ||
        exportDiff.removed.length > 0),
  )

  return (
    <section className={embedded ? 'settings-section' : 'knowledge-page'}>
      <div className="section-head knowledge-actions-head">
        {embedded && <h2>{t('context.title')}</h2>}
        <div className="section-actions">
          <button
            type="button"
            className="button ghost"
            onClick={refreshContext}
            disabled={refreshing || dimsLoading}
          >
            <Icon name="refresh" className={refreshing ? 'spin' : ''} />
            {t('context.action.refresh')}
          </button>
          <button
            type="button"
            className="button ghost"
            onClick={openImport}
            disabled={importing || dimsLoading}
          >
            <Icon name="upload" />
            {t('context.action.import')}
          </button>
          <button
            type="button"
            className="button ghost"
            onClick={exportAll}
            disabled={exporting || dimsLoading || dims.length === 0}
          >
            <Icon name="save" />
            {t('context.action.exportOKF')}
          </button>
        </div>
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept=".md,text/markdown,text/plain"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />
      <div className="context-workbench">
        <div className="context-overview">
          <div className="context-summary-box">
            <div className="context-summary-head">
              <div>
                <strong>{t('context.overview.title')}</strong>
                <span>{t('context.overview.desc')}</span>
              </div>
              <button
                type="button"
                className="icon-button"
                onClick={fetchContextSummary}
                title={t('context.action.refreshSummary')}
              >
                <Icon name="refresh" size={14} className={summaryLoading ? 'spin' : ''} />
              </button>
            </div>
            {summaryLoading ? (
              <div className="empty-page small">{t('context.overview.loading')}</div>
            ) : contextSummary ? (
              <MarkdownRenderer
                source={contextSummary}
                className="context-summary-body compact"
              />
            ) : (
              <div className="context-empty-summary">
                <span>{t('context.overview.empty')}</span>
                <button
                  type="button"
                  className="button ghost compact"
                  onClick={refreshContext}
                  disabled={refreshing || dimsLoading}
                >
                  <Icon name="refresh" className={refreshing ? 'spin' : ''} />
                  {t('context.action.refreshGenerate')}
                </button>
              </div>
            )}
          </div>
          <div className="context-metrics">
            <div>
              <span>{t('context.metrics.dims')}</span>
              <strong>{dimsLoading ? '...' : dims.length}</strong>
            </div>
            <div>
              <span>{t('context.metrics.size')}</span>
              <strong>{dimsLoading ? '...' : formatSize(contextTotalBytes)}</strong>
            </div>
            <div>
              <span>{t('context.metrics.lastUpdated')}</span>
              <strong>{latestContextUpdated ? formatTime(latestContextUpdated) : '-'}</strong>
            </div>
          </div>
        </div>
        <div className="context-callout">
          <Icon name="info" />
          <span>
            {t('context.callout.body')}
            {lastContextRefreshQid && (
              <>
                {' '}{t('context.callout.lastRefresh')}
                <button
                  type="button"
                  className="inline-link-button"
                  onClick={() => onOpenQuest(lastContextRefreshQid)}
                >
                  <code>{lastContextRefreshQid}</code>
                </button>
              </>
            )}
          </span>
        </div>
        {!embedded && (
          <div className="knowledge-preview-panel">
            <div className="context-panel-head">
              <strong>{t('context.exportPreview.title')}</strong>
              <button
                type="button"
                className="button ghost compact"
                onClick={fetchExportPreview}
                disabled={exportPreviewLoading}
              >
                <Icon name="refresh" className={exportPreviewLoading ? 'spin' : ''} />
                {t('context.action.recalculate')}
              </button>
            </div>
            <div className="knowledge-preview-grid">
              <div>
                <span>{t('context.exportPreview.format')}</span>
                <strong>{exportPreview?.format || 'gloop.okf.v0'}</strong>
              </div>
              <div>
                <span>{t('context.exportPreview.files')}</span>
                <strong>{exportPreviewLoading ? '...' : exportPreview?.file_count ?? 0}</strong>
              </div>
              <div>
                <span>{t('context.exportPreview.totalSize')}</span>
                <strong>{exportPreviewLoading ? '...' : formatSize(exportPreview?.total_size_bytes || 0)}</strong>
              </div>
              <div>
                <span>{t('context.exportPreview.lastExport')}</span>
                <strong>{lastExport?.exported_at_ms ? formatTime(lastExport.exported_at_ms) : '-'}</strong>
              </div>
            </div>
            {lastExport?.filename && (
              <div className="knowledge-last-export">
                <Icon name="check" />
                <span>
                  {t('context.exportPreview.lastExportLine', {
                    filename: lastExport.filename,
                    fileCount: lastExport.file_count,
                    size: formatSize(lastExport.total_size_bytes),
                  })}
                </span>
              </div>
            )}
            {exportDiff && (
              <div className="knowledge-diff-strip">
                <div>
                  <span>{t('context.diff.added')}</span>
                  <strong>{exportDiff.added.length}</strong>
                </div>
                <div>
                  <span>{t('context.diff.changed')}</span>
                  <strong>{exportDiff.changed.length}</strong>
                </div>
                <div>
                  <span>{t('context.diff.removed')}</span>
                  <strong>{exportDiff.removed.length}</strong>
                </div>
                <div>
                  <span>{t('context.diff.unchanged')}</span>
                  <strong>{exportDiff.unchanged}</strong>
                </div>
              </div>
            )}
            {hasExportChanges && exportDiff && (
              <div className="knowledge-diff-list">
                {exportDiff.added.map((file) => (
                  <div className="knowledge-diff-row added" key={'added:' + file.path}>
                    <span>{t('context.diff.added')}</span>
                    <code>{file.path}</code>
                    <em>{formatSize(file.size_bytes)}</em>
                  </div>
                ))}
                {exportDiff.changed.map((file) => (
                  <div className="knowledge-diff-row changed" key={'changed:' + file.path}>
                    <span>{t('context.diff.changed')}</span>
                    <code>{file.path}</code>
                    <em>{formatSize(file.size_bytes)}</em>
                  </div>
                ))}
                {exportDiff.removed.map((file) => (
                  <div className="knowledge-diff-row removed" key={'removed:' + file.path}>
                    <span>{t('context.diff.removed')}</span>
                    <code>{file.path}</code>
                    <em>{formatSize(file.size_bytes)}</em>
                  </div>
                ))}
              </div>
            )}
            <div className="knowledge-file-list">
              {exportPreviewLoading && <div className="empty-page small">{t('context.exportPreview.calculating')}</div>}
              {!exportPreviewLoading && previewFiles.length === 0 && (
                <div className="empty-page small">{t('context.exportPreview.empty')}</div>
              )}
              {!exportPreviewLoading && previewFiles.map((file) => (
                <div className="knowledge-file-row" key={file.path}>
                  <code>{file.path}</code>
                  <span>{formatSize(file.size_bytes)}</span>
                </div>
              ))}
            </div>
          </div>
        )}
        {importing && (
          <div className="context-import-panel">
            <div className="import-head">
              <h3>{t('context.importPanel.title')}</h3>
              <button
                type="button"
                className="icon-button"
                onClick={cancelImport}
                title={t('context.action.cancel')}
              >
                <Icon name="x" size={14} />
              </button>
            </div>
            <div className="import-body context-import-body">
              <label className="field">
                <span>{t('context.importPanel.dimName')}</span>
                <input
                  type="text"
                  value={importName}
                  onChange={(e) => setImportName(e.target.value)}
                  placeholder={t('context.importPanel.dimNamePlaceholder')}
                  maxLength={64}
                />
              </label>
              <div className="import-preview">
                <div className="import-preview-head">
                  <span>{t('context.importPanel.contentPreview')}</span>
                  <span className="import-size">
                    {importContent ? formatSize(importContent.length) : '0 B'}
                  </span>
                </div>
                {importContent ? (
                  <pre className="import-preview-body">
                    {importContent.slice(0, 500)}
                    {importContent.length > 500 && '\n...'}
                  </pre>
                ) : (
                  <div className="empty-page small">{t('context.importPanel.selectFile')}</div>
                )}
              </div>
            </div>
            <div className="import-actions">
              <button
                type="button"
                className="button ghost"
                onClick={cancelImport}
              >
                {t('context.action.cancel')}
              </button>
              <button
                type="button"
                className="button primary"
                onClick={confirmImport}
                disabled={!importName.trim() || !importContent.trim()}
              >
                <Icon name="upload" />
                {t('context.action.confirmImport')}
              </button>
            </div>
          </div>
        )}
        <div className="context-layout">
          <div className="context-list-panel">
            <div className="context-panel-head">
              <strong>{t('context.dimList.title')}</strong>
              <span>{dimsLoading ? t('context.dimList.loading') : t('context.dimList.count', { count: dims.length })}</span>
            </div>
            <div className="settings-box divide context-list-box">
              {dimsLoading && <div className="empty-page small">{t('context.dimList.loadingDots')}</div>}
              {!dimsLoading && dims.length === 0 && (
                <div className="context-empty-list">
                  <Icon name="file-text" />
                  <strong>{t('context.dimList.emptyTitle')}</strong>
                  <span>{t('context.dimList.emptyDesc')}</span>
                  <button
                    type="button"
                    className="button primary compact"
                    onClick={refreshContext}
                    disabled={refreshing}
                  >
                    <Icon name="refresh" className={refreshing ? 'spin' : ''} />
                    {t('context.action.generate')}
                  </button>
                </div>
              )}
              {dims.map((d) => (
                <article
                  key={d.name}
                  className={'settings-row context-row' + (selectedDim?.name === d.name ? ' active' : '')}
                  onClick={() => viewDim(d.name)}
                >
                  <div>
                    <strong>{d.title}</strong>
                    <span>{d.name}</span>
                    <em className="dim-meta">
                      {formatSize(d.size_bytes)} · {formatTime(d.updated_at_ms)}
                    </em>
                  </div>
                  <button
                    type="button"
                    className="icon-button"
                    onClick={(e) => {
                      e.stopPropagation()
                      exportDim(d.name)
                    }}
                    title={t('context.action.exportMd')}
                  >
                    <Icon name="save" size={14} />
                  </button>
                </article>
              ))}
            </div>
          </div>
          <div className="context-detail-panel">
            <div className="context-panel-head">
              <strong>{t('context.detail.title')}</strong>
              {selectedDimInfo && (
                <button
                  type="button"
                  className="button ghost compact"
                  onClick={() => exportDim(selectedDimInfo.name)}
                >
                  <Icon name="save" />
                  {t('context.action.export')}
                </button>
              )}
            </div>
            {selectedDimInfo && selectedDim?.name === selectedDimInfo.name ? (
              <div className="context-detail">
                {dimLoading ? (
                  <div className="empty-page small">{t('context.detail.loading')}</div>
                ) : (
                  <>
                    <div className="context-meta context-export-meta">
                      <div>
                        <strong>{selectedDimInfo.title}</strong>
                        <span className="dim-desc">
                          <code>{selectedDimPreviewPath}</code>
                          {selectedDimPreviewFile && <> · {formatSize(selectedDimPreviewFile.size_bytes)}</>}
                        </span>
                      </div>
                      <span className="dim-updated">
                        {t('context.detail.updatedAt', { time: new Date(selectedDimInfo.updated_at_ms).toLocaleString() })}
                      </span>
                    </div>
                    <div className="context-preview-frame">
                      <div className="context-preview-bar">
                        <span>
                          <Icon name="file-text" size={13} />
                          OKF Markdown
                        </span>
                        <code>{selectedDimPreviewPath}</code>
                      </div>
                      <pre className="context-frontmatter-preview">
                        {selectedDimPreviewBody.split('---\n\n')[0] + '---'}
                      </pre>
                    </div>
                    <MarkdownRenderer
                      source={selectedDimPreviewBody}
                      className="context-body compact"
                      stripFrontmatter
                    />
                  </>
                )}
              </div>
            ) : (
              <div className="context-empty-detail">
                <Icon name="file-text" />
                <strong>{t('context.detail.emptyTitle')}</strong>
                <span>{t('context.detail.emptyDesc')}</span>
              </div>
            )}
          </div>
        </div>
      </div>
      <p className="section-hint">
        {t('context.hint')}
      </p>
    </section>
  )
}

export default function ContextSettingsSection(props: Props) {
  return <ContextWorkbench {...props} embedded />
}
