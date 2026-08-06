package cli

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func nowUnix() int64 { return time.Now().Unix() }

// backupTimestampRe 匹配目录名末尾的 unix 时间戳（秒）。
// 兼容 cleanup_backup_1782143508 和 cleanup_backup_blocked_apply_failed_1782143776。
var backupTimestampRe = regexp.MustCompile(`(\d+)$`)

// maxKeptVersions 保留最近多少个历史版本二进制。
// 当前版本必定保留，再保留 2 个旧版本用于回退。
const maxKeptVersions = 3

// PruneOldVersions 清理 bin/versions/ 下超出保留数量的旧版本。
// 当前运行版本必定保留。失败只记日志不阻断启动。
func PruneOldVersions(dataDir string, log *slog.Logger) {
	versionsDir := filepath.Join(dataDir, "bin", "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return // 目录不存在或无权限，静默跳过
	}
	type vEntry struct {
		name string
		path string
	}
	var vers []vEntry
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		vers = append(vers, vEntry{name: e.Name(), path: filepath.Join(versionsDir, e.Name())})
	}
	if len(vers) <= maxKeptVersions {
		return
	}
	// 按版本号降序排序（新的在前）
	sort.Slice(vers, func(i, j int) bool {
		return vers[i].name > vers[j].name
	})
	current := "v" + version.Version
	removed := 0
	for i := maxKeptVersions; i < len(vers); i++ {
		// 当前版本不删（即使排序后在尾部）
		if vers[i].name == current {
			continue
		}
		if err := os.RemoveAll(vers[i].path); err != nil {
			log.Warn("[storage] 删除旧版本失败", "version", vers[i].name, "err", err)
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Info("[storage] 清理旧版本二进制", "removed", removed, "kept", len(vers)-removed)
	}
}

// cleanupBackupTTLDays cleanup_backup 目录的保留天数。
const cleanupBackupTTLDays = 7

// CleanupOldBackups 清理 workspace/ 下过期的 cleanup_backup_* 目录。
// 目录名里的时间戳是创建时的 unix 秒，超过 TTL 的删掉。
func CleanupOldBackups(dataDir string, log *slog.Logger) {
	wsDir := filepath.Join(dataDir, "workspace")
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return
	}
	cutoff := nowUnix() - int64(cleanupBackupTTLDays)*86400
	removed := 0
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "cleanup_backup_") {
			continue
		}
		ts := parseBackupTimestamp(e.Name())
		if ts <= 0 || ts > cutoff {
			continue // 解析失败或还没过期
		}
		if err := os.RemoveAll(filepath.Join(wsDir, e.Name())); err != nil {
			log.Warn("[storage] 删除过期 backup 失败", "dir", e.Name(), "err", err)
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Info("[storage] 清理过期 backup", "removed", removed, "ttl_days", cleanupBackupTTLDays)
	}
}

// parseBackupTimestamp 从 "cleanup_backup_1782143508" 或
// "cleanup_backup_blocked_apply_failed_1782143776" 提取末尾的 unix 时间戳。
func parseBackupTimestamp(name string) int64 {
	m := backupTimestampRe.FindString(strings.TrimPrefix(name, "cleanup_backup_"))
	if m == "" {
		return 0
	}
	ts, err := strconv.ParseInt(m, 10, 64)
	if err != nil {
		return 0
	}
	return ts
}
