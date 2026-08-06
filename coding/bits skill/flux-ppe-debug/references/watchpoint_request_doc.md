## Watchpoint Request Doc

本文件用于记录 Watchpoint 调试链路的请求/响应关键字段约定。

```json
{
  "timeout": 10,
  "response_success_codes": [
    0,
    null,
    "SUCCESS"
  ],
  "endpoints": {
    "create_session": {
      "payload": {
        "psm": "psm",
        "env": "env",
        "cluster": "cluster",
        "instance_name": "pod"
      }
    },
    "delete_session": {
      "params": {
        "session_id": "session_id"
      }
    },
    "create_watchpoint_batch": {
      "payload": {
        "session_id": "session_id",
        "watchpoints_field": "watchpoints",
        "default_action": "CAPTURE"
      }
    },
    "watchpoint_info": {
      "params": {
        "watchpoint_id": "breakpoint_id"
      }
    }
  },
  "response_keys": {
    "session_id": [
      [
        "data",
        "session_id"
      ],
      "session_id"
    ],
    "breakpoint_ids": [
      [
        "data"
      ],
      [
        "data",
        "watchpoint_ids"
      ],
      [
        "data",
        "breakpoint_ids"
      ],
      [
        "data",
        "ids"
      ],
      [
        "data",
        "watchpoints"
      ],
      [
        "data",
        "breakpoints"
      ],
      "watchpoint_ids",
      "breakpoint_ids",
      "ids"
    ]
  }
}
```
