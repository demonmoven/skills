"""后端模块 - 自动导入所有 backend 以触发 @register
导入顺序：dbw → bytedoc → redis → byterds
bytedoc 必须在 dbw 之后，以便覆盖 dbw 的 ByteDoc 注册。
"""
from backends import dbw
from backends import bytedoc
from backends import redis
from backends import byterds
