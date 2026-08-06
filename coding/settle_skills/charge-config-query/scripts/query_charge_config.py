#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""根据主体 ID（contract_inst）查询 caijing_bytepay_charge_union 库中的计费配置，并以 Markdown 表格形式输出。

优先通过平台 RDS MCP 能力在 Region=bytedance、VRegion=China-Pay 上执行查询；
如 RDS MCP 不可用或认证缺失，则退回至使用 PyMySQL 直连的方式，仅按 contract_inst 等值过滤，避免全表扫描。
"""

import csv
import os
import sys
import tempfile
import uuid
from typing import Any, Dict, List, Sequence

try:
    from byted_aime_sdk import call_aime_tool  # type: ignore[import]
except ImportError:  # pragma: no cover - 依赖由运行环境按需安装
    call_aime_tool = None  # type: ignore[assignment]

try:
    import pymysql  # type: ignore[import]
except ImportError:  # pragma: no cover - 依赖由运行环境按需安装
    pymysql = None  # type: ignore[assignment]


# 查询字段顺序（同时作为 Markdown 表头顺序）
COLUMNS: List[str] = [
    "contract_inst",
    "scene",
    "match_rule",
    "charge_rule_content",
    "collect_fee_type",
    "collect_fee_mode",
]


# SQL 模板文件路径（保持与 references/sql_template.sql 一致）
SQL_TEMPLATE_PATH = os.path.join(
    os.path.dirname(os.path.dirname(__file__)),
    "references",
    "sql_template.sql",
)


def load_sql_template() -> str:
    """从 SQL 模板文件中读取查询语句。

    要求模板中包含 {{contract_inst}} 占位符。
    """

    if not os.path.exists(SQL_TEMPLATE_PATH):
        raise RuntimeError(f"SQL 模板文件不存在: {SQL_TEMPLATE_PATH}")

    with open(SQL_TEMPLATE_PATH, "r", encoding="utf-8") as f:
        return f.read().strip()

# RDS 连接信息（Region / VRegion 通过 MCP 能力隐式指定）
DEFAULT_DB_NAME = "caijing_bytepay_charge_union"
RDS_REGION = os.environ.get("CHARGE_CONFIG_RDS_REGION", "bytedance")


class DbConfigError(Exception):
    """数据库连接配置不完整时抛出。"""


class RdsMcpError(Exception):
    """RDS MCP 查询失败时抛出。"""


def render_sql_for_rds(contract_inst: str) -> str:
    """基于 SQL 模板渲染适用于 RDS MCP 的查询语句。

    使用简单转义替换 {{contract_inst}} 占位符，避免引号破坏 SQL 结构。
    """

    template = load_sql_template()
    safe_value = contract_inst.replace("'", "''")
    return template.replace("{{contract_inst}}", safe_value)


def get_connection():
    """基于环境变量创建到 RDS 的直连连接（PyMySQL）。

    期望的环境变量：
    - CHARGE_CONFIG_DB_HOST
    - CHARGE_CONFIG_DB_USER
    - CHARGE_CONFIG_DB_PASSWORD
    - CHARGE_CONFIG_DB_PORT（可选，默认 3306）
    - CHARGE_CONFIG_DB_NAME（可选，默认 caijing_bytepay_charge_union）
    """

    if pymysql is None:  # type: ignore[truthy-function]
        raise RuntimeError("未安装 PyMySQL 库，请先在当前环境中安装 `pymysql`。")

    host = os.environ.get("CHARGE_CONFIG_DB_HOST")
    user = os.environ.get("CHARGE_CONFIG_DB_USER")
    password = os.environ.get("CHARGE_CONFIG_DB_PASSWORD")
    port_str = os.environ.get("CHARGE_CONFIG_DB_PORT", "3306")
    db_name = os.environ.get("CHARGE_CONFIG_DB_NAME", DEFAULT_DB_NAME)

    missing = [
        name
        for name, value in [
            ("CHARGE_CONFIG_DB_HOST", host),
            ("CHARGE_CONFIG_DB_USER", user),
            ("CHARGE_CONFIG_DB_PASSWORD", password),
        ]
        if not value
    ]
    if missing:
        raise DbConfigError("缺少数据库连接配置环境变量: " + ", ".join(missing))

    try:
        port = int(port_str)
    except ValueError as exc:  # pragma: no cover - 配置错误分支
        raise DbConfigError("CHARGE_CONFIG_DB_PORT 必须为整数") from exc

    return pymysql.connect(  # type: ignore[call-arg]
        host=host,
        user=user,
        password=password,
        database=db_name,
        port=port,
        charset="utf8mb4",
        cursorclass=pymysql.cursors.DictCursor,  # type: ignore[attr-defined]
    )


def query_charge_config_via_mysql(contract_inst: str) -> List[Dict[str, Any]]:
    """使用 PyMySQL 直连方式按 contract_inst 精确匹配查询计费配置。"""

    sql_for_mysql = load_sql_template().replace("{{contract_inst}}", "%s")

    conn = get_connection()
    try:
        with conn.cursor() as cursor:  # type: ignore[assignment]
            cursor.execute(sql_for_mysql, (contract_inst,))  # type: ignore[arg-type]
            rows = cursor.fetchall()
            return list(rows)
    finally:
        conn.close()


def query_charge_config_via_rds_mcp(contract_inst: str) -> List[Dict[str, Any]]:
    """通过平台 RDS MCP 工具查询计费配置（首选路径）。

    要求：
    - 运行环境中安装 `byted_aime_sdk`
    - 任务执行时已注入 `AIME_USER_CLOUD_JWT`（通常通过 include_secrets=true 实现）
    - 使用工具集 `rds` 的 `mcp:rds_rds_run_sql` 工具
    """

    if call_aime_tool is None:
        raise RdsMcpError("未安装 byted_aime_sdk，无法调用 RDS MCP 工具")

    if not os.environ.get("AIME_USER_CLOUD_JWT"):
        raise RdsMcpError(
            "缺少 AIME_USER_CLOUD_JWT 环境变量，可能未在执行时启用 include_secrets=true"
        )

    sql = render_sql_for_rds(contract_inst)

    tmp_dir = tempfile.gettempdir()
    filepath = os.path.join(tmp_dir, f"charge_config_query_{uuid.uuid4().hex}.csv")

    params: Dict[str, Any] = {
        "db_name": DEFAULT_DB_NAME,
        "region": RDS_REGION,
        "filepath": filepath,
        "sql": sql,
    }

    try:
        # 结果会写入 filepath 指定的 CSV 文件
        _ = call_aime_tool(
            toolset="rds",
            tool_name="mcp:rds_rds_run_sql",
            parameters=params,
            response_format="text",
        )
    except Exception as exc:  # pragma: no cover - MCP 调用异常兜底
        raise RdsMcpError(f"调用 RDS MCP 工具失败: {exc}") from exc

    if not os.path.exists(filepath):
        raise RdsMcpError(f"RDS MCP 未生成预期的结果文件: {filepath}")

    try:
        with open(filepath, "r", encoding="utf-8") as f:
            reader = csv.DictReader(f)
            rows: List[Dict[str, Any]] = []
            for row in reader:
                # 仅保留关心的字段，且保证列顺序与 COLUMNS 一致
                rows.append({col: row.get(col) for col in COLUMNS})
    finally:
        try:
            os.remove(filepath)
        except OSError:
            # 结果文件删除失败不影响正常输出
            pass

    return rows


def query_charge_config(contract_inst: str) -> List[Dict[str, Any]]:
    """综合查询入口：优先使用 RDS MCP，失败时退回直连数据库。"""

    try:
        return query_charge_config_via_rds_mcp(contract_inst)
    except RdsMcpError as e:
        # 打印到 stderr 便于调试，但不影响兜底路径继续执行
        print(f"RDS MCP 查询失败，将尝试直连数据库: {e}", file=sys.stderr)

    # 兜底：使用直连数据库方式
    return query_charge_config_via_mysql(contract_inst)


def format_markdown(rows: Sequence[Dict[str, Any]]) -> str:
    """将查询结果渲染为 Markdown 表格。

    - 无记录时返回“无匹配记录”
    - None 值渲染为空字符串
    - 列顺序固定为 COLUMNS 中定义的顺序
    """

    if not rows:
        return "无匹配记录"

    header_line = "| " + " | ".join(COLUMNS) + " |"
    separator_line = "| " + " | ".join("---" for _ in COLUMNS) + " |"

    lines = [header_line, separator_line]

    for row in rows:
        values: List[str] = []
        for key in COLUMNS:
            value = row.get(key, "")
            if value is None:
                value = ""
            else:
                value = str(value)
            # 避免换行破坏表格结构
            value = value.replace("\r", " ").replace("\n", " ")
            values.append(value)
        lines.append("| " + " | ".join(values) + " |")

    return "\n".join(lines)


def main(argv: Sequence[str]) -> int:
    if len(argv) != 2:
        script_name = os.path.basename(argv[0] or "query_charge_config.py")
        print(f"用法: python3 {script_name} <contract_inst>", file=sys.stderr)
        return 1

    contract_inst = argv[1].strip()
    if not contract_inst:
        print("错误: contract_inst 不能为空", file=sys.stderr)
        return 1

    try:
        rows = query_charge_config(contract_inst)
    except DbConfigError as e:
        print(f"数据库配置错误: {e}", file=sys.stderr)
        return 1
    except Exception as e:  # pragma: no cover - 运行时异常兜底
        print(f"查询失败: {e}", file=sys.stderr)
        return 1

    markdown = format_markdown(rows)
    # 默认 stdout 即为 UTF-8，直接输出即可
    sys.stdout.write(markdown + "\n")
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main(sys.argv))
