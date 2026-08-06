// scripts/l3-fusion/metric-narrative.ts
//
// Hard-coded Chinese narratives for metric-based findings. These short-circuit
// the LLM fuser because the evidence IS the metric: there's nothing for the
// LLM to "judge" about a 938-line identical overlap or a fan-in of 27.

import type { Finding, Severity } from '../types.js';
import type { Group } from './grouping.js';

/** Returns true if every violation in the group carries the metric hardness flag. */
export function isMetricHardEvidence(group: Group): boolean {
  if (group.violations.length === 0) return false;
  return group.violations.every((v) => (v.evidence.signals ?? []).includes('metric-hard-evidence'));
}

function firstSignalValue(group: Group, name: string): string | undefined {
  for (const v of group.violations) {
    for (const s of v.evidence.signals ?? []) {
      if (s.startsWith(name + '=')) return s.slice(name.length + 1);
    }
  }
  return undefined;
}

function worstSeverity(group: Group): Severity {
  const order: Severity[] = ['critical', 'high', 'medium', 'low'];
  return group.violations.reduce<Severity>(
    (best, v) => (order.indexOf(v.severity) < order.indexOf(best) ? v.severity : best),
    'low',
  );
}

interface Narrative {
  title: string;
  rootCause: string;
  impact: string;
  actions: string[];
}

function buildNarrative(group: Group): Narrative {
  const repr = group.violations[0];
  const ruleId = repr.ruleId;

  if (ruleId === 'go-arch:code-clone') {
    const overlap = firstSignalValue(group, 'overlap-lines') ?? '?';
    const ratio = firstSignalValue(group, 'overlap-ratio') ?? '?';
    const files = group.violations.flatMap((v) => v.locations.map((l) => l.file));
    const uniq = [...new Set(files)];
    return {
      title: `跨包代码克隆：${uniq.slice(0, 2).map((f) => '`' + f + '`').join(' 与 ')} 存在大段重复`,
      rootCause:
        `两个文件有 **${overlap} 行**完全相同的非注释代码，占较小文件的 **${Math.round(Number(ratio) * 100)}%**。` +
        `这是典型的"先复制再演化"信号：两个本应互相独立的模块共享了一段核心逻辑，但实现是复制-粘贴而非提取到公共包。` +
        `修改一处必须同步修改另一处，否则两边行为开始漂移；随着时间推移两份代码一定会分叉，形成难以追溯的 bug 温床。`,
      impact:
        `任意一个改动都需要在两处（甚至更多处）同步实现；" 测不全"风险高。业务边界在代码层被稀释——` +
        `表面上 \`user_creation\` / \`community_creation\` 是兄弟模块，实际上共用内核逻辑。`,
      actions: [
        `把重复代码抽到独立的业务内核包（例如 \`biz/service/creation_content/\` 或 \`biz/model/entity/creation_content/\`）`,
        `让原有两个模块调用新包，而不是各自持有实现`,
        `写一遍单元测试覆盖新包，然后逐步迁移两边的调用`,
        `在 CI 加上 CPD/duplicate-code 门禁（\`cpd --minimum-tokens=100\` 之类），阻止后续克隆蔓延`,
      ],
    };
  }

  if (ruleId === 'go-arch:cycle') {
    const size = firstSignalValue(group, 'cycle-size') ?? String(group.violations[0].evidence.metric?.value ?? '?');
    const members = firstSignalValue(group, 'cycle-members') ?? 'unknown';
    return {
      title: `包级循环依赖：${size} 个包形成互相引用环`,
      rootCause:
        `以下包形成强连通分量（相互直接或间接 import）：${members}。` +
        `Go 编译器允许同一目录下的多文件互相引用，但 **跨目录包的循环依赖在 Go 里是硬错**，只要环成立就必须手工拆解。` +
        `通常根因是层次混乱——在应该是下层的包里引用了上层的类型或函数。`,
      impact: `任何对环中某个包的大改都会波及整个环；单独测试、单独部署几乎不可能；新人理解成本陡增。`,
      actions: [
        `画出环中每条 import 边的具体调用点（\`grep -rn <被依赖包名> <依赖包>\`）`,
        `识别"不该存在的那条边"，把它变成接口注入或事件通信`,
        `把公共数据结构下沉到一个更基础的包`,
        `在 CI 加 \`go list\` 的 SCC 检查，一旦出现新的包级环就阻断\``,
      ],
    };
  }

  if (ruleId === 'go-arch:god-package') {
    const fanIn = firstSignalValue(group, 'fan-in') ?? '?';
    const fanOut = firstSignalValue(group, 'fan-out') ?? '?';
    const I = firstSignalValue(group, 'instability') ?? '?';
    const loc = group.violations[0].locations[0]?.file ?? '?';
    return {
      title: `God Package：\`${loc}\` 同时承担了"共享内核"和"业务模块"两个角色`,
      rootCause:
        `该包被 **${fanIn}** 个其它包依赖（fan-in 高），却自己也依赖 **${fanOut}** 个包（fan-out 高），` +
        `不稳定度 I=${I}（>0.5 即"中间层杂物包"）。Robert Martin 的稳定抽象原则要求：` +
        `被多人依赖的包应尽量"稳定"（I 接近 0，抽象多于具体）；而 I > 0.5 表示该包正在积极地依赖具体实现、` +
        `同时又被当基础设施消费——这是"上帝包"的结构性特征。`,
      impact:
        `任何一点改动会波及 ${fanIn} 个下游；但它自己又会被它依赖的 ${fanOut} 个包变化所波及。` +
        `重构风险高、测试覆盖难、事实上的"谁都不敢动"地带。`,
      actions: [
        `把被外部依赖的"稳定内核"部分抽取成独立小包（纯数据 / 纯接口），暴露给下游`,
        `把依赖其它业务包的"业务编排"部分下沉到一个新的 orchestrator/service`,
        `逐步把下游迁移到新的纯内核包；删除旧包对外的门面`,
        `在 CI 加 fan-in / fan-out 阈值检查，新增类似包直接告警`,
      ],
    };
  }

  if (ruleId.startsWith('go-arch:layering:')) {
    const srcLayer = firstSignalValue(group, 'src-layer');
    const depLayer = firstSignalValue(group, 'dep-layer');
    const convention = firstSignalValue(group, 'convention') ?? '(auto-detected)';
    const edges = group.violations.map((v) => {
      const a = (v.evidence.signals ?? []).find((s) => s.startsWith('src-pkg='))?.slice(8) ?? v.locations[0]?.file ?? '?';
      const b = (v.evidence.signals ?? []).find((s) => s.startsWith('dep-pkg='))?.slice(8) ?? '?';
      return `\`${a}\` → \`${b}\``;
    });
    const uniqEdges = [...new Set(edges)];
    const edgeList = uniqEdges.length > 8
      ? uniqEdges.slice(0, 8).join('; ') + `；（共 ${uniqEdges.length} 条违规边，已截断）`
      : uniqEdges.join('; ');
    return {
      title: `分层违规：${srcLayer} 层包越权引用 ${depLayer} 层（共 ${uniqEdges.length} 条边）`,
      rootCause:
        `在本仓库识别到的分层约定 **${convention}** 中，` +
        `${srcLayer} 层（位于内部/下层）的包反向依赖了 ${depLayer} 层（位于外部/上层）。` +
        `具体违规边：${edgeList}。` +
        `典型原因：下层为了图方便直接调用上层的工具函数、或把业务逻辑搬来复用——短期省事，长期让层边界名存实亡。`,
      impact:
        `层次约定失效后，任何重构都会穿透全栈；单元测试在层级断开时难以注入 mock；` +
        `领域模型被 transport/handler 类型污染；想把某层单独抽成服务变得几乎不可能。`,
      actions: [
        `对每条违规边，先判断是"真的需要反向依赖"还是"搬错了位置"`,
        `若是被复用的工具/常量：下沉到更基础的 \`util\`/\`consts\` 包`,
        `若是被复用的业务逻辑：抽成 service 层接口，由上层通过依赖注入使用`,
        `若违规是某个 \`init.go\` 的启动装配需要，建议抽出独立的 \`cmd/\` 或 \`internal/wire\` 装配包`,
        `修复后让 CI 持续守卫（已修复 arch-go YAML 拼写，也可用本 skill 内置的 go-arch 检查）`,
      ],
    };
  }

  if (ruleId === 'go-arch:horizontal-coupling') {
    const parent = firstSignalValue(group, 'parent') ?? '?';
    const sibling = firstSignalValue(group, 'sibling') ?? '?';
    const targets = firstSignalValue(group, 'depends-on-siblings') ?? '?';
    return {
      title: `业务模块边界被打穿：\`${parent}/${sibling}\` 直接引用了兄弟模块 ${targets}`,
      rootCause:
        `在 \`${parent}/\` 层下，\`${sibling}\` 模块和 \`${targets}\` 模块本应是平行的业务单元，` +
        `但 \`${sibling}\` 直接在代码中引用了 \`${targets}\` 的内部包。这意味着兄弟模块之间没有通过抽象接口或事件通信，` +
        `而是耦合在了具体实现上——业务边界形同虚设。`,
      impact:
        `任何对 \`${targets}\` 的内部重构都会波及 \`${sibling}\`；两个本应独立演化的业务被绑在一起；` +
        `未来想把某个业务模块独立成服务时，将遇到无法切分的代码网。`,
      actions: [
        `识别跨模块调用的具体方法，判断是否能"拉到上层由 facade 编排"`,
        `若确实需要共用：抽成 \`biz/service/<shared>\` 子模块，两边都依赖它而不是彼此`,
        `在业务模块之间用领域事件替代直接调用（解耦选项）`,
        `在 CI 加 sibling-import 检查，禁止 \`biz/service/A\` 直接 import \`biz/service/B\``,
      ],
    };
  }

  if (ruleId === 'goda:high-fan-in' || ruleId === 'high-fan-in') {
    const n = group.violations[0].evidence.metric?.value ?? '?';
    const th = group.violations[0].evidence.metric?.threshold ?? 20;
    const loc = group.violations[0].locations[0]?.file ?? '?';
    return {
      title: `高扇入包：\`${loc}\` 被 ${n} 个包依赖（超过阈值 ${th}）`,
      rootCause:
        `该包是事实上的"共享内核"。本身是否合理要看其内容——如果是纯常量/纯数据/纯接口就没问题；` +
        `但一旦里面混入业务逻辑，它就从"稳定抽象"滑向"上帝包"，变成全局耦合点。`,
      impact: `任何变动都会牵动 ${n} 个下游；发版/测试成本随 fan-in 线性放大。`,
      actions: [
        `检查该包内部是否有混杂的职责（工具函数 + 业务逻辑 + 类型定义）`,
        `如有，按职责拆成多个稳定小包`,
        `为该包冻结 API surface，任何新增公开函数走 code review 门禁`,
      ],
    };
  }

  // Generic metric-hardness fallback: still worth showing, but without the
  // curated narrative. Keeps unknown future hard-metric rules from degrading
  // silently.
  return {
    title: `硬指标发现：${repr.message}`,
    rootCause: `该发现基于确定性指标（${repr.evidence.metric?.name ?? 'metric'}=${repr.evidence.metric?.value ?? '?'}, 阈值 ${repr.evidence.metric?.threshold ?? '?'}）。`,
    impact: '指标超出阈值，需结合业务判断是否可接受。',
    actions: [
      `审阅 \`${repr.locations[0]?.file ?? '?'}\` 的具体内容`,
      '判断是否可以降低该指标（拆分、抽取、重构）',
    ],
  };
}

export function renderMetricHardEvidenceFinding(group: Group): Finding {
  const n = buildNarrative(group);
  const locations = [...new Map(group.violations.flatMap((v) => v.locations).map((l) => [l.file + (l.line ?? ''), l])).values()];
  const severity = worstSeverity(group);
  // Hard-evidence findings always get high confidence — metric is the proof.
  const confidenceScore = 0.95;
  return {
    id: group.id,
    category: group.category,
    title: n.title,
    rootCause: n.rootCause,
    impact: n.impact,
    actions: n.actions,
    confidence: 'high',
    confidenceScore,
    severity,
    signals: group.violations.map((v) => ({ source: v.source, description: v.message || v.ruleId })),
    locations,
    violationIds: group.violations.map((v) => v.id),
  };
}
