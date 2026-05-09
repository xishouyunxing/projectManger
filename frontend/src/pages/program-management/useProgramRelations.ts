import { useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { programApi } from './programApi';
import type { Program, ProgramMapping } from './types';

interface UseProgramRelationsOptions {
  currentProgram: Program | null;
  setCurrentProgram: (program: Program | null) => void;
  setModalLoading: (loading: boolean) => void;
  loadData: () => Promise<void>;
}

export const useProgramRelations = ({
  currentProgram,
  setCurrentProgram,
  setModalLoading,
  loadData,
}: UseProgramRelationsOptions) => {
  const [relationDrawerVisible, setRelationDrawerVisible] = useState(false);
  const [relatedPrograms, setRelatedPrograms] = useState<ProgramMapping[]>([]);
  const [addRelationModalVisible, setAddRelationModalVisible] = useState(false);
  const [relationForm] = Form.useForm();
  const [mappingSearchKeyword, setMappingSearchKeyword] = useState('');
  const [debouncedMappingSearchKeyword, setDebouncedMappingSearchKeyword] =
    useState('');
  const [mappingFilterProductionLine, setMappingFilterProductionLine] =
    useState<number | null>(null);
  const [mappingFilterVehicleModel, setMappingFilterVehicleModel] = useState<
    number | null
  >(null);
  const [mappingFilterStatus, setMappingFilterStatus] = useState<string | null>(
    null,
  );
  const [mappingCandidatePrograms, setMappingCandidatePrograms] = useState<
    Program[]
  >([]);

  const resetMappingFilters = () => {
    relationForm.resetFields();
    setMappingSearchKeyword('');
    setDebouncedMappingSearchKeyword('');
    setMappingFilterProductionLine(null);
    setMappingFilterVehicleModel(null);
    setMappingFilterStatus(null);
  };

  const relatedProgramIds = useMemo(
    () => new Set(relatedPrograms.map((rp) => rp.child_program.id)),
    [relatedPrograms],
  );

  const availableMappingCandidatePrograms = useMemo(
    () =>
      mappingCandidatePrograms
        .filter((p) => p.id !== currentProgram?.id)
        .filter((p) => !p.mapping_info)
        .filter((p) => !relatedProgramIds.has(p.id)),
    [currentProgram?.id, mappingCandidatePrograms, relatedProgramIds],
  );

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      setDebouncedMappingSearchKeyword(mappingSearchKeyword.trim());
    }, 300);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [mappingSearchKeyword]);

  useEffect(() => {
    if (!addRelationModalVisible || !currentProgram) {
      setMappingCandidatePrograms([]);
      return;
    }

    let cancelled = false;
    const loadMappingCandidates = async () => {
      try {
        const candidates = await programApi.listMappingCandidates({
          page: 1,
          page_size: 50,
          keyword: debouncedMappingSearchKeyword || undefined,
          production_line_id: mappingFilterProductionLine || undefined,
          vehicle_model_id: mappingFilterVehicleModel || undefined,
          status: mappingFilterStatus || undefined,
        });
        if (cancelled) {
          return;
        }

        setMappingCandidatePrograms(candidates);
      } catch (error) {
        if (!cancelled) {
          console.error('Failed to load mapping candidates:', error);
          setMappingCandidatePrograms([]);
        }
      }
    };

    void loadMappingCandidates();

    return () => {
      cancelled = true;
    };
  }, [
    addRelationModalVisible,
    currentProgram,
    debouncedMappingSearchKeyword,
    mappingFilterProductionLine,
    mappingFilterVehicleModel,
    mappingFilterStatus,
  ]);

  const refreshRelatedPrograms = async (programId: number) => {
    const mappings = await programApi.listProgramMappings(programId);
    setRelatedPrograms(mappings);
  };

  const handleViewRelations = async (record: Program) => {
    setCurrentProgram(record);
    setModalLoading(true);
    try {
      await refreshRelatedPrograms(record.id);
      setRelationDrawerVisible(true);
    } catch (error) {
      console.error('Failed to load mappings:', error);
      message.error('加载映射程序失败');
      setRelatedPrograms([]);
    } finally {
      setModalLoading(false);
    }
  };

  const handleAddRelation = async (values: { child_program_ids: number[] }) => {
    try {
      await programApi.addProgramMappings(
        currentProgram?.id || 0,
        values.child_program_ids,
      );
      message.success('映射成功');
      setAddRelationModalVisible(false);
      resetMappingFilters();
      if (currentProgram) {
        await refreshRelatedPrograms(currentProgram.id);
      }
      await loadData();
    } catch (error: any) {
      console.error('Failed to add mapping:', error);
      message.error(error.response?.data?.error || '映射失败');
    }
  };

  const handleDeleteRelation = async (mappingId: number) => {
    try {
      await programApi.deleteProgramMapping(mappingId);
      message.success('取消映射成功');
      if (currentProgram) {
        await refreshRelatedPrograms(currentProgram.id);
      }
      await loadData();
    } catch (error) {
      console.error('Failed to delete mapping:', error);
      message.error('取消映射失败');
    }
  };

  const closeAddRelationModal = () => {
    setAddRelationModalVisible(false);
    resetMappingFilters();
  };

  return {
    relationDrawerVisible,
    setRelationDrawerVisible,
    relatedPrograms,
    addRelationModalVisible,
    setAddRelationModalVisible,
    relationForm,
    mappingSearchKeyword,
    setMappingSearchKeyword,
    mappingFilterProductionLine,
    setMappingFilterProductionLine,
    mappingFilterVehicleModel,
    setMappingFilterVehicleModel,
    mappingFilterStatus,
    setMappingFilterStatus,
    availableMappingCandidatePrograms,
    handleViewRelations,
    handleAddRelation,
    handleDeleteRelation,
    closeAddRelationModal,
  };
};
