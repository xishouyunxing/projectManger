import { useEffect, useMemo, useState } from 'react';
import { message } from 'antd';
import api from '../../services/api';
import { buildEnabledCustomFields, normalizeCustomFieldValues } from './utils';
import type {
  Program,
  ProgramCustomFieldDefinition,
  ProductionLine,
  ProgramVersion,
  VehicleModel,
} from './types';

interface UseProgramManagementDataOptions {
  programPage: number;
  programPageSize: number;
  searchKeyword: string;
  filterProductionLine: number | null;
  filterVehicleModel: number | null;
  filterStatus: string | null;
  filterDateRange: [string | null, string | null];
  customFieldFilterValues: Record<string, string>;
  selectedProgramId: number;
  userId?: number;
}

export const useProgramManagementData = ({
  programPage,
  programPageSize,
  searchKeyword,
  filterProductionLine,
  filterVehicleModel,
  filterStatus,
  filterDateRange,
  customFieldFilterValues,
  selectedProgramId,
  userId,
}: UseProgramManagementDataOptions) => {
  const [programs, setPrograms] = useState<Program[]>([]);
  const [productionLines, setProductionLines] = useState<ProductionLine[]>([]);
  const [vehicleModels, setVehicleModels] = useState<VehicleModel[]>([]);
  const [tableLoading, setTableLoading] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [programTotal, setProgramTotal] = useState(0);

  const [versions, setVersions] = useState<ProgramVersion[]>([]);
  const [versionsPage, setVersionsPage] = useState(1);
  const [versionsPageSize, setVersionsPageSize] = useState(20);
  const [versionsTotal, setVersionsTotal] = useState(0);
  const [customFields, setCustomFields] = useState<
    ProgramCustomFieldDefinition[]
  >([]);

  const loadData = async () => {
    setTableLoading(true);
    try {
      const customFieldParams = Object.fromEntries(
        Object.entries(customFieldFilterValues)
          .map(([fieldId, value]) => [fieldId, value.trim()] as const)
          .filter(([, value]) => Boolean(value))
          .map(([fieldId, value]) => [`custom_field_${fieldId}`, value]),
      );

      const programsRes = await api.get('/programs', {
        params: {
          page: programPage,
          page_size: programPageSize,
          keyword: searchKeyword || undefined,
          production_line_id: filterProductionLine || undefined,
          vehicle_model_id: filterVehicleModel || undefined,
          status: filterStatus || undefined,
          date_from: filterDateRange[0] || undefined,
          date_to: filterDateRange[1] || undefined,
          ...customFieldParams,
        },
      });
      setPrograms(programsRes.data?.items || programsRes.data || []);
      setProgramTotal(Number(programsRes.data?.total) || 0);
    } catch (error) {
      console.error('Failed to load data:', error);
      message.error('加载数据失败，请刷新重试');
    } finally {
      setTableLoading(false);
    }
  };

  const loadVersions = async (
    programId: number,
    page = 1,
    pageSize = versionsPageSize,
  ) => {
    const response = await api.get(`/files/program/${programId}`, {
      params: {
        page,
        page_size: pageSize,
      },
    });
    const nextVersions = response.data?.versions || [];
    setVersions(nextVersions);
    setVersionsPage(Number(response.data?.page) || page);
    setVersionsPageSize(Number(response.data?.page_size) || pageSize);
    setVersionsTotal(
      Number(response.data?.total_versions) || nextVersions.length,
    );
    return nextVersions;
  };

  const loadCustomFields = async (productionLineId: number) => {
    const response = await api.get(
      `/production-lines/${productionLineId}/custom-fields`,
    );
    return buildEnabledCustomFields(response.data);
  };

  const loadSelectorData = async () => {
    try {
      const [linesRes, modelsRes] = await Promise.all([
        api.get('/production-lines'),
        api.get('/vehicle-models', {
          params: {
            scope: 'selector',
          },
        }),
      ]);
      setProductionLines(linesRes.data);
      setVehicleModels(modelsRes.data);
    } catch (error) {
      console.error('Failed to load selector data:', error);
      message.error('加载筛选数据失败，请刷新重试');
    }
  };

  useEffect(() => {
    void loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    programPage,
    programPageSize,
    searchKeyword,
    filterProductionLine,
    filterVehicleModel,
    filterStatus,
    filterDateRange,
    customFieldFilterValues,
  ]);

  useEffect(() => {
    void loadSelectorData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filteredPrograms = useMemo(() => {
    return programs.filter((a) => Boolean(a)).sort((a, b) => {
      if (selectedProgramId) {
        if (a.id === selectedProgramId) return -1;
        if (b.id === selectedProgramId) return 1;
      }

      const aEditingByMe =
        a.status === 'in_progress' && a.editing_by === userId;
      const bEditingByMe =
        b.status === 'in_progress' && b.editing_by === userId;
      if (aEditingByMe && !bEditingByMe) return -1;
      if (!aEditingByMe && bEditingByMe) return 1;

      return 0;
    });
  }, [programs, selectedProgramId, userId]);

  return {
    programs,
    setPrograms,
    productionLines,
    vehicleModels,
    tableLoading,
    modalLoading,
    setModalLoading,
    programTotal,
    loadData,
    filteredPrograms,

    versions,
    setVersions,
    versionsPage,
    setVersionsPage,
    versionsPageSize,
    setVersionsPageSize,
    versionsTotal,
    setVersionsTotal,
    loadVersions,

    customFields,
    setCustomFields,
    loadCustomFields,
    normalizeCustomFieldValues,
  };
};
