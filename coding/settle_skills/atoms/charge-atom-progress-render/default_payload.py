# -*- coding: utf-8 -*-
"""默认骨架:9 个 stage(s0~s8)的固定流水。
charge-atom-progress-render 在 init 时使用本骨架,避免调用方每次重复传。
patch 阶段只需要传"这次变了什么",原子内部按 id 找到对应 stage 合并。

⚠️ 本骨架与 auto-develop SKILL v6 的 8 个生命周期节点保持对齐:
  s0  启动                  init 即 done
  s1  PRD 校验              校验飞书 PRD 链接 + 标题/摘要
  s2  仓库路由              charge-atom-repo-router 命中
  s3  子流程标签            complexity / decisions / open_questions
  s4  知识检索              基线知识 + insearch 召回
  s5  编码 + push           Coco sandbox + 本地编译门禁 + push
  s6  BITS 自检流水线       dev_id / pipeline_url / loop_count
  s7  MR 合入               自动化测试 & 合入
  s8  变更报告              最终飞书报告 + MR url

详见: bytepay/settle_skill : feature/ai_native 下的 SKILL.md
"""

DEFAULT_STEPS = [
    {"id": "s0", "name": "启动",              "sub": "鉴权 & 看板初始化",        "status": "pend"},
    {"id": "s1", "name": "PRD 校验",          "sub": "飞书 wiki/docs",          "status": "pend"},
    {"id": "s2", "name": "仓库路由",          "sub": "L1/L2 关键词",            "status": "pend"},
    {"id": "s3", "name": "子流程标签",        "sub": "complexity & decisions",   "status": "pend"},
    {"id": "s4", "name": "知识检索",          "sub": "基线 + insearch",         "status": "pend"},
    {"id": "s5", "name": "编码 + push",       "sub": "Coco sandbox · 编译门禁",  "status": "pend"},
    {"id": "s6", "name": "BITS 自检流水线",   "sub": "等待触发",                 "status": "pend"},
    {"id": "s7", "name": "MR 合入",           "sub": "自动化测试 & 合入",        "status": "pend"},
    {"id": "s8", "name": "变更报告",          "sub": "飞书最终报告",             "status": "pend"},
]

DEFAULT_HERO = [
    {"k": "总耗时",     "v": "0",   "unit": "min",        "fill": 0.0, "tone": "default"},
    {"k": "变更文件",   "v": "0",   "unit": "files",      "fill": 0.0, "tone": "default"},
    {"k": "回环次数",   "v": "0",   "unit": "loops",      "fill": 0.0, "tone": "default"},
    {"k": "流水线状态", "v": "—",   "unit": "pending",    "fill": 0.0, "tone": "default"},
]

DEFAULT_STAGES = [
    {
        "id": "s0",
        "title": "启动",
        "subtitle": "auto-develop SKILL · 鉴权握手 & 看板初始化",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s1",
        "title": "PRD 校验",
        "subtitle": "飞书 wiki/docs · lark-wiki & lark-doc SKILL",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s2",
        "title": "仓库路由",
        "subtitle": "charge-atom-repo-router · L1/L2 关键词命中",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s3",
        "title": "子流程标签",
        "subtitle": "复杂度 / 决策项 / 待澄清问题",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s4",
        "title": "知识检索",
        "subtitle": "基线白皮书 + bytedance-insearch",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s5",
        "title": "编码 + push",
        "subtitle": "bytedance-coco · sandbox 自主编码 + 本地编译门禁 + push 远端",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s6",
        "title": "BITS 自检流水线",
        "subtitle": "charge-atom-bits-pipeline · 自检 & 自动回环 ≤3 轮",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s7",
        "title": "MR 合入",
        "subtitle": "等待流水线 PASS / manual gate 后合入主干",
        "status": "pend",
        "panels": [],
    },
    {
        "id": "s8",
        "title": "变更报告",
        "subtitle": "report 原子 · 飞书最终报告 + MR url",
        "status": "pend",
        "panels": [],
    },
]


def build_initial_snapshot(task_meta):
    """构造首次 init 的快照。task_meta 必须包含 title/prd_url/repo/branch/owner/session_id/started_at/updated_at。"""
    return {
        "task": dict(task_meta),
        "hero": [dict(x) for x in DEFAULT_HERO],
        "steps": [dict(x) for x in DEFAULT_STEPS],
        "stages": [dict(x, panels=list(x.get("panels", []))) for x in DEFAULT_STAGES],
    }
