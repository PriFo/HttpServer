#!/usr/bin/env node

/**
 * Экспорт задач в различные форматы
 * Поддерживает: CSV, JSON, Markdown
 */

const fs = require('fs');
const path = require('path');

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
const OUTPUT_DIR = path.join(PROJECT_ROOT, '.todos', 'exports');

function loadTasks() {
    try {
        const data = fs.readFileSync(TODO_DB, 'utf8');
        const parsed = JSON.parse(data);
        return parsed.tasks || [];
    } catch (error) {
        console.error('Error loading tasks:', error);
        return [];
    }
}

function exportCSV(tasks) {
    const headers = ['ID', 'File', 'Line', 'Type', 'Priority', 'Status', 'Description', 'Assigned To', 'Created At', 'Updated At'];
    const rows = tasks.map(task => [
        task.id,
        task.file,
        task.line,
        task.type,
        task.priority,
        task.status,
        `"${(task.description || '').replace(/"/g, '""')}"`,
        task.assignedTo || 'unassigned',
        task.createdAt,
        task.updatedAt
    ]);
    
    const csv = [headers.join(','), ...rows.map(row => row.join(','))].join('\n');
    return csv;
}

function exportMarkdown(tasks) {
    let md = '# TODO Tasks Export\n\n';
    md += `**Total:** ${tasks.length} tasks\n`;
    md += `**Generated:** ${new Date().toLocaleString('ru-RU')}\n\n`;
    md += '---\n\n';
    
    // Группируем по приоритету
    const byPriority = {
        CRITICAL: tasks.filter(t => t.priority === 'CRITICAL'),
        HIGH: tasks.filter(t => t.priority === 'HIGH'),
        MEDIUM: tasks.filter(t => t.priority === 'MEDIUM'),
        LOW: tasks.filter(t => t.priority === 'LOW')
    };
    
    for (const [priority, priorityTasks] of Object.entries(byPriority)) {
        if (priorityTasks.length === 0) continue;
        
        md += `## ${priority} Priority (${priorityTasks.length})\n\n`;
        
        for (const task of priorityTasks) {
            md += `### \`${task.file}:${task.line}\`\n\n`;
            md += `- **Type:** ${task.type}\n`;
            md += `- **Status:** ${task.status}\n`;
            md += `- **Assigned:** ${task.assignedTo || 'unassigned'}\n`;
            md += `- **Description:** ${task.description}\n`;
            md += `- **Created:** ${new Date(task.createdAt).toLocaleString('ru-RU')}\n\n`;
        }
    }
    
    return md;
}

function main() {
    const format = process.argv[2] || 'json';
    const tasks = loadTasks();
    
    if (tasks.length === 0) {
        console.log('No tasks to export');
        return;
    }
    
    // Создаем директорию для экспорта
    if (!fs.existsSync(OUTPUT_DIR)) {
        fs.mkdirSync(OUTPUT_DIR, { recursive: true });
    }
    
    let content, filename, extension;
    
    switch (format.toLowerCase()) {
        case 'csv':
            content = exportCSV(tasks);
            extension = 'csv';
            filename = `tasks-${new Date().toISOString().split('T')[0]}.csv`;
            break;
        case 'md':
        case 'markdown':
            content = exportMarkdown(tasks);
            extension = 'md';
            filename = `tasks-${new Date().toISOString().split('T')[0]}.md`;
            break;
        case 'json':
        default:
            content = JSON.stringify({ tasks, exported: new Date().toISOString() }, null, 2);
            extension = 'json';
            filename = `tasks-${new Date().toISOString().split('T')[0]}.json`;
            break;
    }
    
    const filepath = path.join(OUTPUT_DIR, filename);
    fs.writeFileSync(filepath, content, 'utf8');
    
    console.log(`✅ Exported ${tasks.length} tasks to ${filepath}`);
}

if (require.main === module) {
    main();
}

module.exports = { exportCSV, exportMarkdown, loadTasks };

