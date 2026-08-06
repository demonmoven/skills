# devclaw_skills_center Makefile
# 核心命令：make install — 安装 xdev 全局 CLI（npm 包名 @byted/xdex）
#
# v0.0.1 起本仓库的 CLI 是 dist/index.js（TypeScript），不再提供旧版"把
# skills 拷到 ~/.claude/skills/"的安装方式。如需安装 skills，使用
# `xdev` 启动对应 coding agent 时由 plugin 自动加载。

.PHONY: install uninstall

## 安装 xdev CLI（全局命令 xdev，npm 包名 @byted/xdex）
install:
	@echo "安装 xdev CLI (@byted/xdex)..."
	@npm install --registry https://bnpm.byted.org
	@npm run build
	@echo ""
	@echo "✓ 已安装全局命令: xdev"
	@echo ""
	@echo "用法："
	@echo "  xdev                            # 交互选 agent 并启动"
	@echo "  xdev --trae-cn                  # 直接启动 Trae CN (IDE)"
	@echo "  xdev --trae                     # 直接启动 Trae (IDE)"
	@echo "  xdev --coco                     # 直接启动 Coco / TRAE CLI"
	@echo "  xdev --cc                       # 直接启动 Claude Code"
	@echo "  xdev --cdx                      # 直接启动 Codex"
	@echo "  xdev --debug --coco             # 从本地源码 build 并启动（需设置 XDEV_DEV_REPO）"
	@echo "  xdev clean                      # 清理 plugin 缓存"
	@echo "  xdev trace auth login           # trace 模块 SSO 登录"
	@echo "  xdev trace forward --file <p>   # trace 模块上报会话"
	@echo ""
	@echo "💡 如果你之前装过旧 bash 版本的 xdev（曾在 /usr/local/bin/xdev），"
	@echo "   或者在同一个 terminal 里跑了 make uninstall + make install，"
	@echo "   bash 可能仍然缓存着旧路径，下次跑 xdev 时会报"
	@echo "     -bash: /usr/local/bin/xdev: No such file or directory"
	@echo "   这不是 xdev 的 bug，是 bash 命令哈希缓存机制。任选一条修复："
	@echo "     hash -r           # 清空全部 bash 命令缓存"
	@echo "     hash -d xdev      # 只清 xdev 这一项"
	@echo "     exec bash         # 用新进程替换当前 shell"
	@echo "   或者直接打开一个新的 terminal 窗口。"

## 卸载 xdev CLI 与本地缓存
## 不会清理 ~/.trace/（含 SSO token、device id），如需重置请手动 rm -rf。
uninstall:
	@echo "卸载 xdev CLI (@byted/xdex)..."
	@npm unlink -g @byted/xdex 2>/dev/null || true
	@npm unlink -g @byted/devclaw-skills-center 2>/dev/null || true
	@sudo rm -f /usr/local/bin/xdev 2>/dev/null || true
	@rm -rf $(HOME)/.xdev
	@echo "  ✓ npm 全局命令已 unlink (含旧 @byted/devclaw-skills-center 残留清理)"
	@echo "  ✓ 已清理 /usr/local/bin/xdev（如有遗留的旧 bash 版本）"
	@echo "  ✓ 已清理 ~/.xdev/"
	@echo "卸载完成。（注意：~/.trace/ 中的 SSO token 与 device id 未清理）"
