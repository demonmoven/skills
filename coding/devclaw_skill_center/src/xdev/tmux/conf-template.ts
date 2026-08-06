export const START_MARKER = '# XDEV-TMUX:START';
export const END_MARKER = '# XDEV-TMUX:END';

export const XDEV_TMUX_CONF_BLOCK = `${START_MARKER}
set -g mouse on

# 鼠标拖选松开后自动复制到系统剪贴板
bind-key -T copy-mode MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "pbcopy"

# 键盘复制也同步到系统剪贴板
bind-key -T copy-mode M-w send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode-vi y send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode-vi Enter send-keys -X copy-pipe-and-cancel "pbcopy"
${END_MARKER}
`;
