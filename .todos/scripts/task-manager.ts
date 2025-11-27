/**
 * Task Manager - Менеджер задач для системы TODO
 */

import { TodoTask, TodoDatabase, TaskStatistics, Priority, TaskStatus, TaskType, FileType } from '../types';

export class TaskManager {
  private dbPath: string = '.todos/tasks.json';

  /**
   * Загрузка базы данных задач
   */
  async loadDatabase(): Promise<TodoDatabase> {
    try {
      const fs = await import('fs/promises');
      const data = await fs.readFile(this.dbPath, 'utf-8');
      return JSON.parse(data) as TodoDatabase;
    } catch (error) {
      // Если файл не существует, создаем новую БД
      return {
        tasks: [],
        version: '1.0.0',
        lastScan: null
      };
    }
  }

  /**
   * Сохранение базы данных задач
   */
  async saveDatabase(db: TodoDatabase): Promise<void> {
    const fs = await import('fs/promises');
    const path = await import('path');
    
    // Создаем директорию если нужно
    const dir = path.dirname(this.dbPath);
    await fs.mkdir(dir, { recursive: true });
    
    await fs.writeFile(this.dbPath, JSON.stringify(db, null, 2), 'utf-8');
  }

  /**
   * Получение всех задач
   */
  async getAllTasks(): Promise<TodoTask[]> {
    const db = await this.loadDatabase();
    return db.tasks;
  }

  /**
   * Получение задачи по ID
   */
  async getTaskById(id: string): Promise<TodoTask | null> {
    const db = await this.loadDatabase();
    return db.tasks.find(task => task.id === id) || null;
  }

  /**
   * Создание новой задачи
   */
  async createTask(task: Omit<TodoTask, 'id' | 'createdAt' | 'updatedAt'>): Promise<TodoTask> {
    const db = await this.loadDatabase();
    
    // Генерируем ID
    const id = this.generateTaskId(task.file, task.line);
    
    // Проверяем, не существует ли уже задача
    const existing = db.tasks.find(t => t.id === id);
    if (existing) {
      throw new Error(`Task with id ${id} already exists`);
    }
    
    const now = new Date().toISOString();
    const newTask: TodoTask = {
      ...task,
      id,
      createdAt: now,
      updatedAt: now
    };
    
    db.tasks.push(newTask);
    await this.saveDatabase(db);
    
    return newTask;
  }

  /**
   * Обновление задачи
   */
  async updateTask(id: string, updates: Partial<TodoTask>): Promise<TodoTask | null> {
    const db = await this.loadDatabase();
    const taskIndex = db.tasks.findIndex(t => t.id === id);
    
    if (taskIndex === -1) {
      return null;
    }
    
    db.tasks[taskIndex] = {
      ...db.tasks[taskIndex],
      ...updates,
      updatedAt: new Date().toISOString()
    };
    
    await this.saveDatabase(db);
    return db.tasks[taskIndex];
  }

  /**
   * Удаление задачи
   */
  async deleteTask(id: string): Promise<boolean> {
    const db = await this.loadDatabase();
    const initialLength = db.tasks.length;
    db.tasks = db.tasks.filter(t => t.id !== id);
    
    if (db.tasks.length < initialLength) {
      await this.saveDatabase(db);
      return true;
    }
    return false;
  }

  /**
   * Получение статистики
   */
  async getStatistics(): Promise<TaskStatistics> {
    const tasks = await this.getAllTasks();
    
    const stats: TaskStatistics = {
      total: tasks.length,
      byPriority: {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0
      },
      byStatus: {
        OPEN: 0,
        IN_PROGRESS: 0,
        RESOLVED: 0,
        TESTING: 0,
        BLOCKED: 0
      },
      byType: {
        TODO: 0,
        FIXME: 0,
        HACK: 0,
        REFACTOR: 0
      },
      byFileType: {
        backend: 0,
        frontend: 0,
        script: 0,
        devops: 0,
        other: 0
      },
      openTasks: 0,
      inProgressTasks: 0,
      resolvedTasks: 0,
      criticalOpen: 0,
      averageAge: 0
    };
    
    let totalAge = 0;
    const now = Date.now();
    
    tasks.forEach(task => {
      stats.byPriority[task.priority]++;
      stats.byStatus[task.status]++;
      stats.byType[task.type]++;
      stats.byFileType[task.fileType]++;
      
      if (task.status === 'OPEN') {
        stats.openTasks++;
        if (task.priority === 'CRITICAL') {
          stats.criticalOpen++;
        }
      }
      
      if (task.status === 'IN_PROGRESS') {
        stats.inProgressTasks++;
      }
      
      if (task.status === 'RESOLVED') {
        stats.resolvedTasks++;
      }
      
      // Вычисляем средний возраст
      const createdAt = new Date(task.createdAt).getTime();
      totalAge += (now - createdAt);
    });
    
    stats.averageAge = tasks.length > 0 ? totalAge / tasks.length / (1000 * 60 * 60 * 24) : 0; // в днях
    
    return stats;
  }

  /**
   * Фильтрация задач
   */
  async filterTasks(filters: {
    priority?: Priority[];
    status?: TaskStatus[];
    type?: TaskType[];
    fileType?: FileType[];
    assignedTo?: string;
    file?: string;
  }): Promise<TodoTask[]> {
    const tasks = await this.getAllTasks();
    
    return tasks.filter(task => {
      if (filters.priority && !filters.priority.includes(task.priority)) {
        return false;
      }
      if (filters.status && !filters.status.includes(task.status)) {
        return false;
      }
      if (filters.type && !filters.type.includes(task.type)) {
        return false;
      }
      if (filters.fileType && !filters.fileType.includes(task.fileType)) {
        return false;
      }
      if (filters.assignedTo && task.assignedTo !== filters.assignedTo) {
        return false;
      }
      if (filters.file && !task.file.includes(filters.file)) {
        return false;
      }
      return true;
    });
  }

  /**
   * Генерация ID задачи
   */
  private generateTaskId(file: string, line: number): string {
    const crypto = require('crypto');
    const content = `${file}:${line}`;
    return crypto.createHash('sha256').update(content).digest('hex').substring(0, 8);
  }
}


