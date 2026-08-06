import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useEventStream, useMicroStream } from '../api/events'
import {
  applyQuestChanges,
  answerQuest,
  cloneQuest,
  commentOnQuest,
  discardQuestChanges,
  getQuestBackup,
  getQuestDetail,
  getQuestDiff,
  getQuestTrace,
  listQuestBackups,
  recoverQuestAgent,
  resolveBlockedQuest,
  resolveQuest,
  spawnExecuteFromDesignQuest,
  startQuest,
  stopQuest,
  upgradeQuestWorkflowMode,
  type QuestDetailResponse,
  type ResolveQuestPayload,
} from '../api/quests'
import type { ApplyBackupMeta, DesignDoc, EvidenceRef, FanoutGroup, KnowledgeExportDiff, KnowledgeExportMeta, LoopStateSpine, QuestArtifact, QuestCheck, QuestEvent, QuestMeta, QuestReports, QuestReview, StructuredReview, ThreadPost, Verdict } from '../api/types'
import AgentMessageRenderer from '../components/AgentMessageRenderer'
import DesignPlanCard from '../components/DesignPlanCard'
import ExecutionTrace from '../components/ExecutionTrace'
import EvidenceList from '../components/EvidenceList'
import FanoutTreePanel from '../components/FanoutTreePanel'
import FinalVerdictCard from '../components/FinalVerdictCard'
import Icon from '../components/Icon'
import ImpactSummaryCard from '../components/ImpactSummaryCard'
import LoopStateCard from '../components/LoopStateCard'
import Logo from '../components/Logo'
import MageReviewBand from '../components/MageReviewBand'
import MageScoreChip from '../components/MageScoreChip'
import MakerReportCard from '../components/MakerReportCard'
import MarkdownRenderer from '../components/MarkdownRenderer'
import PhaseProgressBar from '../components/PhaseProgressBar'
import QuestSummary from '../components/QuestSummary'
import ReviewReportCard from '../components/ReviewReportCard'
import StatusBadge from '../components/StatusBadge'
import ThreadLedgerPanel from '../components/ThreadLedgerPanel'
import { fmtBlockedReason, fmtTime, shortId, typeLabel, debounce } from '../components/util'
import { selectLatestMageReview } from '../domain/mageReview'
import {
  getDesignPlan,
  questActionState,
  questDurationUsageLabel,
  questPhaseNodes,
  questPhaseView,
  questResourceUsage,
  questStatusBadgeInfo,
  questTimelineEntries,
  questTurnUsageLabel,
  reviewActionDraft,
  type TimelineEntry,
  type TimelineVariant,
} from '../domain/questSelectors'
import {
  commentsFromTrace,
  contextPacksFromTrace,
  evidenceFromTraceItems,
  historicEventsFromTraceItems,
  mergeByTimestamp,
  messagesFromTrace,
  nativeEvidenceFromTrace,
  toolResultsFromTrace,
  type EvidenceState,
  type QuestTraceItem,
} from '../features/quest/traceAdapters'
import { emitStuckDiagnosisPath, getStuckDiagnosis, type StuckDiagnosis } from '../features/quest/stuckDiagnosisMetrics'
import type { NativeToolItem, MessageItem, CommentItem, ContextPackItem, ToolResultItem } from '../features/quest/traceModel'
import {
  artifactURL,
  formatArtifactSize,
  outputIconName,
  isTextualArtifact,
  normalizeTextForDedupe,
  fastHash,
  pickSummaryArtifact,
  type ApplyWarningState,
  type BackupDetailState,
  BLOCKED_ACTION,
  type BlockedActionState,
  parseDiffFiles,
  warningsFromError,
  checkTitle,
  checkMeta,
  checkMessage,
  writeClipboardText,
  type DiffFile,
  selectQuestProducerArtifact,
  shouldShowFinalVerdict,
  getWorkflowUpgradeInfo,
  deriveGoalRuntimeView,
} from './questDetailHelpers'
import ChangesSection from './ChangesSection'
import { PhaseNode, PathValue, ResourceBudgetRows } from './questDetailParts'
import { ReviewSheet, BlockedSheet, ApplyWarningSheet, BackupDetailSheet, type ReviewAction } from './QuestDetailSheets'
import CurrentWorkStatus from './CurrentWorkStatus'
import i18n from '../i18n'

type Props = {
  qid: string
  onBack: () => void
  onError: (msg: string) => void
  onChanged: () => void
  onOpenQuest: (qid: string) => void
}

function StuckActionPanel({ diagnosis, onJumpToTrace, onJumpToEvidence }: {
  diagnosis: StuckDiagnosis
  onJumpToTrace: () => void
  onJumpToEvidence: () => void
}) {
  const { t } = useTranslation()
  return (
    <section id="quest-current-action" className={'panel stuck-action-panel source-' + diagnosis.source + ' risk-' + diagnosis.riskLevel}>
      <div className="panel-title">
        <h2>{diagnosis.source === 'runtime' ? t('quest.diagnosis.runtimeStuckState') : t('quest.diagnosis.panelTitle')}</h2>
        <span className="mono">{diagnosis.kind}</span>
      </div>
      <div className="stuck-action-grid">
        <div className="stuck-action-main">
          <div className="stuck-action-label">{diagnosis.title}</div>
          <p>{diagnosis.reason}</p>
          <strong>{diagnosis.recommendedAction}</strong>
        </div>
        <div className="stuck-action-meta">
          <span><em>source</em>{diagnosis.source === 'runtime' ? 'agent_sessions' : 'human_exception'}</span>
          <span><em>risk</em>{diagnosis.riskLevel}</span>
          {diagnosis.sourceStatus && <span><em>status</em>{diagnosis.sourceStatus}</span>}
          {diagnosis.sessionId && <span><em>session</em>{diagnosis.sessionId}</span>}
        </div>
      </div>
      {diagnosis.availableActions.length > 0 && (
        <div className="stuck-action-tokens">
          {diagnosis.availableActions.map((action) => (
            <span key={action} className="chip mono tiny">{action}</span>
          ))}
        </div>
      )}
      {diagnosis.evidenceSummary && (
        <div className="stuck-evidence-summary">
          <Icon name="search" size={13} />
          <span>{diagnosis.evidenceSummary}</span>
        </div>
      )}
      <div className="panel-title-actions stuck-action-buttons">
        <button type="button" className="link-button tiny" onClick={onJumpToEvidence}>
          <Icon name="search" size={12} /> {i18n.t('quest.action.viewEvidence')}
        </button>
        <button type="button" className="link-button tiny" onClick={onJumpToTrace}>
          <Icon name="radio" size={12} /> {i18n.t('quest.action.viewTrace')}
        </button>
      </div>
    </section>
  )
}

export default function QuestDetail({ qid, onBack, onError, onChanged, onOpenQuest }: Props) {
  const { t } = useTranslation()
  const [quest, setQuest] = useState<QuestMeta | null>(null)
  const [designDoc, setDesignDoc] = useState<DesignDoc | null>(null)
  const [checks, setChecks] = useState<QuestCheck[]>([])
  const [reviews, setReviews] = useState<QuestReview[]>([])
  const [reports, setReports] = useState<QuestReports | null>(null)
  const [threadPosts, setThreadPosts] = useState<ThreadPost[]>([])
  const [fanoutGroup, setFanoutGroup] = useState<FanoutGroup | null>(null)
  const [loopStateSpine, setLoopStateSpine] = useState<LoopStateSpine | null>(null)
  const [events, setEvents] = useState<QuestEvent[]>([])
  const [diff, setDiff] = useState<string | null>(null)
  const [diffSource, setDiffSource] = useState<string | null>(null)
  const [diffBackupID, setDiffBackupID] = useState<string | null>(null)
  const [backups, setBackups] = useState<ApplyBackupMeta[]>([])
  const [evidence, setEvidence] = useState<EvidenceState | null>(null)
  const [busy, setBusy] = useState(false)
  const [comment, setComment] = useState('')
  const [answerText, setAnswerText] = useState('')
  const [reviewComment, setReviewComment] = useState('')
  const [reviewError, setReviewError] = useState('')
  const [sendingComment, setSendingComment] = useState(false)
  const [sendingAnswer, setSendingAnswer] = useState(false)
  const [loadingDiff, setLoadingDiff] = useState(false)
  const [loadingBackups, setLoadingBackups] = useState(false)
  const [loadingEvidence, setLoadingEvidence] = useState(false)
  const [blockedAddTurns, setBlockedAddTurns] = useState(10)
  const [blockedAddMinutes, setBlockedAddMinutes] = useState(30)
  const [mobileInfoOpen, setMobileInfoOpen] = useState(false)
  const [queryExpanded, setQueryExpanded] = useState(false)
  const [queryCopied, setQueryCopied] = useState(false)
  const [designPlanText, setDesignPlanText] = useState('')
  const [loadingDesignPlan, setLoadingDesignPlan] = useState(false)
  const [reviewAction, setReviewAction] = useState<ReviewAction | null>(null)
  const [blockedAction, setBlockedAction] = useState<BlockedActionState | null>(null)
  const [applyWarning, setApplyWarning] = useState<ApplyWarningState | null>(null)
  const [backupDetail, setBackupDetail] = useState<BackupDetailState>({ loading: false, item: null })

  const load = useCallback(async () => {
    try {
      const res = await getQuestDetail(qid)
      setQuest(res.quest)
      if (res.quest?.id && res.quest.id !== qid) {
        onOpenQuest(res.quest.id)
      }
      setDesignDoc(res.design || null)
      setChecks(res.checks || [])
      setReviews(res.reviews || [])
      setReports(res.reports || null)
      setThreadPosts(res.thread_posts || [])
      setFanoutGroup(res.fanout_group || null)
      setLoopStateSpine(res.loop_state_spine || null)
      if (res.agent_sessions?.length) {
        setQuest((prev) => prev ? { ...prev, agent_sessions: res.agent_sessions } : res.quest)
      }
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.loadDetail'))
    }
  }, [qid, onError, onOpenQuest])

  const loadBackups = useCallback(async () => {
    setLoadingBackups(true)
    try {
      const res = await listQuestBackups(qid)
      setBackups(res.items || [])
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.loadBackup'))
    } finally {
      setLoadingBackups(false)
    }
  }, [qid, onError])

  useEffect(() => {
    setQuest(null)
    setDesignDoc(null)
    setChecks([])
    setReviews([])
    setReports(null)
    setThreadPosts([])
    setFanoutGroup(null)
    setLoopStateSpine(null)
    setEvents([])
    setDiff(null)
    setBackups([])
    setEvidence(null)
    setQueryExpanded(false)
    setQueryCopied(false)
    setAnswerText('')
    setDesignPlanText('')
    setLoadingDesignPlan(false)
    setBackupDetail({ loading: false, item: null })
    load()
    loadBackups()
  }, [qid, load, loadBackups])

  useEffect(() => {
    if (quest && !evidence && !loadingEvidence) {
      loadEvidence()
    }
    if (quest && !diff && !loadingDiff && quest.diff_changed_files) {
      loadDiff()
    }
  }, [quest])

  const reloadRef = useRef(() => {})
  reloadRef.current = () => {
    load()
    onChanged()
  }
  const debouncedReload = useMemo(() => debounce(() => reloadRef.current(), 200), [])

  const [liveEventAtMs, setLiveEventAtMs] = useState(0)
  const streamState = useEventStream((event) => {
    setEvents((prev) => [...prev.slice(-200), event])
    setLiveEventAtMs(event.ts || event.timestamp || Date.now())
    if (!event.type.startsWith('micro.')) {
      debouncedReload()
    }
  }, qid)

  // 新鲜度只看本次会话收到的实时事件，不混历史事件，避免重开旧 quest 时误报 stale。
  const lastEventAtMs = liveEventAtMs

  const [liveMessages, setLiveMessages] = useState<MessageItem[]>([])
  const [liveContextPacks, setLiveContextPacks] = useState<ContextPackItem[]>([])
  const actionState = questActionState(quest)
  const phaseView = questPhaseView(quest)
  const statusBadge = questStatusBadgeInfo(quest)
  const phaseNodes = questPhaseNodes(quest)
  const designPlan = getDesignPlan(quest)

  useEffect(() => {
    let cancelled = false
    async function loadDesignPlanText() {
      if (!quest || !designPlan) {
        setDesignPlanText('')
        setLoadingDesignPlan(false)
        return
      }
      const isExternal = designPlan.storage_path?.startsWith('http://') || designPlan.storage_path?.startsWith('https://')
      if (isExternal) {
        setDesignPlanText('')
        setLoadingDesignPlan(false)
        return
      }
      setLoadingDesignPlan(true)
      try {
        const res = await fetch(artifactURL(quest.id, designPlan))
        if (!res.ok || !isTextualArtifact(designPlan)) {
          if (!cancelled) setDesignPlanText('')
          return
        }
        const text = await res.text()
        if (!cancelled) setDesignPlanText(text)
      } catch {
        if (!cancelled) setDesignPlanText('')
      } finally {
        if (!cancelled) setLoadingDesignPlan(false)
      }
    }
    loadDesignPlanText()
    return () => {
      cancelled = true
    }
  }, [designPlan, quest])

  useMicroStream((event) => {
    if (event.type === 'micro.assistant_msg') {
      const p = event.payload as Record<string, unknown> | undefined
      if (p && typeof p.content === 'string') {
        setLiveMessages((prev) => [...prev.slice(-30), {
          sid: event.session_id || '',
          content: p.content as string,
          turn: (p.turn as number) || prev.length + 1,
          phase: typeof p.phase === 'number' ? p.phase : 0,
          ts: event.ts || Date.now(),
          meta: { token_input: p.token_input, token_output: p.token_output },
        }])
      }
    }
    if (event.type === 'micro.progress') {
      const p = event.payload as Record<string, unknown> | undefined
      if (p && p.step === 'context_pack') {
        const packSummary = p.context_pack as Record<string, unknown> | undefined
        if (packSummary) {
          setLiveContextPacks((prev) => [...prev.slice(-20), {
            sid: event.session_id || '',
            ts: event.ts || Date.now(),
            turn: (p.turn as number) || 0,
            phase: typeof p.phase === 'number' ? p.phase : 0,
            data: {
              summary: packSummary as unknown as ContextPackItem['data']['summary'],
            },
          }])
        }
      }
    }
  }, qid, actionState.isRunning)

  async function act(fn: () => Promise<unknown>) {
    if (busy) return
    setBusy(true)
    try {
      const result = await fn()
      const maybeQuest = (result as { quest?: QuestMeta } | undefined)?.quest
      if (maybeQuest) {
        setQuest(maybeQuest)
      }
      await load()
      await loadBackups()
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.operation'))
    } finally {
      setBusy(false)
    }
  }

  function openReview(verdict: Verdict, apply?: boolean) {
    // 预填评审意见（从操作栏复制）
    setReviewComment(comment)
    setReviewError('')
    setReviewAction(reviewActionDraft(verdict, hasWorkspaceDiffPending, apply))
  }

  function closeReview() {
    setReviewAction(null)
    setReviewComment('')
    setReviewError('')
  }

  async function confirmReview() {
    if (!reviewAction || busy) return
    setBusy(true)
    try {
      const payload: ResolveQuestPayload = { verdict: reviewAction.verdict, comment: reviewComment.trim() }
      if (reviewAction.verdict === 'pass') {
        payload.apply = reviewAction.apply !== false
      }
      const res = await resolveQuest(qid, payload)
      if (res.quest) {
        setQuest(res.quest)
      }
      closeReview()
      await load()
      await loadBackups()
      onChanged()
    } catch (e) {
      const warnings = warningsFromError(e)
      if (warnings.length > 0) {
        closeReview()
        setApplyWarning({
          message: e instanceof Error ? e.message : i18n.t('quest.error.applyBlocked'),
          warnings,
          fromReview: true,
        })
        await load()
        await loadBackups()
        onChanged()
      } else {
        const message = e instanceof Error ? e.message : i18n.t('quest.error.operation')
        setReviewError(message)
        await load()
        onChanged()
        onError(message)
      }
    } finally {
      setBusy(false)
    }
  }

  async function applyQuest(force = false) {
    if (busy) return
    setBusy(true)
    try {
      await applyQuestChanges(qid, force)
      setApplyWarning(null)
      await load()
      await loadBackups()
      onChanged()
    } catch (e) {
      const warnings = warningsFromError(e)
      if (warnings.length > 0) {
        setApplyWarning({
          message: e instanceof Error ? e.message : i18n.t('quest.error.applyBlocked'),
          warnings,
        })
      } else {
        await load()
        onChanged()
        onError(e instanceof Error ? e.message : i18n.t('quest.error.applyFailed'))
      }
    } finally {
      setBusy(false)
    }
  }

  async function discardQuest() {
    await act(() => discardQuestChanges(qid))
  }

  async function sendComment() {
    if (!comment.trim()) return
    setSendingComment(true)
    try {
      await commentOnQuest(qid, comment.trim())
      setComment('')
      onChanged()
      await load()
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.sendComment'))
    } finally {
      setSendingComment(false)
    }
  }

  async function sendWaitingInputAnswer() {
    if (sendingAnswer || !answerText.trim() || !isWaitingInput) return
    setSendingAnswer(true)
    try {
      await answerQuest(qid, {
        question_id: waitingInput?.question_id,
        answer: answerText.trim(),
      })
      setAnswerText('')
      onChanged()
      await load()
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.sendAnswer'))
    } finally {
      setSendingAnswer(false)
    }
  }

  async function loadDiff() {
    if (loadingDiff) return
    setLoadingDiff(true)
    try {
      const res = await getQuestDiff(qid)
      setDiff(res.diff?.diff || res.diff?.stat || i18n.t('quest.diff.none'))
      setDiffSource(res.diff?.source || null)
      setDiffBackupID(res.diff?.backup_id || null)
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.loadDiff'))
    } finally {
      setLoadingDiff(false)
    }
  }

  async function recoverAgent(action: string) {
    await act(() => recoverQuestAgent(qid, {
      action,
      add_turns: blockedAddTurns,
      add_duration_minutes: blockedAddMinutes,
    }))
  }

  function openBlockedAction(action: BlockedActionState['action']) {
    if (action === BLOCKED_ACTION.continue) {
      setBlockedAction({
        action,
        label: i18n.t('quest.blockedAction.continue'),
        tone: 'primary',
        description: i18n.t('quest.blockedAction.continueDesc'),
      })
      return
    }
    if (action === BLOCKED_ACTION.userReview) {
      setBlockedAction({
        action,
        label: i18n.t('quest.blockedAction.userReview'),
        tone: 'default',
        description: i18n.t('quest.blockedAction.userReviewDesc'),
      })
      return
    }
    setBlockedAction({
      action,
      label: i18n.t('quest.blockedAction.cancel'),
      tone: 'danger',
      description: i18n.t('quest.blockedAction.cancelDesc'),
    })
  }

  async function confirmBlockedAction() {
    if (!blockedAction) return
    const action = blockedAction.action
    setBlockedAction(null)
    await act(() => resolveBlockedQuest(qid, {
      action,
      add_turns: blockedAddTurns,
      add_duration_minutes: blockedAddMinutes,
    }))
  }

  async function loadEvidence() {
    if (loadingEvidence || !quest) return
    setLoadingEvidence(true)
    try {
      const res = await getQuestTrace(qid)
      const items = res.items || []
      setEvents((prev) => prev.length > 0 ? prev : historicEventsFromTraceItems(items))
      setEvidence(evidenceFromTraceItems(items))
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.loadTrace'))
    } finally {
      setLoadingEvidence(false)
    }
  }

  async function showBackupDetail(backupID: string) {
    if (!backupID || backupDetail.loading) return
    setBackupDetail({ loading: true, item: null })
    try {
      const res = await getQuestBackup(qid, backupID)
      setBackupDetail({ loading: false, item: res.item })
    } catch (e) {
      setBackupDetail({ loading: false, item: null })
      onError(e instanceof Error ? e.message : i18n.t('quest.error.loadBackupDetail'))
    }
  }

  async function spawnExecuteFromDesign() {
    if (busy) return
    setBusy(true)
    try {
      const res = await spawnExecuteFromDesignQuest(qid)
      const childID = res.quest?.id || res.qid
      if (childID) {
        await onChanged()
        onOpenQuest(childID)
      } else {
        await load()
        onChanged()
      }
    } catch (e) {
      onError(e instanceof Error ? e.message : i18n.t('quest.error.spawnExecute'))
    } finally {
      setBusy(false)
    }
  }

  // v0.5 slice 3: 手动升级 workflow mode（direct → checked → goal，后端强制单向）。
  // 升级是 system reply：后端 UpgradeWorkflowMode 写 system_workflow_upgrade post（quest_lifecycle.go）。
  // act() 成功后会 load()（含 setThreadPosts），system_workflow_upgrade reply 立即出现在 thread ledger。
  // target 只接受 checked/goal；getWorkflowUpgradeInfo 的 canUpgrade 保证调用时 nextMode 合法。
  async function upgradeWorkflowMode(target: string) {
    if (target !== 'checked' && target !== 'goal') return
    await act(() => upgradeQuestWorkflowMode(qid, { workflow_mode: target }))
  }

  const {
    canReview,
    applyFailed,
    hasWorkspaceDiffPending,
    canApply,
    canDiscard,
    canSpawn,
    canStart,
    canStop,
    isQueued,
    isStarting,
    isBlocked,
    isEnded,
  } = actionState
  const isWaitingInput = quest?.status === 'waiting_input'
  const waitingInput = quest?.waiting_input || null
  const waitingInputQuestion = waitingInput?.question_text || i18n.t('quest.waitingInput.defaultQuestion')

  // v0.5 slice 3: workflow mode 手动升档可见性。
  // 后端强制单向（direct → checked → goal），前端只控制按钮显隐：
  // - blocked / 终态 / goal / 未知 mode 不显示
  // - direct → 显示"升级到 Checked"
  // - checked → 显示"升级到 Goal"
  const workflowInfo = getWorkflowUpgradeInfo(
    quest?.workflow_mode,
    isBlocked,
    isEnded,
    isWaitingInput,
    actionState.isRunning,
  )
  const canUpgradeWorkflow = workflowInfo.canUpgrade
  const nextWorkflowMode = workflowInfo.nextMode
  const nextWorkflowLabel = workflowInfo.nextLabel

  // Goal runtime overview: decision_note 只作为 thread post 投影存在，非独立实体。
  const goalRuntime = deriveGoalRuntimeView(designDoc, quest, threadPosts, canSpawn)
  const showGoalRuntime = goalRuntime.showPanel

  const lastCommentInjectedTs = useMemo(() => {
    if (!evidence?.traces) return 0
    let ts = 0
    for (const trace of evidence.traces) {
      for (const row of trace.rows) {
        if (row.kind === 'comment' && row.ts > ts) ts = row.ts
      }
    }
    return ts
  }, [evidence])

  const nativeEvidence: NativeToolItem[] = useMemo(
    () => evidence?.traces.flatMap((trace) => nativeEvidenceFromTrace(trace)) ?? [],
    [evidence],
  )
  const traceMessages: MessageItem[] = useMemo(
    () => evidence?.traces.flatMap((trace) => messagesFromTrace(trace)) ?? [],
    [evidence],
  )
  const traceComments: CommentItem[] = useMemo(
    () => evidence?.traces.flatMap((trace) => commentsFromTrace(trace)) ?? [],
    [evidence],
  )
  const traceContextPacks: ContextPackItem[] = useMemo(
    () => evidence?.traces.flatMap((trace) => contextPacksFromTrace(trace)) ?? [],
    [evidence],
  )
  const traceToolResults: ToolResultItem[] = useMemo(
    () => evidence?.traces.flatMap((trace) => toolResultsFromTrace(trace)) ?? [],
    [evidence],
  )
  const allMessages: MessageItem[] = useMemo(
    () => mergeByTimestamp(traceMessages, liveMessages),
    [traceMessages, liveMessages],
  )
  const phaseOutputs = useMemo(() => {
    const out = new Map<number, string>()
    for (const msg of allMessages) {
      const text = String(msg.content || '').trim()
      if (!text) continue
      out.set(msg.phase ?? 0, text)
    }
    return out
  }, [allMessages])

  const outputs = quest?.outputs ?? []
  const summaryMatch = useMemo(() => pickSummaryArtifact(outputs), [outputs])
  const [summaryArtifactText, setSummaryArtifactText] = useState<string | null>(null)
  const [summaryArtifactLoading, setSummaryArtifactLoading] = useState(false)

  useEffect(() => {
    if (!quest || !summaryMatch) {
      setSummaryArtifactText(null)
      return
    }
    let cancelled = false
    setSummaryArtifactLoading(true)
    fetch(artifactURL(quest.id, summaryMatch.artifact))
      .then(async (r) => {
        if (!r.ok) throw new Error('HTTP ' + r.status)
        const blob = await r.blob()
        const text = await blob.text()
        if (!cancelled) setSummaryArtifactText(text)
      })
      .catch(() => { if (!cancelled) setSummaryArtifactText(null) })
      .finally(() => { if (!cancelled) setSummaryArtifactLoading(false) })
    return () => { cancelled = true }
  }, [quest, summaryMatch])

  const adventurerConclusion = useMemo(() => selectQuestProducerArtifact({
    quest,
    messages: allMessages,
    contextPacks: traceContextPacks,
    toolResults: traceToolResults,
    summaryArtifact: summaryMatch,
    summaryArtifactText,
  }), [allMessages, quest, summaryMatch, summaryArtifactText, traceContextPacks, traceToolResults])

  // Mage Review 纯函数层：从 reviews[] 挑出法师最新评审，查表得到 tone/state。
  const mageMatch = useMemo(
    () => quest ? selectLatestMageReview(reviews, { mage_id: quest.mage_id }) : null,
    [quest, reviews],
  )
  const latestMakerReport = useMemo(() => {
    const items = reports?.maker_reports || []
    return items.reduce<typeof items[number] | null>((latest, item) =>
      !latest || (item.created_at_ms || 0) > (latest.created_at_ms || 0) ? item : latest, null)
  }, [reports])
  const latestReviewReport = useMemo(() => {
    const items = reports?.review_reports || []
    return items.reduce<typeof items[number] | null>((latest, item) =>
      !latest || (item.created_at_ms || 0) > (latest.created_at_ms || 0) ? item : latest, null)
  }, [reports])
  const stuckDiagnosis = useMemo(() => getStuckDiagnosis(quest), [quest])
  const scrollToTrace = useCallback(() => {
    const el = document.querySelector<HTMLElement>('.exec-trace')
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, [])

  type MessageArtifactLink = { artifactId: string; artifactName: string }
  const [textArtifactContents, setTextArtifactContents] = useState<Record<string, { loading: boolean; text: string | null }>>({})

  const textualOutputs = useMemo(
    () => outputs.filter((a) => isTextualArtifact(a) && String(a.name || '').toLowerCase() !== 'design-plan.md' && a.id !== summaryMatch?.artifact.id),
    [outputs, summaryMatch],
  )

  useEffect(() => {
    if (!quest || textualOutputs.length === 0) return
    let cancelled = false
    setTextArtifactContents((prev) => {
      const next = { ...prev }
      for (const a of textualOutputs) {
        if (!next[a.id]) next[a.id] = { loading: true, text: null }
      }
      return next
    })
    Promise.all(textualOutputs.map(async (a) => {
      try {
        const r = await fetch(artifactURL(quest.id, a))
        if (!r.ok) return [a.id, null] as const
        const text = await (await r.blob()).text()
        return [a.id, text] as const
      } catch {
        return [a.id, null] as const
      }
    })).then((results) => {
      if (cancelled) return
      setTextArtifactContents((prev) => {
        const next = { ...prev }
        for (const [id, text] of results) {
          next[id] = { loading: false, text }
        }
        return next
      })
    })
    return () => { cancelled = true }
  }, [quest, textualOutputs])

  const messageArtifactLinks = useMemo<Record<string, MessageArtifactLink>>(() => {
    const byKey = new Map<string, QuestArtifact>()
    const allTextual = (summaryMatch ? [summaryMatch.artifact] : []).concat(textualOutputs)
    for (const a of allTextual) {
      const loaded = a.id === summaryMatch?.artifact.id
        ? summaryArtifactText
        : textArtifactContents[a.id]?.text ?? null
      if (!loaded) continue
      const norm = normalizeTextForDedupe(loaded)
      if (!norm) continue
      const key = String(norm.length) + ':' + String(fastHash(norm))
      if (!byKey.has(key)) byKey.set(key, a)
    }
    const out: Record<string, MessageArtifactLink> = {}
    for (const m of allMessages) {
      const norm = normalizeTextForDedupe(m.content)
      if (!norm) continue
      const len = norm.length
      if (len < 200) continue
      const key = String(len) + ':' + String(fastHash(norm))
      const match = byKey.get(key)
      if (match) {
        out[m.sid + ':' + m.phase + ':' + m.turn + ':' + String(m.ts)] = { artifactId: match.id, artifactName: match.name }
        continue
      }
      // Loose similarity fallback: same length class + contains first 80 chars
      for (const a of allTextual) {
        const loaded = a.id === summaryMatch?.artifact.id
          ? summaryArtifactText
          : textArtifactContents[a.id]?.text ?? null
        if (!loaded) continue
        const aNorm = normalizeTextForDedupe(loaded)
        const aLen = aNorm.length
        if (aLen < 200) continue
        if (Math.abs(aLen - len) / Math.max(aLen, len) > 0.05) continue
        const head = norm.slice(0, 160)
        if (head && aNorm.includes(head)) {
          out[m.sid + ':' + m.phase + ':' + m.turn + ':' + String(m.ts)] = { artifactId: a.id, artifactName: a.name }
          break
        }
      }
    }
    return out
  }, [allMessages, textualOutputs, summaryMatch, summaryArtifactText, textArtifactContents])

  const flashTokenRef = useRef(0)

  function scrollToArtifact(artifactId: string) {
    const el = document.querySelector<HTMLElement>('[data-artifact-id="' + artifactId + '"]')
    if (!el) return
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    const token = ++flashTokenRef.current
    el.classList.add('quest-output-item--flash')
    window.setTimeout(() => {
      if (flashTokenRef.current === token) el.classList.remove('quest-output-item--flash')
    }, 1800)
  }

  function scrollToQuestSection(id: string) {
    const el = document.getElementById(id)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  function emitDetailDiagnosisPath(resolvedSurface: 'trace' | 'evidence' | 'action') {
    if (!quest || !stuckDiagnosis) return
    emitStuckDiagnosisPath({
      entry_surface: 'quest_detail',
      quest_id: quest.id,
      exception_kind: stuckDiagnosis.kind,
      click_count: 1,
      resolved_surface: resolvedSurface,
      has_recommended_action: stuckDiagnosis.hasRecommendedAction,
      has_evidence_jump: stuckDiagnosis.hasEvidenceJump,
    })
  }

  function jumpToDiagnosisTrace() {
    emitDetailDiagnosisPath('trace')
    scrollToQuestSection('quest-execution-trace')
  }

  function jumpToDiagnosisEvidence() {
    emitDetailDiagnosisPath('evidence')
    if (latestReviewReport?.evidence_refs?.length) {
      scrollToQuestSection('quest-review-report')
      return
    }
    scrollToQuestSection('quest-execution-trace')
  }

  function scrollToMessage(sid?: string, turn?: number) {
    let selector = ''
    if (sid && typeof turn === 'number') {
      selector = `.exec-trace-row.exec-message[data-sid="${sid}"][data-turn="${turn}"]`
    } else if (sid) {
      selector = `.exec-trace-row.exec-message[data-sid="${sid}"]`
    }
    if (selector) {
      const el = document.querySelector<HTMLElement>(selector)
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
        const token = ++flashTokenRef.current
        el.classList.add('exec-trace-row--flash')
        window.setTimeout(() => {
          if (flashTokenRef.current === token) el.classList.remove('exec-trace-row--flash')
        }, 1800)
        return true
      }
    }
    return false
  }

  function handleJumpToEvidence(ref: EvidenceRef, srCtx?: StructuredReview) {
    if (ref.kind === 'artifact' && ref.id) {
      scrollToArtifact(ref.id)
      return
    }
    if (ref.kind === 'warrior_message') {
      const sid = srCtx?.reviewed_warrior_session_id
      const turn = srCtx?.reviewed_warrior_latest_turn
      if (scrollToMessage(sid, turn)) return
    }
    scrollToQuestSection('quest-execution-trace')
  }

  const parsedDiffFiles = useMemo(() => (diff ? parseDiffFiles(diff) : []), [diff])

  const knowledgeArtifacts = useMemo(() => {
    type Candidate = {
      artifact: QuestArtifact
      meta: KnowledgeExportMeta | null
      diff: KnowledgeExportDiff | null
    }
    const outs = quest?.outputs ?? []
    const picked: Candidate[] = []
    for (const a of outs) {
      const n = String(a.name || '').toLowerCase()
      const mime = String(a.mime || '').toLowerCase()
      const source = String(a.source || '').toLowerCase()
      const looks = n.includes('knowledge') || n.includes('context') || mime.includes('knowledge') || source.includes('knowledge')
      if (!looks && !(n.endsWith('.json') && (n.includes('export') || n.includes('package')))) continue
      const anyA = a as { knowledge_export_meta?: unknown; meta?: unknown; knowledge_export_diff?: unknown }
      const rawMeta = anyA.knowledge_export_meta ?? anyA.meta
      const meta = rawMeta && typeof rawMeta === 'object' ? (rawMeta as KnowledgeExportMeta) : null
      const rawDiff = anyA.knowledge_export_diff ?? null
      const diffObj = rawDiff && typeof rawDiff === 'object' ? (rawDiff as KnowledgeExportDiff) : null
      picked.push({ artifact: a, meta, diff: diffObj })
    }
    return picked
  }, [quest])

  const diffPanelBadge = useMemo(() => {
    const effectType: QuestMeta['effect_type'] | undefined = quest?.effect_type
    const isWorkspace = effectType === 'workspace_diff' || (!effectType && ((quest?.workspace_diff_pending) || !!quest?.diff_changed_files))
    const isReadonly = quest?.workspace_mode === 'readonly'
    const isContextStore = effectType === 'context_store'
    const isExternal = effectType === 'external_side_effect'
    return { isWorkspace, isReadonly, isContextStore, isExternal }
  }, [quest])


  const phaseEntries: TimelineEntry[] = useMemo(
    () => questTimelineEntries(quest, events, { lastCommentInjectedTs, phaseOutputs }),
    [quest, events, lastCommentInjectedTs, phaseOutputs],
  )
  const allContextPacks: ContextPackItem[] = useMemo(
    () => mergeByTimestamp(traceContextPacks, liveContextPacks),
    [traceContextPacks, liveContextPacks],
  )
  const copyQuestQuery = useCallback(async () => {
    if (!quest) return
    if (await writeClipboardText(quest.query)) {
      setQueryCopied(true)
      window.setTimeout(() => setQueryCopied(false), 1400)
      return
    }
    onError(i18n.t('quest.error.copyQuery'))
  }, [onError, quest])

  const statPairs: { label: string; value: React.ReactNode; className?: string }[] = quest ? [
    { label: 'ID', value: <span className="mono">#{quest.short_id || shortId(quest.id)}</span> },
    { label: i18n.t('quest.metaLabel.created'), value: fmtTime(quest.created_at_ms) },
    {
      label: i18n.t('quest.metaLabel.baseDir'),
      value: <PathValue value={quest.base_working_dir} fallback={i18n.t('quest.metaLabel.defaultWorkDir')} />,
      className: 'path-kv',
    },
    {
      label: i18n.t('quest.metaLabel.workspace'),
      value: <PathValue value={quest.workspace_path} fallback={quest.status === 'pending' ? i18n.t('quest.metaLabel.createdAfterStart') : '-'} />,
      className: 'path-kv',
    },
    { label: i18n.t('quest.metaLabel.workspaceMode'), value: quest.workspace_mode || 'auto' },
    { label: i18n.t('quest.metaLabel.type'), value: typeLabel(quest.type) || quest.type },
    {
      label: i18n.t('quest.metaLabel.rework'),
      value: (
        <>
          <span className="mono">{quest.rework_count}</span> / <span className="mono">{quest.max_rework}</span>
        </>
      ),
    },
    { label: i18n.t('quest.metaLabel.score'), value: quest.mage_score != null ? <span className="mono">{quest.mage_score} / 10</span> : '-' },
    { label: i18n.t('quest.metaLabel.changes'), value: <span className="mono">{quest.diff_stat || '-'}</span> },
    {
      label: i18n.t('quest.metaLabel.connectors'),
      value:
        quest.connectors && quest.connectors.length > 0 && quest.workspace_mode !== 'readonly' ? (
          <span className="mono">
            <Icon name="git-merge" size={12} />{' '}
            {quest.connectors.map((c) => (c === 'git' ? i18n.t('quest.metaLabel.connectorGit') : c)).join('、')}
          </span>
        ) : (
          i18n.t('quest.metaLabel.none')
        ),
    },
  ] : []

  if (!quest) return <div className="empty-page">{i18n.t('quest.detail.loading')}</div>

  return (
    <div className="detail-page">
      <header className="detail-head">
        <button className="icon-button" onClick={onBack} aria-label={i18n.t('aria.back')}>
          <Icon name="back" />
        </button>
        <span className="mono detail-id">#{quest.short_id || shortId(quest.id)}</span>
        <span className="type-pill">{typeLabel(quest.type)}</span>
        {workflowInfo.pillLabel && (
          <span className={workflowInfo.pillClass || ''}>{workflowInfo.pillLabel}</span>
        )}
        {quest.intensity === 'quick' && (
          <span className="intensity-badge quick">{i18n.t('quest.detail.intensityQuick')}</span>
        )}
        {quest.intensity === 'deep' && (
          <span className="intensity-badge deep">{i18n.t('quest.detail.intensityDeep')}</span>
        )}
        {quest.intensity === 'adversarial' && (
          <span className="intensity-badge adversarial">{i18n.t('quest.detail.intensityAdversarial')}</span>
        )}
        <StatusBadge status={statusBadge.status} muted={statusBadge.muted} title={statusBadge.title} />
        <span className="detail-head-spacer" />
      </header>

      <div className="detail-layout">
        <div className="detail-main-wrap">
          <button
            className={'mobile-info-toggle' + (mobileInfoOpen ? ' open' : '')}
            onClick={() => setMobileInfoOpen(!mobileInfoOpen)}
            aria-expanded={mobileInfoOpen}
          >
            <span>{i18n.t('quest.detail.mobileToggle')}</span>
            <Icon name="chevron-down" size={14} />
          </button>

          {mobileInfoOpen && (
            <div className="mobile-detail-info">
              <div className="info-section">
                <div className="info-section-label">{i18n.t('quest.section.metadata')}</div>
                <div className="info-kv-grid">
                  {statPairs.map((kv) => (
                    <div className={'info-kv ' + (kv.className || '')} key={kv.label}>
                      <span>{kv.label}</span>
                      <div>{kv.value}</div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="info-section">
                <div className="info-section-label">{i18n.t('quest.section.adventurers')}</div>
                <div className="info-kv-grid">
                  <div className="info-kv">
                    <span>{i18n.t('world.warrior')}</span>
                    <div className="row">
                      <Logo variant="knight" size={18} />
                      <span className="faint mono">{quest.warrior_id || '-'}</span>
                    </div>
                  </div>
                  <div className="info-kv">
                    <span>{i18n.t('world.mage')}</span>
                    <div className="row">
                      <Logo variant="mage" size={18} />
                      <span className="faint mono">
                        {quest.intensity === 'quick' ? i18n.t('quest.sidePanel.quickModeNoReview') : quest.mage_id || '-'}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="info-section">
                <div className="info-section-label">{i18n.t('quest.section.budget')}</div>
                <div className="info-kv-grid">
                  <div className="info-kv">
                    <span>{i18n.t('quest.metaLabel.rework')}</span>
                    <div className="side-progress">
                      <div className="side-progress-row">
                        <span className="mono">{quest.rework_count ?? 0}</span>
                        <span className="mono">/ {quest.max_rework ?? '∞'}</span>
                      </div>
                      <div className="side-progress-bar">
                        <i
                          style={{
                            width:
                              Math.min(
                                100,
                                quest.max_rework ? ((quest.rework_count ?? 0) / quest.max_rework) * 100 : 25,
                              ) + '%',
                          }}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {isBlocked && quest.blocked_reason && (
                <div className="alert-box">{fmtBlockedReason(quest)}</div>
              )}
              {isWaitingInput && (
                <div className="alert-box waiting-input-box">
                  <strong>{i18n.t('quest.waitingInput.asking')}</strong>
                  <span>{waitingInputQuestion}</span>
                </div>
              )}
            </div>
          )}

          <CurrentWorkStatus
            quest={quest}
            statusBadge={statusBadge}
            phaseView={phaseView}
            actionState={actionState}
            streamState={streamState}
            lastEventAtMs={lastEventAtMs}
            events={events}
            onJumpToTrace={stuckDiagnosis ? jumpToDiagnosisTrace : () => scrollToQuestSection('quest-execution-trace')}
          />

          {stuckDiagnosis && (
            <StuckActionPanel
              diagnosis={stuckDiagnosis}
              onJumpToTrace={jumpToDiagnosisTrace}
              onJumpToEvidence={jumpToDiagnosisEvidence}
            />
          )}

          <QuestSummary
            query={quest.query}
            variant="detail"
            expanded={queryExpanded}
            copied={queryCopied}
            onCopy={copyQuestQuery}
            onToggleExpanded={() => setQueryExpanded((open) => !open)}
          />


          <MakerReportCard
            report={latestMakerReport}
            fallback={adventurerConclusion}
            loadingSummaryArtifact={!!summaryMatch && summaryArtifactLoading}
            onJumpToArtifact={scrollToArtifact}
            onJumpToTrace={() => scrollToQuestSection('quest-execution-trace')}
          />

          {latestReviewReport && (
            <ReviewReportCard
              report={latestReviewReport}
              onJumpToEvidence={handleJumpToEvidence}
              onJumpToTrace={() => scrollToQuestSection('quest-execution-trace')}
            />
          )}

          {quest && quest.impact_summary && (
            <ImpactSummaryCard impact={quest.impact_summary} className="mt-3" />
          )}

          {quest && shouldShowFinalVerdict(quest) && (
            <details className="panel mt-3 collapsible-section">
              <summary className="collapsible-header">
                <span>{i18n.t('quest.section.verdict')}</span>
                <span className="mono">{quest.final_verdict}</span>
              </summary>
              <FinalVerdictCard
                verdict={quest.final_verdict}
                comment={quest.final_comment}
                finalized_by={quest.finalized_by}
                auto_passed_by_policy={quest.auto_passed_by_policy}
                auto_completed_by_policy={quest.auto_completed_by_policy}
              />
            </details>
          )}

          <div id="quest-thread-ledger">
            <ThreadLedgerPanel
              posts={threadPosts}
              pinnedOutcome={quest.pinned_outcome_summary}
            />
          </div>

          <FanoutTreePanel
            group={fanoutGroup}
            onOpenQuest={onOpenQuest}
          />

          <EvidenceList
            reportEvidence={latestReviewReport?.evidence_refs || []}
            nativeEvidence={nativeEvidence}
            onJumpToTrace={() => scrollToQuestSection('quest-execution-trace')}
          />

          {loopStateSpine && <LoopStateCard spine={loopStateSpine} />}

          {quest && (
            <div id="quest-mage-review" className="mt-3">
              <MageReviewBand
                match={mageMatch}
                quest={{ mage_id: quest.mage_id, intensity: quest.intensity, phases: quest.phases }}
                onJumpToTrace={() => scrollToQuestSection('quest-execution-trace')}
                onJumpToEvidence={handleJumpToEvidence}
              />
            </div>
          )}

          {designPlan && (
            <DesignPlanCard
              artifact={designPlan}
              href={designPlan.storage_path?.startsWith('http') ? designPlan.storage_path : artifactURL(quest.id, designPlan)}
              content={designPlanText}
              loading={loadingDesignPlan}
            />
          )}

          {quest.inputs && quest.inputs.length > 0 && (
            <section className="panel quest-inputs-panel mt-3">
              <div className="panel-title">
                <h2>{i18n.t('quest.section.inputs')}</h2>
                <span>{i18n.t('quest.count.files', { count: quest.inputs.length })}</span>
              </div>
              <div className="quest-input-grid">
                {quest.inputs.map((artifact) => {
                  const url = artifactURL(quest.id, artifact)
                  return (
                    <a className="quest-input-card" href={url} target="_blank" rel="noreferrer" key={artifact.id}>
                      {artifact.kind === 'image' ? (
                        <img src={url} alt={artifact.name} />
                      ) : (
                        <span className="quest-input-file-icon">
                          <Icon name={artifact.kind === 'archive' ? 'package' : 'file-text'} />
                        </span>
                      )}
                      <span className="quest-input-name" title={artifact.name}>{artifact.name}</span>
                      <span className="quest-input-meta">{artifact.kind} · {formatArtifactSize(artifact.size)}</span>
                    </a>
                  )
                })}
              </div>
            </section>
          )}

          {quest.outputs && quest.outputs.length > 0 && (
            <section id="quest-artifacts" className="panel quest-outputs-panel mt-3">
              <div className="panel-title">
                <h2>{i18n.t('quest.section.outputs')}</h2>
                <span>{i18n.t('quest.count.items', { count: quest.outputs.length })}</span>
              </div>
              <div className="quest-output-list">
                {quest.outputs.map((artifact) => {
                  const isExternal = artifact.storage_path?.startsWith('http://') || artifact.storage_path?.startsWith('https://')
                  const url = isExternal ? artifact.storage_path : artifactURL(quest.id, artifact)
                  const textual = !isExternal && isTextualArtifact(artifact)
                  const isSummary = summaryMatch?.artifact.id === artifact.id
                  const loaded = textual
                    ? (isSummary ? { loading: summaryArtifactLoading, text: summaryArtifactText } : (textArtifactContents[artifact.id] ?? { loading: true, text: null }))
                    : null
                  return (
                    <div key={artifact.id} className="quest-output-item" data-artifact-id={artifact.id}>
                      <a className="quest-output-card" href={url} target="_blank" rel="noreferrer">
                        {artifact.kind === 'image' && !isExternal ? (
                          <img src={url} alt={artifact.name} />
                        ) : (
                          <span className="quest-input-file-icon">
                            <Icon name={outputIconName(artifact, isExternal)} />
                          </span>
                        )}
                        <span className="quest-output-card-body">
                          <span className="quest-input-name" title={artifact.name}>{artifact.name}</span>
                          <span className="quest-input-meta">
                            {artifact.kind || 'output'}
                            {!isExternal && ` · ${formatArtifactSize(artifact.size)}`}
                            {isExternal && ` · ${i18n.t('quest.output.external')}`}
                            {isSummary && ` · ${i18n.t('quest.output.summarySource')}`}
                          </span>
                        </span>
                        <Icon name={isExternal ? 'external-link' : 'arrow-up-right'} size={14} className="quest-output-card-arrow" />
                      </a>
                      {textual && loaded && (
                        <details className="quest-output-preview" open={isSummary}>
                          <summary>
                            <Icon name="file-text" size={12} /> {i18n.t('quest.output.previewTitle', { state: loaded.loading ? i18n.t('quest.output.previewLoading') : loaded.text ? i18n.t('quest.count.chars', { count: loaded.text.length }) : i18n.t('quest.output.previewNoContent') })}
                          </summary>
                          {loaded.loading ? (
                            <div className="exec-trace-empty">
                              <Icon name="spinner" className="spin" />
                              <span>{i18n.t('quest.output.previewLoadingText')}</span>
                            </div>
                          ) : loaded.text ? (
                            <MarkdownRenderer source={loaded.text} className="compact" stripFrontmatter emptyLabel={i18n.t('quest.output.emptyText')} />
                          ) : (
                            <div className="quest-output-preview-empty">{i18n.t('quest.output.previewUnreadable')}</div>
                          )}
                        </details>
                      )}
                    </div>
                  )
                })}
              </div>
            </section>
          )}

          <ChangesSection
            quest={quest}
            loading={loadingDiff}
            diff={diff}
            diffSource={diffSource}
            diffBackupID={diffBackupID}
            onJumpToArtifacts={() => scrollToQuestSection('quest-artifacts')}
            onJumpToTrace={() => scrollToQuestSection('quest-execution-trace')}
            onJumpToConclusion={() => scrollToQuestSection('quest-maker-delivery')}
            onJumpToArtifact={scrollToArtifact}
            onCopy={writeClipboardText}
            parsedFiles={parsedDiffFiles}
            knowledgeArtifacts={knowledgeArtifacts}
          />

          {designDoc && (
            <section className="panel design-doc-panel mt-3">
              <div className="panel-title">
                <h2>{i18n.t('quest.section.designDoc')}</h2>
                <span className="mono">{designDoc.schema_version}</span>
              </div>
              <MarkdownRenderer
                source={designDoc.summary || quest.design_summary || i18n.t('quest.designDoc.noSummary')}
                className="review-copy compact"
              />
              <div className="design-doc-grid">
                {[
                  [i18n.t('quest.designDoc.goals'), designDoc.goals],
                  [i18n.t('quest.designDoc.nonGoals'), designDoc.non_goals],
                  [i18n.t('quest.designDoc.steps'), designDoc.implementation_steps],
                  [i18n.t('quest.designDoc.risks'), designDoc.risks],
                  [i18n.t('quest.designDoc.acceptance'), designDoc.acceptance_criteria],
                ].map(([label, items]) => (
                  Array.isArray(items) && items.length > 0 ? (
                    <div className="design-doc-section" key={label as string}>
                      <strong>{label as string}</strong>
                      <ul>
                        {items.map((item, index) => <li key={index}>{item}</li>)}
                      </ul>
                    </div>
                  ) : null
                ))}
              </div>
              {designDoc.execute_prompt && (
                <details className="design-execute-prompt">
                  <summary>{i18n.t('quest.designDoc.executePrompt')}</summary>
                  <AgentMessageRenderer content={designDoc.execute_prompt} compact />
                </details>
              )}
            </section>
          )}

          {showGoalRuntime && (
            <section className="panel goal-runtime-panel mt-3">
              <div className="panel-title">
                <h2>{i18n.t('quest.section.goalRuntime')}</h2>
                {quest.auto_spawn_execute && <span className="chip tiny">{i18n.t('quest.goalRuntime.autoExecute')}</span>}
              </div>
              <div className="goal-runtime-status">
                <div className="goal-status-item">
                  <span className="goal-status-label">{i18n.t('quest.goalRuntime.designDoc')}</span>
                  <span className="goal-status-value">
                    {designDoc
                      ? <><Icon name="circle-check" size={12} /> {i18n.t('quest.goalRuntime.ready')}</>
                      : <><Icon name="circle" size={12} /> {i18n.t('quest.goalRuntime.pending')}</>}
                  </span>
                </div>
                <div className="goal-status-item">
                  <span className="goal-status-label">{i18n.t('quest.goalRuntime.decisionRecord')}</span>
                  <span className="goal-status-value">
                    {goalRuntime.decisionRecorded
                      ? <><Icon name="circle-check" size={12} /> {i18n.t('quest.goalRuntime.recorded')}</>
                      : <><Icon name="circle" size={12} /> {i18n.t('quest.goalRuntime.none')}</>}
                  </span>
                </div>
                <div className="goal-status-item">
                  <span className="goal-status-label">{i18n.t('quest.goalRuntime.executeQuest')}</span>
                  <span className="goal-status-value">
                    {goalRuntime.executeStatus}
                  </span>
                </div>
              </div>
              {goalRuntime.steps.length > 0 && (
                <div className="goal-runtime-steps">
                  <strong>{i18n.t('quest.goalRuntime.steps')}</strong>
                  <ol>
                    {goalRuntime.steps.map((step, i) => (
                      <li key={i}>{step}</li>
                    ))}
                  </ol>
                </div>
              )}
              <div className="goal-runtime-actions">
                {goalRuntime.showSpawnButton && (
                  <button className="button primary" disabled={busy} onClick={spawnExecuteFromDesign}>
                    <Icon name="sword" />
                    {i18n.t('quest.goalRuntime.spawnButton')}
                  </button>
                )}
                {goalRuntime.showOpenChildButton && goalRuntime.childExecuteQuestID && (
                  <button className="button" type="button" onClick={() => onOpenQuest(goalRuntime.childExecuteQuestID!)}>
                    <Icon name="sword" />
                    {i18n.t('quest.goalRuntime.openChild')} <span className="mono">#{shortId(goalRuntime.childExecuteQuestID)}</span>
                  </button>
                )}
                {goalRuntime.showSpawnError && goalRuntime.spawnError && (
                  <div className="alert-box">{i18n.t('quest.goalRuntime.spawnError', { error: goalRuntime.spawnError })}</div>
                )}
                {goalRuntime.showAutoSpawnPlaceholder && (
                  <div className="exec-trace-empty">{i18n.t('quest.goalRuntime.autoSpawnHint')}</div>
                )}
              </div>
              <div className="goal-runtime-footer">
                <button type="button" className="link-button tiny" onClick={() => scrollToQuestSection('quest-thread-ledger')}>
                  {i18n.t('quest.goalRuntime.viewDiscussion')}
                </button>
              </div>
            </section>
          )}

          {checks.length > 0 && (
            <section className="panel checks-panel mt-3">
              <div className="panel-title">
                <h2>{i18n.t('quest.section.checks')}</h2>
                <span>{i18n.t('quest.count.items', { count: checks.length })}</span>
              </div>
              <div className="check-list">
                {checks.map((check, index) => (
                  <div className="check-item" key={index}>
                    <div>
                      <strong>{checkTitle(check)}</strong>
                      {checkMeta(check) && <span className="mono">{checkMeta(check)}</span>}
                    </div>
                    <AgentMessageRenderer
                      content={checkMessage(check)}
                      className="check-message"
                      compact
                    />
                  </div>
                ))}
              </div>
            </section>
          )}

          <div className="vertical-timeline">
            {phaseEntries.map((entry, i) => {
              const isKey = entry.status === 'blocked' || entry.status === 'pass' || entry.variant === 'user' || entry.variant === 'comment' || entry.variant === 'success'
              const isRoutine = !isKey && (entry.variant === 'note' || entry.title.endsWith('汇报'))
              const rowClass = isKey ? ' is-key' : isRoutine ? ' is-routine' : ''
              return (
              <div className={'timeline-col-row' + rowClass} key={i}>
                <div className="timeline-rail">
                  <PhaseNode variant={entry.variant} icon={entry.icon} />
                  {i < phaseEntries.length - 1 && <div className="timeline-connector" />}
                </div>
                <div className="timeline-entry">
                  <div className="timeline-entry-head">
                    <strong>{entry.title}</strong>
                    <span className="meta">{entry.meta}</span>
                  </div>
                  <div className="entry-card">
                    {entry.body && <p>{entry.body}</p>}
                    {entry.chips && (
                      <div className="entry-chips">
                        {entry.chips.map((chip, ci) => (
                          <span key={ci} className="chip mono tiny">{chip}</span>
                        ))}
                      </div>
                    )}
                    {entry.status === 'pass' && (
                      <div className="entry-status pass">
                        <Icon name="circle-check" size={12} /> {i18n.t('quest.timeline.passed')}
                      </div>
                    )}
                    {entry.status === 'blocked' && (
                      <div className="entry-status" style={{ color: 'var(--red)', fontWeight: 500 }}>
                        <Icon name="triangle-alert" size={12} /> {i18n.t('quest.timeline.blocked')}
                      </div>
                    )}
                  </div>
                </div>
              </div>
              )
            })}
          </div>

          <details id="quest-execution-trace" className="panel mt-4 collapsible-section">
            <summary className="collapsible-header">
              <span>{i18n.t('quest.section.trace')}</span>
              <span className="mono">{i18n.t('quest.count.events', { count: events.length })}</span>
            </summary>
            <ExecutionTrace
              events={events}
              nativeEvidence={nativeEvidence}
              messages={allMessages}
              comments={traceComments}
              contextPacks={allContextPacks}
              toolResults={traceToolResults}
              loading={loadingEvidence}
              messageArtifactLinks={messageArtifactLinks}
              onJumpToArtifact={scrollToArtifact}
              onJumpToConclusion={() => scrollToQuestSection('quest-maker-delivery')}
            />
          </details>

        </div>

        <aside className="detail-side-panel">
          <div className="side-section-label">{i18n.t('quest.section.metadata')}</div>
          <div className="side-kv-list">
            {statPairs.map((kv) => (
              <div className={'side-kv ' + (kv.className || '')} key={kv.label}>
                <span>{kv.label}</span>
                <div>{kv.value}</div>
              </div>
            ))}
          </div>

          <div className="side-divider" />
          <div className="side-section-label">{i18n.t('quest.section.adventurers')}</div>
          <div className="side-kv-list">
            <div className="side-kv">
              <span>{i18n.t('world.warrior')}</span>
              <div className="row">
                <Logo variant="knight" size={18} />
                <span className="faint mono">{quest.warrior_id || '-'}</span>
              </div>
            </div>
            <div className="side-kv">
              <span>{i18n.t('world.mage')}</span>
              <div className="row" style={{ flexWrap: 'wrap', gap: 6 }}>
                <Logo variant="mage" size={18} />
                <span className="faint mono">
                  {quest.intensity === 'quick' ? i18n.t('quest.sidePanel.quickModeNoReview') : quest.mage_id || '-'}
                </span>
                {mageMatch && (
                  <button
                    type="button"
                    className="link-button micro"
                    style={{ padding: 0, height: 'auto' }}
                    onClick={() => scrollToQuestSection('quest-mage-review')}
                    title={i18n.t('quest.sidePanel.jumpToMageReview')}
                  >
                    <MageScoreChip
                      match={mageMatch}
                      mageId={quest.mage_id}
                      intensity={quest.intensity}
                      size="sm"
                    />
                  </button>
                )}
              </div>
            </div>
          </div>

          <div className="side-divider" />
          <div className="side-section-label">{i18n.t('quest.section.budget')}</div>
          <div className="side-kv-list">
            <div className="side-kv">
              <span>{i18n.t('quest.metaLabel.rework')}</span>
              <div className="side-progress">
                <div className="side-progress-row">
                  <span className="mono">{quest.rework_count ?? 0}</span>
                  <span className="mono">/ {quest.max_rework ?? '∞'}</span>
                </div>
                <div className="side-progress-bar">
                  <i
                    style={{
                      width:
                        Math.min(
                          100,
                          quest.max_rework ? ((quest.rework_count ?? 0) / quest.max_rework) * 100 : 25,
                        ) + '%',
                    }}
                  />
                </div>
              </div>
            </div>
          </div>

          {isBlocked && quest.blocked_reason && (
            <>
              <div className="side-divider" />
              <div className="alert-box">{fmtBlockedReason(quest)}</div>
            </>
          )}
          {isWaitingInput && (
            <>
              <div className="side-divider" />
              <div className="alert-box waiting-input-box">
                <strong>{i18n.t('quest.waitingInput.asking')}</strong>
                <span>{waitingInputQuestion}</span>
              </div>
            </>
          )}

          <div className="side-divider" />
          <div className="side-section-label">{i18n.t('quest.section.backup')}</div>
          <div className="side-kv-list">
            {loadingBackups ? (
              <div className="side-kv">
                <span className="faint">{i18n.t('quest.sidePanel.loading')}</span>
              </div>
            ) : backups.length > 0 ? (
              backups.map((bk, i) => (
                <button
                  type="button"
                  className="side-kv backup-row"
                  key={i}
                  onClick={() => showBackupDetail(bk.id)}
                >
                  <span className="mono">{bk.id || `backup-${String(i + 1).padStart(3, '0')}`}</span>
                  <em className="faint">
                    {bk.files_changed ? i18n.t('quest.sidePanel.filesChanged', { count: bk.files_changed }) : bk.mode || i18n.t('quest.sidePanel.beforeApply')}
                    {bk.created_at_ms ? ` · ${fmtTime(bk.created_at_ms)}` : ''}
                  </em>
                </button>
              ))
            ) : (
              <div className="side-kv">
                <span className="faint">{i18n.t('quest.sidePanel.noBackup')}</span>
              </div>
            )}
          </div>
        </aside>
      </div>

      {/* Action bar: only render when there are pending actions */}
      {(canReview || canApply || canDiscard || canStart || canStop || canSpawn || isQueued || isStarting || isBlocked || isWaitingInput) && (
        <footer className={'actionbar' + (isBlocked ? ' blocked' : '') + (isWaitingInput ? ' waiting-input' : '')}>
          {isWaitingInput ? (
            <div className="waiting-input-actionbar-wrap">
              <div className="waiting-input-alert">
                <div className="waiting-input-alert-icon">
                  <Icon name="message-circle" size={18} />
                </div>
                <div className="waiting-input-alert-body">
                  <div className="waiting-input-alert-title">{i18n.t('quest.waitingInput.alertTitle')}</div>
                  <div className="waiting-input-alert-desc">{waitingInputQuestion}</div>
                  {waitingInput?.question_id && (
                    <div className="mono waiting-input-question-id">question {waitingInput.question_id}</div>
                  )}
                </div>
              </div>
              <div className="waiting-input-answer">
                <textarea
                  rows={2}
                  aria-label={i18n.t('aria.answerAgent')}
                  placeholder={i18n.t('quest.waitingInput.answerPlaceholder')}
                  value={answerText}
                  onChange={(e) => setAnswerText(e.target.value)}
                  onKeyDown={(e) => {
                    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                      sendWaitingInputAnswer()
                    }
                  }}
                />
                <button
                  type="button"
                  className="button primary"
                  disabled={sendingAnswer || !answerText.trim()}
                  onClick={sendWaitingInputAnswer}
                >
                  {sendingAnswer ? <Icon name="spinner" className="spin" /> : <Icon name="send" />}
                  {i18n.t('quest.waitingInput.submit')}
                </button>
              </div>
            </div>
          ) : (
            <div className="detail-comment">
              <textarea
                rows={1}
                placeholder={i18n.t('quest.actionBar.commentPlaceholder')}
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                onKeyDown={(e) => {
                  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                    sendComment()
                  }
                }}
              />
              <button
                className="icon-button"
                aria-label={i18n.t('quest.actionBar.sendComment')}
                disabled={sendingComment || !comment.trim()}
                onClick={sendComment}
              >
                <Icon name="send" />
              </button>
            </div>
          )}
          {!isWaitingInput && (
          <div className="actionbar-row">
            <div className="actionbar-left">
              {canStop && (
                <div className="actionbar-running">
                  <span className="running-dot" />
                  <span>{i18n.t('quest.actionBar.running')}</span>
                </div>
              )}
              {isQueued && (
                <div className="actionbar-running">
                  <span className="running-dot pending" />
                  <span>{quest?.queue_position ? i18n.t('quest.actionBar.queuedPosition', { position: quest.queue_position }) : i18n.t('quest.actionBar.queued')}</span>
                </div>
              )}
              {isStarting && (
                <div className="actionbar-running">
                  <span className="running-dot pending" />
                  <span>{i18n.t('quest.actionBar.starting')}</span>
                </div>
              )}
            </div>
            <div className="actionbar-right">
              {canUpgradeWorkflow && (
                <button
                  className="button"
                  disabled={busy}
                  onClick={() => upgradeWorkflowMode(nextWorkflowMode)}
                  title={i18n.t('quest.actionBar.upgradeTitle', { from: quest?.workflow_mode || '', to: nextWorkflowMode })}
                >
                  <Icon name="arrow-up" />
                  {nextWorkflowLabel}
                </button>
              )}
              {canStart && (
                <button
                  className="button primary"
                  disabled={busy}
                  onClick={() =>
                    act(() => startQuest(qid))
                  }
                >
                  <Icon name="play" />
                  {i18n.t('quest.actionBar.start')}
                </button>
              )}
              {canReview && (
                <>
                  <button
                    className="button"
                    disabled={busy}
                    onClick={() => openReview('request_changes')}
                  >
                    <Icon name="rotate-ccw" />
                    {i18n.t('quest.actionBar.rework')}
                  </button>
                  <button
                    className="button danger"
                    disabled={busy}
                    onClick={() => openReview('reject')}
                  >
                    <Icon name="trash" />
                    {i18n.t('quest.actionBar.discard')}
                  </button>
                  {hasWorkspaceDiffPending ? (
                    <>
                      <button
                        className="button primary"
                        disabled={busy}
                        onClick={() => openReview('pass', true)}
                      >
                        <Icon name="check" />
                        {i18n.t('quest.actionBar.passAndApply')}
                      </button>
                      <button className="button ghost" onClick={() => openReview("pass", false)} disabled={busy}>
                        <Icon name="check" size={13} />
                        {i18n.t('quest.actionBar.passNoApply')}
                      </button>
                    </>
                  ) : (
                    <button
                      className="button primary"
                      disabled={busy}
                      onClick={() => openReview('pass', false)}
                    >
                      <Icon name="check" />
                      {i18n.t('quest.actionBar.confirmPass')}
                    </button>
                  )}
                </>
              )}
              {canApply && (
                <button className="button primary" disabled={busy} onClick={() => applyQuest(false)}>
                  <Icon name="check" />
                  {i18n.t('quest.actionBar.apply')}
                </button>
              )}
              {canDiscard && (
                <button className="button danger" disabled={busy} onClick={discardQuest}>
                  <Icon name="trash" />
                  {i18n.t('quest.actionBar.discard')}
                </button>
              )}
              {canSpawn && !showGoalRuntime && (
                <button className="button primary" disabled={busy} onClick={spawnExecuteFromDesign}>
                  {i18n.t('quest.actionBar.executeByPlan')}
                </button>
              )}
              {canStop && (
                <button className="button danger" disabled={busy} onClick={() => act(() => stopQuest(qid))}>
                  <Icon name="stop" />
                  {i18n.t('quest.actionBar.cancelExecute')}
                </button>
              )}
            </div>
          </div>
          )}
          {isBlocked && (
            <div className="blocked-actionbar-wrap">
              <div className="blocked-alert">
                <div className="blocked-alert-icon">
                  <Icon name="triangle-alert" size={18} />
                </div>
                <div className="blocked-alert-body">
                  <div className="blocked-alert-title">{i18n.t('quest.blockedBar.title')}</div>
                  <div className="blocked-alert-desc">{fmtBlockedReason(quest)}</div>
                </div>
              </div>
              <div className="blocked-controls">
                <div className="blocked-inputs">
                  <label className="blocked-input-field">
                    <span>{i18n.t('quest.blockedBar.addTurns')}</span>
                    <input type="number" min={1} value={blockedAddTurns} onChange={(e) => setBlockedAddTurns(Number(e.target.value))} />
                  </label>
                  <label className="blocked-input-field">
                    <span>{i18n.t('quest.blockedBar.addMinutes')}</span>
                    <input type="number" min={1} value={blockedAddMinutes} onChange={(e) => setBlockedAddMinutes(Number(e.target.value))} />
                  </label>
                </div>
                <div className="blocked-actions">
                  {quest.blocked_reason?.toLowerCase().includes('auth') && (
                    <>
                      <button className="button primary" disabled={busy} onClick={() => recoverAgent('relay_auth_login')}>
                        {i18n.t('quest.blockedBar.relayLogin')}
                      </button>
                      <button className="button" disabled={busy} onClick={() => recoverAgent('traex_login_status')}>
                        {i18n.t('quest.blockedBar.traexCheck')}
                      </button>
                    </>
                  )}
                  <button className="button primary" disabled={busy} onClick={() => openBlockedAction(BLOCKED_ACTION.continue)}>
                    {i18n.t('quest.blockedBar.continue')}
                  </button>
                  <button className="button" disabled={busy} onClick={() => openBlockedAction(BLOCKED_ACTION.userReview)}>
                    {i18n.t('quest.blockedBar.userReview')}
                  </button>
                  <button className="button danger" disabled={busy} onClick={() => openBlockedAction(BLOCKED_ACTION.cancel)}>
                    {i18n.t('quest.blockedBar.cancel')}
                  </button>
                </div>
              </div>
            </div>
          )}
        </footer>
      )}
      {/* Ended quests: minimal bar with rebuild + comment */}
      {isEnded && !canApply && !canDiscard && !canSpawn && (
        <footer className="actionbar actionbar-ended">
          <div className="detail-comment">
            <textarea
              rows={1}
              placeholder={i18n.t('quest.actionBar.endedCommentPlaceholder')}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  sendComment()
                }
              }}
            />
            <button
              className="icon-button"
              aria-label={i18n.t('quest.actionBar.sendComment')}
              disabled={sendingComment || !comment.trim()}
              onClick={sendComment}
            >
              <Icon name="send" />
            </button>
          </div>
          <div className="actionbar-row">
            <div className="actionbar-left">
              <span className="actionbar-ended-label">{i18n.t('quest.actionBar.questEnded')}</span>
            </div>
            <div className="actionbar-right">
              <button className="button ghost" disabled={busy} onClick={() => act(() => cloneQuest(quest))}>
                <Icon name="copy" />
                {i18n.t('quest.actionBar.clone')}
              </button>
            </div>
          </div>
        </footer>
      )}
      <ReviewSheet
        action={reviewAction}
        comment={reviewComment}
        onCommentChange={setReviewComment}
        error={reviewError}
        busy={busy}
        onConfirm={confirmReview}
        onClose={closeReview}
      />
      <BlockedSheet
        action={blockedAction}
        addTurns={blockedAddTurns}
        addMinutes={blockedAddMinutes}
        quest={quest}
        busy={busy}
        onConfirm={confirmBlockedAction}
        onClose={() => setBlockedAction(null)}
      />
      <ApplyWarningSheet
        warning={applyWarning}
        busy={busy}
        onConfirm={() => applyQuest(true)}
        onClose={() => setApplyWarning(null)}
      />
      <BackupDetailSheet
        detail={backupDetail}
        onClose={() => setBackupDetail({ loading: false, item: null })}
      />
    </div>
  )
}
