import type {
  Program,
  ProgramCustomFieldDefinition,
  ProgramFormValues,
  ProgramListCustomFieldValue,
  ProgramVersion,
} from './types';

export const getVersionSelectionKey = (version: ProgramVersion | null | undefined) => {
  if (!version) {
    return null;
  }
  if (version.id && version.id > 0) {
    return `id:${version.id}`;
  }
  return `version:${version.version}|created_at:${version.created_at}`;
};

export const buildEnabledCustomFields = (data: unknown): ProgramCustomFieldDefinition[] => {
  if (!Array.isArray(data)) {
    return [];
  }

  return data
    .filter(
      (field): field is ProgramCustomFieldDefinition =>
        typeof field === 'object' &&
        field !== null &&
        Boolean((field as ProgramCustomFieldDefinition).enabled),
    )
    .sort((a, b) => a.sort_order - b.sort_order);
};

export const normalizeCustomFieldValues = (
  values: Program['custom_field_values'],
): Record<string, string> => {
  if (Array.isArray(values)) {
    return values.reduce<Record<string, string>>((result, field) => {
      if (typeof field.value === 'string') {
        result[String(field.field_id)] = field.value;
      }
      return result;
    }, {});
  }

  if (!values || typeof values !== 'object') {
    return {};
  }

  return Object.entries(values).reduce<Record<string, string>>((result, [key, value]) => {
    if (typeof value === 'string') {
      result[String(key)] = value;
    }
    return result;
  }, {});
};

export const buildBaseProgramPayload = (values: any, fallbackDescription?: string) => {
  const { custom_field_values, ...baseValues } = values;
  if (typeof baseValues.description !== 'string') {
    baseValues.description = fallbackDescription || '';
  }
  if (
    Object.prototype.hasOwnProperty.call(baseValues, 'vehicle_model_id') &&
    (baseValues.vehicle_model_id === undefined || baseValues.vehicle_model_id === '')
  ) {
    baseValues.vehicle_model_id = null;
  }
  return baseValues;
};

export const buildProgramMutationPayload = (values: any, fallbackDescription?: string) => ({
  ...buildBaseProgramPayload(values, fallbackDescription),
  custom_field_values: buildCustomFieldValuesPayload(values).values,
});

export const buildCustomFieldValuesPayload = (values: any) => {
  const customFieldValues = values?.custom_field_values;

  if (!customFieldValues || typeof customFieldValues !== 'object') {
    return { values: [] };
  }

  return {
    values: Object.entries(customFieldValues).reduce<Array<{ field_id: number; value: string }>>(
      (result, [fieldId, fieldValue]) => {
        if (typeof fieldValue !== 'string') {
          return result;
        }

        const trimmedValue = fieldValue.trim();
        if (!trimmedValue) {
          return result;
        }

        result.push({
          field_id: Number(fieldId),
          value: trimmedValue,
        });
        return result;
      },
      [],
    ),
  };
};

export const getProgramCustomFieldValue = (program: Program, fieldId: number) => {
  if (Array.isArray(program.custom_field_values)) {
    const match = program.custom_field_values.find((field) => field.field_id === fieldId);
    return typeof match?.value === 'string' ? match.value : '';
  }

  if (!program.custom_field_values || typeof program.custom_field_values !== 'object') {
    return '';
  }

  const value = program.custom_field_values[String(fieldId)];
  return typeof value === 'string' ? value : '';
};

export const getProgramCustomFieldSummaries = (
  program: Program,
): ProgramListCustomFieldValue[] => {
  if (Array.isArray(program.custom_field_values)) {
    return [...program.custom_field_values].sort((a, b) => a.sort_order - b.sort_order);
  }

  return [];
};

export const formatFileSize = (size?: number) => {
  if (!size) {
    return '0 KB';
  }

  if (size >= 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  }

  return `${(size / 1024).toFixed(1)} KB`;
};

export const getFileTypeLabel = (fileName: string) => {
  const normalizedName = fileName.toLowerCase();

  if (normalizedName.endsWith('.log')) {
    return 'Log File';
  }

  if (normalizedName.endsWith('.txt')) {
    return 'Text Document';
  }

  if (normalizedName.endsWith('.bin')) {
    return 'Binary File';
  }

  if (normalizedName.endsWith('.nc')) {
    return 'NC Program';
  }

  return 'Program File';
};

export const parseCustomFieldOptions = (optionsJson: string) => {
  if (!optionsJson) {
    return [] as string[];
  }

  try {
    const parsed = JSON.parse(optionsJson);
    return Array.isArray(parsed)
      ? parsed.filter((option): option is string => typeof option === 'string')
      : [];
  } catch {
    return [] as string[];
  }
};

export const buildPropertyDraftSnapshot = (values: ProgramFormValues): ProgramFormValues => ({
  name: values.name,
  code: values.code,
  status: values.status,
  production_line_id: values.production_line_id,
  vehicle_model_id: values.vehicle_model_id,
  custom_field_values: { ...(values.custom_field_values || {}) },
});
