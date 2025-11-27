import { request, APIRequestContext } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// Test API client configuration
const FRONTEND_API_URL = process.env.API_BASE_URL || 'http://localhost:3000/api';
const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:9999';
const TEST_TIMEOUT = 30000;

// Используем бэкенд напрямую для более надежных тестов
const API_BASE_URL = BACKEND_URL + '/api';

/**
 * Creates a new test client
 * @param options - Optional client data
 * @returns Client object with id and name
 */
export async function createTestClient(options?: {
  name?: string;
  legal_name?: string;
  description?: string;
  contact_email?: string;
  contact_phone?: string;
  tax_id?: string;
}) {
  const context = await request.newContext();
  const clientData = {
    name: options?.name || `E2E Test Client ${Date.now()}`,
    legal_name: options?.legal_name || `E2E Test Client ${Date.now()}`,
    description: options?.description || 'Тестовый клиент для E2E теста',
    contact_email: options?.contact_email || 'test@example.com',
    contact_phone: options?.contact_phone || '+79991234567',
    tax_id: options?.tax_id || '',
  };
  
  const response = await context.post(`${API_BASE_URL}/clients`, {
    data: clientData,
    headers: {
      'Content-Type': 'application/json',
    },
  });
  
  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to create test client: ${response.status()} - ${errorText}`);
  }
  
  return await response.json();
}

/**
 * Creates a new test project for a client
 * @param clientId - ID of the client
 * @param options - Optional project data
 * @returns Project object with id and name
 */
export async function createTestProject(
  clientId: string | number,
  options?: {
    name?: string;
    project_type?: string;
    description?: string;
    source_system?: string;
    target_quality_score?: number;
  }
) {
  const context = await request.newContext();
  const projectData = {
    name: options?.name || `E2E Test Project ${Date.now()}`,
    project_type: options?.project_type || 'normalization',
    description: options?.description || 'Тестовый проект для E2E теста',
    source_system: options?.source_system || '1C',
    target_quality_score: options?.target_quality_score || 0.9,
  };
  
  const response = await context.post(`${API_BASE_URL}/clients/${clientId}/projects`, {
    data: projectData,
    headers: {
      'Content-Type': 'application/json',
    },
  });
  
  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to create test project: ${response.status()} - ${errorText}`);
  }
  
  return await response.json();
}

/**
 * Uploads a database file to a project
 * @param clientId - ID of the client
 * @param projectId - ID of the project
 * @param dbPath - Path to the database file
 * @param options - Optional upload options
 * @returns Database object with id
 */
export async function uploadDatabaseFile(
  clientId: string | number,
  projectId: string | number,
  dbPath: string,
  options?: {
    auto_create?: boolean;
  }
) {
  const context = await request.newContext();
  
  if (!fs.existsSync(dbPath)) {
    throw new Error(`Database file not found: ${dbPath}`);
  }
  
  const fileBuffer = fs.readFileSync(dbPath);
  const fileName = path.basename(dbPath);
  
  const formData = new FormData();
  const blob = new Blob([fileBuffer], { type: 'application/x-sqlite3' });
  formData.append('file', blob, fileName);
  formData.append('auto_create', (options?.auto_create ?? false).toString());
  
  const response = await context.post(
    `${API_BASE_URL}/clients/${clientId}/projects/${projectId}/databases`,
    {
      multipart: formData,
    }
  );
  
  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to upload database: ${response.status()} - ${errorText}`);
  }
  
  const result = await response.json();
  return result.database || result;
}

/**
 * Cleans up test data (database, project, client)
 * @param clientId - ID of the client to delete
 * @param projectId - Optional project ID to delete
 * @param databaseId - Optional database ID to delete
 */
export async function cleanupTestData(
  clientId?: string | number,
  projectId?: string | number,
  databaseId?: string | number
) {
  const context = await request.newContext();
  
  try {
    // Delete database if provided
    if (databaseId && clientId && projectId) {
      try {
        await context.delete(
          `${API_BASE_URL}/clients/${clientId}/projects/${projectId}/databases/${databaseId}`
        );
      } catch (error) {
        console.warn(`Failed to delete database ${databaseId}:`, error);
      }
    }
    
    // Delete project if provided
    if (projectId && clientId) {
      try {
        await context.delete(`${API_BASE_URL}/clients/${clientId}/projects/${projectId}`);
      } catch (error) {
        console.warn(`Failed to delete project ${projectId}:`, error);
      }
    }
    
    // Delete client if provided
    if (clientId) {
      try {
        await context.delete(`${API_BASE_URL}/clients/${clientId}`);
      } catch (error) {
        console.warn(`Failed to delete client ${clientId}:`, error);
      }
    }
  } catch (error) {
    console.warn('Error during cleanup:', error);
  }
}

/**
 * Checks normalization status for a project
 * @param clientId - ID of the client
 * @param projectId - ID of the project
 * @returns Normalization status object or null if not found
 */
export async function getNormalizationStatus(
  clientId: string | number,
  projectId: string | number
): Promise<any> {
  const context = await request.newContext();
  const response = await context.get(
    `${API_BASE_URL}/clients/${clientId}/projects/${projectId}/normalization/status`
  );
  
  if (!response.ok()) {
    return null;
  }
  
  return await response.json();
}

/**
 * Starts normalization for a project
 * @param clientId - ID of the client
 * @param projectId - ID of the project
 * @param options - Normalization options
 * @returns Response object
 */
export async function startNormalization(
  clientId: string | number,
  projectId: string | number,
  options?: Record<string, any>
) {
  const context = await request.newContext();
  const response = await context.post(
    `${API_BASE_URL}/clients/${clientId}/projects/${projectId}/normalization/start`,
    {
      data: options || {},
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );
  
  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to start normalization: ${response.status()} - ${errorText}`);
  }
  
  return await response.json();
}

/**
 * Stops normalization for a project
 * @param clientId - ID of the client
 * @param projectId - ID of the project
 * @returns Response object
 */
export async function stopNormalization(
  clientId: string | number,
  projectId: string | number
) {
  const context = await request.newContext();
  const response = await context.post(
    `${API_BASE_URL}/clients/${clientId}/projects/${projectId}/normalization/stop`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );
  
  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to stop normalization: ${response.status()} - ${errorText}`);
  }
  
  return await response.json();
}

/**
 * Finds a test database file in common locations
 * @returns Path to database file or null if not found
 */
export function findTestDatabase(): string | null {
  const possiblePaths = [
    '1c_data.db',
    'data/1c_data.db',
    'test-data.db',
    'data/test-data.db',
    'tests/data/test-data.db',
  ];

  for (const dbPath of possiblePaths) {
    if (fs.existsSync(dbPath)) {
      return dbPath;
    }
  }

  return null;
}

/**
 * Creates a backup of databases
 * @param options - Backup options
 * @returns Backup information object
 */
export async function createBackup(options?: {
  includeMain?: boolean;
  includeUploads?: boolean;
  includeService?: boolean;
  selectedFiles?: string[];
  format?: 'zip' | 'copy' | 'both';
}) {
  const context = await request.newContext();
  const backupData = {
    include_main: options?.includeMain ?? true,
    include_uploads: options?.includeUploads ?? true,
    include_service: options?.includeService ?? false,
    selected_files: options?.selectedFiles || [],
    format: options?.format || 'both',
  };

  const response = await context.post(`${API_BASE_URL}/databases/backup`, {
    data: backupData,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to create backup: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Lists all available backups
 * @returns Array of backup objects
 */
export async function listBackups(): Promise<any[]> {
  const context = await request.newContext();
  const response = await context.get(`${API_BASE_URL}/databases/backups`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to list backups: ${response.status()} - ${errorText}`);
  }

  const result = await response.json();
  return result.backups || [];
}

/**
 * Restores a database from a backup
 * @param backupFileName - Name of the backup file to restore
 * @param targetPath - Optional target path for restoration
 * @returns Response object
 */
export async function restoreBackup(
  backupFileName: string,
  targetPath?: string
) {
  const context = await request.newContext();
  const restoreData: any = {
    backup_file: backupFileName,
  };

  if (targetPath) {
    restoreData.target_path = targetPath;
  }

  const response = await context.post(`${API_BASE_URL}/databases/restore`, {
    data: restoreData,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to restore backup: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets quality duplicates for a database
 * @param databasePath - Path to the database (optional)
 * @param filters - Optional filters (unmerged, limit, offset)
 * @returns Duplicates data
 */
export async function getQualityDuplicates(
  databasePath?: string,
  filters?: {
    unmerged?: boolean;
    limit?: number;
    offset?: number;
  }
) {
  const context = await request.newContext();
  const params = new URLSearchParams();

  if (databasePath) {
    params.append('database', databasePath);
  }

  if (filters?.unmerged) {
    params.append('unmerged', 'true');
  }

  if (filters?.limit) {
    params.append('limit', filters.limit.toString());
  }

  if (filters?.offset) {
    params.append('offset', filters.offset.toString());
  }

  const url = `${API_BASE_URL}/quality/duplicates${params.toString() ? '?' + params.toString() : ''}`;
  const response = await context.get(url);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get quality duplicates: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Merges duplicate records
 * @param groupId - ID of the duplicate group
 * @param masterId - ID of the master record
 * @param mergeIds - Array of IDs to merge into master
 * @returns Response object
 */
export async function mergeDuplicates(
  groupId: number,
  masterId: number,
  mergeIds: number[]
) {
  const context = await request.newContext();
  const response = await context.post(
    `${API_BASE_URL}/quality/duplicates/${groupId}/merge`,
    {
      data: {
        master_id: masterId,
        merge_ids: mergeIds,
      },
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to merge duplicates: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets quality metrics for a database
 * @param databasePath - Path to the database (optional)
 * @returns Quality metrics object
 */
export async function getQualityMetrics(databasePath?: string) {
  const context = await request.newContext();
  const params = new URLSearchParams();

  if (databasePath) {
    params.append('database', databasePath);
  }

  const url = `${API_BASE_URL}/quality/metrics${params.toString() ? '?' + params.toString() : ''}`;
  const response = await context.get(url);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get quality metrics: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets monitoring provider metrics
 * @returns Monitoring data with providers and system stats
 */
export async function getMonitoringProviders() {
  const context = await request.newContext();
  const response = await context.get(`${API_BASE_URL}/monitoring/providers`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get monitoring providers: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets monitoring metrics
 * @returns General monitoring metrics
 */
export async function getMonitoringMetrics() {
  const context = await request.newContext();
  const response = await context.get(`${API_BASE_URL}/monitoring/metrics`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get monitoring metrics: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Lists benchmarks (эталоны)
 * @param entityType - Optional entity type filter (counterparty, nomenclature)
 * @param activeOnly - Show only active benchmarks (default: true)
 * @param limit - Limit results (default: 50)
 * @param offset - Offset for pagination (default: 0)
 * @returns List of benchmarks
 */
export async function listBenchmarks(
  entityType?: string,
  activeOnly: boolean = true,
  limit: number = 50,
  offset: number = 0
) {
  const context = await request.newContext();
  const params = new URLSearchParams();
  if (entityType) params.append('type', entityType);
  if (!activeOnly) params.append('active', 'false');
  if (limit !== 50) params.append('limit', limit.toString());
  if (offset !== 0) params.append('offset', offset.toString());

  const url = `${API_BASE_URL}/benchmarks${params.toString() ? '?' + params.toString() : ''}`;
  const response = await context.get(url);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to list benchmarks: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets benchmark by ID
 * @param benchmarkId - Benchmark ID
 * @returns Benchmark object
 */
export async function getBenchmarkById(benchmarkId: string) {
  const context = await request.newContext();
  const response = await context.get(`${API_BASE_URL}/benchmarks/${benchmarkId}`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get benchmark: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Searches benchmarks
 * @param query - Search query
 * @param entityType - Optional entity type filter
 * @returns Search results
 */
export async function searchBenchmarks(query: string, entityType?: string) {
  const context = await request.newContext();
  const params = new URLSearchParams({ q: query });
  if (entityType) params.append('type', entityType);

  const response = await context.get(`${API_BASE_URL}/benchmarks/search?${params.toString()}`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to search benchmarks: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Creates benchmark from upload
 * @param uploadId - Upload ID
 * @param entityType - Entity type (counterparty, nomenclature)
 * @returns Created benchmark
 */
export async function createBenchmarkFromUpload(uploadId: string, entityType: string = 'counterparty') {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/benchmarks/from-upload`, {
    data: {
      upload_id: uploadId,
      entity_type: entityType,
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to create benchmark from upload: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Creates a new benchmark
 * @param data - Benchmark data
 * @returns Created benchmark
 */
export async function createBenchmark(data: {
  entity_type: string;
  name: string;
  data?: Record<string, any>;
  source_upload_id?: string;
  source_client_id?: number;
  is_active?: boolean;
}) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/benchmarks`, {
    data,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to create benchmark: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Updates a benchmark
 * @param benchmarkId - Benchmark ID
 * @param data - Update data
 * @returns Updated benchmark
 */
export async function updateBenchmark(
  benchmarkId: string,
  data: {
    name?: string;
    data?: Record<string, any>;
    is_active?: boolean;
  }
) {
  const context = await request.newContext();
  const response = await context.put(`${API_BASE_URL}/benchmarks/${benchmarkId}`, {
    data,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to update benchmark: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Deletes a benchmark
 * @param benchmarkId - Benchmark ID
 * @returns Success message
 */
export async function deleteBenchmark(benchmarkId: string) {
  const context = await request.newContext();
  const response = await context.delete(`${API_BASE_URL}/benchmarks/${benchmarkId}`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to delete benchmark: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets KPVED hierarchy
 * @param parentCode - Optional parent code
 * @param level - Optional level
 * @returns KPVED hierarchy
 */
export async function getKpvedHierarchy(parentCode?: string, level?: string) {
  const context = await request.newContext();
  const params = new URLSearchParams();
  if (parentCode) params.append('parent', parentCode);
  if (level) params.append('level', level);

  const url = `${API_BASE_URL}/kpved/hierarchy${params.toString() ? '?' + params.toString() : ''}`;
  const response = await context.get(url);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get KPVED hierarchy: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Searches KPVED
 * @param query - Search query
 * @param limit - Result limit (default: 50)
 * @returns Search results
 */
export async function searchKpved(query: string, limit: number = 50) {
  const context = await request.newContext();
  const params = new URLSearchParams({ q: query });
  if (limit !== 50) params.append('limit', limit.toString());

  const response = await context.get(`${API_BASE_URL}/kpved/search?${params.toString()}`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to search KPVED: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Gets KPVED statistics
 * @returns KPVED statistics
 */
export async function getKpvedStats() {
  const context = await request.newContext();
  const response = await context.get(`${API_BASE_URL}/kpved/stats`);

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to get KPVED stats: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Tests classification
 * @param normalizedName - Normalized name to classify
 * @param model - Optional model name
 * @returns Classification result
 */
export async function testClassification(normalizedName: string, model?: string) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/kpved/classify-test`, {
    data: {
      normalized_name: normalizedName,
      model: model || '',
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to test classification: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Classifies hierarchically
 * @param normalizedName - Normalized name to classify
 * @param category - Category
 * @param model - Optional model name
 * @returns Classification result
 */
export async function classifyHierarchical(
  normalizedName: string,
  category: string,
  model?: string
) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/kpved/classify-hierarchical`, {
    data: {
      normalized_name: normalizedName,
      category: category,
      model: model || '',
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to classify hierarchically: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Resets classification
 * @param normalizedName - Normalized name
 * @param category - Category
 * @returns Success message
 */
export async function resetClassification(normalizedName: string, category: string) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/kpved/reset`, {
    data: {
      normalized_name: normalizedName,
      category: category,
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to reset classification: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Marks classification as incorrect
 * @param normalizedName - Normalized name
 * @param category - Category
 * @returns Success message
 */
export async function markClassificationIncorrect(normalizedName: string, category: string) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/kpved/mark-incorrect`, {
    data: {
      normalized_name: normalizedName,
      category: category,
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to mark as incorrect: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}

/**
 * Marks classification as correct
 * @param normalizedName - Normalized name
 * @param category - Category
 * @returns Success message
 */
export async function markClassificationCorrect(normalizedName: string, category: string) {
  const context = await request.newContext();
  const response = await context.post(`${API_BASE_URL}/kpved/mark-correct`, {
    data: {
      normalized_name: normalizedName,
      category: category,
    },
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok()) {
    const errorText = await response.text();
    throw new Error(`Failed to mark as correct: ${response.status()} - ${errorText}`);
  }

  return await response.json();
}
