#!/usr/bin/env node

/**
 * Generate TODO Report
 * Creates a markdown report with statistics and task lists
 */

const fs = require('fs');
const path = require('path');

// Определяем корень проекта (ищем .todos директорию)
function findProjectRoot() {
    let current = __dirname;
    while (current !== path.dirname(current)) {
        const todosPath = path.join(current, '..', '.todos');
        if (fs.existsSync(todosPath)) {
            return path.join(current, '..');
        }
        current = path.dirname(current);
    }
    return process.cwd();
}

const PROJECT_ROOT = findProjectRoot();
const TODO_DB = path.join(PROJECT_ROOT, '.todos', 'tasks.json');
const REPORT_FILE = path.join(PROJECT_ROOT, 'TODO_REPORT.md');

function loadTasks() {
    try {
        const data = fs.readFileSync(TODO_DB, 'utf8');
        const parsed = JSON.parse(data);
        // Поддержка старой и новой структуры
        if (parsed.metadata) {
            return {
                tasks: parsed.tasks || [],
                lastScan: parsed.metadata.lastScan,
                totalScans: parsed.metadata.totalScans || 0,
                version: parsed.metadata.version || '1.0.0'
            };
        }
        return {
            tasks: parsed.tasks || [],
            lastScan: parsed.lastScan || null,
            totalScans: 0,
            version: parsed.version || '1.0.0'
        };
    } catch (error) {
        console.error('Error loading tasks:', error);
        return { tasks: [], lastScan: null, totalScans: 0, version: '1.0.0' };
    }
}

function generateReport() {
    const data = loadTasks();
    const tasks = data.tasks || [];
    
    const total = tasks.length;
    const open = tasks.filter(t => t.status === 'OPEN').length;
    const inProgress = tasks.filter(t => t.status === 'IN_PROGRESS').length;
    const resolved = tasks.filter(t => t.status === 'RESOLVED').length;
    
    const byPriority = {
        CRITICAL: tasks.filter(t => t.priority === 'CRITICAL' && t.status === 'OPEN'),
        HIGH: tasks.filter(t => t.priority === 'HIGH' && t.status === 'OPEN'),
        MEDIUM: tasks.filter(t => t.priority === 'MEDIUM' && t.status === 'OPEN'),
        LOW: tasks.filter(t => t.priority === 'LOW' && t.status === 'OPEN')
    };
    
    const byType = {
        TODO: tasks.filter(t => t.type === 'TODO'),
        FIXME: tasks.filter(t => t.type === 'FIXME'),
        HACK: tasks.filter(t => t.type === 'HACK'),
        REFACTOR: tasks.filter(t => t.type === 'REFACTOR')
    };
    
    const byFileType = {
        backend: tasks.filter(t => t.fileType === 'backend'),
        frontend: tasks.filter(t => t.fileType === 'frontend'),
        devops: tasks.filter(t => t.fileType === 'devops'),
        unknown: tasks.filter(t => t.fileType === 'unknown')
    };
    
    const completionRate = total > 0 ? Math.round((resolved / total) * 100) : 0;
    
    const report = `# 🎯 Automated TODO Report

**Generated:** ${new Date().toLocaleString('ru-RU')}
**Last Scan:** ${data.lastScan ? new Date(data.lastScan).toLocaleString('ru-RU') : 'Never'}
**Total Scans:** ${data.totalScans || 0}
**Version:** ${data.version || '1.0.0'}

---

## 📈 Quick Stats

| Metric | Count |
|--------|-------|
| **Total Tasks** | ${total} |
| **Open Tasks** | ${open} |
| **In Progress** | ${inProgress} |
| **Resolved** | ${resolved} |
| **Completion Rate** | ${completionRate}% |

---

## 🚨 Critical Tasks (Need Immediate Attention)

${byPriority.CRITICAL.length === 0 
    ? '*No critical tasks found* ✅' 
    : byPriority.CRITICAL.map(task => 
        `### \`${task.file}:${task.line}\`
- **Type:** ${task.type}
- **Description:** ${task.description}
- **Assigned to:** ${task.assignedTo || 'unassigned'}
- **Status:** ${task.status}
- **Created:** ${new Date(task.createdAt).toLocaleDateString('ru-RU')}
`
    ).join('\n')}

---

## ⚠️ High Priority Tasks

${byPriority.HIGH.length === 0 
    ? '*No high priority tasks* ✅' 
    : byPriority.HIGH.slice(0, 10).map(task => 
        `- \`${task.file}:${task.line}\` - ${task.description} (${task.assignedTo || 'unassigned'})`
    ).join('\n')}

${byPriority.HIGH.length > 10 ? `\n*...and ${byPriority.HIGH.length - 10} more high priority tasks*` : ''}

---

## 📊 Distribution by Priority

| Priority | Open Tasks |
|----------|-----------|
| 🔴 **CRITICAL** | ${byPriority.CRITICAL.length} |
| 🟠 **HIGH** | ${byPriority.HIGH.length} |
| 🟡 **MEDIUM** | ${byPriority.MEDIUM.length} |
| 🟢 **LOW** | ${byPriority.LOW.length} |

---

## 📋 Distribution by Type

| Type | Count |
|------|-------|
| **TODO** | ${byType.TODO.length} |
| **FIXME** | ${byType.FIXME.length} |
| **HACK** | ${byType.HACK.length} |
| **REFACTOR** | ${byType.REFACTOR.length} |

---

## 🎯 Distribution by File Type

| Type | Count |
|------|-------|
| **Backend** | ${byFileType.backend.length} |
| **Frontend** | ${byFileType.frontend.length} |
| **DevOps** | ${byFileType.devops.length} |
| **Unknown** | ${byFileType.unknown.length} |

---

## 🎯 Next Actions

1. **Review critical tasks** - ${byPriority.CRITICAL.length} critical tasks need immediate attention
2. **Assign unassigned tasks** - ${tasks.filter(t => !t.assignedTo || t.assignedTo === 'unassigned').length} tasks are unassigned
3. **Update status** - ${inProgress} tasks are in progress, consider updating their status
4. **Resolve completed tasks** - Mark resolved tasks as completed

---

## 📝 Recent Tasks

${tasks
    .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
    .slice(0, 10)
    .map(task => 
        `- [${task.status}] \`${task.file}:${task.line}\` - ${task.description} (${task.priority})`
    ).join('\n')}

---

*This report is automatically generated. Run \`npm run todos:scan\` to update tasks.*
`;

    fs.writeFileSync(REPORT_FILE, report, 'utf8');
    console.log(`✅ Report generated: ${REPORT_FILE}`);
    console.log(`📊 Total tasks: ${total}, Open: ${open}, Critical: ${byPriority.CRITICAL.length}`);
}

if (require.main === module) {
    generateReport();
}

module.exports = { generateReport, loadTasks };

