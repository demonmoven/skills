/**
 * Vanilla JS for client-side interactivity in the analyze report.
 *
 * Responsibilities (kept deliberately small for v1):
 *   - Tab switching (Summary / Trajectory)
 *   - Click-to-toggle expand/collapse on .node-header and .tool-header
 *   - Expand-all / Collapse-all buttons
 *   - Simple text-based trajectory filter
 *   - Keyboard: `/` focuses filter, `Escape` clears
 */
export const INTERACTIVITY_JS = `
(function() {
  'use strict';

  function onReady(fn) {
    if (document.readyState !== 'loading') fn();
    else document.addEventListener('DOMContentLoaded', fn);
  }

  onReady(function() {
    // Tab switching
    var tabButtons = document.querySelectorAll('.tab-button');
    var tabPanes = document.querySelectorAll('.tab-pane');
    tabButtons.forEach(function(btn) {
      btn.addEventListener('click', function() {
        var target = btn.getAttribute('data-tab');
        tabButtons.forEach(function(b) { b.classList.remove('active'); });
        tabPanes.forEach(function(p) { p.classList.remove('active'); });
        btn.classList.add('active');
        var pane = document.querySelector('.tab-pane[data-tab="' + target + '"]');
        if (pane) pane.classList.add('active');
      });
    });

    // Click-to-toggle on node headers
    document.body.addEventListener('click', function(ev) {
      var target = ev.target;
      while (target && target !== document.body) {
        if (target.classList && (target.classList.contains('node-header') || target.classList.contains('tool-header'))) {
          var container = target.parentElement;
          if (container) container.classList.toggle('expanded');
          return;
        }
        target = target.parentElement;
      }
    });

    // Expand all / collapse all
    var expandAllBtn = document.querySelector('.expand-all');
    var collapseAllBtn = document.querySelector('.collapse-all');
    var trajectoryRoot = document.querySelector('.trajectory');

    if (expandAllBtn && trajectoryRoot) {
      expandAllBtn.addEventListener('click', function() {
        trajectoryRoot.querySelectorAll('.node, .tool-call').forEach(function(el) {
          el.classList.add('expanded');
        });
      });
    }
    if (collapseAllBtn && trajectoryRoot) {
      collapseAllBtn.addEventListener('click', function() {
        trajectoryRoot.querySelectorAll('.node, .tool-call').forEach(function(el) {
          el.classList.remove('expanded');
        });
      });
    }

    // Filter input — simple substring match on summary text
    var filterInput = document.querySelector('.filter-input');
    if (filterInput && trajectoryRoot) {
      filterInput.addEventListener('input', function() {
        var q = filterInput.value.trim().toLowerCase();
        var nodes = trajectoryRoot.querySelectorAll('.node');
        if (!q) {
          nodes.forEach(function(n) { n.classList.remove('filtered-out'); });
          return;
        }
        nodes.forEach(function(n) {
          var txt = (n.textContent || '').toLowerCase();
          if (txt.indexOf(q) === -1) {
            n.classList.add('filtered-out');
          } else {
            n.classList.remove('filtered-out');
          }
        });
      });
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', function(ev) {
      if (ev.key === '/' && document.activeElement !== filterInput) {
        if (filterInput) {
          ev.preventDefault();
          filterInput.focus();
        }
      } else if (ev.key === 'Escape' && document.activeElement === filterInput) {
        filterInput.value = '';
        filterInput.dispatchEvent(new Event('input'));
        filterInput.blur();
      }
    });

    // Default expand: top-level main-thread nodes,
    // all Agent tool-calls (so subagents are visible without hunting),
    // and the top-level nodes inside subagent drill-downs.
    if (trajectoryRoot) {
      trajectoryRoot.querySelectorAll(':scope > .node').forEach(function(el) {
        el.classList.add('expanded');
      });
      document.querySelectorAll('.tool-call[data-tool="Agent"]').forEach(function(el) {
        el.classList.add('expanded');
      });
      document.querySelectorAll('.subagent-trajectory > .node').forEach(function(el) {
        el.classList.add('expanded');
      });
    }

    // ===== Tool filter chips (multi-select) =====
    var activeTools = new Set();
    var toolChips = document.querySelectorAll('.tool-chip');
    var allChip = document.querySelector('.tool-chip.chip-all');

    function applyToolFilter() {
      var hasFilter = activeTools.size > 0;
      // First pass: hide everything if filter active
      document.querySelectorAll('.tool-call, .trajectory .node').forEach(function(el) {
        el.classList.toggle('tool-filter-hidden', hasFilter);
      });
      if (!hasFilter) return;
      // Second pass: unhide matching tool-calls and walk up to unhide ancestors
      activeTools.forEach(function(tool) {
        document.querySelectorAll('.tool-call[data-tool="' + CSS.escape(tool) + '"]').forEach(function(tc) {
          tc.classList.remove('tool-filter-hidden');
          var el = tc.parentElement;
          while (el) {
            if (el.classList && (el.classList.contains('node') || el.classList.contains('tool-call'))) {
              el.classList.remove('tool-filter-hidden');
              // Also ensure the ancestor is expanded so the matched child is visible
              el.classList.add('expanded');
            }
            el = el.parentElement;
          }
        });
      });
    }

    toolChips.forEach(function(chip) {
      chip.addEventListener('click', function(ev) {
        ev.stopPropagation();
        var tool = chip.getAttribute('data-tool');
        if (tool === '__all__') {
          // Clear all filters
          activeTools.clear();
          toolChips.forEach(function(c) { c.classList.remove('active'); });
          if (allChip) allChip.classList.add('active');
        } else {
          if (allChip) allChip.classList.remove('active');
          if (activeTools.has(tool)) {
            activeTools.delete(tool);
            chip.classList.remove('active');
          } else {
            activeTools.add(tool);
            chip.classList.add('active');
          }
          if (activeTools.size === 0 && allChip) {
            allChip.classList.add('active');
          }
        }
        applyToolFilter();
      });
    });

    // ===== Work Directory tab =====
    var workdirFilesEl = document.getElementById('workdir-files');
    var workdirFiles = {};
    if (workdirFilesEl) {
      try { workdirFiles = JSON.parse(workdirFilesEl.textContent || '{}'); }
      catch (err) { console.error('Failed to parse workdir files:', err); }
    }

    // Dir collapse/expand
    document.querySelectorAll('.workdir-tree .node-dir').forEach(function(dir) {
      dir.addEventListener('click', function(ev) {
        ev.stopPropagation();
        var li = dir.parentElement;
        if (!li || li.tagName !== 'LI') return;
        li.classList.toggle('collapsed');
        var icon = dir.querySelector('.dir-icon');
        if (icon) icon.textContent = li.classList.contains('collapsed') ? '▶' : '▼';
      });
    });

    // File click → show content in right pane
    var contentPane = document.getElementById('workdir-content');
    var currentFileEl = null;
    document.querySelectorAll('.workdir-tree .node-file').forEach(function(file) {
      file.addEventListener('click', function(ev) {
        ev.stopPropagation();
        var path = file.getAttribute('data-path');
        var skipped = file.getAttribute('data-skipped');
        if (currentFileEl) currentFileEl.classList.remove('active');
        file.classList.add('active');
        currentFileEl = file;

        if (!contentPane) return;
        if (skipped) {
          contentPane.innerHTML = '<h2>' + escapeHtmlJs(path || '') + '</h2>' +
            '<div class="file-meta">Content not embedded: ' + escapeHtmlJs(skipped) + '</div>';
          return;
        }
        var content = workdirFiles[path || ''];
        if (content === undefined) {
          contentPane.innerHTML = '<h2>' + escapeHtmlJs(path || '') + '</h2>' +
            '<div class="file-meta">(no content available)</div>';
          return;
        }
        var sizeStr = content.length + ' chars';
        contentPane.innerHTML = '<h2>' + escapeHtmlJs(path || '') + '</h2>' +
          '<div class="file-meta">' + sizeStr + '</div>' +
          '<pre></pre>';
        var pre = contentPane.querySelector('pre');
        if (pre) pre.textContent = content;
      });
    });

    function escapeHtmlJs(s) {
      return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    }
  });
})();
`;
