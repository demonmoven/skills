---
name: rds-query
description: 查询 RDS（关系型数据库服务）元数据的 Agent。当需要搜索数据库、查看数据库信息、列出表结构时使用此 SKILL。仅支持元数据查询，不支持执行 SQL。
---

> **session_id 传递**：如果之前已经调用过 `gdpa-cli`，你将获得一个 SessionID，务必在之后的调用中都携带上此 session_id，并在 input 中加上此参数: `--input '{..., "session_id": "sess_xxx_xxx"}'`

# RDS Query Agent

Query RDS (Relational Database Service) metadata and information from BOE environment.

## Overview

This agent provides access to RDS OpenAPI for querying database **metadata** in the BOE environment. It supports searching databases, listing databases, getting database details, and listing table structures.

**Important**: This agent queries **metadata only** (database info, table lists, schema information). It does **NOT** execute SQL queries to fetch actual table data. For querying table data, you need to connect to the database directly using MySQL client or other tools.

## Supported Operations

✅ **What this agent CAN do:**
- Search for databases by keyword
- List all databases with pagination
- Get database information (owners, version, region, etc.)
- List tables in a database

❌ **What this agent CANNOT do:**
- Execute SELECT queries to fetch table data
- Execute INSERT/UPDATE/DELETE statements
- Run arbitrary SQL commands

## Features

- **Search Databases**: Search for databases by keyword
- **List Databases**: List all databases in a region with pagination
- **Get Database Info**: Get detailed information about a specific database
- **List Tables**: List all tables in a specific database

## Usage

### Command Line

```bash
./output/gdpa-cli run rds_query --input '{
  "action": "search",
  "keyword": "my_database",
  "region": "China-BOE"
}'
```

### Input Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `action` | string | Yes | - | Action to perform: `search`, `list`, `get_info`, `list_tables` |
| `keyword` | string | Conditional | - | Search keyword (required for `search` action) |
| `dbname` | string | Conditional | - | Database name (required for `get_info` and `list_tables` actions) |
| `region` | string | No | `cn` | Region (e.g., `cn` for China, `China-East` for East region) |
| `page` | integer | No | 1 | Page number for pagination |
| `page_size` | integer | No | 10 | Page size for pagination (max: 100) |

## Actions

### 1. Search Databases (`action: "search"`)

Search for databases by keyword.

**Required Parameters:**
- `keyword`: Search keyword

**Example:**
```bash
CONSUL_HTTP_HOST=10.37.45.130 ./output/gdpa-cli run rds_query --input '{
  "action": "search",
  "keyword": "gdpa",
  "region": "cn"
}'
```

**Output:**
```json
{
  "action": "search",
  "success": true,
  "data": {
    "databases": [
      {
        "region": "China-BOE",
        "dbname": "gdpa_service",
        "owners": ["user1", "user2"],
        "department": "backend",
        "consuls": ["gdpa.service"],
        "subscribed": false
      }
    ],
    "total": 1
  }
}
```

### 2. List Databases (`action: "list"`)

List all databases in a region with pagination.

**Example:**
```bash
CONSUL_HTTP_HOST=10.37.45.130 ./output/gdpa-cli run rds_query --input '{
  "action": "list",
  "region": "cn",
  "page": 1,
  "page_size": 20
}'
```

**Output:**
```json
{
  "action": "list",
  "success": true,
  "data": {
    "databases": [
      {
        "region": "China-BOE",
        "dbname": "database1",
        "owners": ["user1"]
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 3. Get Database Info (`action: "get_info"`)

Get detailed information about a specific database.

**Required Parameters:**
- `dbname`: Database name

**Example:**
```bash
CONSUL_HTTP_HOST=10.37.45.130 ./output/gdpa-cli run rds_query --input '{
  "action": "get_info",
  "dbname": "gdpa_service",
  "region": "cn"
}'
```

**Output:**
```json
{
  "action": "get_info",
  "success": true,
  "data": {
    "region": "China-BOE",
    "dbname": "gdpa_service",
    "owners": ["user1", "user2"],
    "department": "backend",
    "basic": {
      "db_type": "mysql",
      "db_version": "8.0",
      "charset": "utf8mb4"
    }
  }
}
```

### 4. List Tables (`action: "list_tables"`)

List all tables in a specific database.

**Required Parameters:**
- `dbname`: Database name

**Example:**
```bash
CONSUL_HTTP_HOST=10.37.45.130 ./output/gdpa-cli run rds_query --input '{
  "action": "list_tables",
  "dbname": "gdpa_service",
  "region": "cn",
  "page": 1,
  "page_size": 50
}'
```

**Output:**
```json
{
  "action": "list_tables",
  "success": true,
  "data": {
    "tables": [
      {
        "table_name": "users"
      },
      {
        "table_name": "orders"
      }
    ],
    "total": 15,
    "page": 1,
    "page_size": 50
  }
}
```

## Environment Variables

- `CONSUL_HTTP_HOST`: Consul host for JWT token retrieval (e.g., `10.37.45.130`)
- `DEBUG`: Set to `1` to enable debug logging

## Authentication

The agent uses internal JWT token authentication (`GetCNJwt()`) and DevFlow JWT secret for RDS API access.

## Region Support

**⚠️ PENDING: Region parameter needs validation**
- Current implementation uses `"cn"` as default based on SDK test examples
- Need to verify if this correctly queries BOE environment data
- May need additional configuration or different region values for BOE

Supported regions:
- `cn` (default) - China region
- `China-East` - China East region (requires special configuration)

Note: Use lowercase `"cn"` instead of `"China-BOE"` for BOE environment queries.

## Best Practices

1. **Use specific searches**: Provide detailed keywords when searching databases
2. **Paginate large results**: Use `page` and `page_size` parameters for large datasets
3. **Check region**: Ensure you're querying the correct region
4. **Error handling**: Check the `success` field in the output

## Error Handling

If query fails, the output will contain:

```json
{
  "action": "search",
  "success": false,
  "error": "error message"
}
```

Common errors:
- `failed to get JWT token`: Ensure `CONSUL_HTTP_HOST` is set correctly
- `keyword is required for search action`: Provide keyword parameter
- `dbname is required for get_info action`: Provide dbname parameter

## Logging

Enable debug logging to see detailed request/response information:

```bash
DEBUG=1 CONSUL_HTTP_HOST=10.37.45.130 ./output/gdpa-cli run rds_query --input '{...}' 2> error.log
```

The log file will contain:
- Full SDK request parameters
- Complete API responses
- Authentication status
- Query execution details

## Notes

- This agent queries **metadata only** (database structure, table lists, schema info)
- **Cannot execute SQL queries** to fetch actual table data (e.g., `SELECT * FROM table`)
- To query actual data, you need to:
  1. Use `get_info` action to get database connection info
  2. Connect to the database using MySQL client or other tools
  3. Execute SQL queries directly
- All operations are read-only
- BOE environment only (production access requires separate configuration)
