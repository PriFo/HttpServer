/**
 * TypeScript интерфейсы для системы управления TODO
 */

export type Priority = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
export type TaskType = 'TODO' | 'FIXME' | 'HACK' | 'REFACTOR';
export type TaskStatus = 'OPEN' | 'IN_PROGRESS' | 'RESOLVED' | 'TESTING' | 'BLOCKED';
export type FileType = 'backend' | 'frontend' | 'script' | 'devops' | 'other';

export interface TodoTask {
  id: string;
  file: string;
  line: number;
  description: string;
  type: TaskType;
  priority: Priority;
  status: TaskStatus;
  assignedTo?: string;
  createdAt: string;
  updatedAt: string;
  fileType: FileType;
  estimatedHours: number;
  actualHours?: number;
  dependencies: string[];
  relatedFiles: string[];
  tags?: string[];
  comments?: TaskComment[];
}

export interface TaskComment {
  id: string;
  author: string;
  content: string;
  createdAt: string;
}

export interface TodoConfig {
  todoSystem: {
    enabled: boolean;
    scanInterval: string;
    autoAssign: boolean;
    notifications: {
      slack: boolean;
      email: boolean;
    };
    priorities: {
      [key in Priority]: {
        color: string;
        label: string;
        blockCommit?: boolean;
      };
    };
    filePatterns: {
      [key: string]: string[];
    };
    excludePatterns: string[];
  };
}

export interface TeamConfig {
  team: {
    [teamName: string]: string[];
  };
  specialties: {
    [technology: string]: string[];
  };
  workload: {
    [developer: string]: number;
  };
}

export interface TodoDatabase {
  tasks: TodoTask[];
  version: string;
  lastScan: string | null;
}

export interface TaskStatistics {
  total: number;
  byPriority: {
    [key in Priority]: number;
  };
  byStatus: {
    [key in TaskStatus]: number;
  };
  byType: {
    [key in TaskType]: number;
  };
  byFileType: {
    [key in FileType]: number;
  };
  openTasks: number;
  inProgressTasks: number;
  resolvedTasks: number;
  criticalOpen: number;
  averageAge: number;
}

export interface AssignmentEngine {
  assignTask(task: TodoTask, teamConfig: TeamConfig): string;
  calculateWorkload(developer: string, teamConfig: TeamConfig): number;
  findBestAssignee(task: TodoTask, teamConfig: TeamConfig): string;
}


