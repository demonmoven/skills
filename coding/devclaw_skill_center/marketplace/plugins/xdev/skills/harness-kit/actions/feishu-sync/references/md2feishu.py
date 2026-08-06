"""
Markdown → 飞书文档转换器
用法: FEISHU_APP_ID=xxx FEISHU_APP_SECRET=xxx python3 md2feishu.py <markdown_file> [--grant <open_id>] [--email <email>]

--grant <open_id>  授权指定用户并转移所有权
--email <email>    通过邮箱查找用户 open_id，然后授权并转移所有权

优化策略:
- 简单块(文本/标题/列表/引用/分割线) → descendant API 批量插入
- 代码块 → children API 逐个(descendant 不支持 \n)
- 表格 → descendant API 一次性创建(含内容)
- 图片 → children API 三步(创建→上传→绑定)
- 1s 间隔避免限频
"""

import urllib.request
import urllib.error
import urllib.parse
import json
import os
import re
import sys
import time
import uuid

# ─── CONFIG ───────────────────────────────────────────────────────────────────
APP_ID = os.environ.get("FEISHU_APP_ID", "")
APP_SECRET = os.environ.get("FEISHU_APP_SECRET", "")
BASE_URL = "https://open.feishu.cn/open-apis"
EDIT_DELAY = 1.0
TABLE_TOTAL_WIDTH = 820  # 飞书文档表格总宽度(px)
TABLE_MIN_COL_WIDTH = 50  # API 最小列宽

# 官方 language enum
LANG_MAP = {
    "python": 49,
    "py": 49,
    "bash": 7,
    "sh": 60,
    "shell": 60,
    "go": 22,
    "java": 29,
    "javascript": 30,
    "js": 30,
    "typescript": 63,
    "ts": 63,
    "json": 28,
    "yaml": 67,
    "yml": 67,
    "xml": 66,
    "html": 24,
    "css": 12,
    "sql": 56,
    "rust": 53,
    "c": 10,
    "cpp": 9,
    "c++": 9,
    "csharp": 8,
    "c#": 8,
    "kotlin": 32,
    "swift": 61,
    "ruby": 52,
    "php": 43,
    "dockerfile": 18,
    "makefile": 38,
    "markdown": 39,
    "md": 39,
    "toml": 75,
    "graphql": 71,
    "protobuf": 48,
    "proto": 48,
}


def log(msg):
    print(f"[{time.strftime('%H:%M:%S')}] {msg}", flush=True)


# ─── HTTP ─────────────────────────────────────────────────────────────────────
def _h(token):
    return {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json; charset=utf-8",
    }


def api(token, method, path, payload=None, retries=3):
    url = path if path.startswith("http") else f"{BASE_URL}{path}"
    data = json.dumps(payload).encode() if payload else None
    for attempt in range(retries):
        req = urllib.request.Request(url, data=data, headers=_h(token), method=method)
        try:
            with urllib.request.urlopen(req, timeout=15) as r:
                return json.load(r)
        except urllib.error.HTTPError as e:
            body = e.read().decode()
            try:
                return json.loads(body)
            except:
                return {"code": e.code}
        except Exception as e:
            if attempt < retries - 1:
                time.sleep((attempt + 1) * 3)
            else:
                log(f"  ❌ {type(e).__name__}: {e}")
                return {"code": -1}


def get_token():
    req = urllib.request.Request(
        f"{BASE_URL}/auth/v3/tenant_access_token/internal",
        data=json.dumps({"app_id": APP_ID, "app_secret": APP_SECRET}).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=10) as r:
        result = json.load(r)
    return result["tenant_access_token"]


def resolve_email(token, email):
    """通过邮箱查找用户 open_id"""
    r = api(token, "POST", "/contact/v3/users/batch_get_id", {"emails": [email]})
    if r.get("code") != 0:
        log(f"❌ 邮箱查找失败: {r.get('msg', r)}")
        return None
    user_list = r.get("data", {}).get("user_list", [])
    if not user_list or not user_list[0].get("user_id"):
        log(f"❌ 未找到邮箱对应的用户: {email}")
        return None
    open_id = user_list[0]["user_id"]
    log(f"✅ 邮箱 {email} → open_id: {open_id}")
    return open_id


def transfer_owner(token, doc_id, open_id):
    """将文档所有权转移给指定用户"""
    r = api(
        token,
        "POST",
        f"/drive/v1/permissions/{doc_id}/members/transfer_owner?type=docx&need_notification=false",
        {"member_type": "openid", "member_id": open_id},
    )
    if r.get("code") == 0:
        log("✅ 已转移所有权")
    else:
        log(f"⚠️ 所有权转移失败: {r.get('msg', r)}")


# ─── BLOCK BUILDERS ──────────────────────────────────────────────────────────
def nid():
    return str(uuid.uuid4()).replace("-", "")[:16]


FIELD_MAP = {
    2: "text",
    3: "heading1",
    4: "heading2",
    5: "heading3",
    6: "heading4",
    7: "heading5",
    8: "heading6",
    12: "bullet",
    13: "ordered",
    15: "quote",
}


def text_block(bt, text, bid=None):
    """简单文本块(用于 descendant)，支持 inline markdown 格式"""
    bid = bid or nid()
    if bt == 22:
        return {"block_id": bid, "block_type": 22, "divider": {}, "children": []}
    f = FIELD_MAP.get(bt, "text")
    elements = parse_inline(text)
    return {
        "block_id": bid,
        "block_type": bt,
        f: {"elements": elements},
        "children": [],
    }


def calc_column_widths(
    data, cols, total_width=TABLE_TOTAL_WIDTH, min_width=TABLE_MIN_COL_WIDTH
):
    """按每列最长文本字符数比例分配列宽(中文字符算2宽度)"""
    max_lens = []
    for c in range(cols):
        max_len = 1  # 至少1
        for row in data:
            if c < len(row):
                # 中文字符宽度约为英文的2倍
                w = sum(2 if ord(ch) > 127 else 1 for ch in row[c])
                max_len = max(max_len, w)
        max_lens.append(max_len)
    total = sum(max_lens) or 1
    widths = [max(min_width, int(total_width * l / total)) for l in max_lens]
    # 修正总宽度误差(加到最宽的列)
    diff = total_width - sum(widths)
    if diff != 0:
        widest = widths.index(max(widths))
        widths[widest] += diff
    return widths


def table_blocks(data, header_row=True):
    """表格块(用于 descendant，含 cells 和内容)，自动按内容比例分配列宽"""
    rows = len(data)
    cols = max(len(r) for r in data) if data else 0
    table_id = nid()
    cell_ids, descendants = [], []
    col_widths = calc_column_widths(data, cols)

    for r in range(rows):
        for c in range(cols):
            cell_id, text_id = nid(), nid()
            text = data[r][c] if r < len(data) and c < len(data[r]) else ""
            cell_ids.append(cell_id)
            descendants.append(
                {
                    "block_id": cell_id,
                    "block_type": 32,
                    "table_cell": {},
                    "children": [text_id],
                }
            )
            elements = parse_inline(text)
            descendants.append(
                {
                    "block_id": text_id,
                    "block_type": 2,
                    "text": {"elements": elements},
                    "children": [],
                }
            )

    table = {
        "block_id": table_id,
        "block_type": 31,
        "table": {
            "property": {
                "row_size": rows,
                "column_size": cols,
                "header_row": header_row,
                "column_width": col_widths,
            }
        },
        "children": cell_ids,
    }
    return table, [table] + descendants


# ─── MARKDOWN PARSER ──────────────────────────────────────────────────────────
def strip_md(text):
    """去除 inline markdown 标记(纯文本)"""
    for pat, rep in [
        (r"\*\*([^*]+)\*\*", r"\1"),
        (r"\*([^*]+)\*", r"\1"),
        (r"`([^`]+)`", r"\1"),
        (r"\[([^\]]+)\]\([^\)]+\)", r"\1"),
    ]:
        text = re.sub(pat, rep, text)
    return text.strip()


def parse_inline(text):
    """解析 inline markdown 为飞书 text elements(保留格式)
    - **bold** → bold style
    - `code` → inline_code style
    - *italic* → italic style
    - [text](url) → link style (需 http/https，url_encode)
    """
    elements = []
    pattern = re.compile(
        r"(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*]+\*)|(\[[^\]]+\]\([^\)]+\))"
    )
    last = 0
    for m in pattern.finditer(text):
        if m.start() > last:
            elements.append({"text_run": {"content": text[last : m.start()]}})
        if m.group(1):  # `inline code`
            elements.append(
                {
                    "text_run": {
                        "content": m.group(1)[1:-1],
                        "text_element_style": {"inline_code": True},
                    }
                }
            )
        elif m.group(2):  # **bold**
            elements.append(
                {
                    "text_run": {
                        "content": m.group(2)[2:-2],
                        "text_element_style": {"bold": True},
                    }
                }
            )
        elif m.group(3):  # *italic*
            elements.append(
                {
                    "text_run": {
                        "content": m.group(3)[1:-1],
                        "text_element_style": {"italic": True},
                    }
                }
            )
        elif m.group(4):  # [text](url)
            lm = re.match(r"\[([^\]]+)\]\(([^\)]+)\)", m.group(4))
            if lm:
                url = lm.group(2)
                if url.startswith("http://") or url.startswith("https://"):
                    # 完整 URL → link(需 url_encode)
                    encoded_url = urllib.parse.quote(url, safe="")
                    elements.append(
                        {
                            "text_run": {
                                "content": lm.group(1),
                                "text_element_style": {"link": {"url": encoded_url}},
                            }
                        }
                    )
                else:
                    # 锚点/相对路径 → 纯文本(若 link text 有反引号则转 inline code)
                    link_text = lm.group(1)
                    code_m = re.match(r"^`(.+)`$", link_text)
                    if code_m:
                        elements.append(
                            {
                                "text_run": {
                                    "content": code_m.group(1),
                                    "text_element_style": {"inline_code": True},
                                }
                            }
                        )
                    else:
                        elements.append({"text_run": {"content": link_text}})
        last = m.end()
    if last < len(text):
        remaining = text[last:]
        if remaining:
            elements.append({"text_run": {"content": remaining}})
    if not elements:
        elements.append({"text_run": {"content": text}})
    return elements


def parse_markdown(md_path):
    """解析 Markdown 为结构化 segments 列表
    返回: [(type, data), ...] 其中 type 为:
      "text" -> (block_type, text)
      "code" -> (lang_str, code_content)
      "table" -> [[row], [row], ...]
      "image" -> image_path
      "divider" -> None
    """
    with open(md_path) as f:
        lines = f.readlines()

    segments = []
    i = 0
    title = None

    while i < len(lines):
        line = lines[i].rstrip("\n")

        # 代码块
        if line.startswith("```"):
            lang_str = line[3:].strip()
            i += 1
            code_lines = []
            while i < len(lines) and not lines[i].rstrip("\n").startswith("```"):
                code_lines.append(lines[i].rstrip("\n"))
                i += 1
            i += 1  # skip closing ```
            code = "\n".join(code_lines)
            if code.strip():
                segments.append(("code", lang_str, code))
            continue

        # 表格
        if line.startswith("|"):
            table_rows = []
            while i < len(lines) and lines[i].rstrip("\n").startswith("|"):
                row_line = lines[i].rstrip("\n")
                cells = [c.strip() for c in row_line.split("|")[1:-1]]
                # 跳过分隔行 (|---|---|)
                if cells and all(re.match(r"^[-:]+$", c) for c in cells):
                    i += 1
                    continue
                if cells:
                    table_rows.append(cells)
                i += 1
            if table_rows:
                segments.append(("table", table_rows))
            continue

        # 分割线
        if re.match(r"^---+\s*$", line.strip()):
            segments.append(("divider",))
            i += 1
            continue

        # 标题
        m = re.match(r"^(#{1,6})\s+(.*)", line)
        if m:
            level = len(m.group(1))
            raw = m.group(2).strip()
            if level == 1 and title is None:
                title = strip_md(raw)  # H1 → 文档标题(纯文本)
            else:
                bt = level + 2  # H1→3, H2→4, H3→5, ...
                if raw:
                    segments.append(("text", bt, raw))
            i += 1
            continue

        # 无序列表
        m = re.match(r"^\s*[-*+]\s+(.*)", line)
        if m:
            text = m.group(1).strip()
            if text:
                segments.append(("text", 12, text))
            i += 1
            continue

        # 有序列表
        m = re.match(r"^\s*\d+\.\s+(.*)", line)
        if m:
            text = m.group(1).strip()
            if text:
                segments.append(("text", 13, text))
            i += 1
            continue

        # 图片
        m = re.match(r"!\[.*?\]\((.+?)\)", line)
        if m:
            segments.append(("image", m.group(1)))
            i += 1
            continue

        # 引用
        m = re.match(r"^>\s*(.*)", line)
        if m:
            text = m.group(1).strip()
            if text:
                segments.append(("text", 15, text))
            i += 1
            continue

        # 普通段落
        text = line.strip()
        if text:
            segments.append(("text", 2, text))
        i += 1

    return title or "Untitled", segments


# ─── WRITER ───────────────────────────────────────────────────────────────────
def desc_insert(token, doc_id, blocks):
    """descendant API 批量插入"""
    all_child_refs = set()
    for b in blocks:
        for c in b.get("children", []):
            all_child_refs.add(c)
    top_ids = [b["block_id"] for b in blocks if b["block_id"] not in all_child_refs]

    url = f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/descendant?document_revision_id=-1"
    r = api(token, "POST", url, {"children_id": top_ids, "descendants": blocks})
    if r.get("code") != 0:
        log(f"  ⚠️ desc_insert 失败: {r.get('msg', r)}")
    time.sleep(EDIT_DELAY)
    return r


def child_insert(token, doc_id, children):
    url = (
        f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/children?document_revision_id=-1"
    )
    r = api(token, "POST", url, {"children": children})
    time.sleep(EDIT_DELAY)
    return r


def insert_code(token, doc_id, code, lang_str):
    lang_code = LANG_MAP.get(lang_str.lower().strip(), 1)
    return child_insert(
        token,
        doc_id,
        [
            {
                "block_type": 14,
                "code": {
                    "style": {"language": lang_code, "wrap": False},
                    "elements": [{"text_run": {"content": code}}],
                },
            }
        ],
    )


def insert_diagram(token, doc_id, code, syntax_type=2):
    """插入 Mermaid/PlantUML 图表(画板 block_type 43)
    syntax_type: 1=PlantUML, 2=Mermaid
    三步: 创建 board block → 获取 whiteboard_id → 写入图表代码
    """
    # Step 1: 创建 board block
    r = child_insert(token, doc_id, [{"block_type": 43, "board": {}}])
    if r.get("code") != 0:
        log(f"  ❌ 创建画板失败: {r.get('msg', r)}")
        return r
    wid = r["data"]["children"][0]["board"]["token"]
    log(f"  画板 whiteboard_id: {wid}")

    # Step 2: 写入图表代码
    r2 = api(
        token,
        "POST",
        f"/board/v1/whiteboards/{wid}/nodes/plantuml",
        {"plant_uml_code": code, "syntax_type": syntax_type, "diagram_type": 0},
    )
    time.sleep(EDIT_DELAY)
    if r2.get("code") != 0:
        log(f"  ❌ 写入图表失败: {r2.get('msg', r2)}")
    return r2


def insert_image(token, doc_id, image_path):
    if not os.path.exists(image_path):
        log(f"  ⚠️ 图片不存在: {image_path}")
        return
    r = child_insert(token, doc_id, [{"block_type": 27, "image": {}}])
    if r.get("code") != 0:
        return
    bid = r["data"]["children"][0]["block_id"]
    with open(image_path, "rb") as f:
        file_data = f.read()
    ext = os.path.splitext(image_path)[1].lower().lstrip(".")
    mime = {
        "jpg": "image/jpeg",
        "jpeg": "image/jpeg",
        "png": "image/png",
        "gif": "image/gif",
        "webp": "image/webp",
    }.get(ext, "image/jpeg")
    boundary = "BoundaryFeishuDocAPI"

    def mk(n, v):
        return f'--{boundary}\r\nContent-Disposition: form-data; name="{n}"\r\n\r\n{v}\r\n'.encode()

    body = (
        mk("file_name", os.path.basename(image_path))
        + mk("parent_type", "docx_image")
        + mk("parent_node", bid)
        + mk("size", str(len(file_data)))
        + f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="{os.path.basename(image_path)}"\r\nContent-Type: {mime}\r\n\r\n'.encode()
        + file_data
        + f"\r\n--{boundary}--\r\n".encode()
    )
    req = urllib.request.Request(
        f"{BASE_URL}/drive/v1/medias/upload_all",
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": f"multipart/form-data; boundary={boundary}",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as r2:
            ur = json.load(r2)
        if ur.get("code") == 0:
            ft = ur["data"]["file_token"]
            api(
                token,
                "PATCH",
                f"/docx/v1/documents/{doc_id}/blocks/{bid}?document_revision_id=-1",
                {"replace_image": {"token": ft}},
            )
            time.sleep(EDIT_DELAY)
    except Exception as e:
        log(f"  ❌ 图片上传失败: {e}")


def write_to_feishu(token, doc_id, segments):
    """将解析后的 segments 写入飞书文档
    策略: 积累简单块 → 遇到代码/表格/图片时先 flush → 处理特殊块 → 继续积累"""
    batch = []  # 待批量插入的简单块

    def flush():
        if batch:
            log(f"  desc({len(batch)} blocks)")
            desc_insert(token, doc_id, list(batch))
            batch.clear()

    for seg in segments:
        if seg[0] == "text":
            _, bt, text = seg
            batch.append(text_block(bt, text))

        elif seg[0] == "divider":
            batch.append(text_block(22, ""))

        elif seg[0] == "code":
            flush()
            _, lang_str, code = seg
            lang_lower = lang_str.lower().strip()
            if lang_lower == "mermaid":
                log(f"  diagram(mermaid)")
                insert_diagram(token, doc_id, code, syntax_type=2)
            elif lang_lower == "plantuml":
                log(f"  diagram(plantuml)")
                insert_diagram(token, doc_id, code, syntax_type=1)
            else:
                log(f"  code({lang_str})")
                insert_code(token, doc_id, code, lang_str)

        elif seg[0] == "table":
            flush()
            _, data = seg
            log(f"  table({len(data)}x{len(data[0])})")
            tbl, tbl_desc = table_blocks(data)
            desc_insert(token, doc_id, tbl_desc)

        elif seg[0] == "image":
            flush()
            _, path = seg
            log(f"  image({os.path.basename(path)})")
            insert_image(token, doc_id, path)

    flush()  # 最后 flush 剩余


# ─── MAIN ─────────────────────────────────────────────────────────────────────
def main():
    if len(sys.argv) < 2:
        print(
            "用法: FEISHU_APP_ID=xxx FEISHU_APP_SECRET=xxx python3 md2feishu.py <markdown_file> [--grant <open_id>] [--email <email>]"
        )
        sys.exit(1)

    md_path = sys.argv[1]
    grant_id = None
    email = None
    if "--grant" in sys.argv:
        idx = sys.argv.index("--grant")
        if idx + 1 < len(sys.argv):
            grant_id = sys.argv[idx + 1]
    if "--email" in sys.argv:
        idx = sys.argv.index("--email")
        if idx + 1 < len(sys.argv):
            email = sys.argv[idx + 1]

    if not APP_ID or not APP_SECRET:
        print("请设置环境变量 FEISHU_APP_ID 和 FEISHU_APP_SECRET")
        sys.exit(1)

    if not os.path.exists(md_path):
        print(f"文件不存在: {md_path}")
        sys.exit(1)

    start = time.time()
    md_dir = os.path.dirname(os.path.abspath(md_path))
    log(f"解析 {md_path}...")
    title, segments = parse_markdown(md_path)

    # 解析图片相对路径为绝对路径
    for i, seg in enumerate(segments):
        if seg[0] == "image":
            img_path = seg[1]
            if not os.path.isabs(img_path):
                img_path = os.path.join(md_dir, img_path)
            segments[i] = ("image", img_path)
    log(f"标题: {title}")
    log(f"内容: {len(segments)} 个 segments")

    token = get_token()
    log("✅ Token OK")

    # 通过邮箱解析 open_id
    if email and not grant_id:
        grant_id = resolve_email(token, email)
        if not grant_id:
            log("⚠️ 邮箱解析失败，文档将不授权给用户")

    # 创建文档
    r = api(token, "POST", "/docx/v1/documents", {"title": title})
    if r.get("code") != 0:
        log(f"❌ 创建文档失败: {r.get('msg')}")
        sys.exit(1)
    doc_id = r["data"]["document"]["document_id"]
    doc_url = f"https://feishu.cn/docx/{doc_id}"
    log(f"✅ 文档: {doc_url}")

    # 创建后立即授权并转移所有权，这样用户可以实时看到内容写入
    if grant_id:
        api(
            token,
            "POST",
            f"/drive/v1/permissions/{doc_id}/members?type=docx&need_notification=false",
            {"member_type": "openid", "member_id": grant_id, "perm": "full_access"},
        )
        log("✅ 已授权")
        transfer_owner(token, doc_id, grant_id)
    time.sleep(EDIT_DELAY)

    # 写入内容
    log("写入内容...")
    write_to_feishu(token, doc_id, segments)

    elapsed = time.time() - start
    log(f"\n完成! {elapsed:.1f}s")
    log(f"文档: {doc_url}")


if __name__ == "__main__":
    main()
