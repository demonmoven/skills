"""
feishu-doc-api: Complete Feishu document management via Open API.

Import this file or copy functions as needed.
All functions use urllib (stdlib only, no dependencies).
"""

import urllib.request
import urllib.error
import json
import os

# ─── CONFIG ──────────────────────────────────────────────────────────────────

APP_ID = os.environ.get("FEISHU_APP_ID", "your_app_id")
APP_SECRET = os.environ.get("FEISHU_APP_SECRET", "your_app_secret")
BASE_URL = "https://open.feishu.cn/open-apis"


# ─── AUTH ─────────────────────────────────────────────────────────────────────

def get_token(app_id=None, app_secret=None):
    """Get tenant_access_token. Valid for 7200 seconds."""
    req = urllib.request.Request(
        f"{BASE_URL}/auth/v3/tenant_access_token/internal",
        data=json.dumps({"app_id": app_id or APP_ID, "app_secret": app_secret or APP_SECRET}).encode(),
        headers={"Content-Type": "application/json"},
        method="POST"
    )
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.load(r)["tenant_access_token"]


# ─── HTTP HELPERS ─────────────────────────────────────────────────────────────

def _headers(token):
    return {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}


def api_get(token, path):
    url = path if path.startswith("http") else f"{BASE_URL}{path}"
    req = urllib.request.Request(url, headers=_headers(token), method="GET")
    with urllib.request.urlopen(req, timeout=15) as r:
        return json.load(r)


def api_post(token, path, payload):
    url = path if path.startswith("http") else f"{BASE_URL}{path}"
    req = urllib.request.Request(url, data=json.dumps(payload).encode(), headers=_headers(token), method="POST")
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.load(r)


def api_patch(token, path, payload):
    url = path if path.startswith("http") else f"{BASE_URL}{path}"
    req = urllib.request.Request(url, data=json.dumps(payload).encode(), headers=_headers(token), method="PATCH")
    with urllib.request.urlopen(req, timeout=15) as r:
        return json.load(r)


def api_delete(token, path, payload):
    url = path if path.startswith("http") else f"{BASE_URL}{path}"
    req = urllib.request.Request(url, data=json.dumps(payload).encode(), headers=_headers(token), method="DELETE")
    with urllib.request.urlopen(req, timeout=15) as r:
        return json.load(r)


def _children_url(doc_id, block_id, index=None):
    url = f"/docx/v1/documents/{doc_id}/blocks/{block_id}/children?document_revision_id=-1"
    if index is not None:
        url += f"&index={index}"
    return url


# ─── DOCUMENT ─────────────────────────────────────────────────────────────────

def create_document(token, title, folder_token=None):
    """Create a new Feishu document. Returns document_id and URL."""
    payload = {"title": title}
    if folder_token:
        payload["folder_token"] = folder_token
    result = api_post(token, "/docx/v1/documents", payload)
    doc = result["data"]["document"]
    return doc["document_id"], f"https://feishu.cn/docx/{doc['document_id']}"


def read_document(token, doc_id):
    """Read document title and plain text content."""
    result = api_get(token, f"/docx/v1/documents/{doc_id}")
    return result["data"]


def list_blocks(token, doc_id, page_size=200):
    """List all blocks in a document."""
    items = []
    page_token = None
    while True:
        url = f"/docx/v1/documents/{doc_id}/blocks?page_size={page_size}"
        if page_token:
            url += f"&page_token={page_token}"
        result = api_get(token, url)
        items.extend(result["data"]["items"])
        if not result["data"].get("has_more"):
            break
        page_token = result["data"].get("page_token")
    return items


def get_block(token, doc_id, block_id):
    """Get a single block by ID."""
    return api_get(token, f"/docx/v1/documents/{doc_id}/blocks/{block_id}")["data"]


# ─── INSERT HELPERS ───────────────────────────────────────────────────────────

def _insert(token, doc_id, parent_id, children, index=None):
    payload = {"children": children}
    if index is not None:
        payload["index"] = index
    result = api_post(token, _children_url(doc_id, parent_id), payload)
    return result["data"]["children"]


# ─── TEXT & HEADINGS ──────────────────────────────────────────────────────────

def insert_text(token, doc_id, text, block_type=2, index=None, parent_id=None, bold=False, italic=False):
    """
    Insert a text or heading block.
    block_type: 2=paragraph, 3=H1, 4=H2, 5=H3, 6=H4, 7=H5, 8=H6
    """
    style = {}
    if bold or italic:
        style = {"bold": bold, "italic": italic}
    element = {"text_run": {"content": text}}
    if style:
        element["text_run"]["text_element_style"] = style
    return _insert(token, doc_id, parent_id or doc_id, [{
        "block_type": block_type,
        "text": {"elements": [element], "style": {}}
    }], index=index)


def insert_divider(token, doc_id, index=None):
    """Insert a horizontal divider (block_type 22)."""
    return _insert(token, doc_id, doc_id, [{"block_type": 22}], index=index)


# ─── CODE BLOCK ───────────────────────────────────────────────────────────────

def insert_code_block(token, doc_id, code, language=7, index=None):
    """
    Insert a code block.
    language: 1=PlainText 7=Bash 10=C 9=C++ 8=C# 22=Go 29=Java
              30=JavaScript 28=JSON 32=Kotlin 49=Python 53=Rust
              60=Shell 61=Swift 56=SQL 63=TypeScript 66=XML 67=YAML
    """
    return _insert(token, doc_id, doc_id, [{
        "block_type": 14,
        "code": {
            "style": {"language": language, "wrap": False},
            "elements": [{"text_run": {"content": code}}]
        }
    }], index=index)


# ─── TABLE ────────────────────────────────────────────────────────────────────

def _clear_cell_placeholder(token, doc_id, cell_id):
    """Delete the auto-generated empty placeholder block in a table cell."""
    children = api_get(token, f"/docx/v1/documents/{doc_id}/blocks/{cell_id}/children")["data"]["items"]
    for i, child in enumerate(children):
        if child.get("block_type") == 2:
            content = "".join(
                el.get("text_run", {}).get("content", "")
                for el in child.get("text", {}).get("elements", [])
            )
            if content == "":
                api_delete(token,
                    f"/docx/v1/documents/{doc_id}/blocks/{cell_id}/children/batch_delete?document_revision_id=-1",
                    {"start_index": i, "end_index": i + 1})
                return


def _write_cell(token, doc_id, cell_id, text):
    api_post(token, _children_url(doc_id, cell_id), {
        "children": [{
            "block_type": 2,
            "text": {"elements": [{"text_run": {"content": text}}], "style": {}}
        }]
    })


def insert_table_with_data(token, doc_id, data, header_row=True, index=None):
    """
    Insert a table and fill with data.
    data: list of lists, e.g. [["Header A", "Header B"], ["val1", "val2"]]
    Returns list of cell block_ids (row-major order).
    """
    rows = len(data)
    cols = max(len(row) for row in data) if data else 0

    # Step 1: Create table block
    result = _insert(token, doc_id, doc_id, [{
        "block_type": 31,
        "table": {"property": {"row_size": rows, "column_size": cols, "header_row": header_row}}
    }], index=index)
    cells = result[0]["table"]["cells"]

    # Step 2: Delete auto-generated empty placeholders
    for cell_id in cells:
        _clear_cell_placeholder(token, doc_id, cell_id)

    # Step 3: Write content
    for idx, cell_id in enumerate(cells):
        row, col = idx // cols, idx % cols
        if row < len(data) and col < len(data[row]):
            _write_cell(token, doc_id, cell_id, data[row][col])

    return cells


# ─── IMAGE ────────────────────────────────────────────────────────────────────

def insert_image(token, doc_id, image_path, index=None):
    """
    Insert an image. Follows required 3-step order:
    1. Create empty image block (get block_id)
    2. Upload image bound to that block_id
    3. Patch block with file_token
    """
    # Step 1: Create empty image block
    result = _insert(token, doc_id, doc_id, [{"block_type": 27, "image": {}}], index=index)
    block_id = result[0]["block_id"]

    # Step 2: Upload image bound to block_id
    file_name = os.path.basename(image_path)
    file_size = os.path.getsize(image_path)
    ext = os.path.splitext(file_name)[1].lower().lstrip(".")
    mime = {"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
            "gif": "image/gif", "webp": "image/webp"}.get(ext, "image/jpeg")

    with open(image_path, "rb") as f:
        file_data = f.read()

    boundary = "BoundaryFeishuDocAPI"

    def mk_field(name, value):
        return f"--{boundary}\r\nContent-Disposition: form-data; name=\"{name}\"\r\n\r\n{value}\r\n".encode()

    body = (
        mk_field("file_name", file_name) +
        mk_field("parent_type", "docx_image") +
        mk_field("parent_node", block_id) +
        mk_field("size", str(file_size)) +
        f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"{file_name}\"\r\nContent-Type: {mime}\r\n\r\n".encode() +
        file_data +
        f"\r\n--{boundary}--\r\n".encode()
    )

    upload_req = urllib.request.Request(
        f"{BASE_URL}/drive/v1/medias/upload_all",
        data=body,
        headers={"Authorization": f"Bearer {token}", "Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST"
    )
    with urllib.request.urlopen(upload_req, timeout=30) as r:
        file_token = json.load(r)["data"]["file_token"]

    # Step 3: Bind image to block
    api_patch(token, f"/docx/v1/documents/{doc_id}/blocks/{block_id}?document_revision_id=-1",
              {"replace_image": {"token": file_token}})
    return block_id


# ─── DIAGRAM BOARD (Mermaid / PlantUML) ──────────────────────────────────────

def insert_diagram(token, doc_id, diagram_code, syntax_type=2, diagram_type=0, index=None):
    """
    Insert a Mermaid or PlantUML diagram as an interactive board block.
    Requires app permission: board:whiteboard:node:create

    syntax_type: 1=PlantUML  2=Mermaid  3=SVG
    diagram_type: 0=auto-detect (recommended for Mermaid)
                  For PlantUML: 1=mindmap 2=sequence 3=activity 4=class
    style_type (not exposed): 1=board style  2=classic/re-editable (PlantUML only)
    """
    # Step 1: Create board block — "board":{} must be present
    result = _insert(token, doc_id, doc_id, [{"block_type": 43, "board": {}}], index=index)
    whiteboard_id = result[0]["board"]["token"]

    # Step 2: Write diagram to board
    api_post(token,
             f"{BASE_URL}/board/v1/whiteboards/{whiteboard_id}/nodes/plantuml",
             {"plant_uml_code": diagram_code, "syntax_type": syntax_type, "diagram_type": diagram_type})

    return whiteboard_id


# ─── UPDATE ───────────────────────────────────────────────────────────────────

def update_text_block(token, doc_id, block_id, text, bold=False, italic=False):
    """Update a text block's content."""
    style = {}
    if bold:
        style["bold"] = True
    if italic:
        style["italic"] = True
    element = {"text_run": {"content": text}}
    if style:
        element["text_run"]["text_element_style"] = style
    return api_patch(token, f"/docx/v1/documents/{doc_id}/blocks/{block_id}?document_revision_id=-1", {
        "update_text_elements": {"elements": [element]}
    })


# ─── DELETE ───────────────────────────────────────────────────────────────────

def delete_blocks(token, doc_id, parent_id, start_index, end_index):
    """Delete a range of child blocks from a parent block."""
    return api_delete(token,
        f"/docx/v1/documents/{doc_id}/blocks/{parent_id}/children/batch_delete?document_revision_id=-1",
        {"start_index": start_index, "end_index": end_index})


# ─── DEMO ─────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    token = get_token()
    DOC_ID = "your_document_id"

    # Create a document
    # doc_id, url = create_document(token, "My Doc")

    # Text and headings
    insert_text(token, DOC_ID, "Introduction", block_type=3)  # H1
    insert_text(token, DOC_ID, "This is a paragraph.")

    # Code block
    insert_code_block(token, DOC_ID, "print('Hello, World!')", language=49)

    # Table
    insert_table_with_data(token, DOC_ID, [
        ["错误码", "说明", "解决方案"],
        ["1770001", "invalid param", "检查 block 结构"],
        ["1770013", "relation mismatch", "图片上传 parent_node 用 image block_id"],
    ])

    # Image
    # insert_image(token, DOC_ID, "/path/to/image.jpg")

    # Mermaid diagram
    insert_diagram(token, DOC_ID, "graph TD\n    A[开始] --> B[处理] --> C[结束]", syntax_type=2)


# ─── PERMISSIONS ──────────────────────────────────────────────────────────────

def grant_permission(token, doc_id, member_id, member_type="openid", perm="edit"):
    """
    Grant a user permission to a document created by the app.

    IMPORTANT: App-created documents are only accessible by the app by default.
    Always call this after create_document() to give the user access.

    member_type: "openid" | "email" | "userid" | "unionid"
    perm: "view" | "edit" | "full_access"
    """
    return api_post(token, f"/drive/v1/permissions/{doc_id}/members?type=docx&need_notification=false", {
        "member_type": member_type,
        "member_id": member_id,
        "perm": perm
    })


def grant_permission_to_current_user(token, doc_id, user_open_id, perm="edit"):
    """Convenience wrapper: grant edit permission to a specific user by open_id."""
    return grant_permission(token, doc_id, user_open_id, member_type="openid", perm=perm)


# ─── BULK INSERT (descendant API) ────────────────────────────────────────────

def bulk_insert_simple_blocks(token, doc_id, blocks):
    """
    Bulk insert up to 1000 simple blocks in a single request using the descendant API.
    Much faster than inserting one by one.

    IMPORTANT CONSTRAINTS:
    1. Only works for simple blocks: text, headings, bullet, ordered, divider (block_type 22).
       Code blocks (block_type 14) contain newlines and MUST use insert_code_block() instead.
    2. Each block_id must be unique within a single request. Use uuid-based IDs.
    3. If any block in the batch is invalid, the entire batch fails with 1770001.
       Use fallback_insert_one_by_one() to recover.

    blocks: list of dicts with keys: block_type, content (str), block_id (unique str)
    """
    import uuid

    def nid():
        return str(uuid.uuid4()).replace("-", "")[:16]

    def kft(bt):
        return {2: "text", 3: "heading1", 4: "heading2", 5: "heading3",
                6: "heading4", 7: "heading5", 8: "heading6",
                12: "bullet", 13: "ordered", 15: "quote"}.get(bt, "text")

    def make_desc_block(bt, text, block_id=None):
        bid = block_id or nid()
        if bt == 22:
            return {"block_id": bid, "block_type": 22, "divider": {}, "children": []}
        field = kft(bt)
        return {
            "block_id": bid,
            "block_type": bt,
            field: {"elements": [{"text_run": {"content": text}}], "style": {}},
            "children": []
        }

    descendants = []
    for blk in blocks:
        bt = blk["block_type"]
        text = blk.get("content", "")
        bid = blk.get("block_id") or nid()
        descendants.append(make_desc_block(bt, text, bid))

    children_id = [d["block_id"] for d in descendants]
    url = f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/descendant?document_revision_id=-1"

    try:
        result = api_post(token, url, {"children_id": children_id, "descendants": descendants})
        if result.get("code") == 0:
            return result
        # Fallback: insert one by one via children API
        _fallback_one_by_one(token, doc_id, blocks)
        return {"code": 0, "fallback": True}
    except Exception:
        _fallback_one_by_one(token, doc_id, blocks)
        return {"code": 0, "fallback": True}


def _fallback_one_by_one(token, doc_id, blocks):
    """Fallback: insert blocks one at a time via children API."""
    import time

    def kft(bt):
        return {2: "text", 3: "heading1", 4: "heading2", 5: "heading3",
                6: "heading4", 7: "heading5", 8: "heading6",
                12: "bullet", 13: "ordered", 15: "quote"}.get(bt, "text")

    url = f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/children?document_revision_id=-1"
    for blk in blocks:
        bt = blk["block_type"]
        text = blk.get("content", "")
        if bt == 22:
            child = {"block_type": 22}
        else:
            field = kft(bt)
            child = {"block_type": bt, field: {"elements": [{"text_run": {"content": text}}], "style": {}}}
        try:
            api_post(token, url, {"children": [child]})
            time.sleep(0.35)
        except Exception:
            pass


def convert_markdown_to_doc(token, doc_id, md_path,
                             lang_map=None, batch_size=200):
    """
    Convert a Markdown file to a Feishu document.

    Strategy:
    - Simple blocks (text/headings/lists/dividers) → batched via descendant API
    - Code blocks → individual requests via children API (descendant doesn't support newlines)
    - Markdown tables → skipped (use insert_table_with_data() separately)
    - H1 heading → skipped (already the document title)

    lang_map: override language string → language enum mapping
    batch_size: max blocks per descendant request (max 1000, default 200 for safety)
    """
    import re
    import time
    import uuid

    DEFAULT_LANG_MAP = {
        "typescript": 63, "ts": 63, "javascript": 30, "js": 30,
        "python": 49, "py": 49, "bash": 7, "sh": 60, "shell": 60, "go": 22,
        "java": 29, "rust": 53, "sql": 56, "json": 28, "yaml": 67, "xml": 66,
        "c": 10, "cpp": 9, "c++": 9, "csharp": 8, "c#": 8,
        "kotlin": 32, "swift": 61, "ruby": 52, "php": 43, "html": 24, "css": 12,
    }
    lm = {**DEFAULT_LANG_MAP, **(lang_map or {})}

    def strip_md(t):
        for pat, rep in [(r'\*\*([^*]+)\*\*', r'\1'), (r'\*([^*]+)\*', r'\1'),
                         (r'`([^`]+)`', r'\1'), (r'\[([^\]]+)\]\([^\)]+\)', r'\1')]:
            t = re.sub(pat, rep, t)
        return t.strip()

    def nid():
        return str(uuid.uuid4()).replace("-", "")[:16]

    def kft(bt):
        return {2: "text", 3: "heading1", 4: "heading2", 5: "heading3",
                6: "heading4", 7: "heading5", 8: "heading6",
                12: "bullet", 13: "ordered", 15: "quote"}.get(bt, "text")

    with open(md_path) as f:
        lines = f.readlines()

    # Parse into segments
    segments = []
    i = 0
    title_skipped = False
    while i < len(lines):
        line = lines[i].rstrip('\n')
        if line.startswith("```"):
            lang_str = line[3:].strip()
            i += 1
            cl = []
            while i < len(lines) and not lines[i].rstrip('\n').startswith("```"):
                cl.append(lines[i].rstrip('\n'))
                i += 1
            i += 1
            if "\n".join(cl).strip():
                segments.append(("code", lang_str, "\n".join(cl)))
            continue
        if line.startswith("|"):
            i += 1
            continue  # skip tables
        if re.match(r'^---+$', line.strip()):
            segments.append(("div",))
            i += 1
            continue
        m = re.match(r'^(#{1,6})\s+(.*)', line)
        if m:
            lv = len(m.group(1))
            text = strip_md(m.group(2))
            if lv == 1 and not title_skipped:
                title_skipped = True
                i += 1
                continue
            bt = min(lv + 2, 11)
            if text:
                segments.append(("s", bt, text))
            i += 1
            continue
        matched = False
        for pat, bt in [(r'^\s*[-*+]\s+(.*)', 12), (r'^\s*\d+\.\s+(.*)', 13)]:
            m = re.match(pat, line)
            if m:
                text = strip_md(m.group(1))
                if text:
                    segments.append(("s", bt, text))
                matched = True
                break
        if not matched:
            m = re.match(r'^>\s*(.*)', line)
            text = strip_md(m.group(1) if m else line)
            if text:
                segments.append(("s", 2, text))
        i += 1

    # Flush helpers
    code_url = f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/children?document_revision_id=-1"
    desc_url = f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/descendant?document_revision_id=-1"
    total = [0]
    simple_batch = []

    def flush_simple():
        if not simple_batch:
            return
        children_id = [b["block_id"] for b in simple_batch]
        result = api_post(token, desc_url, {"children_id": children_id, "descendants": list(simple_batch)})
        time.sleep(0.4)
        if result.get("code") != 0:
            # Fallback one by one
            for blk in simple_batch:
                bt = blk["block_type"]
                if bt == 22:
                    r2 = api_post(token, code_url, {"children": [{"block_type": 22}]})
                else:
                    field = kft(bt)
                    c = blk.get(field, {}).get("elements", [{}])[0].get("text_run", {}).get("content", "")
                    if not c:
                        continue
                    r2 = api_post(token, code_url, {"children": [{"block_type": bt,
                        field: {"elements": [{"text_run": {"content": c}}], "style": {}}}]})
                time.sleep(0.35)
                if r2.get("code") == 0:
                    total[0] += 1
        else:
            total[0] += len(simple_batch)
        simple_batch.clear()

    for seg in segments:
        if seg[0] == "code":
            flush_simple()
            lang = lm.get(seg[1].lower().strip(), 1)
            result = api_post(token, code_url, {"children": [{"block_type": 14, "code": {
                "style": {"language": lang, "wrap": False},
                "elements": [{"text_run": {"content": seg[2]}}]
            }}]})
            time.sleep(0.4)
            if result.get("code") == 0:
                total[0] += 1
        elif seg[0] == "div":
            simple_batch.append({"block_id": nid(), "block_type": 22, "divider": {}, "children": []})
            if len(simple_batch) >= batch_size:
                flush_simple()
        else:
            bt, text = seg[1], seg[2]
            field = kft(bt)
            simple_batch.append({
                "block_id": nid(), "block_type": bt,
                field: {"elements": [{"text_run": {"content": text}}], "style": {}},
                "children": []
            })
            if len(simple_batch) >= batch_size:
                flush_simple()

    flush_simple()
    return total[0]
