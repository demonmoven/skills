import argparse
import json
import sys
import jsonschema

# User provided schema for stages
STAGES_SCHEMA = {
    "type": "array",
    "maxItems": 16,
    "items": {
        "type": "object",
        "additionalProperties": False,
        "required": [
            "id",
            "name",
            "jobs",
            "if"
        ],
        "properties": {
            "id": {
                "type": "string"
            },
            "name": {
                "type": "object",
                "properties": {
                    "value": {
                        "type": "string"
                    },
                    "lang": {
                        "type": "string"
                    },
                    "texts": {
                        "type": "object"
                    }
                },
                "additionalProperties": False
            },
            "if": {
                "type": [
                    "string"
                ]
            },
            "jobs": {
                "type": "array",
                "minItems": 1,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": [
                        "id",
                        "name",
                        "uses",
                        "if",
                        "depends_on"
                    ],
                    "properties": {
                        "id": {
                            "type": "string",
                            "pattern": "^[a-zA-Z][a-zA-Z0-9_.-]{0,63}$"
                        },
                        "name": {
                                "type": "object",
                                "properties": {
                                    "value": {
                                        "type": "string"
                                    },
                                    "lang": {
                                        "type": "string"
                                    },
                                    "texts": {
                                        "type": "object"
                                    }
                                },
                                "additionalProperties": False
                            },
                        "if": {
                            "type": [
                                "string"
                            ]
                        },
                        "if_skip": {
                            "type": [
                                "string",
                                "boolean"
                            ]
                        },
                        "depends_on": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            }
                        },
                        "extra_if": {
                            "type": [
                                "string"
                            ]
                        },
                        "manual": {
                            "type": "boolean"
                        },
                        "uses": {
                            "type": "string",
                            "minLength": 1
                        },
                        "inputs": {
                            "type": [
                                "object",
                                "null"
                            ]
                        },
                        "support_timeout": {
                            "type": "boolean"
                        },
                        "timeout": {
                            "type": "integer",
                            "minimum": 0,
                            "maximum": 7776000
                        },
                        "retry": {
                            "type": [
                                "object",
                                "null"
                            ],
                            "properties": {
                                "max": {
                                    "enum": [
                                        1,
                                        2,
                                        3,
                                        4,
                                        5
                                    ]
                                },
                                "interval": {
                                    "type": [
                                        "string",
                                        "number"
                                    ]
                                }
                            },
                            "additionalProperties": False
                        },
                        "on_failed": {
                            "enum": [
                                "ON_FAILED_UNSPECIFIED",
                                "fail",
                                "retry",
                                "continue",
                                "ignore"
                            ]
                        },
                        "on_timeout": {
                            "enum": [
                                "ON_TIMEOUT_UNSPECIFIED",
                                "fail",
                                "skip",
                                "cancel"
                            ]
                        },
                        "on_ignored": {
                            "enum": [
                                "ON_IGNORED_UNSPECIFIED",
                                "abort",
                                "continue"
                            ]
                        },
                        "enable_pipeline_rollback": {
                            "type": "boolean"
                        },
                        "can_operations": {
                            "type": [
                                "array",
                                "object",
                                "null"
                            ]
                        },
                        "disable_operations": {
                            "type": [
                                "array",
                                "object",
                                "null"
                            ]
                        },
                        "auto_go_module_proxy": {
                            "type": "boolean"
                        },
                        "notifications": {
                            "type": [
                                "array",
                                "null"
                            ],
                            "items": {
                                "type": [
                                    "object",
                                    "null"
                                ],
                                "properties": {
                                    "type": {
                                        "enum": [
                                            "webhook",
                                            "lark"
                                        ]
                                    },
                                    "name": {
                                        "type": [
                                            "string",
                                            "number"
                                        ],
                                        "maxLength": 99
                                    },
                                    "when": {
                                        "type": [
                                            "object",
                                            "null"
                                        ],
                                        "properties": {
                                            "status": {
                                                "type": [
                                                    "array",
                                                    "null"
                                                ],
                                                "items": {
                                                    "enum": [
                                                        "start",
                                                        "waiting",
                                                        "running",
                                                        "blocking",
                                                        "cancelled",
                                                        "rollbacked",
                                                        "failed",
                                                        "succeeded",
                                                        "skipped",
                                                        "ignored"
                                                    ]
                                                }
                                            },
                                            "timeout": {
                                                "type": "integer",
                                                "maximum": 2678400,
                                                "minimum": 0
                                            }
                                        }
                                    },
                                    "lark": {
                                        "type": [
                                            "object",
                                            "null"
                                        ],
                                        "properties": {
                                            "users": {
                                                "type": "array",
                                                "items": {
                                                    "type": "string"
                                                }
                                            },
                                            "notifier_types": {
                                                "type": "array",
                                                "items": {
                                                    "type": "string"
                                                }
                                            },
                                            "notification_method": {
                                                "enum": [
                                                    "normal",
                                                    "mentioned",
                                                    "urgent"
                                                ]
                                            },
                                            "groups": {
                                                "type": "array",
                                                "items": {
                                                    "type": [
                                                        "string",
                                                        "integer"
                                                    ]
                                                }
                                            },
                                            "group_notification_method": {
                                                "enum": [
                                                    "normal",
                                                    "mentioned",
                                                    "urgent"
                                                ]
                                            },
                                            "cards": {
                                                "type": "array",
                                                "items": {
                                                    "type": "object",
                                                    "properties": {
                                                        "title": {
                                                            "type": [
                                                                "string",
                                                                "number"
                                                            ]
                                                        },
                                                        "content": {
                                                            "type": [
                                                                "string",
                                                                "number"
                                                            ]
                                                        }
                                                    }
                                                }
                                            },
                                            "repeat": {
                                                "type": [
                                                    "object",
                                                    "null"
                                                ],
                                                "properties": {
                                                    "count": {
                                                        "type": "integer",
                                                        "minimum": 0
                                                    },
                                                    "interval": {
                                                        "type": "integer",
                                                        "enum": [
                                                            10,
                                                            120,
                                                            300,
                                                            600,
                                                            1800,
                                                            3600,
                                                            86400
                                                        ]
                                                    }
                                                }
                                            }
                                        }
                                    },
                                    "webhook": {
                                        "type": [
                                            "object",
                                            "null"
                                        ],
                                        "properties": {
                                            "action_type": {
                                                "enum": [
                                                    "http",
                                                    "pipeline"
                                                ]
                                            },
                                            "http_action": {
                                                "type": [
                                                    "object",
                                                    "null"
                                                ],
                                                "properties": {
                                                    "url": {
                                                        "type": "string"
                                                    },
                                                    "method": {
                                                        "enum": [
                                                            "GET",
                                                            "POST",
                                                            "PUT"
                                                        ]
                                                    },
                                                    "headers": {
                                                        "type": "object"
                                                    },
                                                    "body": {
                                                        "type": "string"
                                                    }
                                                }
                                            },
                                            "pipelineAction": {
                                                "type": [
                                                    "integer",
                                                    "null"
                                                ]
                                            }
                                        }
                                    }
                                }
                            }
                        },
                        "run_env": {
                            "type": "string"
                        },
                        "job_run_operations_used": {
                            "type": [
                                "string",
                                "null"
                            ]
                        },
                        "job_run_operations": {
                            "type": [
                                "array",
                                "object",
                                "null"
                            ]
                        }
                    }
                }
            }
        }
    }
}

PIPELINE_SCHEMA = {
    "type": "object",
    "required": ["name", "stages"],
    "properties": {
        "name": {
            "type": "object",
            "required": ["value"],
            "properties": {
                "value": {"type": "string"},
                "lang": {"type": "string"},
                "texts": {"type": "object"}
            }
        },
        "stages": STAGES_SCHEMA
    }
}

def _load_payload(payload_json: str, payload_file: str) -> dict:
    if payload_json:
        try:
            return json.loads(payload_json)
        except json.JSONDecodeError as e:
            return {"_error": f"Invalid JSON string: {str(e)}"}
    if payload_file:
        try:
            with open(payload_file, "r", encoding="utf-8") as f:
                return json.load(f)
        except Exception as e:
            return {"_error": f"Failed to read file: {str(e)}"}
    return {}

def validate_pipeline(payload: dict) -> list:
    """
    对流水线编排进行静态校验，使用 jsonschema 库。
    返回错误信息列表，如果为空则表示校验通过。
    """
    errors = []
    
    # 0. 检查 JSON 解析错误
    if "_error" in payload:
        return [payload["_error"]]

    # 1. 结构和类型校验 (using jsonschema)
    validator = jsonschema.Draft7Validator(PIPELINE_SCHEMA)
    for error in validator.iter_errors(payload):
        path = ".".join([str(p) for p in error.path])
        message = error.message
        errors.append(f"{path}: {message}" if path else message)

    # 如果结构校验失败太多，可能没必要做业务逻辑校验，但为了兼容之前逻辑，继续做业务校验
    
    # 2. 业务逻辑校验 (Uniqueness, Dependencies)
    # 这些很难用 JSON Schema 表达，保留部分逻辑
    if "stages" in payload and isinstance(payload["stages"], list):
        seen_stage_ids = set()
        seen_all_job_ids = set()
        
        for i, stage in enumerate(payload["stages"]):
            if not isinstance(stage, dict): continue
            
            # Stage ID Uniqueness
            stage_id = stage.get("id")
            if stage_id:
                if stage_id in seen_stage_ids:
                    errors.append(f"stages[{i}].id: duplicate stage id '{stage_id}'")
                else:
                    seen_stage_ids.add(stage_id)
            
            # Job checks
            jobs = stage.get("jobs")
            if isinstance(jobs, list):
                # 获取当前 stage 下所有的 job id，用于校验 depends_on 是否在同一个 stage 下
                current_stage_job_ids = {j.get("id") for j in jobs if isinstance(j, dict) and j.get("id")}

                for j, job in enumerate(jobs):
                    if not isinstance(job, dict): continue

                    job_id = job.get("id")
                    if job_id:
                        # Global Job ID Uniqueness
                        if job_id in seen_all_job_ids:
                            errors.append(f"duplicate job id {job_id}")
                        else:
                            seen_all_job_ids.add(job_id)

                        # Self Dependency
                        depends_on = job.get("depends_on")
                        if depends_on is None:
                            errors.append(f"Job '{job_id}' depends_on required")
                            continue
                        if not isinstance(depends_on, list):
                            errors.append(f"Job '{job_id}' depends_on must be a list")
                            continue
                        if job_id in depends_on:
                            errors.append(f"Job '{job_id}' cannot depend on itself")

                        # 校验 depends_on 的 job 是否在同一个 stage 下
                        for dep_id in depends_on:
                            if dep_id not in current_stage_job_ids:
                                errors.append(f"Job '{job_id}' depends on '{dep_id}' which is not in the same stage")

    return errors



def main():
    parser = argparse.ArgumentParser(description="验证 Bits 流水线编排配置 (Draft)")
    parser.add_argument(
        "--payload-json",
        default="",
        help="请求体 JSON 字符串",
    )
    parser.add_argument(
        "--payload-file",
        default="",
        help="请求体 JSON 文件路径",
    )
    # 虽然是本地校验，预留 jwt_token 参数位置以保持接口一致性
    parser.add_argument("jwt_token", nargs="?", help="用户的 JWT Token (Optional for local validation)")

    args = parser.parse_args()
    
    # 必须提供 payload
    if not args.payload_json and not args.payload_file:
        print(json.dumps({"valid": False, "errors": ["Must provide --payload-json or --payload-file"]}, ensure_ascii=False))
        sys.exit(1)

    payload = _load_payload(args.payload_json, args.payload_file)
    errors = validate_pipeline(payload)
    
    result = {
        "valid": len(errors) == 0,
        "errors": errors
    }
    
    print(json.dumps(result, indent=4, ensure_ascii=False))
    
    if not result["valid"]:
        sys.exit(1)

if __name__ == "__main__":
    main()
