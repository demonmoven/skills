import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { questTimelineEntries, questActionState, deriveQuestAttention, deriveQuestProgressLine } from '../domain/questSelectors'
import { actorMeta } from '../domain/actorMeta'
import { automationPolicyWarnings, automationTriageMode, flowFromAutomationConfig, patchFromAutomationFlow, automationFlowLabel, automationFlowIcon, buildAutomationCreatePayload, buildAutomationUpdatePayload } from '../components/AutomationPolicySummary'
import { capabilityTierAdvice, capabilityTierRank, compareCapabilityTiers } from '../components/CapabilityTierBadge'
import FeedCard, { cardShouldHandleKeydown } from '../components/FeedCard'
import { visibleFanoutLeaves } from '../components/FanoutTreePanel'
import ThreadLedgerPanel from '../components/ThreadLedgerPanel'
import { getStuckDiagnosis, emitStuckDiagnosisPath, humanExceptionSourceKind } from '../features/quest/stuckDiagnosisMetrics'
import { buildSettingsPatch } from '../features/settings/settingsModel'
import { activityMatchesFeedSearch } from '../pages/FeedView'
import { automationCandidateMeta, isAutomationCandidate, splitInboxWorkQueues } from '../pages/inboxHelpers'
import { buildAttentionSections } from '../pages/questsBoardHelpers'
import { selectQuestProducerArtifact, shouldShowFinalVerdict, getWorkflowUpgradeInfo, deriveGoalRuntimeView } from '../pages/questDetailHelpers'
import { buildCommandItems } from '../features/commandPalette/buildCommandItems'
import { COMMAND_PRIORITY, type CommandItem } from '../features/commandPalette/types'
import { orderCommandItemsForPalette } from '../components/CommandPalette'

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) throw new Error(message)
}

function run() {
  const warrior = selectQuestProducerArtifact({
    quest: { design_summary: 'design fallback' },
    messages: [],
    contextPacks: [{
      ts: 20,
      turn: 1,
      phase: 1,
      data: { blocks: [{ name: 'warrior_artifact', source: 'warrior', trust: 'agent_output', content: 'warrior delivery' }] },
    }],
    toolResults: [{ content: '{"data":{"summary":"tool summary"}}', phase: 0, ts: 10, turn: 1, toolName: 'gloop_cli_signal' }],
    summaryArtifact: null,
    summaryArtifactText: null,
  })
  assert(warrior?.kind === 'warrior_context_artifact', 'warrior_artifact should beat tool summary')
  assert(warrior.content === 'warrior delivery', 'producer artifact must use warrior content')

  const noMageFallback = selectQuestProducerArtifact({
    quest: { design_summary: 'design fallback' },
    messages: [{ content: 'mage review', phase: 1, ts: 99, turn: 1 }],
    contextPacks: [],
    toolResults: [],
    summaryArtifact: null,
    summaryArtifactText: null,
  })
  assert(noMageFallback?.kind === 'design_summary', 'mage output must not be used as producer artifact')
  assert(noMageFallback.content === 'design fallback', 'design summary is the safe fallback')

  assert(!shouldShowFinalVerdict({ finalized_by: '' }), 'candidate final_verdict is not a final decision')
  assert(!shouldShowFinalVerdict({ finalized_by: undefined }), 'missing finalizer is not final')
  assert(shouldShowFinalVerdict({ finalized_by: 'user' }), 'user finalizer should show final verdict')

  const fanoutLeaves = visibleFanoutLeaves({
    group_id: 'grp_real',
    leaves: [
      {
        quest_id: 'qst_fake',
        query: 'looks like a leaf but has no backend leaf id',
        status: 'running',
        created_at_ms: 1,
      } as any,
      {
        quest_id: 'qst_leaf_b',
        fanout_leaf_id: 'leaf-b',
        query: 'leaf b',
        status: 'running',
        ownership_scopes: ['web/src'],
        merge_strategy: 'single_leaf',
        merge_owner_leaf_id: 'leaf-a',
        created_at_ms: 2,
      },
      {
        quest_id: 'qst_leaf_a',
        fanout_leaf_id: 'leaf-a',
        query: 'leaf a',
        status: 'running',
        ownership_scopes: ['internal/server'],
        merge_strategy: 'single_leaf',
        merge_owner_leaf_id: 'leaf-a',
        is_current: true,
        created_at_ms: 3,
      },
    ],
  } as any)
  assert(fanoutLeaves.length === 2, 'fanout tree must not fabricate leaves without backend fanout_leaf_id')
  assert(fanoutLeaves[0].fanout_leaf_id === 'leaf-a', 'current real leaf should sort first')
  assert(fanoutLeaves[1].fanout_leaf_id === 'leaf-b', 'non-current real leaves should remain visible')

  const fanoutFoldItem = {
    kind: 'user',
    query: '展开 2 个分支',
    summary: '展开 2 个分支',
    warrior_name: '',
    group_id: 'grp_feed_fold',
    fanout_fold: true,
    fanout_leaf_ids: ['leaf-a', 'leaf-b'],
    fanout_quest_ids: ['qst_leaf_a', 'qst_leaf_b'],
    merge_owner_leaf_id: 'leaf-a',
  } as any
  assert(activityMatchesFeedSearch(fanoutFoldItem, 'grp_feed'), 'feed search should match fanout group id')
  assert(activityMatchesFeedSearch(fanoutFoldItem, 'leaf-b'), 'feed search should match real fanout leaf id')
  assert(activityMatchesFeedSearch(fanoutFoldItem, 'qst_leaf_a'), 'feed search should match real fanout quest id')

  const threadPreviewItem = {
    kind: 'user',
    query: 'root post',
    summary: 'root post',
    reply_previews: [
      { post_id: 'post_checker', author_role: 'checker', summary: 'checker found the edge case' },
    ],
  } as any
  assert(activityMatchesFeedSearch(threadPreviewItem, 'edge case'), 'feed search should match reply preview content')
  assert(activityMatchesFeedSearch(threadPreviewItem, 'post_checker'), 'feed search should match reply preview post id')

  const postKindMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: {
      id: 'evt_rework',
      quest_id: 'qst_rework',
      short_id: 'rework',
      kind: 'post',
      post_kind: 'system_rework',
      author_role: 'system',
      query: 'system rework',
      summary: 'system asked the maker to rework',
      thread_id: 'qst_rework',
      reply_to: 'root_qst_rework',
      ts: Date.now(),
    } as any,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  // F2: "一种 Post" — post_kind 不再渲染强类型 badge，统一为轻状态点。
  assert(postKindMarkup.includes('feed-state-dot'), 'post_kind should remain visible only as a lightweight state dot')
  assert(!postKindMarkup.includes('feed-kind-chip'), 'post_kind must not drive the primary feed card tone')
  assert(!postKindMarkup.includes('feed-post-kind-mark'), 'strong post_kind card badge must not render')
  assert(!postKindMarkup.includes('feed-post-kind-badge'), 'old prominent post_kind badge must not render')
  assert(!postKindMarkup.includes('kind-rework'), 'system_rework must not turn the whole card into a rework card type')

  // F-slice1D: Feed 社区感 polish — 工程字段（causal count）不占主视觉，只作 title hover；
  // root post 正文有 feed-root-body 层次，区别于 reply。
  const polishItem = {
    id: 'evt_polish',
    quest_id: 'qst_polish',
    short_id: 'polish',
    kind: 'user',
    query: 'root post for community polish',
    summary: 'root post body',
    causal_refs: ['evt_root', 'evt_phase1'],
    ts: Date.now(),
  } as any
  const polishMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: polishItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(!polishMarkup.includes('causal 2'), 'feed card must not surface causal count as primary text — move to title hover')
  assert(polishMarkup.includes('feed-root-body'), 'root post body must carry feed-root-body class for visual hierarchy')

  // F-slice1C: inline thread expand — reply_count 多于 reply_previews 时必须露出可点击的展开按钮，
  // 且展开前不得伪造未拉取的 thread post（不应出现 source_event_id 的 event # 标记）。
  const threadExpandItem = {
    id: 'evt_thread',
    quest_id: 'qst_thread',
    short_id: 'thread',
    kind: 'user',
    query: 'root post with hidden replies',
    summary: 'root post body',
    reply_count: 5,
    reply_previews: [
      { post_id: 'post_r1', author_role: 'maker', summary: 'first reply' },
      { post_id: 'post_r2', author_role: 'checker', summary: 'second reply' },
    ],
    ts: Date.now(),
  } as any
  const expandMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: threadExpandItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(expandMarkup.includes('展开 3 条回复'), 'feed card must surface hidden reply count (5 - 2 = 3) as an expand button')
  assert(expandMarkup.includes('feed-reply-preview-more'), 'expand affordance must keep the preview-more class for styling')
  assert(/<button[^>]*feed-reply-preview-more/.test(expandMarkup), 'expand affordance must be a clickable button, not a static div')
  assert(!expandMarkup.includes('event #'), 'feed card must not fabricate thread post source_event_id before the expand fetch')

  // F-slice1C: 键盘可达性 —— 子按钮（展开/查看完整 thread/收起）的 Enter/Space 冒泡到卡片主体时，
  // 不得触发整卡 onOpen 跳 Detail；只有事件来自卡片自身（currentTarget === target）才处理。
  const cardEl: unknown = 'card'
  const buttonEl: unknown = 'button'
  assert(cardShouldHandleKeydown({ currentTarget: cardEl, target: cardEl, key: 'Enter' }) === true, 'card own Enter should open the thread')
  assert(cardShouldHandleKeydown({ currentTarget: cardEl, target: cardEl, key: ' ' }) === true, 'card own Space should open the thread')
  assert(cardShouldHandleKeydown({ currentTarget: cardEl, target: buttonEl, key: 'Enter' }) === false, 'child button Enter must not bubble to open the card')
  assert(cardShouldHandleKeydown({ currentTarget: cardEl, target: buttonEl, key: ' ' }) === false, 'child button Space must not bubble to open the card')
  assert(cardShouldHandleKeydown({ currentTarget: cardEl, target: cardEl, key: 'ArrowDown' }) === false, 'non-activation keys must not open the card')

  // F-slice2: Automation as Feed Actor — automation-created root post + automation_run reply preview
  // 必须作为社区流成员自然呈现：root 作者是 Automation 不是「你」，preview 走 automation 角色，
  // 不得把后端 no_finding/candidate/exception outcome 伪造成强类型卡片 chip。前端只消费后端已投影的事实。
  const automationItem = {
    id: 'evt_auto',
    quest_id: 'qst_auto',
    short_id: 'auto',
    kind: 'user',
    author_role: 'automation',
    source: 'automation:auto_weekly_review',
    warrior_name: '',
    query: '每周回顾自动发起的委托',
    summary: 'Automation auto_weekly_review 创建的委托 · triage_mode=candidate',
    reply_count: 1,
    reply_previews: [
      { post_id: 'auto_run_qst_auto_auto_weekly_review', author_role: 'automation', post_kind: 'automation_run', summary: 'Automation auto_weekly_review created this quest | triage_mode=candidate' },
    ],
    ts: Date.now(),
  } as any
  const automationMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: automationItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  // 1. root 作者显示 Automation，不得出现人类发帖的「你」主语
  assert(automationMarkup.includes('Automation'), 'automation-created root post must show Automation as author')
  // 精确锁定作者名容器：只断言 feed-author-name 渲染成 Automation，不用宽正则 />你</
  // （否则 summary 正文里自然出现的「你」会假红）。反向只禁「不是你」太弱，正向断言「是 Automation」更强。
  assert(automationMarkup.includes('feed-author-name">Automation<'), 'automation root author slot must render Automation, not the human-authored 你')
  assert(automationMarkup.includes('feed-role-chip automation'), 'automation root must carry the automation role chip, not a human/maker role')
  // 2. reply preview 走 automation 角色（role-automation / Automation label），不是强类型 outcome card
  assert(automationMarkup.includes('feed-reply-preview role-automation'), 'automation_run preview must use the automation reply preview tone')
  assert(automationMarkup.includes('自动化·auto_weekly_review') || automationMarkup.includes('auto_weekly_review'), 'automation root must surface the automation source name lightly')
  // 3. 不得出现前端伪造的 no_finding/candidate/exception outcome chip。
  // 注意：automation_run 的真实 summary 会含 triage_mode=candidate 文本（后端 projectAutomationRunThreadPost 投影），
  // 那是消费后端事实、原样显示在 preview 文本里，不算伪造。禁止的是把它们做成结构化 outcome chip/card（feed-kind-chip、
  // role-outcome 之类），即前端不该把后端没投成 ThreadPost 的 outcome 自行类型化。
  assert(!automationMarkup.includes('no_finding'), 'feed must not fabricate a no_finding outcome chip from the frontend')
  assert(!automationMarkup.includes('exception'), 'feed must not fabricate an exception outcome chip from the frontend')
  assert(!automationMarkup.includes('role-candidate') && !automationMarkup.includes('role-no_finding') && !automationMarkup.includes('role-exception'), 'feed must not typecast automation outcomes into structured outcome role chips')
  assert(!automationMarkup.includes('feed-kind-chip'), 'feed must not render automation outcomes as strong typed kind chips')

  // F-slice2B: candidate quest 在 Feed 暴露确认入口（triage_mode=candidate + status=blocked）
  const candidateItem = {
    id: 'evt_candidate',
    quest_id: 'qst_candidate',
    short_id: 'cand',
    kind: 'user',
    query: 'candidate quest for feed confirmation',
    summary: 'candidate quest body',
    triage_mode: 'candidate',
    status: 'blocked',
    ts: Date.now(),
  } as any
  const candidateMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: candidateItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(candidateMarkup.includes('feed-candidate-actions'), 'candidate quest must show candidate tag in Feed')
  assert(candidateMarkup.includes('候选·待确认'), 'candidate quest must show 候选·待确认 label')
  assert(!candidateMarkup.includes('feed-candidate-btn confirm'), 'candidate quest must NOT have a confirm button (decision belongs to Inbox)')
  assert(!candidateMarkup.includes('feed-candidate-btn cancel'), 'candidate quest must NOT have a cancel button (decision belongs to Inbox)')
  assert(candidateMarkup.includes('feed-candidate-tag'), 'candidate quest must show candidate tag for deep-link to Inbox')
  // non-candidate quest should not show candidate actions
  const nonCandidateMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: { ...candidateItem, triage_mode: 'direct', status: 'success' },
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(!nonCandidateMarkup.includes('feed-candidate-actions'), 'non-candidate quest must not show candidate actions')

  const attentionMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: {
      id: 'evt_attention',
      quest_id: 'qst_attention',
      short_id: 'attn',
      kind: 'attention',
      query: 'blocked quest',
      summary: 'agent auth expired',
      status: 'blocked',
      risk_level: 'high',
      recommended_action: '完成 Agent 登录或授权后恢复执行。',
      available_actions: ['continue', 'user-review', 'cancel'],
      action_endpoint_hint: 'resolve-blocked',
      blocked_reason_code: 'agent_error_auth',
      blocked_category: 'configuration',
      ts: Date.now(),
    } as any,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(attentionMarkup.includes('feed-card is-attention risk-high'), 'attention item must use the dedicated attention card tone')
  assert(attentionMarkup.includes('需要你处理'), 'attention card must show the action-oriented author label')
  assert(attentionMarkup.includes('完成 Agent 登录或授权后恢复执行。'), 'attention card must surface recommended_action')
  assert(attentionMarkup.includes('feed-attention-action primary'), 'attention card must render the continue action as a primary button')
  assert(attentionMarkup.includes('feed-attention-action danger'), 'attention card must render cancel as a danger action')

  const threadLedgerMarkup = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: [{
      post_id: 'post_delivery',
      thread_id: 'qst_thread',
      root_post_id: 'root_qst_thread',
      parent_reply_id: 'root_qst_thread',
      causal_refs: ['root_qst_thread', 'review_hints'],
      author_role: 'maker',
      author_identity: 'adv_maker',
      source_event_id: 42,
      kind: 'maker_report',
      content: 'delivery summary',
      created_at_ms: Date.now(),
      artifact_refs: [{
        id: 'artifact_delivery',
        artifact_type: 'DeliveryArtifact',
        ref: 'outputs/main.patch',
        role: 'maker',
      }],
    }],
  }))
  assert(threadLedgerMarkup.includes('event #42'), 'thread ledger should expose source_event_id from backend data')
  assert(threadLedgerMarkup.includes('causal root_qst_thread'), 'thread ledger should expose causal refs, not only counts')
  assert(threadLedgerMarkup.includes('artifact DeliveryArtifact'), 'thread ledger should expose artifact type')
  assert(threadLedgerMarkup.includes('outputs/main.patch'), 'thread ledger should expose artifact ref')

  const quickTimeline = questTimelineEntries({
    id: 'qst_quick',
    query: 'quick',
    type: 'design',
    status: 'success',
    intensity: 'quick',
    mage_id: '',
    rework_count: 0,
    max_rework: 2,
    phases: [
      { phase_idx: 0, status: 'done' },
      { phase_idx: 1, status: 'done' },
    ],
    pipeline_def: [
      { phase_idx: 0, role: 'warrior', class: 'warrior', name: 'warrior', display_name: '剑士执行' },
      { phase_idx: 1, role: 'mage', class: 'mage', name: 'mage', display_name: '法师评审' },
    ],
  } as any, [])
  const magePhase = quickTimeline.find((entry) => entry.title === '法师评审')
  assert(magePhase?.body === '快速模式已跳过法师评审，剑士交付直接进入用户终审。', 'quick mage phase should explain skipped review')
  assert(magePhase?.chips?.includes('已跳过'), 'quick mage phase should show skipped chip')

  const patch = buildSettingsPatch({ runtime_idle_warning_ms: 45_000 }, [])
  assert(patch.runtime_idle_warning_ms === 45_000, 'runtime idle warning setting must be persisted')

  const attention = buildAttentionSections([
    {
      id: 'qst_human',
      short_id: 'human',
      query: 'needs action',
      type: 'execute',
      status: 'blocked',
      rework_count: 0,
      max_rework: 1,
      created_by: 'user',
      created_at_ms: Date.now(),
      human_exception: {
        id: 'hex_qst_human',
        quest_id: 'qst_human',
        reason: 'Agent auth expired',
        recommended_action: '完成 Agent 登录或授权后恢复执行。',
        risk_level: 'high',
        available_actions: ['recover'],
        audit_ref: '/quests/qst_human',
        source_status: 'blocked',
      },
    } as any,
  ])
  assert(attention.some((section) => section.key === 'blocked' && section.items.length === 1), 'human exception quest should remain in attention queue')

  const blockedDiagnosis = getStuckDiagnosis({
    id: 'qst_blocked',
    query: 'blocked task',
    type: 'execute',
    status: 'blocked',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
    human_exception: {
      id: 'hex_qst_blocked',
      quest_id: 'qst_blocked',
      reason: 'Agent auth expired',
      recommended_action: '完成 Agent 登录或授权后恢复执行。',
      risk_level: 'high',
      priority: 0,
      available_actions: ['recover', 'cancel'],
      audit_ref: '/quests/qst_blocked',
      source_status: 'blocked',
      evidence_summary: 'session auth check failed',
    },
  } as any)
  assert(blockedDiagnosis?.kind === 'blocked', 'blocked quest should be diagnosed as blocked')
  assert(blockedDiagnosis.source === 'human_exception', 'blocked diagnosis should use HumanException source')
  assert(blockedDiagnosis.recommendedAction === '完成 Agent 登录或授权后恢复执行。', 'blocked diagnosis should preserve recommended action')
  assert(blockedDiagnosis.hasEvidenceJump, 'blocked diagnosis should expose evidence jump from human exception evidence')

  const waitingDiagnosis = getStuckDiagnosis({
    id: 'qst_waiting',
    query: 'waiting task',
    type: 'execute',
    status: 'waiting_input',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
    waiting_input: {
      question_id: 'ask_1',
      question_text: 'Need target branch?',
      phase_idx: 0,
    },
    human_exception: {
      id: 'hex_qst_waiting',
      quest_id: 'qst_waiting',
      reason: 'Agent asked for missing branch',
      recommended_action: '回答 Agent 的问题后继续。',
      risk_level: 'medium',
      available_actions: ['answer'],
      audit_ref: '/quests/qst_waiting',
      source_status: 'waiting_input',
    },
  } as any)
  assert(waitingDiagnosis?.kind === 'waiting_input', 'waiting input quest should be diagnosed as waiting_input')
  assert(waitingDiagnosis.reason === 'Agent asked for missing branch', 'waiting diagnosis should prefer HumanException reason')

  const applyFailedDiagnosis = getStuckDiagnosis({
    id: 'qst_apply',
    query: 'apply task',
    type: 'execute',
    status: 'success',
    apply_status: 'failed',
    apply_error: 'patch rejected',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
    human_exception: {
      id: 'hex_qst_apply',
      quest_id: 'qst_apply',
      reason: 'Apply failed after successful quest',
      recommended_action: '查看应用错误，选择重试或放弃变更。',
      risk_level: 'high',
      available_actions: ['apply', 'discard'],
      audit_ref: '/quests/qst_apply',
      source_status: 'apply_failed',
      evidence_summary: 'patch rejected',
    },
  } as any)
  assert(applyFailedDiagnosis?.kind === 'apply_failed', 'success quest with failed apply should be diagnosed as apply_failed')
  assert(applyFailedDiagnosis.source === 'human_exception', 'apply_failed diagnosis should use HumanException source when present')

  const missingProjectionDiagnosis = getStuckDiagnosis({
    id: 'qst_missing_projection',
    query: 'blocked without projection',
    type: 'execute',
    status: 'blocked',
    blocked_reason: 'auth expired',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
  } as any)
  assert(!missingProjectionDiagnosis, 'blocked/waiting/apply_failed must not be faked without HumanException projection')

  const missingWaitingProjectionDiagnosis = getStuckDiagnosis({
    id: 'qst_missing_waiting_projection',
    query: 'waiting without projection',
    type: 'execute',
    status: 'waiting_input',
    waiting_input: {
      question_id: 'ask_missing',
      question_text: 'Need more context?',
      phase_idx: 0,
    },
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
  } as any)
  assert(!missingWaitingProjectionDiagnosis, 'waiting_input must not be faked without HumanException projection')

  const missingApplyProjectionDiagnosis = getStuckDiagnosis({
    id: 'qst_missing_apply_projection',
    query: 'apply failed without projection',
    type: 'execute',
    status: 'success',
    apply_status: 'failed',
    apply_error: 'patch rejected',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
  } as any)
  assert(!missingApplyProjectionDiagnosis, 'apply_failed must not be faked without HumanException projection')

  const idleDiagnosis = getStuckDiagnosis({
    id: 'qst_idle',
    query: 'idle task',
    type: 'execute',
    status: 'running',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
    agent_sessions: [{
      session_id: 'sess_idle',
      quest_id: 'qst_idle',
      health_status: 'idle',
      updated_at_ms: 123,
    }],
  } as any)
  assert(idleDiagnosis?.kind === 'runtime.idle_warning', 'idle session should be diagnosed as runtime idle warning')
  assert(idleDiagnosis.source === 'runtime', 'runtime idle warning must not be treated as HumanException')

  const protocolDiagnosis = getStuckDiagnosis({
    id: 'qst_protocol',
    query: 'protocol task',
    type: 'execute',
    status: 'reviewing',
    rework_count: 0,
    max_rework: 1,
    created_by: 'user',
    created_at_ms: Date.now(),
    agent_sessions: [{
      session_id: 'sess_protocol',
      quest_id: 'qst_protocol',
      health_status: 'protocol_risk',
      updated_at_ms: 456,
    }],
  } as any)
  assert(protocolDiagnosis?.kind === 'runtime.protocol_warning', 'protocol_risk session should be diagnosed as runtime protocol warning')
  assert(protocolDiagnosis.source === 'runtime', 'runtime protocol warning must not be treated as HumanException')

  const storage: { rows: string[] } = { rows: [] }
  const fakeStorage = {
    getItem: () => storage.rows[0] || null,
    setItem: (_key: string, value: string) => { storage.rows[0] = value },
  }
  const emitted = emitStuckDiagnosisPath({
    entry_surface: 'feed',
    quest_id: 'qst_idle',
    exception_kind: 'runtime.idle_warning',
    click_count: 1,
    resolved_surface: 'quest_detail',
    has_recommended_action: true,
    has_evidence_jump: true,
  }, fakeStorage)
  assert(emitted, 'stuck diagnosis path event should be emitted')
  assert(storage.rows[0].includes('ui.stuck_diagnosis_path'), 'stored metric should include event name')
  assert(humanExceptionSourceKind('user_review') === null, 'unknown/non-stuck human exception statuses must not emit stuck diagnosis metrics')

  const candidateMeta = automationCandidateMeta({
    id: 'qst_candidate_meta',
    query: 'candidate meta',
    type: 'execute',
    status: 'pending',
    triage_mode: 'candidate',
    candidate_source: 'automation:auto_weekly_review',
    created_by: 'automation:auto_weekly_review',
    rework_count: 0,
    max_rework: 1,
    created_at_ms: Date.now(),
  } as any)
  assert(isAutomationCandidate({ created_by: 'automation:auto_weekly_review' } as any), 'automation created inbox item should be candidate')
  assert(candidateMeta.source === 'auto_weekly_review', 'candidate source should strip automation prefix')
  assert(candidateMeta.triageMode === 'candidate', 'candidate triage mode should be surfaced')
  assert(candidateMeta.holdReason.includes('需要确认'), 'candidate card should explain why it waits in Inbox')

  const splitInbox = splitInboxWorkQueues([
    {
      id: 'qst_candidate_meta',
      query: 'candidate meta',
      type: 'execute',
      status: 'pending',
      created_by: 'automation:auto_weekly_review',
      rework_count: 0,
      max_rework: 1,
      created_at_ms: Date.now(),
    } as any,
    {
      id: 'qst_manual_pending',
      query: 'manual pending',
      type: 'execute',
      status: 'pending',
      created_by: 'user',
      rework_count: 0,
      max_rework: 1,
      created_at_ms: Date.now(),
    } as any,
    {
      id: 'qst_exception_overlap',
      query: 'overlap',
      type: 'execute',
      status: 'blocked',
      created_by: 'automation:auto_weekly_review',
      rework_count: 0,
      max_rework: 1,
      created_at_ms: Date.now(),
    } as any,
  ], [{
    id: 'hex_overlap',
    quest_id: 'qst_exception_overlap',
    reason: 'blocked',
    recommended_action: 'recover',
    risk_level: 'high',
    available_actions: ['recover'],
    audit_ref: '/quests/qst_exception_overlap',
    source_status: 'blocked',
  }])
  assert(splitInbox.automationCandidates.length === 1 && splitInbox.automationCandidates[0].id === 'qst_candidate_meta', 'automation lane should contain only automation candidates and exclude human exception overlaps')
  assert(splitInbox.pendingDelegations.length === 1 && splitInbox.pendingDelegations[0].id === 'qst_manual_pending', 'pending lane should contain non-automation pending items')

  assert(automationTriageMode({ autoStart: false }) === 'candidate', 'auto_start=false should default to candidate triage mode')
  assert(automationTriageMode({ autoStart: true }) === 'direct', 'auto_start=true should default to direct triage mode')
  assert(automationPolicyWarnings({
    triageMode: 'direct',
    autoStart: true,
    capabilityTier: 'tier_c',
    connectors: ['git'],
    enabledConnectors: [],
  }).length === 2, 'policy summary should warn for direct Tier C and disabled connectors')
  assert(capabilityTierRank('tier_a') < capabilityTierRank('tier_b'), 'tier A should rank ahead of tier B')
  assert(capabilityTierRank('tier_b') < capabilityTierRank('tier_c'), 'tier C should sort behind observable tiers')
  assert(capabilityTierAdvice('tier_c').includes('低可观测'), 'tier C advice should explain low observability')
  const sortedTiers = ['tier_c', 'unknown', 'tier_b', 'tier_a'].sort(compareCapabilityTiers)
  assert(sortedTiers[0] === 'tier_a', 'tier A should be the first recommended capability tier')
  assert(sortedTiers[1] === 'tier_b', 'tier B should be recommended before unknown and tier C')
  assert(sortedTiers[2] === 'unknown', 'unknown tier should remain ahead of tier C but behind observable tiers')
  assert(sortedTiers[3] === 'tier_c', 'tier C should sort last among normal tiers')
  assert(compareCapabilityTiers(undefined, 'tier_c') < 0, 'missing tier should still rank ahead of tier C')

  // ===== v0.5 Slice 3: workflow mode upgrade tests =====

  // S3-1: FeedCard system_workflow_upgrade → system 色 dot（不是 blocker 色），
  // 与 system_escalation 区分：升档是正常系统行为，不应显示为阻塞告警色。
  const wfUpgradeItem = {
    id: 'evt_wf_up',
    quest_id: 'qst_wf',
    short_id: 'wfup',
    kind: 'post',
    post_kind: 'system_workflow_upgrade',
    author_role: 'system',
    query: 'workflow upgraded',
    summary: 'Upgraded from direct to checked',
    thread_id: 'qst_wf',
    reply_to: 'root_qst_wf',
    ts: Date.now(),
  } as any
  const wfUpgradeMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: wfUpgradeItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(wfUpgradeMarkup.includes('kind-system'), 'system_workflow_upgrade must render as system dot, not blocker')
  assert(!wfUpgradeMarkup.includes('kind-blocker'), 'system_workflow_upgrade must not use blocker dot color')

  // S3-2: FeedCard system_escalation → blocker 色 dot（不是 system 色），
  // blocked 投影是需要人工介入的告警，保持阻塞色。
  const escalationItem = {
    id: 'evt_esc',
    quest_id: 'qst_esc',
    short_id: 'esc',
    kind: 'post',
    post_kind: 'system_escalation',
    author_role: 'system',
    query: 'blocked escalation',
    summary: 'Escalation: blocked (auth_expired)',
    thread_id: 'qst_esc',
    reply_to: 'root_qst_esc',
    ts: Date.now(),
  } as any
  const escalationMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: escalationItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(escalationMarkup.includes('kind-blocker'), 'system_escalation must render as blocker dot')
  assert(!escalationMarkup.includes('kind-system'), 'system_escalation must not use system dot color')

  // S3-3: ThreadLedgerPanel system_workflow_upgrade → WorkflowUpgrade label
  const wfUpgradeLedger = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: [{
      post_id: 'post_wf_up',
      thread_id: 'qst_wf',
      root_post_id: 'root_qst_wf',
      parent_reply_id: 'root_qst_wf',
      author_role: 'system',
      kind: 'system_workflow_upgrade',
      content: 'Upgraded workflow mode from direct to checked (reason: external write requires review)',
      created_at_ms: Date.now(),
    }],
  }))
  assert(wfUpgradeLedger.includes('WorkflowUpgrade'), 'system_workflow_upgrade post must show WorkflowUpgrade kind label')
  assert(wfUpgradeLedger.includes('Upgraded workflow mode'), 'system_workflow_upgrade content must be visible')

  // S3-4: ThreadLedgerPanel system_escalation → Escalation label（与 WorkflowUpgrade 区分）
  const escalationLedger = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: [{
      post_id: 'post_esc',
      thread_id: 'qst_esc',
      root_post_id: 'root_qst_esc',
      parent_reply_id: 'root_qst_esc',
      author_role: 'system',
      kind: 'system_escalation',
      content: 'Escalation: blocked. auth expired',
      created_at_ms: Date.now(),
    }],
  }))
  assert(escalationLedger.includes('Escalation'), 'system_escalation post must show Escalation kind label')
  assert(!escalationLedger.includes('WorkflowUpgrade'), 'system_escalation must not show WorkflowUpgrade label')

  // S3-5: ThreadLedgerPanel decision_note + pinnedOutcome 不退化
  const decisionNoteLedger = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: [{
      post_id: 'post_decision',
      thread_id: 'qst_dec',
      root_post_id: 'root_qst_dec',
      parent_reply_id: 'root_qst_dec',
      author_role: 'system',
      kind: 'decision_note',
      content: 'Outcome: success | verdict=pass | applied=true | finalized_by=user',
      created_at_ms: Date.now(),
      artifact_refs: [{
        id: 'decision_qst_dec',
        artifact_type: 'DecisionNote',
        ref: 'pinned_outcome_summary',
        role: 'system',
      }],
    }],
    pinnedOutcome: 'Outcome: success | verdict=pass | applied=true | finalized_by=user',
  }))
  assert(decisionNoteLedger.includes('DecisionNote'), 'decision_note post must show DecisionNote kind label')
  assert(decisionNoteLedger.includes('pinned-outcome') || decisionNoteLedger.includes('pinned_outcome') || decisionNoteLedger.includes('Outcome: success'), 'pinned outcome must be visible')
  assert(decisionNoteLedger.includes('artifact DecisionNote'), 'decision_note artifact ref must be exposed')

  // S3-6: QuestDetail workflow mode pill — Direct / Checked / Goal 渲染
  const directPill = getWorkflowUpgradeInfo('direct', false, false, false)
  assert(directPill.pillLabel === 'Direct', 'direct mode pill must show Direct')
  assert(directPill.pillClass === 'workflow-pill direct', 'direct mode pill must use direct class')
  const checkedPill = getWorkflowUpgradeInfo('checked', false, false, false)
  assert(checkedPill.pillLabel === 'Checked', 'checked mode pill must show Checked')
  assert(checkedPill.pillClass === 'workflow-pill checked', 'checked mode pill must use checked class')
  const goalPill = getWorkflowUpgradeInfo('goal', false, false, false)
  assert(goalPill.pillLabel === 'Goal', 'goal mode pill must show Goal')
  assert(goalPill.pillClass === 'workflow-pill goal', 'goal mode pill must use goal class')
  const emptyPill = getWorkflowUpgradeInfo('', false, false, false)
  assert(emptyPill.pillLabel === null, 'empty mode must not render pill')
  assert(emptyPill.pillClass === null, 'empty mode must not render pill class')
  const unknownPill = getWorkflowUpgradeInfo('legacy', false, false, false)
  assert(unknownPill.pillLabel === null, 'unknown mode must not render pill')

  // S3-7: QuestDetail 升级按钮可见性 — direct 显示"升级到 Checked"
  const directUpgrade = getWorkflowUpgradeInfo('direct', false, false, false)
  assert(directUpgrade.canUpgrade === true, 'direct mode must show upgrade button')
  assert(directUpgrade.nextMode === 'checked', 'direct → next must be checked')
  assert(directUpgrade.nextLabel === '升级到 Checked', 'direct upgrade label must be 升级到 Checked')

  // S3-8: QuestDetail 升级按钮可见性 — checked 显示"升级到 Goal"
  const checkedUpgrade = getWorkflowUpgradeInfo('checked', false, false, false)
  assert(checkedUpgrade.canUpgrade === true, 'checked mode must show upgrade button')
  assert(checkedUpgrade.nextMode === 'goal', 'checked → next must be goal')
  assert(checkedUpgrade.nextLabel === '升级到 Goal', 'checked upgrade label must be 升级到 Goal')

  // S3-9: QuestDetail 升级按钮可见性 — goal 不显示
  const goalUpgrade = getWorkflowUpgradeInfo('goal', false, false, false)
  assert(goalUpgrade.canUpgrade === false, 'goal mode must NOT show upgrade button')
  assert(goalUpgrade.nextMode === '', 'goal next must be empty')

  // S3-10: QuestDetail 升级按钮可见性 — blocked / ended / waiting_input 不显示
  assert(getWorkflowUpgradeInfo('direct', true, false, false).canUpgrade === false, 'blocked direct must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('direct', false, true, false).canUpgrade === false, 'ended (success/failed/cancelled) must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('direct', false, false, true).canUpgrade === false, 'waiting_input must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('checked', true, false, false).canUpgrade === false, 'blocked checked must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('checked', false, true, false).canUpgrade === false, 'ended checked must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('direct', false, false, false, true).canUpgrade === false, 'runtime-active direct must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('checked', false, false, false, true).canUpgrade === false, 'runtime-active checked must NOT show upgrade')

  // S3-11: QuestDetail 升级按钮可见性 — 未知/空 mode 不显示（防 legacy 数据误触发）
  assert(getWorkflowUpgradeInfo('legacy', false, false, false).canUpgrade === false, 'unknown mode must NOT show upgrade')
  assert(getWorkflowUpgradeInfo('', false, false, false).canUpgrade === false, 'empty mode must NOT show upgrade')
  assert(getWorkflowUpgradeInfo(undefined, false, false, false).canUpgrade === false, 'undefined mode must NOT show upgrade')

  // ===== v0.5 Automation 三档: flow mapping tests =====

  // AF-1: flowFromAutomationConfig 正向推导 — direct (execute + quick)
  assert(flowFromAutomationConfig({ quest_type: 'execute', intensity: 'quick', with_design_phase: false }) === 'direct', 'execute+quick+noDesign must be direct')

  // AF-2: flowFromAutomationConfig 正向推导 — checked (execute + standard, no design phase)
  assert(flowFromAutomationConfig({ quest_type: 'execute', intensity: 'standard', with_design_phase: false }) === 'checked', 'execute+standard+noDesign must be checked')

  // AF-3: flowFromAutomationConfig 正向推导 — goal (quest_type=design)
  assert(flowFromAutomationConfig({ quest_type: 'design', intensity: 'standard' }) === 'goal', 'design quest must be goal')

  // AF-4: flowFromAutomationConfig 正向推导 — goal (with_design_phase=true 即使 quest_type=execute，归为 legacy goal)
  assert(flowFromAutomationConfig({ quest_type: 'execute', intensity: 'standard', with_design_phase: true }) === 'goal', 'execute+withDesignPhase must be goal (legacy design-phase)')

  // AF-5: flowFromAutomationConfig 正向推导 — 无 intensity 默认 checked
  assert(flowFromAutomationConfig({ quest_type: 'execute' }) === 'checked', 'execute without intensity defaults to checked')

  // AF-6: flowFromAutomationConfig 正向推导 — quick 但有 design_phase → goal（design phase 优先于 intensity）
  assert(flowFromAutomationConfig({ quest_type: 'execute', intensity: 'quick', with_design_phase: true }) === 'goal', 'quick+designPhase must be goal, not direct')

  // AF-7: patchFromAutomationFlow 反向生成 — direct
  const directPatch = patchFromAutomationFlow('direct')
  assert(directPatch.quest_type === 'execute', 'direct → quest_type=execute')
  assert(directPatch.intensity === 'quick', 'direct → intensity=quick')
  assert(directPatch.with_design_phase === false, 'direct → with_design_phase=false')

  // AF-8: patchFromAutomationFlow 反向生成 — checked（关键：with_design_phase 必须 false）
  const checkedPatch = patchFromAutomationFlow('checked')
  assert(checkedPatch.quest_type === 'execute', 'checked → quest_type=execute')
  assert(checkedPatch.intensity === 'standard', 'checked → intensity=standard')
  assert(checkedPatch.with_design_phase === false, 'checked → with_design_phase must be false (Critical: cannot map checked to design pipeline)')

  // AF-9: patchFromAutomationFlow 反向生成 — goal
  const goalPatch = patchFromAutomationFlow('goal')
  assert(goalPatch.quest_type === 'design', 'goal → quest_type=design')
  assert(goalPatch.with_design_phase === false, 'goal → with_design_phase=false (quest_type=design is sufficient)')

  // AF-10: 双向 round-trip 一致（flow → patch → flow）
  for (const flow of ['direct', 'checked', 'goal'] as const) {
    const patch = patchFromAutomationFlow(flow)
    const roundTrip = flowFromAutomationConfig(patch)
    assert(roundTrip === flow, `round-trip ${flow} → patch → ${roundTrip} must match`)
  }

  // AF-11: automationFlowLabel
  assert(automationFlowLabel('direct') === 'Direct', 'direct label')
  assert(automationFlowLabel('checked') === 'Checked', 'checked label')
  assert(automationFlowLabel('goal') === 'Goal', 'goal label')

  // AF-12: automationFlowIcon（非空即可，不锁具体图标名）
  assert(automationFlowIcon('direct').length > 0, 'direct icon must be non-empty')
  assert(automationFlowIcon('checked').length > 0, 'checked icon must be non-empty')
  assert(automationFlowIcon('goal').length > 0, 'goal icon must be non-empty')

  // ===== v0.5 Automation 三档: consumer payload builder tests =====

  const baseCreateForm = {
    name: 'test',
    description: 'desc',
    query: 'do thing',
    workingDir: '/tmp',
    workspaceMode: 'auto',
    trigger: 'manual',
    cron: '',
    warriorId: '',
    mageId: '',
    autoStart: true,
    autoApply: false,
    allowL2: false,
    planOnly: false,
    connectors: [] as string[],
  }

  // AF-13: Direct create payload → execute/quick/with_design_phase=false，不含 auto_spawn_execute
  const directPayload = buildAutomationCreatePayload(baseCreateForm, 'direct')
  assert(directPayload.quest_type === 'execute', 'Direct create must send quest_type=execute')
  assert(directPayload.intensity === 'quick', 'Direct create must send intensity=quick (Critical: must use flowPatch intensity, not stale state)')
  assert(directPayload.with_design_phase === false, 'Direct create must send with_design_phase=false')
  assert(!('auto_spawn_execute' in directPayload), 'Direct create payload must NOT include auto_spawn_execute')

  // AF-14: Checked create payload → execute/standard/with_design_phase=false，不含 auto_spawn_execute
  const checkedPayload = buildAutomationCreatePayload(baseCreateForm, 'checked')
  assert(checkedPayload.quest_type === 'execute', 'Checked create must send quest_type=execute')
  assert(checkedPayload.intensity === 'standard', 'Checked create must send intensity=standard')
  assert(checkedPayload.with_design_phase === false, 'Checked create must send with_design_phase=false (Critical: checked ≠ design pipeline)')
  assert(!('auto_spawn_execute' in checkedPayload), 'Checked create payload must NOT include auto_spawn_execute')

  // AF-15: Goal create payload (默认 planOnly=false) → design/standard/auto_spawn_execute=true
  const goalPayload = buildAutomationCreatePayload(baseCreateForm, 'goal')
  assert(goalPayload.quest_type === 'design', 'Goal create must send quest_type=design')
  assert(goalPayload.intensity === 'standard', 'Goal create must send intensity=standard')
  assert(goalPayload.auto_spawn_execute === true, 'Goal create default must send auto_spawn_execute=true (planOnly=false means auto-spawn after design success)')

  // AF-16: Goal planOnly create payload → auto_spawn_execute=false
  const goalPlanOnlyPayload = buildAutomationCreatePayload({ ...baseCreateForm, planOnly: true }, 'goal')
  assert(goalPlanOnlyPayload.quest_type === 'design', 'Goal planOnly must still be design')
  assert(goalPlanOnlyPayload.auto_spawn_execute === false, 'Goal planOnly=true must send auto_spawn_execute=false')

  // AF-17: Goal quick intensity 不展示 planOnly，payload 不含 auto_spawn_execute
  // 注意：Goal 默认 intensity=standard，所以 canPlanOnlyToggle=true；这里通过 helper 直接验证
  // quick intensity 的 goal 在 flowPatch 里 intensity='standard'（patchFromAutomationFlow 固定），
  // 所以 Goal 档 canPlanOnlyToggle 始终 true。Direct/Checked 档不含此字段已在 AF-13/AF-14 验证。

  // AF-18: Automations save — 非 Goal 不含 auto_spawn_execute
  const baseUpdateForm = {
    name: 'updated',
    description: '',
    query: 'updated query',
    quest_type: 'execute',
    with_design_phase: false,
    intensity: 'standard',
    working_dir: '',
    workspace_mode: 'auto',
    trigger: 'manual',
    cron: '',
    auto_start: true,
    auto_apply: false,
    allow_l2: false,
    warrior_id: '',
    mage_id: '',
    connectors: [] as string[],
    priority: 0,
  }
  // AF-18: Automations save — 非 Goal 不含 auto_spawn_execute（previous=checked, current=checked, 无 stale）
  const checkedUpdate = buildAutomationUpdatePayload(baseUpdateForm, 'checked', 'checked', undefined, false, false, true)
  assert(!('auto_spawn_execute' in checkedUpdate), 'Non-Goal save must NOT include auto_spawn_execute')
  const directUpdate = buildAutomationUpdatePayload(baseUpdateForm, 'direct', 'direct', undefined, false, false, true)
  assert(!('auto_spawn_execute' in directUpdate), 'Direct save must NOT include auto_spawn_execute')

  // AF-19: Automations save — Goal 写入 auto_spawn_execute（previous=goal, current=goal, canPlanOnlyToggle）
  const goalUpdateForm = { ...baseUpdateForm, quest_type: 'design' as const, auto_spawn_execute: true }
  const goalUpdate = buildAutomationUpdatePayload(goalUpdateForm, 'goal', 'goal', false, true, false, true)
  assert(goalUpdate.auto_spawn_execute === true, 'Goal save must write auto_spawn_execute=true when form says so')

  // AF-20: Automations save — guarded automation 强制 auto_apply=false/allow_l2=false
  const guardedUpdate = buildAutomationUpdatePayload(
    { ...baseUpdateForm, auto_apply: true, allow_l2: true },
    'checked', 'checked', undefined, false, true, true,
  )
  assert(guardedUpdate.auto_apply === false, 'Guarded automation must force auto_apply=false')
  assert(guardedUpdate.allow_l2 === false, 'Guarded automation must force allow_l2=false')

  // AF-21: Automations save — 真正的 flow transition：previous=goal, current=direct/checked → 清 stale
  const goalToDirectUpdate = buildAutomationUpdatePayload(baseUpdateForm, 'direct', 'goal', true, false, false, true)
  assert(goalToDirectUpdate.auto_spawn_execute === false, 'Actual transition goal→direct must clear stale auto_spawn_execute')
  const goalToCheckedUpdate = buildAutomationUpdatePayload(baseUpdateForm, 'checked', 'goal', true, false, false, true)
  assert(goalToCheckedUpdate.auto_spawn_execute === false, 'Actual transition goal→checked must clear stale auto_spawn_execute')

  // AF-22: Automations save — 非 Goal 且无 flow transition，即使 stale=true 也不发此字段
  const staleButNoTransition = buildAutomationUpdatePayload(baseUpdateForm, 'checked', 'checked', true, false, false, true)
  assert(!('auto_spawn_execute' in staleButNoTransition), 'Non-Goal without goal→non-goal transition must NOT include auto_spawn_execute even if stale value exists')
  const directStaleNoTransition = buildAutomationUpdatePayload(baseUpdateForm, 'direct', 'direct', true, false, false, true)
  assert(!('auto_spawn_execute' in directStaleNoTransition), 'Direct without goal→direct transition must NOT include auto_spawn_execute even if stale value exists')

  // S3-7: Goal runtime skeleton — deriveGoalRuntimeView 真实展示决策
  // 7a: design quest + designDoc + implementation_steps → showPanel + steps 输出
  const designDocWithSteps = {
    quest_id: 'qst_gr1', schema_version: 'design_doc.v1',
    original_query: 'do thing', summary: 'plan summary',
    goals: ['g1'], non_goals: ['ng1'],
    implementation_steps: ['step 1', 'step 2', 'step 3'],
    risks: ['r1'], acceptance_criteria: ['ac1'],
    execute_prompt: 'execute this', created_at_ms: 1,
  }
  const designQuest = { type: 'design', status: 'success', child_execute_quest_id: undefined, spawn_error: undefined, auto_spawn_execute: false, pinned_outcome_summary: undefined } as any
  const view1 = deriveGoalRuntimeView(designDocWithSteps, designQuest, [], true)
  assert(view1.showPanel === true, 'design quest with designDoc must show panel')
  assert(view1.docReady === true, 'with designDoc must be docReady')
  assert(view1.steps.length === 3, 'implementation_steps must be exposed as steps')
  assert(view1.steps[0] === 'step 1', 'steps must preserve order')
  assert(view1.showSpawnButton === true, 'canSpawn=true must show spawn button')
  assert(view1.executeStatus === '可发起', 'canSpawn with no child must show 可发起')

  // 7b: 有 decision_note post → decisionRecorded=true
  const postsWithDecision = [{
    post_id: 'post_dec', thread_id: 'qst_gr1', root_post_id: 'root_qst_gr1',
    parent_reply_id: 'root_qst_gr1', author_role: 'system', kind: 'decision_note',
    content: 'outcome', created_at_ms: 1, artifact_refs: [],
  }]
  const view2 = deriveGoalRuntimeView(designDocWithSteps, designQuest, postsWithDecision, false)
  assert(view2.decisionRecorded === true, 'decision_note post must mark decisionRecorded')

  // 7c: 有 pinned_outcome_summary → decisionRecorded=true（DecisionNote 投影等价物）
  const questWithPinned = { ...designQuest, pinned_outcome_summary: 'Outcome: success' }
  const view3 = deriveGoalRuntimeView(designDocWithSteps, questWithPinned, [], false)
  assert(view3.decisionRecorded === true, 'pinned_outcome_summary must mark decisionRecorded')

  // 7d: 有 child execute quest → showOpenChildButton + executeStatus 含 ID
  const questWithChild = { ...designQuest, child_execute_quest_id: 'qst_child_abc123' }
  const view4 = deriveGoalRuntimeView(designDocWithSteps, questWithChild, [], false)
  assert(view4.showOpenChildButton === true, 'with child must show open button')
  assert(view4.showSpawnButton === false, 'with child must NOT show spawn button')
  assert(view4.executeStatus.includes('已创建'), 'with child must show 已创建 status')
  assert(view4.childExecuteQuestID === 'qst_child_abc123', 'childExecuteQuestID must be exposed')

  // 7e: spawn_error → showSpawnError
  const questWithError = { ...designQuest, spawn_error: 'network fail' }
  const view5 = deriveGoalRuntimeView(designDocWithSteps, questWithError, [], false)
  assert(view5.showSpawnError === true, 'spawn_error must show error flag')
  assert(view5.spawnError === 'network fail', 'spawnError must be exposed')
  assert(view5.executeStatus === '创建失败', 'spawn_error must show 创建失败 status')

  // 7f: auto_spawn_execute 且无 child → showAutoSpawnPlaceholder
  const questAutoSpawn = { ...designQuest, auto_spawn_execute: true, status: 'running' }
  const view6 = deriveGoalRuntimeView(designDocWithSteps, questAutoSpawn, [], false)
  assert(view6.showAutoSpawnPlaceholder === true, 'auto_spawn without child must show placeholder')
  assert(view6.showSpawnButton === false, 'not canSpawn must NOT show spawn button')

  // 7g: 非 design 且无 designDoc → showPanel=false
  const executeQuest = { type: 'execute', status: 'running', child_execute_quest_id: undefined, spawn_error: undefined, auto_spawn_execute: false, pinned_outcome_summary: undefined } as any
  const view7 = deriveGoalRuntimeView(null, executeQuest, [], false)
  assert(view7.showPanel === false, 'non-design without designDoc must NOT show panel')

  // 7h: 无 implementation_steps → steps 为空
  const designDocNoSteps = { ...designDocWithSteps, implementation_steps: [] }
  const view8 = deriveGoalRuntimeView(designDocNoSteps, designQuest, [], true)
  assert(view8.steps.length === 0, 'empty implementation_steps must yield empty steps')

  // S3-8: Goal runtime — helper 不输出假语义字段
  const viewKeys = Object.keys(view1)
  const fakeKeys = ['planReview', 'plan_review', 'sliceStatus', 'slice_status', 'approvePlan', 'decisionNote']
  for (const fake of fakeKeys) {
    assert(!viewKeys.includes(fake), `deriveGoalRuntimeView must NOT expose fake semantics: "${fake}"`)
  }
  // canSpawn 边界守护
  const designSuccessNoChild = questActionState({ type: 'design', status: 'success', child_execute_quest_id: undefined } as any)
  assert(designSuccessNoChild.canSpawn === true, 'design success without child must allow spawn-execute')
  const designSuccessHasChild = questActionState({ type: 'design', status: 'success', child_execute_quest_id: 'qst_child_123' } as any)
  assert(designSuccessHasChild.canSpawn === false, 'design with child execute quest must NOT allow spawn-execute')

  // ── Phase Authority: questActionState() phase runtime gating ──────────

  // PA-legacy: no pipeline fields → fallback to status-only, running canStop
  const legacyRunning = questActionState({ status: 'running' } as any)
  assert(legacyRunning.canStop === true, 'legacy running quest must allow stop (status-only fallback)')
  assert(legacyRunning.isRunning === true, 'legacy running quest must be isRunning')

  const legacyReviewing = questActionState({ status: 'reviewing' } as any)
  assert(legacyReviewing.canStop === true, 'legacy reviewing quest must allow stop')

  // PA-pipeline: pipeline available + current phase running → canStop
  const pipelineRunning = questActionState({
    status: 'running',
    pipeline_version: 1,
    phases: [{ status: 'running', phase_idx: 0 }, { status: 'pending', phase_idx: 1 }],
    current_phase_idx: 0,
  } as any)
  assert(pipelineRunning.canStop === true, 'pipeline + current phase running must allow stop')

  // PA-pipeline: pipeline available + current phase NOT running → cannot stop
  const pipelineIdlePhase = questActionState({
    status: 'running',
    pipeline_version: 1,
    phases: [{ status: 'done', phase_idx: 0 }, { status: 'pending', phase_idx: 1 }],
    current_phase_idx: 0,
  } as any)
  assert(pipelineIdlePhase.canStop === false, 'pipeline + current phase done must NOT allow stop (status stale)')

  // PA-pipeline-fallback: current_phase_idx missing but phase[1] is running → fallback finds it
  const pipelineMissingIdx = questActionState({
    status: 'running',
    pipeline_version: 1,
    phases: [{ status: 'done', phase_idx: 0 }, { status: 'running', phase_idx: 1 }],
    // current_phase_idx intentionally omitted
  } as any)
  assert(pipelineMissingIdx.canStop === true, 'pipeline + missing idx + running phase must allow stop (fallback)')

  // PA-ended-stale: ended quest with stale running phase → isRunning must be false
  const endedStaleRunning = questActionState({
    status: 'success',
    pipeline_version: 1,
    phases: [{ status: 'running', phase_idx: 0 }],
    current_phase_idx: 0,
  } as any)
  assert(endedStaleRunning.isRunning === false, 'ended quest with stale running phase must NOT be isRunning')
  assert(endedStaleRunning.canStop === false, 'ended quest with stale running phase must NOT allow stop')

  // PA-canStart: pending + first phase already running → cannot start
  const pendingFirstRunning = questActionState({
    status: 'pending',
    pipeline_version: 1,
    phases: [{ status: 'running', phase_idx: 0 }],
    current_phase_idx: 0,
  } as any)
  assert(pendingFirstRunning.canStart === false, 'pending quest with first phase running must NOT allow start')

  // PA-canStart: pending + no phase running → can start
  const pendingClean = questActionState({
    status: 'pending',
    pipeline_version: 1,
    phases: [{ status: 'pending', phase_idx: 0 }],
  } as any)
  assert(pendingClean.canStart === true, 'pending quest with no running phase must allow start')

  // ── P1a: actorMeta ──────────────────────────────────────────────────────
  const makerMeta = actorMeta('maker')!
  assert(makerMeta.label === 'Maker', 'actorMeta maker label')
  assert(makerMeta.icon === 'sword', 'actorMeta maker icon')
  assert(makerMeta.tone === 'maker', 'actorMeta maker tone')
  const checkerMeta = actorMeta('checker')!
  assert(checkerMeta.label === 'Checker', 'actorMeta checker label')
  assert(checkerMeta.icon === 'wand-sparkles', 'actorMeta checker icon')
  const unknownMeta = actorMeta('nonexistent_role')
  assert(unknownMeta === null, 'unknown role returns null — caller must handle fallback')
  const nullMeta = actorMeta(null)
  assert(nullMeta === null, 'null role returns null')

  // ── P1a: deriveQuestAttention — priority order ──────────────────────────
  // Priority: apply_failed > blocked > waiting_input > runtime_warning > review > none

  const applyFailedQuest = { status: 'running', apply_failed: true, apply_status: 'failed' } as any
  assert(deriveQuestAttention(applyFailedQuest).kind === 'apply_failed', 'apply_failed takes priority')
  assert(deriveQuestAttention(applyFailedQuest).priority === 1, 'apply_failed priority=1')

  const blockedQuest = { status: 'blocked', human_exception: { source_status: 'blocked', reason: 'stuck' } } as any
  assert(deriveQuestAttention(blockedQuest).kind === 'blocked', 'blocked detected')
  assert(deriveQuestAttention(blockedQuest).priority === 2, 'blocked priority=2')

  const waitingInputQuest = { status: 'waiting_input' } as any
  assert(deriveQuestAttention(waitingInputQuest).kind === 'waiting_input', 'waiting_input detected')
  assert(deriveQuestAttention(waitingInputQuest).priority === 3, 'waiting_input priority=3')

  const reviewQuest = { status: 'user_review', workspace_diff_pending: true } as any
  assert(deriveQuestAttention(reviewQuest).kind === 'review', 'review detected')
  assert(deriveQuestAttention(reviewQuest).priority === 5, 'review priority=5')

  const noneQuest = { status: 'running' } as any
  assert(deriveQuestAttention(noneQuest).kind === 'none', 'running quest has none attention')
  assert(deriveQuestAttention(noneQuest).priority === 999, 'none priority=999')

  // apply_failed beats blocked when both present
  const bothQuest = { status: 'blocked', apply_failed: true, apply_status: 'failed', human_exception: { source_status: 'blocked' } } as any
  assert(deriveQuestAttention(bothQuest).kind === 'apply_failed', 'apply_failed beats blocked')

  // blocked beats waiting_input
  const blockedWaiting = { status: 'waiting_input', human_exception: { source_status: 'blocked' } } as any
  assert(deriveQuestAttention(blockedWaiting).kind === 'blocked', 'blocked beats waiting_input via human_exception')

  // null quest returns none
  assert(deriveQuestAttention(null).kind === 'none', 'null quest returns none')

  // ── P1a: deriveQuestProgressLine — i18n keys, not text ──────────────────
  const progressApplyFailed = deriveQuestProgressLine({ status: 'running', apply_failed: true, apply_status: 'failed' } as any)
  assert(progressApplyFailed.i18nKey === 'quest.progress.applyFailed', 'apply_failed progress key')

  const progressBlocked = deriveQuestProgressLine({ status: 'blocked', human_exception: { source_status: 'blocked' } } as any)
  assert(progressBlocked.i18nKey === 'quest.progress.blocked', 'blocked progress key')

  const progressWaiting = deriveQuestProgressLine({ status: 'waiting_input' } as any)
  assert(progressWaiting.i18nKey === 'quest.progress.waitingInput', 'waiting_input progress key')

  const progressReview = deriveQuestProgressLine({ status: 'user_review', workspace_diff_pending: true } as any)
  assert(progressReview.i18nKey === 'quest.progress.waitingReview', 'review progress key')

  const progressReviewing = deriveQuestProgressLine({ status: 'reviewing' } as any)
  assert(progressReviewing.i18nKey === 'quest.progress.mageReviewing', 'reviewing progress key')

  const progressRunning = deriveQuestProgressLine({ status: 'running', intensity: 'standard' } as any)
  assert(progressRunning.i18nKey === 'quest.progress.warriorRunning', 'running progress key')

  const progressQuick = deriveQuestProgressLine({ status: 'running', intensity: 'quick' } as any)
  assert(progressQuick.i18nKey === 'quest.progress.warriorRunningQuick', 'quick running progress key')

  const progressQueued = deriveQuestProgressLine({ status: 'pending', queued: true, queue_position: 3 } as any)
  assert(progressQueued.i18nKey === 'quest.progress.queuedPosition', 'queued with position key')
  assert(progressQueued.params?.position === 3, 'queued position param')

  const progressSuccess = deriveQuestProgressLine({ status: 'success' } as any)
  assert(progressSuccess.i18nKey === 'quest.progress.success', 'success progress key')

  const progressNull = deriveQuestProgressLine(null)
  assert(progressNull.i18nKey === 'quest.progress.default', 'null quest returns default progress')

  // progress line must NOT return raw display text — only i18n keys
  assert(!progressRunning.i18nKey.includes('正在'), 'progress line must not contain Chinese text')
  assert(!progressRunning.i18nKey.includes('运行'), 'progress line must not contain Chinese text')

  // P1a fix: progress must align with attention for human_exception blocked
  const humanBlockedWaitingInput = {
    status: 'waiting_input',
    human_exception: { source_status: 'blocked', reason: 'stuck' },
  } as any
  assert(
    deriveQuestAttention(humanBlockedWaitingInput).kind === 'blocked',
    'attention: human_exception blocked detected even when status=waiting_input',
  )
  assert(
    deriveQuestProgressLine(humanBlockedWaitingInput).i18nKey === 'quest.progress.blocked',
    'progress: human_exception blocked must yield blocked key, not waitingInput — must align with attention',
  )

  const humanBlockedRunning = {
    status: 'running',
    human_exception: { source_status: 'blocked', reason: 'stuck' },
  } as any
  assert(
    deriveQuestAttention(humanBlockedRunning).kind === 'blocked',
    'attention: human_exception blocked detected even when status=running',
  )
  assert(
    deriveQuestProgressLine(humanBlockedRunning).i18nKey === 'quest.progress.blocked',
    'progress: human_exception blocked with status=running must yield blocked key',
  )

  // P1a fix: actorMeta returns null for unknown roles
  assert(actorMeta('some_future_role') === null, 'actorMeta unknown role returns null — caller must handle fallback')
  assert(actorMeta(null) === null, 'actorMeta null returns null')
  assert(actorMeta(undefined) === null, 'actorMeta undefined returns null')
  assert(actorMeta('maker') !== null, 'actorMeta known role returns non-null')

  // FeedCard: unknown author_role should show warrior_name, not raw role string
  const unknownRoleItem = {
    id: 'evt_unknown_role',
    quest_id: 'qst_unknown',
    short_id: 'unk',
    kind: 'agent',
    author_role: 'some_future_role',
    warrior_name: 'Agent Alice',
    warrior_class: 'warrior',
    query: 'test query',
    summary: 'test summary',
    ts: Date.now(),
  } as any
  const unknownRoleMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: unknownRoleItem,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(
    unknownRoleMarkup.includes('Agent Alice'),
    'FeedCard unknown role must display warrior_name as author, not raw role string',
  )
  assert(
    !unknownRoleMarkup.includes('>some_future_role<'),
    'FeedCard must NOT expose raw unknown role string as author name',
  )

  // FeedCard: candidate tag is interactive when onOpenInbox provided
  const candidateWithInbox = {
    id: 'evt_cand_inbox',
    quest_id: 'qst_cand_inbox',
    short_id: 'cand2',
    kind: 'user',
    query: 'candidate with inbox handler',
    summary: 'candidate body',
    triage_mode: 'candidate',
    status: 'blocked',
    ts: Date.now(),
  } as any
  let inboxOpened = false
  const candWithInboxMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: candidateWithInbox,
    onOpen: () => {},
    onOpenAdventurer: () => {},
    onOpenInbox: () => { inboxOpened = true },
  }))
  assert(
    candWithInboxMarkup.includes('feed-candidate-tag'),
    'candidate with onOpenInbox renders candidate tag',
  )
  assert(
    !candWithInboxMarkup.includes('feed-candidate-btn'),
    'candidate must NOT render confirm/cancel buttons',
  )

  // candidate without onOpenInbox: tag is non-interactive (em, not button)
  const candWithoutInboxMarkup = renderToStaticMarkup(createElement(FeedCard, {
    item: candidateWithInbox,
    onOpen: () => {},
    onOpenAdventurer: () => {},
  }))
  assert(
    candWithoutInboxMarkup.includes('feed-candidate-tag'),
    'candidate without onOpenInbox still renders tag',
  )

  // ── P3: Command Palette data source contract ────────────────────────────
  const commandRuns: string[] = []
  const commandItems = buildCommandItems({
    quests: [
      {
        id: 'qst_attention',
        short_id: 'attn',
        query: 'Fix production checkout',
        type: 'execute',
        status: 'blocked',
        rework_count: 0,
        max_rework: 1,
        created_by: 'user',
        created_at_ms: Date.now() - 120_000,
        updated_at_ms: Date.now() - 60_000,
        base_working_dir: '/repo/gloop',
      } as any,
      {
        id: 'qst_normal',
        short_id: 'norm',
        query: 'Write routine notes',
        type: 'execute',
        status: 'running',
        rework_count: 0,
        max_rework: 1,
        created_by: 'user',
        created_at_ms: Date.now() - 240_000,
      } as any,
    ],
    inbox: [
      {
        id: 'qst_candidate_palette',
        short_id: 'candp',
        query: 'Confirm automation candidate',
        type: 'execute',
        status: 'pending',
        rework_count: 0,
        max_rework: 1,
        created_by: 'automation:auto_context_refresh',
        created_at_ms: Date.now(),
      } as any,
    ],
    humanExceptions: [
      {
        id: 'hex_attention',
        quest_id: 'qst_attention',
        reason: 'Agent auth expired',
        recommended_action: 'Login and resume.',
        risk_level: 'high',
        priority: 10,
        available_actions: ['continue'],
        audit_ref: '/quests/qst_attention',
        source_status: 'blocked',
        created_at_ms: Date.now(),
      },
    ],
    automations: [
      { id: 'auto_context_refresh', name: 'Context refresh', query: 'refresh context', enabled: true, trigger: 'schedule' } as any,
    ],
    adventurers: [
      { id: 'adv_mage', name: 'Strict Mage', class: 'mage', status: 'active', agent: 'traex', level: 3, exp: 10, win_count: 2, lose_count: 1, created_at_ms: 1 } as any,
    ],
    skills: [
      { name: 'gloop-quest-review', description: 'Review quest output', category: 'review' },
    ],
    prompts: [
      { name: 'roles/mage.md', source: 'builtin', size_bytes: 120, preview: 'Mage prompt', is_overridden: false },
    ],
    actions: {
      navigate: (tab) => commandRuns.push('navigate:' + tab),
      createQuest: () => commandRuns.push('create:quest'),
      createAutomation: () => commandRuns.push('create:automation'),
      createAdventurer: () => commandRuns.push('create:adventurer'),
      openQuest: (qid) => commandRuns.push('open:quest:' + qid),
      openAutomation: (id) => commandRuns.push('open:automation:' + id),
      openAdventurer: (id) => commandRuns.push('open:adventurer:' + id),
      openSkill: (name) => commandRuns.push('open:skill:' + name),
      openPrompt: (name) => commandRuns.push('open:prompt:' + name),
    },
  })

  const byId = new Map(commandItems.map((item) => [item.id, item]))
  assert(byId.get('open:inbox:hex_attention')?.priority === COMMAND_PRIORITY.decideAttention, 'human exception command must be top-priority Decide')
  assert(byId.get('open:inbox:hex_attention')?.section === 'Decide', 'human exception command belongs to Decide section')
  assert(byId.get('open:quest:qst_attention')?.priority === COMMAND_PRIORITY.openExact, 'attention quest command must use openExact priority')
  assert(byId.get('open:quest:qst_normal')?.priority === COMMAND_PRIORITY.openNormal, 'normal quest command must not outrank create/navigate')
  assert(byId.get('open:resource:skill:gloop-quest-review')?.priority === COMMAND_PRIORITY.inspectResource, 'skill command must use inspectResource priority')
  assert(byId.get('open:resource:prompt:roles/mage.md')?.section === 'Inspect', 'prompt command belongs to Inspect section')
  assert(byId.get('create:quest')?.title.startsWith('新建'), 'create command title must start with a verb')
  assert(byId.get('open:automation:auto_context_refresh')?.subtitle?.includes('已启用'), 'automation command subtitle must carry entity metadata')
  byId.get('open:inbox:hex_attention')?.run()
  byId.get('open:quest:qst_attention')?.run()
  byId.get('open:automation:auto_context_refresh')?.run()
  byId.get('open:resource:skill:gloop-quest-review')?.run()
  assert(commandRuns.includes('navigate:inbox'), 'inbox command must navigate to Inbox primary surface')
  assert(commandRuns.includes('open:quest:qst_attention'), 'quest command must open quest detail')
  assert(commandRuns.includes('open:automation:auto_context_refresh'), 'automation command must select automation surface')
  assert(commandRuns.includes('open:skill:gloop-quest-review'), 'skill command must select resource surface')

  const orderedCommands = orderCommandItemsForPalette([
    commandStub('open:normal', '打开设置相关委托', COMMAND_PRIORITY.openNormal, 'Open'),
    commandStub('create:quest:test', '新建设置委托', COMMAND_PRIORITY.create, 'Create'),
    commandStub('navigate:settings:test', '打开设置', COMMAND_PRIORITY.navigatePrimary, 'Navigate'),
  ], '设置')
  assert(
    orderedCommands.map((item) => item.id).join(',') === 'create:quest:test,navigate:settings:test,open:normal',
    'palette ordering must keep global priority before text score or section grouping',
  )
}

function commandStub(id: string, title: string, priority: number, section: CommandItem['section']): CommandItem {
  return {
    id,
    kind: id.startsWith('create:') ? 'create' : id.startsWith('navigate:') ? 'navigate' : 'open',
    entity: 'quest',
    intent: 'monitor',
    section,
    title,
    priority,
    run: () => {},
  }
}

run()
console.log('quest-display-semantics tests: PASSED')
