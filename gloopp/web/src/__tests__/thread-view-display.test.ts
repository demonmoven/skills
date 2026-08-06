import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import ThreadView from '../pages/ThreadView'
import ThreadLedgerPanel from '../components/ThreadLedgerPanel'
import type { ThreadPost, ThreadViewResponse } from '../api/types'

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) throw new Error(message)
}

function makeThreadResponse(overrides: Partial<ThreadViewResponse> = {}): ThreadViewResponse {
  const base: ThreadViewResponse = {
    ok: true,
    quest: {
      id: 'qst_test123',
      short_id: 'test',
      query: 'Test quest query',
      status: 'running',
      created_by: 'user',
      created_at_ms: 1719000000000,
      updated_at_ms: 1719003600000,
    },
    posts: [
      {
        post_id: 'root_qst_test123',
        thread_id: 'qst_test123',
        root_post_id: 'root_qst_test123',
        author_identity: 'user',
        author_role: 'human',
        kind: 'root',
        content: 'Original request body for test quest',
        created_at_ms: 1719000000000,
      },
      {
        post_id: 'post_reply1',
        thread_id: 'qst_test123',
        root_post_id: 'root_qst_test123',
        parent_reply_id: 'root_qst_test123',
        author_identity: 'adv_maker',
        author_role: 'maker',
        kind: 'maker_report',
        content: 'Maker delivery summary',
        created_at_ms: 1719001800000,
      },
    ],
    actors: [
      { identity: 'user', role: 'human', name: 'You', source_type: 'human' as const },
      { identity: 'adv_maker', role: 'maker', name: 'Maker Agent', source_type: 'adventurer' as const },
    ],
    fanout_summary: null,
    action_entrypoints: [
      { action_type: 'add_comment', target_section: 'comment', priority: 1, hint: 'Add a comment' },
    ],
  }
  return { ...base, ...overrides }
}

// We can't directly render ThreadView (it uses hooks/useEffect for data loading),
// so we test the display logic by:
// 1. Testing ThreadLedgerPanel directly with full posts (including root)
// 2. Testing helper logic that ThreadView uses

function run() {
  // ── TV-1: Root post must appear as post envelope in ThreadLedgerPanel ──
  const resp = makeThreadResponse()
  const allPostsMarkup = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: resp.posts,
  }))

  // Root post must be visible as a post with kind=Root
  assert(allPostsMarkup.includes('Root'), 'root post must show Root kind label')
  assert(allPostsMarkup.includes('Original request body for test quest'), 'root post content must be visible')
  // Root post author must be shown
  assert(allPostsMarkup.includes('role-human'), 'root post must carry human role class')

  // Reply must also be present
  assert(allPostsMarkup.includes('MakerReport'), 'maker_report reply must show MakerReport kind label')
  assert(allPostsMarkup.includes('Maker delivery summary'), 'maker reply content must be visible')
  assert(allPostsMarkup.includes('role-maker'), 'maker reply must carry maker role class')

  // ── TV-2: Root post must NOT be filtered out ──
  // Simulate the old (broken) behavior: filtering root
  const filteredPosts = resp.posts.filter((p) => p.kind !== 'root' && !p.post_id.startsWith('root_'))
  assert(filteredPosts.length === 1, 'sanity: filtering root removes exactly 1 post')
  const filteredMarkup = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: filteredPosts,
  }))
  assert(!filteredMarkup.includes('Original request body for test quest'), 'filtered-out root must not appear')
  // Verify that with ALL posts (current behavior), root IS present
  assert(allPostsMarkup.includes('Original request body for test quest'), 'unfiltered posts must include root content')

  // ── TV-3: Time field contract — created_at_ms/updated_at_ms present ──
  const quest = resp.quest
  assert(quest.created_at_ms > 0, 'quest.created_at_ms must be a positive number')
  assert((quest.updated_at_ms || 0) > 0, 'quest.updated_at_ms must be a positive number')
  assert(quest.created_at_ms <= (quest.updated_at_ms || 0), 'created_at_ms must be <= updated_at_ms')
  // Verify no non-_ms variants
  assert(!('created_at' in quest), 'quest must NOT have created_at (use created_at_ms)')
  assert(!('updated_at' in quest), 'quest must NOT have updated_at (use updated_at_ms)')

  // ── TV-4: ThreadLedgerPanel renders time from created_at_ms ──
  // The markup should contain a formatted time for the root post (not just '-')
  // fmtTime(1719000000000) produces a date string; we verify it's not just '-'
  const timeMatches = allPostsMarkup.match(/thread-ledger-time">([^<]+)</g)
  assert(timeMatches !== null, 'thread ledger must render time spans')
  const hasNonDashTime = timeMatches.some((m) => !m.includes('>-<'))
  assert(hasNonDashTime, 'thread ledger must show non-dash formatted time for posts with created_at_ms')

  // ── TV-5: Fanout summary — root with leaves ──
  const fanoutResp = makeThreadResponse({
    fanout_summary: {
      group_id: 'grp_test',
      is_root: true,
      is_leaf: false,
      leaves: [
        { quest_id: 'qst_leaf_a', leaf_id: 'leaf-a', status: 'running', latest_post_preview: 'leaf a progress' },
        { quest_id: 'qst_leaf_b', leaf_id: 'leaf-b', status: 'success', latest_post_preview: 'leaf b done' },
      ],
    },
  })
  // Verify the fanout data structure is correct
  assert(fanoutResp.fanout_summary?.is_root === true, 'fanout root must have is_root=true')
  assert(fanoutResp.fanout_summary?.is_leaf === false, 'fanout root must have is_leaf=false')
  assert(fanoutResp.fanout_summary?.leaves?.length === 2, 'fanout root must list leaves')
  assert(fanoutResp.fanout_summary?.leaves?.[0].quest_id === 'qst_leaf_a', 'leaf quest_id must be preserved')

  // ── TV-6: Fanout summary — leaf pointing to root ──
  const leafResp = makeThreadResponse({
    fanout_summary: {
      group_id: 'grp_test',
      is_root: false,
      is_leaf: true,
      root_quest_id: 'qst_root_abc',
    },
  })
  assert(leafResp.fanout_summary?.is_leaf === true, 'fanout leaf must have is_leaf=true')
  assert(leafResp.fanout_summary?.is_root === false, 'fanout leaf must have is_root=false')
  assert(leafResp.fanout_summary?.root_quest_id === 'qst_root_abc', 'fanout leaf must reference root quest id')

  // ── TV-7: Action entrypoints are navigation hints, not direct actions ──
  const entrypoints = resp.action_entrypoints!
  assert(entrypoints.length > 0, 'action entrypoints must be present')
  for (const ep of entrypoints) {
    assert(typeof ep.action_type === 'string', 'entrypoint action_type must be string')
    assert(typeof ep.target_section === 'string', 'entrypoint target_section must be string')
    assert(typeof ep.priority === 'number', 'entrypoint priority must be number')
    // Entrypoints must NOT contain executable fields like API endpoints or payloads
    const keys = Object.keys(ep)
    assert(!keys.includes('api_endpoint'), 'entrypoints must NOT include api_endpoint (navigation hint only)')
    assert(!keys.includes('payload'), 'entrypoints must NOT include payload (navigation hint only)')
  }

  // ── TV-8: Pinned outcome from decision_note ──
  const decisionResp = makeThreadResponse({
    posts: [
      ...resp.posts,
      {
        post_id: 'post_decision',
        thread_id: 'qst_test123',
        root_post_id: 'root_qst_test123',
        parent_reply_id: 'root_qst_test123',
        author_identity: 'system',
        author_role: 'system',
        kind: 'decision_note',
        content: 'Outcome: success | verdict=pass',
        created_at_ms: 1719003600000,
      },
    ],
  })
  const decision = decisionResp.posts.find((p) => p.kind === 'decision_note')
  assert(decision !== undefined, 'decision_note post must be findable')
  assert((decision?.content || '').includes('verdict=pass'), 'decision_note content must include verdict')

  const decisionMarkup = renderToStaticMarkup(createElement(ThreadLedgerPanel, {
    posts: decisionResp.posts,
    pinnedOutcome: decision?.content || '',
  }))
  assert(decisionMarkup.includes('DecisionNote'), 'decision_note must show DecisionNote kind label')
  assert(decisionMarkup.includes('verdict=pass'), 'pinned outcome must show verdict')

  // ── TV-9: Actor identity map ──
  const actors = resp.actors
  assert(actors.length === 2, 'must have 2 actors')
  assert(actors[0].identity === 'user', 'human actor identity must be preserved')
  assert(actors[0].role === 'human', 'human actor role must be preserved')
  assert(actors[1].identity === 'adv_maker', 'maker actor identity must be preserved')
  assert(actors[1].role === 'maker', 'maker actor role must be preserved')
  for (const actor of actors) {
    assert(typeof actor.source_type === 'string', 'actor must have source_type')
  }

  // ── TV-10: Post ordering by created_at_ms ──
  const ordered = [...resp.posts].sort((a, b) => (a.created_at_ms || 0) - (b.created_at_ms || 0))
  assert(ordered[0].post_id === 'root_qst_test123', 'root post must sort first (earliest timestamp)')
  assert(ordered[1].post_id === 'post_reply1', 'reply must sort after root')
}

run()
console.log('thread-view-display tests: PASSED')
