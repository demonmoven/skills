import json
import logging
import re
from typing import Any, AsyncGenerator, Literal, Optional

import litellm
from litellm.integrations.custom_logger import CustomLogger
from litellm.proxy.proxy_server import DualCache, UserAPIKeyAuth
from litellm.types.utils import ModelResponseStream

logger = logging.getLogger(__name__)

class ImageContentTransformer(CustomLogger):
    """
    自定义回调处理器，用于转换MCP ImageContent格式为OpenRouter格式
    """

    # Anthropic/OpenRouter 支持的图片 media_type
    VALID_MEDIA_TYPES = {'image/jpeg', 'image/png', 'image/gif', 'image/webp'}

    # media_type 映射表（将不支持的类型映射到支持的类型）
    MEDIA_TYPE_MAPPING = {
        'image/jpg': 'image/jpeg',
        'image/svg+xml': 'image/png',
        'image/bmp': 'image/png',
        'image/tiff': 'image/png',
        'image/x-icon': 'image/png',
        'image/ico': 'image/png',
        'image/avif': 'image/png',
        'image/heic': 'image/png',
        'image/heif': 'image/png',
    }

    def __init__(self):
        super().__init__()
        logger.info("ImageContentTransformer initialized")

    def _normalize_media_type(self, media_type: Optional[str]) -> str:
        """
        将 media_type 规范化为 Anthropic/OpenRouter 支持的格式。

        Args:
            media_type: 原始的 media_type，可能为 None 或不支持的格式

        Returns:
            规范化后的 media_type，如果无法识别则默认返回 'image/png'
        """
        if not media_type:
            # 没有 media_type，默认使用 png
            logger.warning(f"[ImageContentTransformer] Missing media_type, defaulting to image/png")
            return 'image/png'

        # 转换为小写并去除空格
        normalized = media_type.lower().strip()

        # 如果已经是有效类型，直接返回
        if normalized in self.VALID_MEDIA_TYPES:
            return normalized

        # 尝试从映射表中查找
        if normalized in self.MEDIA_TYPE_MAPPING:
            mapped_type = self.MEDIA_TYPE_MAPPING[normalized]
            logger.info(f"[ImageContentTransformer] Mapped media_type '{media_type}' to '{mapped_type}'")
            return mapped_type

        # 未知类型，默认使用 png 并记录警告
        logger.warning(
            f"[ImageContentTransformer] Unknown media_type '{media_type}', defaulting to image/png. "
            f"Supported types: {self.VALID_MEDIA_TYPES}"
        )
        return 'image/png'

    def _normalize_image_source(self, image_item: dict, convert_to_openai_format: bool = False) -> dict:
        """
        规范化图片的 source，确保 media_type 是有效的格式。

        支持多种图片格式：
        1. Anthropic 格式 v1: {"type": "image", "source": {"type": "base64", "media_type": "...", "data": "..."}}
        2. Anthropic 格式 v2: {"type": "image", "source": {"base64": {"media_type": "...", "data": "..."}}}
        3. OpenAI 格式: {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}

        关键修复（针对 OpenRouter -> Bedrock 兼容性问题）：
        - Amazon Bedrock 通过 OpenRouter 期望特定的图片格式
        - 错误: "messages.16.content.1.image.source.base64.media_type: Input should be..."
        - 当 convert_to_openai_format=True 时，将 Anthropic 格式转换为 OpenAI data URL 格式
        - OpenAI 格式是更通用的格式，被大多数 API 和网关支持

        Args:
            image_item: 包含 type='image' 或 type='image_url' 的字典
            convert_to_openai_format: 是否将 Anthropic 格式转换为 OpenAI data URL 格式

        Returns:
            规范化后的图片字典（创建副本，不修改原始数据）
        """
        import copy

        normalized = copy.deepcopy(image_item)
        item_type = normalized.get("type", "")

        # 处理 Anthropic 格式
        if item_type == "image":
            source = normalized.get("source", {})
            logger.info(f"[ImageContentTransformer] _normalize_image_source called, source keys: {list(source.keys()) if isinstance(source, dict) else type(source)}")
            if isinstance(source, dict):
                # 提取 media_type 和 data
                original_media_type = None
                original_data = None

                # 检查格式 v2: source.base64.media_type
                if "base64" in source and isinstance(source["base64"], dict):
                    base64_obj = source["base64"]
                    original_media_type = base64_obj.get("media_type")
                    original_data = base64_obj.get("data")
                    logger.info(f"[ImageContentTransformer] v2 format detected, original media_type: '{original_media_type}'")
                # 检查格式 v1: source.media_type
                elif "media_type" in source:
                    original_media_type = source.get("media_type")
                    original_data = source.get("data")
                    logger.info(f"[ImageContentTransformer] v1 format detected, original media_type: '{original_media_type}'")

                if original_media_type and original_data:
                    normalized_media_type = self._normalize_media_type(original_media_type)

                    if convert_to_openai_format:
                        # 关键修复：转换为 OpenAI data URL 格式
                        # 这是最通用的格式，被 OpenRouter、Bedrock 等大多数 API 支持
                        data_url = f"data:{normalized_media_type};base64,{original_data}"
                        normalized = {
                            "type": "image_url",
                            "image_url": {
                                "url": data_url
                            }
                        }
                        logger.info(
                            f"[ImageContentTransformer] Converted Anthropic image to OpenAI data URL format, "
                            f"media_type: '{original_media_type}' -> '{normalized_media_type}'"
                        )
                    else:
                        # 保持 Anthropic v1 格式，只规范化 media_type
                        new_source = {
                            "type": "base64",
                            "media_type": normalized_media_type,
                            "data": original_data
                        }
                        normalized["source"] = new_source
                        if original_media_type != normalized_media_type:
                            logger.info(
                                f"[ImageContentTransformer] Normalized Anthropic image media_type: "
                                f"'{original_media_type}' -> '{normalized_media_type}'"
                            )
                        else:
                            logger.info(f"[ImageContentTransformer] media_type already valid: '{original_media_type}'")

        # 处理 OpenAI 格式 (data URL)
        elif item_type == "image_url":
            image_url_obj = normalized.get("image_url", {})
            if isinstance(image_url_obj, dict):
                url = image_url_obj.get("url", "")
                if url.startswith("data:"):
                    # 解析 data URL: data:image/xxx;base64,...
                    try:
                        # 格式: data:media_type;base64,data
                        header_end = url.find(",")
                        if header_end > 0:
                            header = url[5:header_end]  # 去掉 "data:"
                            parts = header.split(";")
                            if parts:
                                original_media_type = parts[0]
                                normalized_media_type = self._normalize_media_type(original_media_type)

                                if original_media_type != normalized_media_type:
                                    # 重建 data URL
                                    new_header = normalized_media_type
                                    if len(parts) > 1:
                                        new_header += ";" + ";".join(parts[1:])
                                    new_url = "data:" + new_header + url[header_end:]
                                    image_url_obj["url"] = new_url
                                    normalized["image_url"] = image_url_obj
                                    logger.info(
                                        f"[ImageContentTransformer] Normalized OpenAI image_url media_type: "
                                        f"'{original_media_type}' -> '{normalized_media_type}'"
                                    )
                    except Exception as e:
                        logger.warning(f"[ImageContentTransformer] Failed to parse data URL: {e}")

        return normalized

    def _get_image_media_type(self, item: dict) -> Optional[str]:
        """
        从图片项中提取 media_type，支持多种格式。

        Args:
            item: 图片字典

        Returns:
            media_type 字符串，如果找不到则返回 None
        """
        item_type = item.get("type", "")

        if item_type == "image":
            source = item.get("source", {})
            if isinstance(source, dict):
                # 格式 v2: source.base64.media_type
                if "base64" in source and isinstance(source["base64"], dict):
                    return source["base64"].get("media_type")
                # 格式 v1: source.media_type
                return source.get("media_type")

        elif item_type == "image_url":
            image_url_obj = item.get("image_url", {})
            if isinstance(image_url_obj, dict):
                url = image_url_obj.get("url", "")
                if url.startswith("data:"):
                    header_end = url.find(",")
                    if header_end > 0:
                        header = url[5:header_end]
                        parts = header.split(";")
                        if parts:
                            return parts[0]

        return None

    def _process_content_item(self, item: Any, convert_to_openai_format: bool = False) -> tuple[list, bool]:
        """
        处理单个 content 项，规范化其中的图片。

        对于包含图片的 tool_result，会将其拆分为：
        1. 只包含文本的 tool_result
        2. 独立的图片项（规范化后的）

        Args:
            item: content 中的单个项
            convert_to_openai_format: 是否将 Anthropic 格式图片转换为 OpenAI data URL 格式
                                       （用于解决 OpenRouter -> Bedrock 兼容性问题）

        Returns:
            (处理后的项列表, 是否修改过)
            注意：返回的是列表，因为一个 tool_result 可能被拆分为多个项
        """
        if not isinstance(item, dict):
            return [item], False

        item_type = item.get("type", "")

        # 处理 Anthropic 格式图片
        if item_type == "image" and "source" in item:
            original_media_type = self._get_image_media_type(item)
            normalized = self._normalize_image_source(item, convert_to_openai_format=convert_to_openai_format)
            # 如果转换为 OpenAI 格式，类型会变化
            if convert_to_openai_format:
                logger.info(f"[ImageContentTransformer] Converted image to OpenAI format")
                return [normalized], True
            new_media_type = self._get_image_media_type(normalized)
            was_modified = original_media_type != new_media_type
            if was_modified:
                logger.debug(f"[ImageContentTransformer] Image modified: {original_media_type} -> {new_media_type}")
            return [normalized], was_modified or convert_to_openai_format

        # 处理 OpenAI 格式图片
        if item_type == "image_url" and "image_url" in item:
            original_media_type = self._get_image_media_type(item)
            normalized = self._normalize_image_source(item, convert_to_openai_format=convert_to_openai_format)
            new_media_type = self._get_image_media_type(normalized)
            was_modified = original_media_type != new_media_type
            if was_modified:
                logger.debug(f"[ImageContentTransformer] Image URL modified: {original_media_type} -> {new_media_type}")
            return [normalized], was_modified

        # 处理 tool_result（其 content 可能包含图片）
        # 重要：不要将图片从 tool_result 中拆分出来！
        # LiteLLM 的 translate_anthropic_messages_to_openai 原生支持在 tool_result.content
        # 列表中处理 type=image 的项，会将它们转换为 combined_content_parts。
        # 如果拆分出来，image_url 类型的独立块会被 LiteLLM 的翻译器静默丢弃，
        # 因为翻译器只处理 text/image/tool_result，不处理 image_url。
        if item_type == "tool_result" and "content" in item:
            tool_content = item.get("content")
            if isinstance(tool_content, list):
                import copy
                new_tool_result = copy.deepcopy(item)
                new_content = []
                was_modified = False
                has_text = False
                has_image = False

                for sub_item in tool_content:
                    if isinstance(sub_item, dict):
                        sub_type = sub_item.get("type", "")
                        if sub_type == "text":
                            has_text = True
                        if sub_type == "image" and "source" in sub_item:
                            has_image = True
                            # 只规范化 media_type，保持 Anthropic image 格式不变
                            # 不要转换为 OpenAI image_url 格式，让 LiteLLM 原生翻译处理
                            normalized_image = self._normalize_image_source(
                                sub_item, convert_to_openai_format=False
                            )
                            new_content.append(normalized_image)
                            original_mt = self._get_image_media_type(sub_item)
                            new_mt = self._get_image_media_type(normalized_image)
                            if original_mt != new_mt:
                                was_modified = True
                            logger.info(
                                f"[ImageContentTransformer] Normalized image in tool_result in-place, "
                                f"media_type: '{original_mt}' -> '{new_mt}', "
                                f"tool_use_id: {item.get('tool_use_id')}"
                            )
                        elif sub_type == "image_url" and "image_url" in sub_item:
                            has_image = True
                            normalized_image = self._normalize_image_source(sub_item)
                            new_content.append(normalized_image)
                            was_modified = True
                        else:
                            new_content.append(sub_item)
                    else:
                        new_content.append(sub_item)

                # Workaround for LiteLLM bug: when tool_result.content has
                # exactly 1 image item (no text), the translator puts the
                # data-URL as a plain string in content= instead of using
                # the list format with image_url objects. The model then
                # treats the base64 blob as text and hallucinates.
                # Fix: prepend a placeholder text item so the translator
                # takes the multi-item path which correctly uses list format.
                if has_image and not has_text:
                    new_content.insert(0, {"type": "text", "text": "[image]"})
                    was_modified = True
                    logger.warning(
                        f"[ImageContentTransformer] Inserted placeholder text into "
                        f"image-only tool_result to work around LiteLLM single-image bug, "
                        f"tool_use_id: {item.get('tool_use_id')}"
                    )

                new_tool_result["content"] = new_content
                return [new_tool_result], was_modified
            else:
                return [item], False

        return [item], False

    # 需要将图片转换为 OpenAI 格式的模型前缀列表
    # 这些模型通过 OpenRouter 路由到 Bedrock 等后端，可能有格式兼容性问题
    OPENROUTER_MODELS_REQUIRING_CONVERSION = [
        "openrouter/",           # 所有 OpenRouter 模型
        "openrouter_",           # 另一种 OpenRouter 前缀格式
        "anthropic/",            # Anthropic 模型通过 OpenRouter
        "amazon/",               # Amazon Bedrock 模型
        "bedrock/",              # Bedrock 模型
    ]

    # def _dump_image_messages(self, data: dict, call_type: str, phase: str):
    #     """Dump messages containing images to a file for debugging."""
    #     import json, os, copy
    #     from datetime import datetime
    #     messages = data.get("messages", [])
    #     has_image = False
    #     for msg in messages:
    #         content = msg.get("content", [])
    #         if isinstance(content, list):
    #             for item in content:
    #                 if isinstance(item, dict):
    #                     if item.get("type") in ("image", "image_url"):
    #                         has_image = True
    #                         break
    #                     if item.get("type") == "tool_result":
    #                         tc = item.get("content", [])
    #                         if isinstance(tc, list):
    #                             for sub in tc:
    #                                 if isinstance(sub, dict) and sub.get("type") in ("image", "image_url"):
    #                                     has_image = True
    #                                     break
    #                 if has_image:
    #                     break
    #         if has_image:
    #             break
    #     if not has_image:
    #         return
    #     def truncate_data(obj):
    #         if isinstance(obj, dict):
    #             result = {}
    #             for k, v in obj.items():
    #                 if k == "data" and isinstance(v, str) and len(v) > 200:
    #                     result[k] = v[:80] + f"...[TRUNCATED {len(v)} chars total]"
    #                 elif k == "url" and isinstance(v, str) and v.startswith("data:") and len(v) > 200:
    #                     result[k] = v[:80] + f"...[TRUNCATED {len(v)} chars total]"
    #                 else:
    #                     result[k] = truncate_data(v)
    #             return result
    #         elif isinstance(obj, list):
    #             return [truncate_data(v) for v in obj]
    #         return obj
    #     dump = {
    #         "timestamp": datetime.now().isoformat(),
    #         "phase": phase,
    #         "call_type": call_type,
    #         "model": data.get("model", "unknown"),
    #         "messages": truncate_data(copy.deepcopy(messages)),
    #     }
    #     dump_dir = "/tmp/litellm_image_dumps"
    #     os.makedirs(dump_dir, exist_ok=True)
    #     ts = datetime.now().strftime("%Y%m%d_%H%M%S_%f")
    #     dump_file = f"{dump_dir}/{ts}_{phase}_{call_type}.json"
    #     with open(dump_file, "w") as f:
    #         json.dump(dump, f, indent=2, default=str)
    #     logger.warning(f"[ImageContentTransformer] DUMPED image messages to {dump_file}")

    async def async_pre_call_hook( # type: ignore
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
            "anthropic_messages",
        ]
    ):
        """
        在调用LLM API之前，规范化所有图片的 media_type，确保符合 API 要求。

        处理范围：
        1. 所有角色的消息（user, assistant, system）
        2. Anthropic 格式图片（type=image）
        3. OpenAI 格式图片（type=image_url）
        4. tool_result 中嵌套的图片

        特殊处理：
        - 对于 OpenRouter 等需要格式转换的模型，将 Anthropic 格式图片转换为 OpenAI data URL 格式
        - 这解决了 OpenRouter -> Bedrock 的图片格式兼容性问题
        """
        model = data.get('model', 'unknown')
        logger.warning(f"[ImageContentTransformer] ========== PRE_CALL_HOOK START ==========")
        logger.warning(f"[ImageContentTransformer] call_type={call_type}, model={model}")

        try:
            # # Dump 包含图片的消息到文件以便调试
            # self._dump_image_messages(data, call_type, "pre_call")

            # 处理所有类型的调用
            if call_type not in ("anthropic_messages", "completion"):
                logger.warning(f"[ImageContentTransformer] Skipping call_type={call_type}")
                return data

            messages = data.get("messages", [])
            if not messages:
                logger.warning(f"[ImageContentTransformer] No messages found")
                return data

            # 检测是否需要将图片转换为 OpenAI 格式
            # 重要：当 call_type == "anthropic_messages" 时，消息仍为 Anthropic 格式，
            # 后续会经过 LiteLLM 的 translate_anthropic_messages_to_openai 翻译。
            # 如果在翻译前将 type="image" 转为 type="image_url"（OpenAI 格式），
            # 翻译器不认识 "image_url" 类型会静默丢弃图片！
            # 因此只在 call_type == "completion"（消息已是 OpenAI 格式）时才做格式转换。
            convert_to_openai_format = False
            if call_type != "anthropic_messages":
                for prefix in self.OPENROUTER_MODELS_REQUIRING_CONVERSION:
                    if model.lower().startswith(prefix.lower()):
                        convert_to_openai_format = True
                        logger.warning(
                            f"[ImageContentTransformer] Model '{model}' matches prefix '{prefix}', "
                            f"will convert images to OpenAI format"
                        )
                        break
            else:
                logger.warning(
                    f"[ImageContentTransformer] call_type=anthropic_messages, "
                    f"skipping OpenAI format conversion (will be handled by LiteLLM translator)"
                )

            logger.warning(f"[ImageContentTransformer] Processing {len(messages)} messages, convert_to_openai_format={convert_to_openai_format}")

            # # 调试：扫描所有图片并打印其格式和 media_type
            # for msg_idx, message in enumerate(messages):
            #     content = message.get("content", [])
            #     if isinstance(content, list):
            #         for item_idx, item in enumerate(content):
            #             if isinstance(item, dict):
            #                 item_type = item.get("type", "")
            #                 if item_type == "image":
            #                     source = item.get("source", {})
            #                     if isinstance(source, dict):
            #                         if "base64" in source and isinstance(source["base64"], dict):
            #                             mt = source["base64"].get("media_type", "NOT_FOUND")
            #                         else:
            #                             mt = source.get("media_type", "NOT_FOUND")
            #                     else:
            #                         mt = "SOURCE_NOT_DICT"
            #                     logger.warning(
            #                         f"[ImageContentTransformer] SCAN: messages[{msg_idx}].content[{item_idx}] "
            #                         f"type=image, source_keys={list(source.keys()) if isinstance(source, dict) else 'N/A'}, "
            #                         f"media_type='{mt}'"
            #                     )
            #                 elif item_type == "tool_result":
            #                     tool_content = item.get("content", [])
            #                     if isinstance(tool_content, list):
            #                         for sub_idx, sub_item in enumerate(tool_content):
            #                             if isinstance(sub_item, dict) and sub_item.get("type") == "image":
            #                                 sub_source = sub_item.get("source", {})
            #                                 if isinstance(sub_source, dict):
            #                                     if "base64" in sub_source and isinstance(sub_source["base64"], dict):
            #                                         mt = sub_source["base64"].get("media_type", "NOT_FOUND")
            #                                     else:
            #                                         mt = sub_source.get("media_type", "NOT_FOUND")
            #                                 else:
            #                                     mt = "SOURCE_NOT_DICT"
            #                                 logger.warning(
            #                                     f"[ImageContentTransformer] SCAN: messages[{msg_idx}].content[{item_idx}].content[{sub_idx}] "
            #                                     f"type=image (in tool_result), source_keys={list(sub_source.keys()) if isinstance(sub_source, dict) else 'N/A'}, "
            #                                     f"media_type='{mt}'"
            #                                 )

            modified = False
            image_count = 0

            for msg_idx, message in enumerate(messages):
                if "content" not in message:
                    continue

                content = message["content"]

                # content 可能是字符串或列表
                if isinstance(content, str):
                    continue

                if isinstance(content, list):
                    new_content = []
                    msg_modified = False

                    for item in content:
                        # _process_content_item 现在返回列表（因为 tool_result 可能被拆分）
                        processed_items, was_modified = self._process_content_item(
                            item,
                            convert_to_openai_format=convert_to_openai_format
                        )
                        new_content.extend(processed_items)  # 使用 extend 而不是 append
                        if was_modified:
                            msg_modified = True
                            image_count += 1

                    if msg_modified:
                        message["content"] = new_content
                        modified = True
                        logger.info(f"[ImageContentTransformer] Processed message {msg_idx} (role={message.get('role')})")

            if modified:
                logger.warning(f"[ImageContentTransformer] Normalized {image_count} image(s) media_type in messages")
            else:
                logger.warning(f"[ImageContentTransformer] No modifications made")

            logger.warning(f"[ImageContentTransformer] ========== PRE_CALL_HOOK END ==========")
            return data

        except Exception as e:
            logger.error(f"[ImageContentTransformer] Error in async_pre_call_hook: {e}", exc_info=True)
            # 出错时返回原始数据，避免中断请求
            return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败的hook"""
        logger.error(f"LLM call failed: {original_exception}")

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功的hook（非流式）"""
        pass

    async def async_moderation_hook( # type: ignore
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        call_type: Literal["completion", "embeddings", "image_generation", "moderation", "audio_transcription"],
    ):
        """与LLM调用并行运行的审核hook"""
        pass

    async def async_post_call_streaming_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        response: str,
    ):
        """处理流式响应的hook"""
        pass

    async def async_post_call_streaming_iterator_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        response: Any,
        request_data: dict,
    ) -> AsyncGenerator[ModelResponseStream, None]:
        """
        处理整个流的hook
        """
        async for item in response:
            yield item

class ToolResultContentFlattener(CustomLogger):
    """
    自定义回调处理器，用于将 tool_result 的 content 列表扁平化为字符串。

    解决错误: "each tool_use must have a single result. Found multiple tool_result blocks with id: xxx"

    问题根因：
    当 tool_result 的 content 是一个列表时，LiteLLM 的 translate_anthropic_messages_to_openai
    方法会为列表中的每个项目创建一个独立的 ChatCompletionToolMessage，每个都使用相同的 tool_use_id。
    当这些消息被转回 Anthropic 格式时，就会产生多个具有相同 tool_use_id 的 tool_result 块。

    解决方案：
    在 LiteLLM 转换之前，将 tool_result 的 content 列表合并为单个字符串，
    这样 LiteLLM 只会创建一个 ChatCompletionToolMessage。
    """

    def __init__(self):
        super().__init__()
        logger.info("ToolResultContentFlattener initialized")

    def _contains_image(self, content: Any) -> bool:
        """
        检查 tool_result 的 content 是否包含图片。

        Args:
            content: tool_result 的 content

        Returns:
            True 如果包含图片，False 否则
        """
        if not isinstance(content, list):
            return False

        for item in content:
            if isinstance(item, dict):
                item_type = item.get("type", "")
                if item_type == "image" or item_type == "image_url":
                    return True
        return False

    def _flatten_tool_result_content(self, content: Any) -> str:
        """
        将 tool_result 的 content 扁平化为字符串。

        注意：此方法只应在 content 不包含图片时调用。
        包含图片的 tool_result 应该跳过扁平化处理。

        Args:
            content: tool_result 的 content，可能是字符串、列表或 None

        Returns:
            扁平化后的字符串
        """
        if content is None:
            return ""
        elif isinstance(content, str):
            return content
        elif isinstance(content, list):
            parts = []
            for item in content:
                if isinstance(item, str):
                    parts.append(item)
                elif isinstance(item, dict):
                    item_type = item.get("type", "")
                    if item_type == "text":
                        text = item.get("text", "")
                        if text:
                            parts.append(text)
                    elif item_type == "image" or item_type == "image_url":
                        # 不应该到达这里，因为调用前应该检查是否包含图片
                        logger.warning("[ToolResultContentFlattener] Unexpected image in flatten, skipping")
                        continue
                    else:
                        # 其他类型，尝试转为字符串
                        parts.append(str(item))
                else:
                    parts.append(str(item))
            return "".join(parts) if parts else ""
        else:
            return str(content)

    def _process_message_content(self, content: list) -> tuple[list, bool]:
        """
        处理消息的 content 列表，将其中 tool_result 的 content 列表扁平化。

        注意：包含图片的 tool_result 会被跳过，不进行扁平化处理。
        这样可以保留图片内容，由 ImageContentTransformer 单独处理。

        Args:
            content: 消息的 content 列表

        Returns:
            (处理后的 content 列表, 是否进行了修改)
        """
        if not isinstance(content, list):
            return content, False

        modified = False
        new_content = []

        for item in content:
            if isinstance(item, dict) and item.get("type") == "tool_result":
                tool_content = item.get("content")
                # 只有当 content 是列表时才需要处理
                if isinstance(tool_content, list):
                    # 检查是否包含图片，如果包含则跳过扁平化
                    if self._contains_image(tool_content):
                        logger.info(
                            f"[ToolResultContentFlattener] Skipping tool_result with images, "
                            f"tool_use_id: {item.get('tool_use_id')}"
                        )
                        new_content.append(item)
                    else:
                        # 不包含图片，可以安全扁平化
                        flattened = self._flatten_tool_result_content(tool_content)
                        new_item = dict(item)
                        new_item["content"] = flattened
                        new_content.append(new_item)
                        modified = True
                        logger.info(
                            f"[ToolResultContentFlattener] Flattened tool_result content for "
                            f"tool_use_id: {item.get('tool_use_id')}"
                        )
                else:
                    new_content.append(item)
            else:
                new_content.append(item)

        return new_content, modified

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
            "anthropic_messages",
        ]
    ):
        """
        在调用 LLM API 之前，将 tool_result 的 content 列表扁平化为字符串。

        这可以防止 LiteLLM 的 translate_anthropic_messages_to_openai 方法
        为列表中的每个项目创建独立的 ChatCompletionToolMessage。
        """
        logger.info(f"[ToolResultContentFlattener] call_type={call_type}")

        try:
            # 处理 anthropic_messages 和 completion 类型的请求
            if call_type not in ("anthropic_messages", "completion"):
                return data

            messages = data.get("messages", [])
            if not messages:
                return data

            logger.info(f"[ToolResultContentFlattener] Processing {len(messages)} messages")
            modified = False

            for idx, message in enumerate(messages):
                # 检查是否是 user 消息，其 content 中可能包含 tool_result
                if message.get("role") == "user" and "content" in message:
                    content = message["content"]

                    # content 可能是字符串或列表
                    if isinstance(content, list):
                        new_content, was_modified = self._process_message_content(content)
                        if was_modified:
                            message["content"] = new_content
                            modified = True

            if modified:
                logger.info(f"[ToolResultContentFlattener] Flattened tool_result content in user messages")

            return data

        except Exception as e:
            logger.error(f"Error in ToolResultContentFlattener.async_pre_call_hook: {e}", exc_info=True)
            # 出错时返回原始数据，避免中断请求
            return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败的 hook"""
        pass

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功的 hook（非流式）"""
        pass


class EmptyChunkFilter(CustomLogger):
    """
    过滤流式响应中的空 chunk，防止空 chunk 被累计计入 output token 统计。

    问题根因：
    在某些情况下，OpenRouter 或 LiteLLM 在转发流式响应时会产生大量空的 delta chunk。
    这些空 chunk 虽然没有实际内容，但可能被 Claude Agent SDK 累计计入 output token，
    导致 "Claude's response exceeded the 32000 output token maximum" 错误。

    解决方案：
    在流式响应的迭代器 hook 中过滤掉没有实际内容的 chunk，只转发有效的 chunk。

    注意：
    - 此过滤器仅对 ENABLED_MODELS 列表中的模型生效
    - 其他模型不进行过滤，直接透传所有 chunk
    """

    # 启用过滤的模型列表（只有这些模型会触发空 chunk 过滤）
    ENABLED_MODELS = [
        "glm-4.6",
        "eval-doubao-dev",
        "doubao-dev-1215-v2-builder-json-nfc",
    ]

    def __init__(self):
        super().__init__()
        self.filtered_count = 0
        self.passed_count = 0
        logger.info(f"EmptyChunkFilter initialized - enabled_models: {self.ENABLED_MODELS}")

    def _has_content(self, chunk: Any) -> bool:
        """
        检查 chunk 是否包含实际内容。

        Args:
            chunk: 流式响应的 chunk

        Returns:
            True 如果 chunk 有实际内容，False 如果是空 chunk
        """
        try:
            # 检查 OpenAI 格式的流式响应
            if hasattr(chunk, 'choices') and chunk.choices:
                has_any_content = False
                for choice in chunk.choices:
                    delta = getattr(choice, 'delta', None)
                    if delta:
                        # 检查是否有文本内容（必须非空）
                        content = getattr(delta, 'content', None)
                        if content is not None and content != '':
                            has_any_content = True
                            break

                        # 检查 reasoning_content（doubao 等推理模型）
                        reasoning_content = getattr(delta, 'reasoning_content', None)
                        if reasoning_content is not None and reasoning_content != '':
                            has_any_content = True
                            break

                        # 检查是否有 tool_calls
                        tool_calls = getattr(delta, 'tool_calls', None)
                        if tool_calls:
                            has_any_content = True
                            break

                        # 检查是否有 function_call (旧格式)
                        function_call = getattr(delta, 'function_call', None)
                        if function_call:
                            has_any_content = True
                            break

                        # 检查是否有 role (第一个 chunk 通常只有 role)
                        # 只有当这是真正的第一个 chunk 时才保留
                        role = getattr(delta, 'role', None)
                        if role:
                            has_any_content = True
                            break

                    # 检查 finish_reason (最后一个 chunk)
                    finish_reason = getattr(choice, 'finish_reason', None)
                    if finish_reason:
                        has_any_content = True
                        break

                # 如果检查了 choices 但没有任何内容，则过滤掉
                return has_any_content

            # 检查是否是 Anthropic 格式
            if hasattr(chunk, 'type'):
                chunk_type = chunk.type
                # 这些类型的 chunk 都应该保留
                if chunk_type in ['message_start', 'content_block_start', 'content_block_delta',
                                  'content_block_stop', 'message_delta', 'message_stop']:
                    # 对于 content_block_delta，进一步检查是否有实际内容
                    if chunk_type == 'content_block_delta':
                        delta = getattr(chunk, 'delta', None)
                        if delta:
                            text = getattr(delta, 'text', None)
                            if text is not None and text != '':
                                return True
                            # 检查 tool_use 的 partial_json
                            partial_json = getattr(delta, 'partial_json', None)
                            if partial_json:
                                return True
                        return False
                    return True

            # 如果没有 choices 且没有 type，说明是异常格式，过滤掉
            logger.debug(f"[EmptyChunkFilter] Unknown chunk format, filtering: {type(chunk)}")
            return False

        except Exception as e:
            logger.warning(f"[EmptyChunkFilter] Error checking chunk content: {e}")
            # 出错时保留 chunk（保守策略）
            return True

    async def async_post_call_streaming_iterator_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        response: Any,
        request_data: dict,
    ) -> AsyncGenerator[ModelResponseStream, None]:
        """
        过滤流式响应中的空 chunk。
        仅对 ENABLED_MODELS 列表中的模型生效。
        """
        # 检查模型是否在启用列表中
        model = request_data.get("model", "")
        model_enabled = any(enabled in model for enabled in self.ENABLED_MODELS)

        if not model_enabled:
            # 模型不在启用列表中，直接透传所有 chunk
            async for chunk in response:
                yield chunk
            return

        filtered_in_request = 0
        passed_in_request = 0

        try:
            async for chunk in response:
                if self._has_content(chunk):
                    passed_in_request += 1
                    yield chunk
                else:
                    filtered_in_request += 1
                    # 每过滤 100 个空 chunk 记录一次日志
                    if filtered_in_request % 100 == 0:
                        logger.warning(
                            f"[EmptyChunkFilter] Filtered {filtered_in_request} empty chunks so far "
                            f"(passed: {passed_in_request})"
                        )
        finally:
            # 请求结束时记录统计
            self.filtered_count += filtered_in_request
            self.passed_count += passed_in_request

            if filtered_in_request > 0:
                logger.info(
                    f"[EmptyChunkFilter] Request completed - "
                    f"filtered: {filtered_in_request}, passed: {passed_in_request}, "
                    f"total filtered: {self.filtered_count}, total passed: {self.passed_count}"
                )

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
            "anthropic_messages",
        ]
    ):
        """Pre-call hook - 不做处理"""
        return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败的 hook"""
        pass

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功的 hook（非流式）"""
        pass


class OutputTokenPrefillContinuation(CustomLogger):
    """
    Output Token Prefill 续接处理器。

    当检测到输出 token 接近上限时，截断当前响应并使用 Prefill 机制发起续接请求，
    然后将多次响应合并返回给客户端。

    工作原理：
    1. 监控流式响应中累计的文本长度（估算 token 数）
    2. 当接近阈值时，记录已生成的内容
    3. 截断当前响应，构造带 Prefill 的续接请求
    4. 将续接响应无缝衔接到原响应流中（跳过第一个 role chunk）

    注意：
    - 每轮续接只统计该轮新生成的 token，不累计之前的
    - 续接响应的第一个 chunk（通常只有 role）会被跳过
    """

    # 配置参数
    # 估算：约 3-4 个字符 = 1 个 token（用于日志统计）
    CHARS_PER_TOKEN = 4.0
    # 主动触发续接的 token 阈值（当输出达到此值时主动截断并续接）
    # Claude Opus 4.5 的 output token 上限是 32000，设置为 28000 留出安全余量
    PROACTIVE_CUTOFF_THRESHOLD = 28000
    # 最大续接轮数（每轮最多 16000 token，5 轮 = 80000 token）
    MAX_CONTINUATION_ROUNDS = 5
    # 启用续接的模型列表（只有这些模型会触发续接机制）
    # 只对 opus_auto_prefill 模型生效，触发项目中的 Context Auto-Continue Skill
    ENABLED_MODELS = [
        "opus_auto_prefill",  # Claude Opus 4.5 with auto prefill continuation
    ]

    def __init__(self):
        super().__init__()
        self.continuation_count = 0
        logger.info(
            f"OutputTokenPrefillContinuation initialized - "
            f"cutoff_threshold: {self.PROACTIVE_CUTOFF_THRESHOLD}, "
            f"max_rounds: {self.MAX_CONTINUATION_ROUNDS}, "
            f"enabled_models: {self.ENABLED_MODELS}"
        )

    def _estimate_tokens(self, text: str) -> int:
        """估算文本的 token 数"""
        return int(len(text) / self.CHARS_PER_TOKEN)

    def _extract_text_from_chunk(self, chunk: Any) -> str:
        """从 chunk 中提取文本内容"""
        try:
            # OpenAI 格式
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    delta = getattr(choice, 'delta', None)
                    if delta:
                        # 首先检查 content（标准输出）
                        content = getattr(delta, 'content', None)
                        if content:
                            return content
                        # 然后检查 reasoning_content（推理模型的思考过程，如 doubao）
                        reasoning_content = getattr(delta, 'reasoning_content', None)
                        if reasoning_content:
                            return reasoning_content

            # Anthropic 格式
            if hasattr(chunk, 'type') and chunk.type == 'content_block_delta':
                delta = getattr(chunk, 'delta', None)
                if delta:
                    text = getattr(delta, 'text', None)
                    if text:
                        return text

            return ""
        except Exception:
            return ""

    def _is_first_chunk_with_role_only(self, chunk: Any) -> bool:
        """检查是否是只包含 role 的第一个 chunk（续接时需要跳过）"""
        try:
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    delta = getattr(choice, 'delta', None)
                    if delta:
                        role = getattr(delta, 'role', None)
                        content = getattr(delta, 'content', None)
                        # 只有 role，没有 content
                        if role and not content:
                            return True
            return False
        except Exception:
            return False

    def _smart_truncate(self, content: str) -> str:
        """智能截断，尽量在合适的边界截断，同时保留尽可能多的内容用于 Prefill"""
        if not content:
            return content

        # 对于 Prefill，我们希望保留尽可能多的内容（至少 90%）
        min_pos = int(len(content) * 0.9)

        # 按优先级查找截断点（找到第一个有效的就使用）
        truncation_markers = [
            "```\n",      # 代码块结束（最佳）
            "\n\n",       # 段落结束
            "。\n",       # 中文句子
            ".\n",        # 英文句子
            "。",         # 中文句号
            ". ",         # 英文句号
            "\n",         # 换行
        ]

        for marker in truncation_markers:
            pos = content.rfind(marker, min_pos)
            if pos > 0:
                return content[:pos + len(marker)]

        # 找不到好的截断点，直接返回原内容（不截断）
        return content

    def _build_continuation_messages(
        self,
        original_messages: list,
        all_accumulated_text: str,
        model: str
    ) -> list:
        """
        构建带 Prefill 的续接消息。

        将已生成的全部输出作为 assistant 消息添加到消息列表末尾，
        模型会从这个位置继续生成。

        注意：
        - 对于 doubao 等推理模型，reasoning_content 是思考过程，最终会输出到 content
        - Prefill 时统一使用 content 字段，因为这是标准的 assistant 消息格式
        - 添加续接提示词，让模型知道需要继续之前的输出
        """
        import copy

        # 深拷贝原始消息
        messages = copy.deepcopy(original_messages)

        # 智能截断（用于 Prefill）
        truncated_output = self._smart_truncate(all_accumulated_text)

        # 添加 assistant 的部分响应作为 Prefill
        # 使用 content 字段，这是标准格式
        messages.append({
            "role": "assistant",
            "content": truncated_output
        })

        # 添加 user 消息触发上下文自动续接 skill
        # 这只是触发器，实际的续接逻辑由 .claude/skills/context-auto-continue/skill.md 定义
        messages.append({
            "role": "user",
            "content": """⚠️ 输出 token 已达到阈值（约 28000 tokens），请立即执行上下文自动续接。

请按照「上下文自动续接技能 (Context Auto-Continue Skill) v2.0」执行：

1. **确定任务 ID**: 检查 CONTINUATION_TASK_ID 环境变量，或根据当前任务生成新 ID
2. **生成结构化状态**: 保存到 `eval_results/meta-learning/sessions/{task-id}/state.md`
3. **保存续接记录**: 保存到 `continuations/` 目录
4. **更新任务索引**: 更新 `eval_results/meta-learning/index.json`
5. **启动新窗口**: 执行 `~/.claude/scripts/context-continue.sh`

详细流程参见: `.claude/skills/context-auto-continue/skill.md`

请立即开始执行续接流程，不要继续当前输出。"""
        })

        logger.info(
            f"[PrefillContinuation] Built continuation messages - "
            f"total_accumulated_length: {len(all_accumulated_text)}, "
            f"truncated_for_prefill: {len(truncated_output)}, "
            f"total_messages: {len(messages)}"
        )

        return messages, truncated_output

    async def _make_continuation_request(
        self,
        model: str,
        messages: list,
        original_request: dict,
    ):
        """
        发起续接请求。

        通过 HTTP 请求发送到 LiteLLM proxy，而不是直接调用 litellm.acompletion()。
        这样可以确保请求经过 proxy 的路由，支持 eval-doubao-dev 等需要 proxy 路由的模型。
        """
        import aiohttp
        import json
        import os

        # LiteLLM proxy 地址
        proxy_url = os.environ.get("LITELLM_PROXY_URL", "http://0.0.0.0:4000")
        endpoint = f"{proxy_url}/chat/completions"

        # 构建请求参数
        # 添加 x-prefill-continuation 标记，让 proxy 识别这是续接请求，避免重复触发续接
        request_body = {
            "model": model,
            "messages": messages,
            "stream": True,
            "metadata": {
                "x-prefill-continuation": True
            }
        }

        # 保留原始请求的关键参数
        preserve_keys = [
            "temperature", "max_tokens", "top_p", "top_k",
            "stop", "presence_penalty", "frequency_penalty",
            "system",  # system prompt（某些 API 格式会单独传递）
            "tools",   # 工具定义
            "tool_choice",  # 工具选择策略
        ]
        for key in preserve_keys:
            if key in original_request and original_request[key] is not None:
                request_body[key] = original_request[key]

        logger.info(f"[PrefillContinuation] Making continuation request via proxy: {endpoint}, model: {model}")

        # 使用 aiohttp 发送 HTTP 请求
        headers = {
            "Content-Type": "application/json",
        }

        # 如果有 API key，添加到 header
        api_key = os.environ.get("LITELLM_API_KEY") or os.environ.get("OPENAI_API_KEY")
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"

        async def stream_response():
            """流式读取 SSE 响应并解析为 chunk 对象"""
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    endpoint,
                    json=request_body,
                    headers=headers,
                    timeout=aiohttp.ClientTimeout(total=600)  # 10 分钟超时
                ) as resp:
                    if resp.status != 200:
                        error_text = await resp.text()
                        raise Exception(f"Continuation request failed with status {resp.status}: {error_text}")

                    # 读取 SSE 流
                    async for line in resp.content:
                        line = line.decode('utf-8').strip()
                        if not line:
                            continue
                        if line.startswith('data: '):
                            data = line[6:]  # 去掉 'data: ' 前缀
                            if data == '[DONE]':
                                break
                            try:
                                chunk_dict = json.loads(data)
                                # 转换为 ModelResponseStream 对象
                                chunk = self._dict_to_model_response_stream(chunk_dict)
                                yield chunk
                            except json.JSONDecodeError as e:
                                logger.warning(f"[PrefillContinuation] Failed to parse SSE data: {e}, data: {data}")
                                continue

        return stream_response()

    def _dict_to_model_response_stream(self, chunk_dict: dict) -> Any:
        """将字典转换为 ModelResponseStream 类似的对象"""
        from litellm.types.utils import ModelResponseStream, StreamingChoices, Delta

        try:
            choices = []
            for choice_dict in chunk_dict.get("choices", []):
                delta_dict = choice_dict.get("delta", {})
                delta = Delta(
                    content=delta_dict.get("content"),
                    role=delta_dict.get("role"),
                    tool_calls=delta_dict.get("tool_calls"),
                    function_call=delta_dict.get("function_call"),
                )
                # 添加 reasoning_content 支持（doubao 模型）
                if "reasoning_content" in delta_dict:
                    delta.reasoning_content = delta_dict.get("reasoning_content")

                choice = StreamingChoices(
                    index=choice_dict.get("index", 0),
                    delta=delta,
                    finish_reason=choice_dict.get("finish_reason"),
                )
                choices.append(choice)

            response = ModelResponseStream(
                id=chunk_dict.get("id", ""),
                choices=choices,
                created=chunk_dict.get("created", 0),
                model=chunk_dict.get("model", ""),
                object=chunk_dict.get("object", "chat.completion.chunk"),
            )
            return response
        except Exception as e:
            logger.warning(f"[PrefillContinuation] Failed to convert chunk dict: {e}")
            # 返回一个简单的对象，让上层能够继续处理
            class SimpleChunk:
                def __init__(self, data):
                    self.choices = []
                    if "choices" in data:
                        for c in data["choices"]:
                            self.choices.append(SimpleChoice(c))

            class SimpleChoice:
                def __init__(self, data):
                    self.delta = SimpleDelta(data.get("delta", {}))
                    self.finish_reason = data.get("finish_reason")
                    self.index = data.get("index", 0)

            class SimpleDelta:
                def __init__(self, data):
                    self.content = data.get("content")
                    self.role = data.get("role")
                    self.reasoning_content = data.get("reasoning_content")
                    self.tool_calls = data.get("tool_calls")
                    self.function_call = data.get("function_call")

            return SimpleChunk(chunk_dict)

    def _is_model_enabled(self, model: str) -> bool:
        """检查模型是否启用续接功能"""
        if not model:
            return False
        # 检查模型名称是否匹配（支持部分匹配，如 "openrouter/glm-4.6" 也能匹配 "glm-4.6"）
        for enabled_model in self.ENABLED_MODELS:
            if enabled_model in model:
                return True
        return False

    def _get_finish_reason(self, chunk: Any) -> Optional[str]:
        """从 chunk 中提取 finish_reason"""
        try:
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    finish_reason = getattr(choice, 'finish_reason', None)
                    if finish_reason:
                        return finish_reason
            return None
        except Exception:
            return None

    async def async_post_call_streaming_iterator_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        response: Any,
        request_data: dict,
    ) -> AsyncGenerator[ModelResponseStream, None]:
        """
        监控流式响应，主动在 token 达到阈值时触发续接。
        仅对 ENABLED_MODELS 中的模型生效。

        工作原理：
        1. 实时统计输出 token 数量
        2. 当达到 PROACTIVE_CUTOFF_THRESHOLD 阈值时，主动截断并发起续接
        3. 当检测到 finish_reason='length' 时，也触发续接（备用逻辑）
        4. 将续接响应无缝衔接到原响应流中
        """
        model = request_data.get("model", "")

        # 检查是否是续接请求（避免递归）
        metadata = request_data.get("metadata", {}) or {}
        if metadata.get("x-prefill-continuation"):
            logger.debug("[PrefillContinuation] Skipping continuation request (already a continuation)")
            async for chunk in response:
                yield chunk
            return

        # 检查模型是否启用续接功能
        if not self._is_model_enabled(model):
            # 不启用续接，直接透传所有 chunk
            async for chunk in response:
                yield chunk
            return

        # 全部累计的文本（用于构建 Prefill）
        all_accumulated_text = ""

        continuation_round = 0
        original_messages = request_data.get("messages", [])

        current_response = response
        is_continuation = False  # 标记是否是续接响应

        while True:
            should_continue = False
            last_finish_reason = None
            first_chunk_skipped = False  # 续接时跳过第一个只有 role 的 chunk

            try:
                async for chunk in current_response:
                    # 续接响应：跳过第一个只有 role 的 chunk
                    if is_continuation and not first_chunk_skipped:
                        if self._is_first_chunk_with_role_only(chunk):
                            first_chunk_skipped = True
                            logger.debug("[PrefillContinuation] Skipped first role-only chunk in continuation")
                            continue
                        first_chunk_skipped = True  # 即使不是 role-only，也标记为已处理

                    # 提取文本内容
                    text = self._extract_text_from_chunk(chunk)
                    if text:
                        all_accumulated_text += text

                    # 主动监控：检查是否达到 token 阈值
                    current_tokens = self._estimate_tokens(all_accumulated_text)
                    if current_tokens >= self.PROACTIVE_CUTOFF_THRESHOLD:
                        if continuation_round < self.MAX_CONTINUATION_ROUNDS:
                            should_continue = True
                            continuation_round += 1
                            self.continuation_count += 1

                            logger.warning(
                                f"[PrefillContinuation] Proactive truncation triggered - "
                                f"token threshold reached ({current_tokens} >= {self.PROACTIVE_CUTOFF_THRESHOLD}), "
                                f"initiating continuation round {continuation_round}"
                            )
                            # 输出当前 chunk 后停止，进行续接
                            yield chunk
                            break
                        else:
                            logger.warning(
                                f"[PrefillContinuation] Max continuation rounds reached ({self.MAX_CONTINUATION_ROUNDS}), "
                                f"continuing without truncation despite threshold"
                            )

                    # 检查 finish_reason（备用逻辑）
                    finish_reason = self._get_finish_reason(chunk)
                    if finish_reason:
                        last_finish_reason = finish_reason

                        # 如果是 'length' 结束且还有续接配额，触发续接
                        # 不 yield 这个 chunk，否则 Claude SDK 会收到 stop_reason=max_tokens 并报错
                        if finish_reason == 'length' and continuation_round < self.MAX_CONTINUATION_ROUNDS:
                            should_continue = True
                            continuation_round += 1
                            self.continuation_count += 1

                            logger.warning(
                                f"[PrefillContinuation] Detected truncation (finish_reason=length), "
                                f"initiating continuation round {continuation_round} - "
                                f"current_tokens_estimate: {current_tokens}, "
                                f"NOT yielding this chunk to avoid SDK error"
                            )
                            # 不 yield 这个带有 finish_reason='length' 的 chunk，直接 break 去续接
                            break
                        else:
                            # 正常结束或已达最大续接次数，输出 chunk
                            yield chunk
                    else:
                        # 正常输出 chunk（没有 finish_reason）
                        yield chunk

                # 检查是否需要续接
                if not should_continue:
                    # 正常结束
                    if continuation_round > 0:
                        total_tokens = self._estimate_tokens(all_accumulated_text)
                        logger.info(
                            f"[PrefillContinuation] Completed with {continuation_round} continuation(s) - "
                            f"total_tokens_estimate: {total_tokens}, finish_reason: {last_finish_reason}"
                        )
                    break

                # 发起续接请求
                continuation_messages, prefill_text = self._build_continuation_messages(
                    original_messages,
                    all_accumulated_text,
                    model
                )

                try:
                    current_response = await self._make_continuation_request(
                        model,
                        continuation_messages,
                        request_data
                    )
                    is_continuation = True
                    logger.info(f"[PrefillContinuation] Continuation request succeeded, starting round {continuation_round}")

                except Exception as e:
                    logger.error(
                        f"[PrefillContinuation] Continuation request failed: {e}, "
                        f"round: {continuation_round}, accumulated_length: {len(all_accumulated_text)}",
                        exc_info=True
                    )
                    # 续接失败，返回已有内容，不抛出异常
                    break

            except Exception as e:
                logger.error(f"[PrefillContinuation] Error during streaming: {e}", exc_info=True)
                raise

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
            "anthropic_messages",
        ]
    ):
        """Pre-call hook - 不做处理"""
        return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败的 hook"""
        pass

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功的 hook（非流式）"""
        pass


class ProactiveStreamTruncation(CustomLogger):
    """
    主动流式截断处理器。

    解决问题：
    Claude Code SDK 有 32000/64000 output token 硬限制，当输出超过限制时会立即报错：
    "Claude's response exceeded the 32000 output token maximum"

    原有的 OutputTokenPrefillContinuation 只在 finish_reason='length' 时触发续接，
    但 Claude Code SDK 会在达到限制时立即中断流，导致续接机制来不及触发。

    解决方案：
    在流式输出过程中**主动监控 token 数量**，当接近阈值时：
    1. 主动停止输出当前流
    2. 构造续接请求，让模型继续生成
    3. 将续接的内容无缝拼接到原流中

    注意：
    - 此处理器应该放在回调链的最后，在其他处理器之后执行
    - 对所有模型生效（不限于特定模型列表）
    """

    # 配置参数
    # 估算：约 3-4 个字符 = 1 个 token
    CHARS_PER_TOKEN = 4.0

    # 主动截断的 token 阈值（必须低于 Claude Code SDK 的 32000 限制）
    # 设置为 28000，留出安全余量
    PROACTIVE_CUTOFF_THRESHOLD = 28000

    # 最大续接轮数
    MAX_CONTINUATION_ROUNDS = 10

    # 启用主动截断的模型列表
    # 仅对配置的模型启用主动截断和续接机制
    # 设置为空列表表示禁用此功能，不对任何模型生效
    ENABLED_MODELS = []

    def __init__(self):
        super().__init__()
        self.truncation_count = 0
        logger.info(
            f"ProactiveStreamTruncation initialized - "
            f"cutoff_threshold: {self.PROACTIVE_CUTOFF_THRESHOLD} tokens, "
            f"max_rounds: {self.MAX_CONTINUATION_ROUNDS}, "
            f"enabled_models: {'all' if self.ENABLED_MODELS is None else self.ENABLED_MODELS}"
        )

    def _estimate_tokens(self, text: str) -> int:
        """估算文本的 token 数"""
        return int(len(text) / self.CHARS_PER_TOKEN)

    def _extract_text_from_chunk(self, chunk: Any) -> str:
        """从 chunk 中提取文本内容"""
        try:
            # OpenAI 格式
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    delta = getattr(choice, 'delta', None)
                    if delta:
                        content = getattr(delta, 'content', None)
                        if content:
                            return content
                        # 支持 reasoning_content（doubao 等推理模型）
                        reasoning_content = getattr(delta, 'reasoning_content', None)
                        if reasoning_content:
                            return reasoning_content

            # Anthropic 格式
            if hasattr(chunk, 'type') and chunk.type == 'content_block_delta':
                delta = getattr(chunk, 'delta', None)
                if delta:
                    text = getattr(delta, 'text', None)
                    if text:
                        return text

            return ""
        except Exception:
            return ""

    def _is_first_chunk_with_role_only(self, chunk: Any) -> bool:
        """检查是否是只包含 role 的第一个 chunk"""
        try:
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    delta = getattr(choice, 'delta', None)
                    if delta:
                        role = getattr(delta, 'role', None)
                        content = getattr(delta, 'content', None)
                        if role and not content:
                            return True
            return False
        except Exception:
            return False

    def _get_finish_reason(self, chunk: Any) -> Optional[str]:
        """从 chunk 中提取 finish_reason"""
        try:
            if hasattr(chunk, 'choices') and chunk.choices:
                for choice in chunk.choices:
                    finish_reason = getattr(choice, 'finish_reason', None)
                    if finish_reason:
                        return finish_reason
            return None
        except Exception:
            return None

    def _smart_truncate(self, content: str) -> str:
        """智能截断，在合适的边界截断"""
        if not content:
            return content

        # 保留至少 90% 的内容
        min_pos = int(len(content) * 0.9)

        # 按优先级查找截断点
        truncation_markers = [
            "```\n",      # 代码块结束
            "\n\n",       # 段落结束
            "。\n",       # 中文句子
            ".\n",        # 英文句子
            "。",         # 中文句号
            ". ",         # 英文句号
            "\n",         # 换行
        ]

        for marker in truncation_markers:
            pos = content.rfind(marker, min_pos)
            if pos > 0:
                return content[:pos + len(marker)]

        return content

    def _is_model_enabled(self, model: str) -> bool:
        """检查模型是否启用主动截断"""
        if self.ENABLED_MODELS is None:
            return True  # 对所有模型生效
        if not model:
            return False
        for enabled_model in self.ENABLED_MODELS:
            if enabled_model in model:
                return True
        return False

    def _build_continuation_messages(
        self,
        original_messages: list,
        accumulated_text: str,
    ) -> list:
        """构建续接消息"""
        import copy

        messages = copy.deepcopy(original_messages)
        truncated_output = self._smart_truncate(accumulated_text)

        # 添加 assistant 的部分响应
        messages.append({
            "role": "assistant",
            "content": truncated_output
        })

        # 添加续接提示
        messages.append({
            "role": "user",
            "content": "请继续，从你上次停止的地方继续输出，不要重复已经输出的内容。"
        })

        logger.info(
            f"[ProactiveTruncation] Built continuation - "
            f"accumulated: {len(accumulated_text)}, truncated: {len(truncated_output)}"
        )

        return messages, truncated_output

    async def _make_continuation_request(
        self,
        model: str,
        messages: list,
        original_request: dict,
    ):
        """发起续接请求"""
        import aiohttp
        import json
        import os

        proxy_url = os.environ.get("LITELLM_PROXY_URL", "http://0.0.0.0:4000")
        endpoint = f"{proxy_url}/chat/completions"

        request_body = {
            "model": model,
            "messages": messages,
            "stream": True,
            "metadata": {
                "x-proactive-truncation": True  # 标记为续接请求，避免递归
            }
        }

        # 保留原始请求参数
        preserve_keys = [
            "temperature", "max_tokens", "top_p", "top_k",
            "stop", "presence_penalty", "frequency_penalty",
            "system", "tools", "tool_choice",
        ]
        for key in preserve_keys:
            if key in original_request and original_request[key] is not None:
                request_body[key] = original_request[key]

        logger.info(f"[ProactiveTruncation] Making continuation request to {endpoint}")

        headers = {"Content-Type": "application/json"}
        api_key = os.environ.get("LITELLM_API_KEY") or os.environ.get("OPENAI_API_KEY")
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"

        async def stream_response():
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    endpoint,
                    json=request_body,
                    headers=headers,
                    timeout=aiohttp.ClientTimeout(total=600)
                ) as resp:
                    if resp.status != 200:
                        error_text = await resp.text()
                        raise Exception(f"Continuation failed: {resp.status} - {error_text}")

                    async for line in resp.content:
                        line = line.decode('utf-8').strip()
                        if not line:
                            continue
                        if line.startswith('data: '):
                            data = line[6:]
                            if data == '[DONE]':
                                break
                            try:
                                chunk_dict = json.loads(data)
                                chunk = self._dict_to_chunk(chunk_dict)
                                yield chunk
                            except json.JSONDecodeError:
                                continue

        return stream_response()

    def _dict_to_chunk(self, chunk_dict: dict) -> Any:
        """将字典转换为 chunk 对象"""
        from litellm.types.utils import ModelResponseStream, StreamingChoices, Delta

        try:
            choices = []
            for choice_dict in chunk_dict.get("choices", []):
                delta_dict = choice_dict.get("delta", {})
                delta = Delta(
                    content=delta_dict.get("content"),
                    role=delta_dict.get("role"),
                    tool_calls=delta_dict.get("tool_calls"),
                    function_call=delta_dict.get("function_call"),
                )
                if "reasoning_content" in delta_dict:
                    delta.reasoning_content = delta_dict.get("reasoning_content")

                choice = StreamingChoices(
                    index=choice_dict.get("index", 0),
                    delta=delta,
                    finish_reason=choice_dict.get("finish_reason"),
                )
                choices.append(choice)

            return ModelResponseStream(
                id=chunk_dict.get("id", ""),
                choices=choices,
                created=chunk_dict.get("created", 0),
                model=chunk_dict.get("model", ""),
                object=chunk_dict.get("object", "chat.completion.chunk"),
            )
        except Exception as e:
            logger.warning(f"[ProactiveTruncation] Failed to convert chunk: {e}")
            # 返回简单对象
            class SimpleChunk:
                def __init__(self, data):
                    self.choices = [SimpleChoice(c) for c in data.get("choices", [])]

            class SimpleChoice:
                def __init__(self, data):
                    self.delta = SimpleDelta(data.get("delta", {}))
                    self.finish_reason = data.get("finish_reason")
                    self.index = data.get("index", 0)

            class SimpleDelta:
                def __init__(self, data):
                    self.content = data.get("content")
                    self.role = data.get("role")
                    self.reasoning_content = data.get("reasoning_content")
                    self.tool_calls = data.get("tool_calls")
                    self.function_call = data.get("function_call")

            return SimpleChunk(chunk_dict)

    async def async_post_call_streaming_iterator_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        response: Any,
        request_data: dict,
    ) -> AsyncGenerator[ModelResponseStream, None]:
        """
        主动监控流式响应，在接近 token 限制时主动截断并续接。

        工作流程：
        1. 实时统计输出 token 数
        2. 当达到阈值时，停止当前流
        3. 发起续接请求
        4. 将续接内容无缝拼接
        """
        model = request_data.get("model", "")

        # 检查是否是续接请求（避免递归）
        metadata = request_data.get("metadata", {}) or {}
        if metadata.get("x-proactive-truncation"):
            logger.debug("[ProactiveTruncation] Skipping (already a continuation)")
            async for chunk in response:
                yield chunk
            return

        # 检查模型是否启用
        if not self._is_model_enabled(model):
            async for chunk in response:
                yield chunk
            return

        accumulated_text = ""
        continuation_round = 0
        original_messages = request_data.get("messages", [])
        current_response = response
        is_continuation = False

        while True:
            should_continue = False
            first_chunk_skipped = False

            try:
                async for chunk in current_response:
                    # 续接响应：跳过第一个只有 role 的 chunk
                    if is_continuation and not first_chunk_skipped:
                        if self._is_first_chunk_with_role_only(chunk):
                            first_chunk_skipped = True
                            continue
                        first_chunk_skipped = True

                    # 提取并累计文本
                    text = self._extract_text_from_chunk(chunk)
                    if text:
                        accumulated_text += text

                    # 检查是否达到阈值（主动截断的核心逻辑）
                    current_tokens = self._estimate_tokens(accumulated_text)
                    if current_tokens >= self.PROACTIVE_CUTOFF_THRESHOLD:
                        if continuation_round < self.MAX_CONTINUATION_ROUNDS:
                            should_continue = True
                            continuation_round += 1
                            self.truncation_count += 1

                            logger.warning(
                                f"[ProactiveTruncation] Token threshold reached ({current_tokens} >= {self.PROACTIVE_CUTOFF_THRESHOLD}), "
                                f"initiating continuation round {continuation_round}"
                            )

                            # 输出当前 chunk 后停止
                            yield chunk
                            break
                        else:
                            logger.warning(
                                f"[ProactiveTruncation] Max rounds reached ({self.MAX_CONTINUATION_ROUNDS}), "
                                f"continuing without truncation"
                            )

                    # 检查是否正常结束
                    finish_reason = self._get_finish_reason(chunk)
                    if finish_reason:
                        # 如果是 length 结束且还有续接配额，触发续接
                        # 关键：不要 yield 带有 finish_reason='length' 的 chunk，
                        # 否则 Claude SDK 会收到 stop_reason=max_tokens 并报错
                        if finish_reason == 'length' and continuation_round < self.MAX_CONTINUATION_ROUNDS:
                            should_continue = True
                            continuation_round += 1
                            logger.warning(
                                f"[ProactiveTruncation] finish_reason=length detected, "
                                f"initiating continuation round {continuation_round}, "
                                f"NOT yielding this chunk to avoid SDK error"
                            )
                            # 不 yield 这个 chunk，直接 break 去续接
                            break
                        else:
                            # 正常结束或已达最大续接次数，输出 chunk
                            yield chunk
                    else:
                        # 正常输出 chunk（没有 finish_reason）
                        yield chunk

                # 检查是否需要续接
                if not should_continue:
                    if continuation_round > 0:
                        total_tokens = self._estimate_tokens(accumulated_text)
                        logger.info(
                            f"[ProactiveTruncation] Completed with {continuation_round} truncation(s), "
                            f"total tokens: {total_tokens}"
                        )
                    break

                # 发起续接请求
                continuation_messages, _ = self._build_continuation_messages(
                    original_messages,
                    accumulated_text,
                )

                try:
                    current_response = await self._make_continuation_request(
                        model,
                        continuation_messages,
                        request_data
                    )
                    is_continuation = True

                    # 重置累计文本为截断点（续接会从这里继续）
                    # 不重置，继续累计

                    logger.info(f"[ProactiveTruncation] Continuation round {continuation_round} started")

                except Exception as e:
                    logger.error(f"[ProactiveTruncation] Continuation failed: {e}", exc_info=True)
                    break

            except Exception as e:
                logger.error(f"[ProactiveTruncation] Streaming error: {e}", exc_info=True)
                raise

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
            "anthropic_messages",
        ]
    ):
        """Pre-call hook - 不做处理"""
        return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败"""
        pass

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功（非流式）"""
        pass


class ContextWindowInjector(CustomLogger):
    """
    Context Window 信息注入器。

    在每次请求前计算当前消息的 token 使用量，并将剩余 context window 信息
    以 <system><ctx_window>xxx tokens left</ctx_window></system> 格式注入到消息中。

    这使得 Claude 能够感知当前对话的 context 使用情况，从而：
    1. 更好地规划输出长度
    2. 在 context 紧张时主动压缩回复
    3. 提前预警 context 即将耗尽

    注入位置：
    - 在第一条 user 消息的 content 开头注入
    - 使用 XML 标签格式，Claude 能够识别和理解

    排除模型：
    - opus_auto_prefill 模型使用自动续接机制，不需要注入 context window 信息
    """

    # 排除的模型列表（这些模型不会注入 context window 信息）
    # opus_auto_prefill 使用自动续接机制替代 context window 提示
    EXCLUDED_MODELS = [
        "opus_auto_prefill",
    ]

    # 各模型的 context window 大小配置
    # 格式: 模型名称(部分匹配) -> max_context_tokens
    MODEL_CONTEXT_LIMITS = {
        # Claude 系列
        "claude-opus-4": 200000,
        "claude-sonnet-4": 200000,
        "claude-haiku-4": 200000,
        "claude-3": 200000,
        "claude-2": 100000,

        # OpenAI 系列
        "gpt-5": 256000,
        "gpt-4-turbo": 128000,
        "gpt-4o": 128000,
        "gpt-4": 8192,
        "gpt-3.5-turbo": 16385,

        # Gemini 系列
        "gemini-2": 2000000,
        "gemini-1.5": 2000000,
        "gemini-pro": 32000,

        # GLM 系列
        "glm-4": 128000,

        # Doubao 系列
        "doubao": 128000,

        # Kimi 系列
        "kimi": 128000,

        # 默认值
        "default": 128000,
    }

    # 每个字符估算的 token 数（用于粗略估算）
    # 英文约 4 字符/token，中文约 1.5 字符/token，取中间值
    CHARS_PER_TOKEN = 3.0

    def __init__(self):
        super().__init__()
        logger.info(f"ContextWindowInjector initialized, excluded models: {self.EXCLUDED_MODELS}")

    def _is_model_excluded(self, model: str) -> bool:
        """
        检查模型是否被排除（不注入 context window 信息）。

        Args:
            model: 模型名称

        Returns:
            True 如果模型被排除，否则 False
        """
        if not model:
            return False

        model_lower = model.lower()

        for excluded_model in self.EXCLUDED_MODELS:
            excluded_lower = excluded_model.lower()
            # 精确匹配或前缀匹配
            if model_lower == excluded_lower or model_lower.startswith(excluded_lower):
                return True

        return False

    def _get_model_context_limit(self, model: str) -> int:
        """
        获取模型的 context window 大小。

        Args:
            model: 模型名称

        Returns:
            模型的最大 context token 数
        """
        if not model:
            return self.MODEL_CONTEXT_LIMITS["default"]

        model_lower = model.lower()

        # 按优先级匹配
        for pattern, limit in self.MODEL_CONTEXT_LIMITS.items():
            if pattern != "default" and pattern in model_lower:
                return limit

        return self.MODEL_CONTEXT_LIMITS["default"]

    def _estimate_message_tokens(self, message: dict) -> int:
        """
        估算单条消息的 token 数。

        Args:
            message: 消息字典，包含 role 和 content

        Returns:
            估算的 token 数
        """
        tokens = 0

        # role 本身占用一些 token
        tokens += 4  # 角色标记的固定开销

        content = message.get("content", "")

        if isinstance(content, str):
            tokens += int(len(content) / self.CHARS_PER_TOKEN)
        elif isinstance(content, list):
            for item in content:
                if isinstance(item, dict):
                    item_type = item.get("type", "")
                    if item_type == "text":
                        text = item.get("text", "")
                        tokens += int(len(text) / self.CHARS_PER_TOKEN)
                    elif item_type == "image" or item_type == "image_url":
                        # 图片估算为固定 token 数
                        tokens += 1000
                    elif item_type == "tool_use":
                        # 工具调用估算
                        tokens += 100 + int(len(str(item.get("input", {}))) / self.CHARS_PER_TOKEN)
                    elif item_type == "tool_result":
                        tool_content = item.get("content", "")
                        if isinstance(tool_content, str):
                            tokens += int(len(tool_content) / self.CHARS_PER_TOKEN)
                        elif isinstance(tool_content, list):
                            for sub_item in tool_content:
                                if isinstance(sub_item, dict) and sub_item.get("type") == "text":
                                    tokens += int(len(sub_item.get("text", "")) / self.CHARS_PER_TOKEN)
                                elif isinstance(sub_item, str):
                                    tokens += int(len(sub_item) / self.CHARS_PER_TOKEN)
                elif isinstance(item, str):
                    tokens += int(len(item) / self.CHARS_PER_TOKEN)

        return tokens

    def _estimate_total_tokens(self, messages: list, system: str = None) -> int:
        """
        估算所有消息的总 token 数。

        Args:
            messages: 消息列表
            system: system prompt（可选）

        Returns:
            估算的总 token 数
        """
        total = 0

        # system prompt
        if system:
            total += int(len(system) / self.CHARS_PER_TOKEN) + 4

        # 所有消息
        for message in messages:
            total += self._estimate_message_tokens(message)

        # 消息间的固定开销
        total += len(messages) * 3

        return total

    def _inject_context_info(self, messages: list, remaining_tokens: int, total_tokens: int, max_tokens: int) -> bool:
        """
        将 context window 信息注入到消息中。

        Args:
            messages: 消息列表（会被修改）
            remaining_tokens: 剩余 token 数
            total_tokens: 当前使用的 token 数
            max_tokens: 最大 context window

        Returns:
            是否成功注入
        """
        # 构造注入的标签
        ctx_tag = f"""<system>
<total_tokens>{max_tokens} tokens left</total_tokens>
<ctx_window>{remaining_tokens} tokens left</ctx_window>
</system>

"""

        # 查找第一条 user 消息并注入
        for message in messages:
            if message.get("role") == "user":
                content = message.get("content")

                if isinstance(content, str):
                    # 字符串类型，直接在开头添加
                    message["content"] = ctx_tag + content
                    return True
                elif isinstance(content, list):
                    # 列表类型，在第一个 text 项前添加，或创建新的 text 项
                    for i, item in enumerate(content):
                        if isinstance(item, dict) and item.get("type") == "text":
                            item["text"] = ctx_tag + item.get("text", "")
                            return True
                        elif isinstance(item, str):
                            content[i] = ctx_tag + item
                            return True

                    # 如果没有 text 项，在列表开头插入
                    content.insert(0, {"type": "text", "text": ctx_tag})
                    return True

        return False

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: str,  # 使用 str 以支持所有 call_type 值
    ):
        """
        在 LLM 调用前注入 context window 信息。

        注意：排除列表中的模型（如 opus_auto_prefill）不会注入，
        因为它们使用自动续接机制替代 context window 提示。
        """
        try:
            model = data.get("model", "")
            print(f"[ContextWindowInjector] async_pre_call_hook called, call_type={call_type}, model={model}", flush=True)

            # 检查模型是否被排除
            if self._is_model_excluded(model):
                print(f"[ContextWindowInjector] Skipping excluded model={model} (uses auto continuation instead)", flush=True)
                return data

            # 只处理对话类型的请求
            # 支持同步和异步版本: completion, acompletion, anthropic_messages
            if call_type not in ("completion", "acompletion", "anthropic_messages"):
                print(f"[ContextWindowInjector] Skipping call_type={call_type}", flush=True)
                return data

            messages = data.get("messages", [])
            system = data.get("system", "")

            if not messages:
                return data

            # 获取模型的 context limit
            max_context = self._get_model_context_limit(model)

            # 估算当前使用的 token 数
            current_tokens = self._estimate_total_tokens(messages, system)

            # 计算剩余 token
            remaining_tokens = max(0, max_context - current_tokens)

            # 注入信息
            injected = self._inject_context_info(messages, remaining_tokens, current_tokens, max_context)

            if injected:
                usage_percent = (current_tokens / max_context) * 100 if max_context > 0 else 0
                print(
                    f"[ContextWindowInjector] Injected context info - "
                    f"model: {model}, used: {current_tokens}, "
                    f"remaining: {remaining_tokens}, max: {max_context}, "
                    f"usage: {usage_percent:.1f}%",
                    flush=True
                )
            else:
                print(
                    f"[ContextWindowInjector] Failed to inject - no suitable user message found",
                    flush=True
                )

            return data

        except Exception as e:
            print(f"[ContextWindowInjector] Error in async_pre_call_hook: {e}", flush=True)
            import traceback
            traceback.print_exc()
            return data

    async def async_post_call_failure_hook(
        self,
        request_data: dict,
        original_exception: Exception,
        user_api_key_dict: UserAPIKeyAuth,
        traceback_str: Optional[str] = None,
    ):
        """处理调用失败"""
        pass

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response,
    ):
        """处理调用成功（非流式）"""
        pass


# 创建处理器实例
proxy_handler_instance = ImageContentTransformer()
tool_result_content_flattener_instance = ToolResultContentFlattener()
empty_chunk_filter_instance = EmptyChunkFilter()
output_token_prefill_continuation_instance = OutputTokenPrefillContinuation()
proactive_stream_truncation_instance = ProactiveStreamTruncation()
context_window_injector_instance = ContextWindowInjector()


class ToolCacheControlRemover(CustomLogger):
    """
    自定义回调处理器，用于移除工具定义中的 cache_control 字段。
    
    解决错误: "tools.19.custom.cache_control.ephemeral.scope: Extra inputs are not permitted"
    
    原因：
    Claude Code 会在发送给 API 的工具定义中添加 cache_control 参数用于优化缓存。
    但 OpenRouter 通过某些后端（如 Google）时，不支持这个扩展字段。
    
    解决方案：
    在调用 API 之前，移除 tools 列表中每个工具定义的 cache_control 字段。
    """

    def __init__(self):
        super().__init__()
        logger.info("ToolCacheControlRemover initialized")

    def _remove_cache_control_from_tools(self, tools: list) -> tuple[list, int]:
        """
        从工具列表中移除 cache_control 字段。
        
        Args:
            tools: 工具定义列表
            
        Returns:
            (处理后的工具列表, 移除的 cache_control 数量)
        """
        if not tools or not isinstance(tools, list):
            return tools, 0
            
        import copy
        cleaned_tools = []
        removed_count = 0
        
        for tool in tools:
            if not isinstance(tool, dict):
                cleaned_tools.append(tool)
                continue
                
            cleaned_tool = copy.deepcopy(tool)
            
            # 移除顶层的 cache_control
            if "cache_control" in cleaned_tool:
                del cleaned_tool["cache_control"]
                removed_count += 1
                
            # 移除 custom 字段中的 cache_control
            if "custom" in cleaned_tool and isinstance(cleaned_tool["custom"], dict):
                if "cache_control" in cleaned_tool["custom"]:
                    del cleaned_tool["custom"]["cache_control"]
                    removed_count += 1
                # 如果 custom 变成空字典，也移除它
                if not cleaned_tool["custom"]:
                    del cleaned_tool["custom"]
                    
            # 移除 function 字段中的 cache_control (OpenAI 格式)
            if "function" in cleaned_tool and isinstance(cleaned_tool["function"], dict):
                if "cache_control" in cleaned_tool["function"]:
                    del cleaned_tool["function"]["cache_control"]
                    removed_count += 1
                    
            cleaned_tools.append(cleaned_tool)
            
        return cleaned_tools, removed_count


    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: str,  # 使用 str 以支持所有 call_type
    ) -> Optional[dict]:
        """在调用 LLM API 之前，移除工具定义中的 cache_control 字段。"""
        try:
            tools = data.get("tools", [])
            print(f"[ToolCacheControlRemover] called, call_type={call_type}, num_tools={len(tools) if tools else 0}", flush=True)
            
            if call_type not in ("completion", "responses", "anthropic_messages"):
                print(f"[ToolCacheControlRemover] Skipping call_type={call_type}", flush=True)
                return data
                
            if tools:
                cleaned_tools, removed_count = self._remove_cache_control_from_tools(tools)
                if removed_count > 0:
                    data["tools"] = cleaned_tools
                    print(
                        f"[ToolCacheControlRemover] Removed {removed_count} cache_control field(s) "
                        f"from {len(tools)} tool(s)",
                        flush=True
                    )
                else:
                    print(f"[ToolCacheControlRemover] No cache_control found in {len(tools)} tools", flush=True)
                    
            return data
            
        except Exception as e:
            print(f"[ToolCacheControlRemover] Error in async_pre_call_hook: {e}", flush=True)
            import traceback
            traceback.print_exc()
            return data


# 创建实例
tool_cache_control_remover_instance = ToolCacheControlRemover()


class ThinkingStrategyFixer(CustomLogger):
    """
    修复 Anthropic thinking/context_management 参数在 OpenRouter chat completions 格式下的兼容性问题。

    问题链路：
    1. Claude Code 发送 Anthropic Messages 格式请求，包含 thinking + context_management
    2. LiteLLM 转为 OpenAI chat completions 格式发给 OpenRouter
    3. OpenRouter chat completions 不认 Anthropic 原生的 thinking/context_management
    4. Google Vertex 收到 context_management 但找不到 thinking → 400 错误

    解决方案：
    在 async_pre_call_hook 中（proxy 层，litellm.completion 调用之前）：
    1. 将 Anthropic 格式的 thinking 转为 OpenAI 格式的 reasoning（OpenRouter chat completions 支持）
    2. 移除 context_management（chat completions 格式不支持）
    """

    def __init__(self):
        super().__init__()
        logger.info("ThinkingStrategyFixer initialized")

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: str,
    ) -> Optional[dict]:
        """将 Anthropic thinking 转为 OpenRouter reasoning，移除 context_management。"""
        try:
            thinking = data.get("thinking")
            context_mgmt = data.get("context_management")

            if not thinking and not context_mgmt:
                return data

            if thinking and isinstance(thinking, dict):
                thinking_type = thinking.get("type", "")
                budget_tokens = thinking.get("budget_tokens")
                if thinking_type in ("enabled", "adaptive") and "reasoning" not in data:
                    reasoning_param: dict = {"enabled": True}
                    if budget_tokens and isinstance(budget_tokens, int):
                        reasoning_param["max_tokens"] = budget_tokens
                    data["reasoning"] = reasoning_param
                del data["thinking"]

            if context_mgmt:
                del data["context_management"]

            return data

        except Exception as e:
            logger.warning(f"ThinkingStrategyFixer error: {e}")
            return data


thinking_strategy_fixer_instance = ThinkingStrategyFixer()


class SessionIdHeaderInjector(CustomLogger):
    DEFAULT_SESSION_ID = "1fadsfjklkfsdvxc1"
    SESSION_PATTERN = re.compile(r"_session_([^_\s]+)$")

    def __init__(self):
        super().__init__()
        logger.info("SessionIdHeaderInjector initialized")

    def _extract_session_id(self, user_value: Any, metadata_value: Any = None) -> str:
        for candidate in (user_value, metadata_value):
            if isinstance(candidate, str):
                match = self.SESSION_PATTERN.search(candidate)
                if match:
                    return match.group(1)
        return self.DEFAULT_SESSION_ID

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: str,
    ) -> Optional[dict]:
        try:
            user_value = data.get("user")
            metadata = data.get("metadata")
            metadata_user_id = metadata.get("user_id") if isinstance(metadata, dict) else None
            session_id = self._extract_session_id(user_value, metadata_user_id)
            extra_headers = data.get("extra_headers")
            if not isinstance(extra_headers, dict):
                extra_headers = {}
            extra_headers["extra"] = json.dumps({"session_id": session_id}, separators=(",", ":"))
            data["extra_headers"] = extra_headers
            return data
        except Exception as e:
            logger.warning(f"SessionIdHeaderInjector error: {e}")
            return data


session_id_header_injector_instance = SessionIdHeaderInjector()
