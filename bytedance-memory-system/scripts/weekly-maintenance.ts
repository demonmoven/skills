#!/usr/bin/env node
/**
 * @fileoverview 每周记忆维护脚本
 * @author suying.1111
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const WORKSPACE_DIR = process.env.WORKSPACE_DIR || process.cwd();
const DAYS_TO_ARCHIVE = 7;
const DAYS_TO_DELETE = 30;

function getFileAgeDays(filePath) {
  const stats = fs.statSync(filePath);
  const now = Date.now();
  const mtime = stats.mtime.getTime();
  return Math.floor((now - mtime) / (1000 * 60 * 60 * 24));
}

function archiveOldLogs() {
  console.log('🗄️  归档旧日志...');
  const logsDir = path.join(WORKSPACE_DIR, 'logs');
  const archiveDir = path.join(logsDir, 'archive');
  
  if (!fs.existsSync(logsDir)) return;
  
  ensureDir(archiveDir);
  
  let archived = 0;
  fs.readdirSync(logsDir).forEach(file => {
    const filePath = path.join(logsDir, file);
    if (fs.statSync(filePath).isFile() && file !== 'archive') {
      const age = getFileAgeDays(filePath);
      if (age > DAYS_TO_ARCHIVE) {
        const dest = path.join(archiveDir, file);
        fs.renameSync(filePath, dest);
        archived++;
      }
    }
  });
  
  console.log(`  ✅ 归档 ${archived} 个日志文件`);
}

function archiveOldMemory() {
  console.log('📝 归档旧记忆日志...');
  const memoryDir = path.join(WORKSPACE_DIR, 'memory');
  const archiveDir = path.join(memoryDir, 'archive');
  
  if (!fs.existsSync(memoryDir)) return;
  
  ensureDir(archiveDir);
  
  let archived = 0;
  fs.readdirSync(memoryDir).forEach(file => {
    if (file.match(/^\d{4}-\d{2}-\d{2}\.md$/)) {
      const filePath = path.join(memoryDir, file);
      const age = getFileAgeDays(filePath);
      if (age > DAYS_TO_ARCHIVE) {
        const dest = path.join(archiveDir, file);
        fs.renameSync(filePath, dest);
        archived++;
      }
    }
  });
  
  console.log(`  ✅ 归档 ${archived} 个记忆日志`);
}

function cleanOldArchive() {
  console.log('🧹 清理过期归档...');
  const archiveDirs = [
    path.join(WORKSPACE_DIR, 'logs/archive'),
    path.join(WORKSPACE_DIR, 'memory/archive')
  ];
  
  let deleted = 0;
  archiveDirs.forEach(dir => {
    if (!fs.existsSync(dir)) return;
    
    fs.readdirSync(dir).forEach(file => {
      const filePath = path.join(dir, file);
      if (fs.statSync(filePath).isFile()) {
        const age = getFileAgeDays(filePath);
        if (age > DAYS_TO_DELETE) {
          fs.unlinkSync(filePath);
          deleted++;
        }
      }
    });
  });
  
  console.log(`  ✅ 删除 ${deleted} 个过期归档文件`);
}

function cleanStateCache() {
  console.log('💾 清理 state 缓存...');
  const stateDir = path.join(WORKSPACE_DIR, 'state');
  
  if (!fs.existsSync(stateDir)) return;
  
  let cleaned = 0;
  fs.readdirSync(stateDir).forEach(file => {
    const filePath = path.join(stateDir, file);
    if (fs.statSync(filePath).isFile()) {
      const age = getFileAgeDays(filePath);
      if (age > DAYS_TO_ARCHIVE) {
        fs.unlinkSync(filePath);
        cleaned++;
      }
    }
  });
  
  console.log(`  ✅ 清理 ${cleaned} 个过期 state 文件`);
}

function checkMemoryMd() {
  console.log('📄 检查 MEMORY.md 长度...');
  const memoryMd = path.join(WORKSPACE_DIR, 'MEMORY.md');
  
  if (!fs.existsSync(memoryMd)) {
    console.log('  ⚠️ MEMORY.md 不存在');
    return;
  }
  
  const content = fs.readFileSync(memoryMd, 'utf8');
  const lines = content.split('\n').length;
  
  if (lines > 40) {
    console.log(`  ⚠️  MEMORY.md 过长 (${lines} 行)，建议整理归档`);
  } else {
    console.log(`  ✅ MEMORY.md 长度正常 (${lines} 行)`);
  }
}

function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

function gitCommit() {
  console.log('📦 提交 Git 变更...');
  try {
    execSync('git add -A', { cwd: WORKSPACE_DIR });
    const date = new Date().toISOString().split('T')[0];
    execSync(`git commit -m "chore: weekly memory maintenance ${date}"`, { cwd: WORKSPACE_DIR });
    console.log('  ✅ Git 提交成功');
  } catch (e) {
    console.log('  ⏭️  无变更需要提交');
  }
}

function runMaintenance() {
  console.log('🔧 每周记忆维护\n');
  console.log('='.repeat(60));
  
  archiveOldLogs();
  archiveOldMemory();
  cleanOldArchive();
  cleanStateCache();
  checkMemoryMd();
  
  console.log('='.repeat(60));
  gitCommit();
  
  console.log('\n🎉 维护完成！');
  console.log('\n建议：');
  console.log('  • 阅读最近7天的 memory/ 日志');
  console.log('  • 提取要点更新到 MEMORY.md');
  console.log('  • 检查 OpenViking 记忆健康状况');
}

runMaintenance();