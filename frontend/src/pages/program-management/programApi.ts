import api from '../../services/api';
import type {
  BatchImportPreview,
  BatchImportStatus,
  Program,
  ProgramCustomFieldDefinition,
  ProgramFormValues,
  ProgramMapping,
  ProgramVersion,
  ProductionLine,
  VehicleModel,
  WorkstationMapping,
} from './types';

export interface ProgramListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  production_line_id?: number;
  vehicle_model_id?: number;
  status?: string;
  date_from?: string;
  date_to?: string;
  [key: `custom_field_${string}`]: string | number | undefined;
}

export interface ProgramListResult {
  items: Program[];
  total: number;
}

export interface ProgramVersionListResult {
  versions: ProgramVersion[];
  page: number;
  pageSize: number;
  totalVersions: number;
}

const extractListData = <T>(payload: unknown): T[] => {
  if (Array.isArray(payload)) {
    return payload as T[];
  }
  if (
    payload &&
    typeof payload === 'object' &&
    Array.isArray((payload as { items?: unknown }).items)
  ) {
    return (payload as { items: T[] }).items;
  }
  return [];
};

const extractPagedListData = <T>(payload: unknown) => {
  const items = extractListData<T>(payload);
  const pagePayload = payload as
    | { total?: unknown; page?: unknown; page_size?: unknown }
    | undefined;
  return {
    items,
    total:
      typeof pagePayload?.total === 'number' ? pagePayload.total : items.length,
  };
};

export const programApi = {
  async listPrograms(params: ProgramListParams): Promise<ProgramListResult> {
    const response = await api.get('/programs', { params });
    const page = extractPagedListData<Program>(response.data);
    return { items: page.items, total: page.total };
  },

  async listProductionLines(): Promise<ProductionLine[]> {
    const response = await api.get('/production-lines');
    return response.data || [];
  },

  async listVehicleModelsForSelector(): Promise<VehicleModel[]> {
    const response = await api.get('/vehicle-models', {
      params: { scope: 'selector' },
    });
    return response.data || [];
  },

  async listProgramVersions(
    programId: number,
    page: number,
    pageSize: number,
  ): Promise<ProgramVersionListResult> {
    const response = await api.get(`/files/program/${programId}`, {
      params: { page, page_size: pageSize },
    });
    const versions = response.data?.versions || [];
    return {
      versions,
      page: Number(response.data?.page) || page,
      pageSize: Number(response.data?.page_size) || pageSize,
      totalVersions: Number(response.data?.total_versions) || versions.length,
    };
  },

  async listCustomFields(
    productionLineId: number,
  ): Promise<ProgramCustomFieldDefinition[]> {
    const response = await api.get(
      `/production-lines/${productionLineId}/custom-fields`,
    );
    return response.data || [];
  },

  async exportExcel(params: ProgramListParams) {
    return api.get('/programs/export/excel', {
      params,
      responseType: 'blob',
    });
  },

  async listMappingCandidates(params: ProgramListParams): Promise<Program[]> {
    const response = await api.get('/programs', { params });
    return extractListData<Program>(response.data);
  },

  async listProgramMappings(parentProgramId: number): Promise<ProgramMapping[]> {
    const response = await api.get(`/program-mappings/by-parent/${parentProgramId}`);
    return response.data || [];
  },

  async addProgramMappings(
    parentProgramId: number,
    childProgramIds: number[],
  ): Promise<void> {
    await api.post(`/program-mappings?parent_program_id=${parentProgramId}`, {
      child_program_ids: childProgramIds,
    });
  },

  async deleteProgramMapping(mappingId: number): Promise<void> {
    await api.delete(`/program-mappings/${mappingId}`);
  },

  async createProgram(payload: ProgramFormValues): Promise<void> {
    await api.post('/programs', payload);
  },

  async updateProgram(
    programId: number,
    payload: ProgramFormValues,
  ): Promise<void> {
    await api.put(`/programs/${programId}`, payload);
  },

  async deleteProgram(programId: number): Promise<void> {
    await api.delete(`/programs/${programId}`);
  },

  async uploadBatchArchive(file: File): Promise<BatchImportPreview> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await api.post('/programs/batch-upload', formData, {
      headers: { 'Content-Type': undefined },
    });
    return response.data;
  },

  async startBatchImport(
    previewId: string | undefined,
    mappings: WorkstationMapping[],
  ): Promise<{ task_id: number }> {
    const response = await api.post('/programs/batch-import', {
      preview_id: previewId,
      mappings,
    });
    return response.data;
  },

  async getTaskStatus(taskId: number): Promise<BatchImportStatus> {
    const response = await api.get(`/tasks/${taskId}/status`);
    return response.data;
  },

  async uploadProgramFiles(formData: FormData) {
    return api.post('/files/upload', formData, {
      headers: { 'Content-Type': undefined },
    });
  },

  async downloadBlob(url: string) {
    return api.get(url, { responseType: 'blob' });
  },

  async deleteFile(fileId: number): Promise<void> {
    await api.delete(`/files/${fileId}`);
  },

  async updateVersionDescription(
    versionId: number,
    changeLog: string,
  ): Promise<void> {
    await api.put(`/versions/${versionId}`, { change_log: changeLog });
  },
};
