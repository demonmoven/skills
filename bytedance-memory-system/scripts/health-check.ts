#!/usr/bin/env node
/**
 * @fileoverview 三层记忆架构健康检查脚本
 * @author suying.1111
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const http = require('http');

const WORKSPACE_DIR = process.env.WORKSPACE_DIR || process.cwd();

const CHECKS = {
  workspace: { passed: 0, failed: 0, items: [] },
  memory: { passed: 0, failed: 0, items: [] },
  openviking: { passed: 0, failed: 0, items: [] },
  git: { passed: 0, failed: 0, items: [] }
};

function check(condition, category, message, detail = '') {
  const status = condition ? '✅' : '❌';
  const result = condition ? 'passed' : 'failed';
  CHECKS[category][result]++;
  CHECKS[category].items.push({ status, message, detail });
  return condition;
}

function checkDir(dir, category, name) {
  const exists = fs.existsSync(path.join(WORKSPACE_DIR, dir));
  check(exists, category, `${name}目录存在`, dir);
  return exists;
}

function checkFile(file, category, name) {
  const exists = fs.existsSync(path.join(WORKSPACE_DIR, file));
  check(exists, category, `${name}文件存在`, file);
  return exists;
}

function checkOpenViking() {
  return new Promise((resolve) => {
    const req = http.get('http://127.0.0.1:1933/health', (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const result = JSON.parse(data);
          check(result.status === 'ok', 'openviking', 'OpenViking服务运行正常', 'http://127.0.0.1:1933');
          resolve(true);
        } catch {
          check(false, 'openviking', 'OpenViking服务响应异常', data);
          resolve(false);
        }
      });
    });
    req.on('error', () => {
      check(false, 'openviking', 'OpenViking服务未启动', '请运行: openviking-server');
      resolve(false);
    });
    req.setTimeout(3000, () => {
      check(false, 'openviking', 'OpenViking服务连接超时', '检查服务状态');
      req.destroy();
      resolve(false);
    });
  });
}

function checkGitStatus() {
  try {
    const status = execSync('git status --short', { cwd: WORKSPACE_DIR, encoding: 'utf8' });
    const isClean = status.trim() === '';
    check(isClean, 'git', isClean ? 'Git工作区干净' : 'Git工作区有未提交变更', 
      isClean ? '' : `${status.split('\n').length} 个文件待处理`);
    
    const log = execSync('git log --oneline -3', { cwd: WORKSPACE_DIR, encoding: 'utf8' });
    check(true, 'git', 'Git仓库正常', `最近提交: ${log.split('\n')[0]}`);
  } catch (e) {
    check(false, 'git', 'Git检查失败', e.message);
  }
}

function checkMemorySize() {
  try {
    const memoryDir = path.join(WORKSPACE_DIR, 'memory');
    if (fs.existsSync(memoryDir)) {
      const files = fs.readdirSync(memoryDir);
      const size = files.reduce((acc, f) => {
        const stat = fs.statSync(path.join(memoryDir, f));
        return acc + (stat.isFile() ? stat.size : 0);
      }, 0);
      const sizeKB = Math.round(size / 1024);
      check(sizeKB < 500, 'memory', `memory/ 大小: ${sizeKB}KB`, sizeKB > 500 ? '建议归档旧日志' : '健康');
    }
  } catch (e) {
    check(false, 'memory', 'memory/ 大小检查失败', e.message);
  }
}

async function runChecks() {
  console.log('🔍 三层记忆架构健康检查\n');
  console.log('=' .repeat(60));

  // Workspace 检查
  console.log('\n📁 Workspace 目录结构');
  checkDir('memory', 'workspace', 'memory/');
  checkDir('.learnings', 'workspace', '.learnings/');
  checkDir('state', 'workspace', 'state/');
  checkDir('logs', 'workspace', 'logs/');
  checkDir('tmp', 'workspace', 'tmp/');
  
  checkFile('DIRECTORIES.md', 'workspace', 'DIRECTORIES.md');
  checkFile('HEARTBEAT.md', 'workspace', 'HEARTBEAT.md');
  checkFile('EVOLUTION.md', 'workspace', 'EVOLUTION.md');
  checkFile('MEMORY.md', 'workspace', 'MEMORY.md');
  checkFile('AGENTS.md', 'workspace', 'AGENTS.md');

  // Memory 检查
  console.log('\n🧠 手动记忆层');
  checkFile('.learnings/LEARNINGS.md', 'memory', 'LEARNINGS.md');
  checkFile('memory/projects.md', 'memory', 'projects.md');
  checkMemorySize();

  // Git 检查
  console.log('\n📦 Git 版本控制');
  checkGitStatus();

  // OpenViking 检查
  console.log('\n🤖 OpenViking 自动记忆');
  await checkOpenViking();

  // 输出摘要
  console.log('\n' + '='.repeat(60));
  console.log('📊 检查摘要');
  console.log('='.repeat(60));
  
  let totalPassed = 0;
  let totalFailed = 0;
  
  Object.entries(CHECKS).forEach(([category, data]) => {
    const total = data.passed + data.failed;
    const status = data.failed === 0 ? '✅' : '⚠️';
    console.log(`${status} ${category.padEnd(12)}: ${data.passed}/${total} 通过`);
    totalPassed += data.passed;
    totalFailed += data.failed;
  });

  console.log('\n' + '-'.repeat(60));
  const allPassed = totalFailed === 0;
  console.log(`${allPassed ? '✅' : '⚠️'} 总计: ${totalPassed} 通过, ${totalFailed} 失败`);
  
  if (!allPassed) {
    console.log('\n❌ 失败项详情:');
    Object.entries(CHECKS).forEach(([category, data]) => {
      data.items.forEach(item => {
        if (item.status === '❌') {
          console.log(`  [${category}] ${item.message}`);
          if (item.detail) console.log(`    → ${item.detail}`);
        }
      });
    });
    process.exit(1);
  }
  
  console.log('\n🎉 所有检查通过！三层记忆架构运行正常。');
  process.exit(0);
}

runChecks();