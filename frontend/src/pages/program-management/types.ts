export interface ProductionLine {
  id: number;
  name: string;
}

export interface VehicleModel {
  id: number;
  name: string;
}

export interface ProgramFile {
  id: number;
  file_name: string;
  file_size: number;
  created_at: string;
  file_exists?: boolean;
  uploader?: { name: string };
}

export interface ProgramVersion {
  id: number;
  version: string;
  is_current: boolean;
  change_log?: string;
  created_at: string;
  file_count?: number;
  files?: ProgramFile[];
  uploader?: { name: string };
}

export interface ProgramCustomFieldDefinition {
  id: number;
  name: string;
  field_type: 'text' | 'select';
  options_json: string;
  sort_order: number;
  enabled: boolean;
}

export interface ProgramListCustomFieldValue {
  field_id: number;
  field_name: string;
  field_type: 'text' | 'select';
  sort_order: number;
  value: string;
}

export interface Program {
  id: number;
  name: string;
  code: string;
  version?: string;
  status: string;
  production_line_id: number;
  vehicle_model_id?: number;
  production_line?: ProductionLine;
  vehicle_model?: VehicleModel;
  description?: string;
  created_at: string;
  editing_by?: number | null;
  editing_user?: { id: number; name: string } | null;
  custom_field_values?: ProgramListCustomFieldValue[] | Record<string, unknown>;
  own_version_count?: number;
  own_file_count?: number;
  mapping_info?: {
    mapping_id: number;
    parent_program_id: number;
    parent_program_name: string;
    parent_program_code: string;
  } | null;
}

export type ProgramMapping = {
  id: number;
  parent_program: Program;
  child_program: Program;
};

export interface WorkstationInfo {
  name: string;
  programs: {
    name: string;
    files: { name: string; size: number; path: string }[];
  }[];
}

export interface BatchImportPreview {
  workstations: WorkstationInfo[];
  total_programs: number;
  total_files: number;
  temp_dir: string;
}

export interface WorkstationMapping {
  workstation_name: string;
  production_line_id: number | null;
  vehicle_model_id: number | null;
}

export interface BatchImportStatus {
  status: 'idle' | 'processing' | 'completed' | 'failed';
  total: number;
  processed: number;
  success: number;
  failed: number;
  progress: number;
  current_item: string;
  error_message: string;
}

export interface ProgramFormValues {
  name?: string;
  code?: string;
  status?: string;
  production_line_id?: number;
  vehicle_model_id?: number;
  description?: string;
  custom_field_values?: Record<string, string>;
}
