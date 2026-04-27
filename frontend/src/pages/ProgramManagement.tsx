import { useEffect, useMemo, useState } from 'react';
import type { TablePaginationConfig } from 'antd/es/table';
import { useSearchParams } from 'react-router-dom';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  message,
  Typography,
  Tag,
  Popconfirm,
  Tooltip,
  Progress,
  Alert,
  Steps,
  Card,
  Divider,
  Statistic,
  Drawer,
  Dropdown,
  Pagination,
} from 'antd';
import {
  PlusOutlined,
  UploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileOutlined,
  ClockCircleOutlined,
  ClockCircleFilled,
  FolderAddOutlined,
  CheckCircleOutlined,
  LoadingOutlined,
  EditOutlined,
  UserOutlined,
  LinkOutlined,
  SettingOutlined,
  AppstoreOutlined,
  CloseOutlined,
  FileTextOutlined,
  LeftOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import ProgramHeader from './program-management/ProgramHeader';
import ProgramFilterPanel from './program-management/ProgramFilterPanel';
import { useProgramManagementData } from './program-management/useProgramManagementData';
import {
  buildProgramMutationPayload,
  buildPropertyDraftSnapshot,
  formatFileSize,
  getFileTypeLabel,
  getProgramCustomFieldSummaries,
  getVersionSelectionKey,
  normalizeCustomFieldValues,
  parseCustomFieldOptions,
} from './program-management/utils';
import type {
  BatchImportPreview,
  BatchImportStatus,
  Program,
  ProgramCustomFieldDefinition,
  ProgramFormValues,
  ProgramMapping,
  WorkstationInfo,
  WorkstationMapping,
} from './program-management/types';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Step } = Steps;

const ProgramManagement = () => {
  const [searchParams] = useSearchParams();
  const { user } = useAuth();
  const selectedProgramId = Number(searchParams.get('id') || 0);
  const initialSearchKeyword = searchParams.get('keyword') || '';

  const [modalVisible, setModalVisible] = useState(false);
  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [fileModalVisible, setFileModalVisible] = useState(false);
  const [currentProgram, setCurrentProgram] = useState<Program | null>(null);
  const [selectedVersionKey, setSelectedVersionKey] = useState<string | null>(
    null,
  );
  const [isEditingProperties, setIsEditingProperties] = useState(false);
  const [propertyDraftSnapshot, setPropertyDraftSnapshot] =
    useState<ProgramFormValues | null>(null);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [descriptionDraftSnapshot, setDescriptionDraftSnapshot] =
    useState<string>('');
  const [versionDescriptionValue, setVersionDescriptionValue] = useState('');
  const [form] = Form.useForm();
  const [uploadForm] = Form.useForm();

  // 筛选相关状态
  const [filterProductionLine, setFilterProductionLine] = useState<
    number | null
  >(null);
  const [filterVehicleModel, setFilterVehicleModel] = useState<number | null>(
    null,
  );
  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [searchInputValue, setSearchInputValue] = useState(initialSearchKeyword);
  const [searchKeyword, setSearchKeyword] = useState(initialSearchKeyword);
  const [filterDateRange, setFilterDateRange] = useState<
    [string | null, string | null]
  >([null, null]);
  const [customFieldFilters, setCustomFieldFilters] = useState<
    ProgramCustomFieldDefinition[]
  >([]);
  const [customFieldFilterValues, setCustomFieldFilterValues] = useState<
    Record<string, string>
  >({});
  const [programPage, setProgramPage] = useState(1);
  const [programPageSize, setProgramPageSize] = useState(20);

  const {
    productionLines,
    vehicleModels,
    tableLoading,
    modalLoading,
    setModalLoading,
    programTotal,
    loadData,
    filteredPrograms: sortedPrograms,
    versions,
    setVersions,
    versionsPage,
    setVersionsPage,
    versionsPageSize,
    versionsTotal,
    setVersionsTotal,
    loadVersions,
    customFields,
    setCustomFields,
    loadCustomFields,
  } = useProgramManagementData({
    programPage,
    programPageSize,
    searchKeyword,
    filterProductionLine,
    filterVehicleModel,
    filterStatus,
    filterDateRange,
    customFieldFilterValues,
    selectedProgramId,
    userId: user?.id,
  });

  // 关联管理相关状态
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

  const filteredPrograms = sortedPrograms;

  // 重置筛选
  const handleResetFilter = () => {
    setProgramPage(1);
    setFilterProductionLine(null);
    setFilterVehicleModel(null);
    setFilterStatus(null);
    setSearchInputValue('');
    setSearchKeyword('');
    setFilterDateRange([null, null]);
    setCustomFieldFilters([]);
    setCustomFieldFilterValues({});
  };

  const applySearchKeyword = () => {
    setProgramPage(1);
    setSearchKeyword(searchInputValue.trim());
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

  const handleFilterProductionLineChange = async (value?: number) => {
    const nextValue = value ?? null;
    setProgramPage(1);
    setFilterProductionLine(nextValue);
    setCustomFieldFilterValues({});

    if (!nextValue) {
      setCustomFieldFilters([]);
      return;
    }

    try {
      setCustomFieldFilters(await loadCustomFields(nextValue));
    } catch (error) {
      console.error('Failed to load custom fields:', error);
      setCustomFieldFilters([]);
    }
  };

  const handleCustomFieldFilterChange = (fieldId: string, value: string) => {
    setProgramPage(1);
    setCustomFieldFilterValues((current) => {
      if (!value) {
        const nextValues = { ...current };
        delete nextValues[fieldId];
        return nextValues;
      }

      return {
        ...current,
        [fieldId]: value,
      };
    });
  };

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      setProgramPage(1);
      setSearchKeyword(searchInputValue.trim());
    }, 300);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [searchInputValue]);

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
        const response = await api.get('/programs', {
          params: {
            page: 1,
            page_size: 50,
            keyword: debouncedMappingSearchKeyword || undefined,
            production_line_id: mappingFilterProductionLine || undefined,
            vehicle_model_id: mappingFilterVehicleModel || undefined,
            status: mappingFilterStatus || undefined,
          },
        });
        if (cancelled) {
          return;
        }

        setMappingCandidatePrograms(
          response.data?.items || response.data || [],
        );
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

  const handleModalProductionLineChange = async (
    productionLineId: number,
    options?: { preserveValues?: boolean },
  ) => {
    if (!options?.preserveValues) {
      form.setFieldValue('custom_field_values', {});
    }

    if (!productionLineId) {
      setCustomFields([]);
      return;
    }

    try {
      const nextFields = await loadCustomFields(productionLineId);
      if (form.getFieldValue('production_line_id') === productionLineId) {
        setCustomFields(nextFields);
      }
    } catch (error) {
      console.error('Failed to load custom fields:', error);
      if (form.getFieldValue('production_line_id') === productionLineId) {
        setCustomFields([]);
      }
    }
  };

  // 导出Excel
  const handleExportExcel = async () => {
    try {
      const params: any = {};
      if (filterProductionLine)
        params.production_line_id = filterProductionLine;
      if (filterVehicleModel) params.vehicle_model_id = filterVehicleModel;
      if (filterStatus) params.status = filterStatus;
      if (searchKeyword.trim()) params.keyword = searchKeyword.trim();
      if (filterDateRange[0]) params.date_from = filterDateRange[0];
      if (filterDateRange[1]) params.date_to = filterDateRange[1];
      Object.entries(customFieldFilterValues).forEach(([fieldId, value]) => {
        const trimmedValue = value.trim();
        if (trimmedValue) {
          params[`custom_field_${fieldId}`] = trimmedValue;
        }
      });

      const response = await api.get('/programs/export/excel', {
        params,
        responseType: 'blob',
      });

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;

      const contentDisposition = response.headers['content-disposition'];
      let fileName = '程序列表.xlsx';
      if (contentDisposition) {
        const match = contentDisposition.match(/filename=(.+)/);
        if (match) {
          fileName = match[1];
        }
      }

      link.setAttribute('download', fileName);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);

      message.success('导出成功');
    } catch (error) {
      console.error('Failed to export:', error);
      message.error('导出失败');
    }
  };

  const handleViewRelations = async (record: Program) => {
    setCurrentProgram(record);
    setModalLoading(true);
    try {
      const response = await api.get(
        `/program-mappings/by-parent/${record.id}`,
      );
      setRelatedPrograms(response.data || []);
      setRelationDrawerVisible(true);
    } catch (error) {
      console.error('Failed to load mappings:', error);
      message.error('加载映射程序失败');
      setRelatedPrograms([]);
    } finally {
      setModalLoading(false);
    }
  };

  const handleAddRelation = async (values: any) => {
    try {
      await api.post(
        `/program-mappings?parent_program_id=${currentProgram?.id}`,
        {
          child_program_ids: values.child_program_ids,
        },
      );
      message.success('映射成功');
      setAddRelationModalVisible(false);
      relationForm.resetFields();
      setMappingSearchKeyword('');
      setDebouncedMappingSearchKeyword('');
      setMappingFilterProductionLine(null);
      setMappingFilterVehicleModel(null);
      setMappingFilterStatus(null);
      if (currentProgram) {
        const response = await api.get(
          `/program-mappings/by-parent/${currentProgram.id}`,
        );
        setRelatedPrograms(response.data || []);
      }
      await loadData();
    } catch (error: any) {
      console.error('Failed to add mapping:', error);
      message.error(error.response?.data?.error || '映射失败');
    }
  };

  const handleDeleteRelation = async (mappingId: number) => {
    try {
      await api.delete(`/program-mappings/${mappingId}`);
      message.success('取消映射成功');
      if (currentProgram) {
        const response = await api.get(
          `/program-mappings/by-parent/${currentProgram.id}`,
        );
        setRelatedPrograms(response.data || []);
      }
      await loadData();
    } catch (error) {
      console.error('Failed to delete mapping:', error);
      message.error('取消映射失败');
    }
  };

  const [versionChangeLogSupported, setVersionChangeLogSupported] =
    useState(true);
  const [batchImportSupported, setBatchImportSupported] = useState(true);

  const [batchImportVisible, setBatchImportVisible] = useState(false);
  const [batchImportStep, setBatchImportStep] = useState(0);
  const [batchImportPreview, setBatchImportPreview] =
    useState<BatchImportPreview | null>(null);
  const [workstationMappings, setWorkstationMappings] = useState<
    WorkstationMapping[]
  >([]);
  const [batchImportStatus, setBatchImportStatus] =
    useState<BatchImportStatus | null>(null);
  const [batchImportPolling, setBatchImportPolling] = useState<ReturnType<
    typeof setInterval
  > | null>(null);
  const [uploadLoading, setUploadLoading] = useState(false);

  useEffect(() => {
    if (!fileModalVisible && !modalVisible) {
      return;
    }

    if (versions.length === 0) {
      setSelectedVersionKey(null);
      return;
    }

    if (
      selectedVersionKey &&
      versions.some((v) => getVersionSelectionKey(v) === selectedVersionKey)
    ) {
      return;
    }

    const preferredVersion = versions.find((v) => v.is_current) ?? versions[0];
    setSelectedVersionKey(getVersionSelectionKey(preferredVersion));
  }, [fileModalVisible, modalVisible, versions, selectedVersionKey]);

  useEffect(() => {
    const keyword = searchParams.get('keyword');
    const nextKeyword = keyword || '';
    setSearchInputValue(nextKeyword);
    setSearchKeyword(nextKeyword);
  }, [searchParams]);

  useEffect(() => {
    if (!modalVisible || !currentProgram) {
      return;
    }

    form.setFieldValue(
      'custom_field_values',
      normalizeCustomFieldValues(currentProgram.custom_field_values),
    );
  }, [currentProgram, form, modalVisible, customFields]);

  const handleAdd = () => {
    setCurrentProgram(null);
    setCustomFields([]);
    setIsEditingProperties(true);
    setPropertyDraftSnapshot(null);
    setIsEditingDescription(true);
    setDescriptionDraftSnapshot('');
    form.resetFields();
    form.setFieldValue('status', 'in_progress');
    setModalVisible(true);
  };

  // 小铅笔入口: 编辑程序弹窗，控制 `modalVisible`
  const handleEdit = async (record: Program) => {
    setCurrentProgram(record);
    setCustomFields([]);
    setVersions([]);
    setVersionsPage(1);
    setVersionsTotal(0);
    setSelectedVersionKey(null);
    setIsEditingProperties(false);
    setPropertyDraftSnapshot(null);
    setIsEditingDescription(false);
    setDescriptionDraftSnapshot('');
    form.setFieldsValue({
      ...record,
      custom_field_values: normalizeCustomFieldValues(
        record.custom_field_values,
      ),
    });
    setModalVisible(true);

    try {
      await loadVersions(record.id, 1);
    } catch (error) {
      console.error('Failed to load editor versions:', error);
      setVersions([]);
      setVersionsTotal(0);
      setSelectedVersionKey(null);
    }

    if (record.production_line_id) {
      await handleModalProductionLineChange(record.production_line_id, {
        preserveValues: true,
      });
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/programs/${id}`);
      message.success('删除成功');
      loadData();
    } catch (error: any) {
      console.error('Failed to delete:', error);
      const serverMsg = error?.response?.data?.error;
      message.error(
        serverMsg ? `删除失败: ${serverMsg}` : '删除失败，请稍后重试',
      );
    }
  };

  const handleSubmit = async (values: any) => {
    const payload = buildProgramMutationPayload(
      values,
      currentProgram?.description || '',
    );

    try {
      if (currentProgram) {
        await api.put(`/programs/${currentProgram.id}`, payload);
        message.success('??????');
        setIsEditingProperties(false);
        setPropertyDraftSnapshot(null);
        setIsEditingDescription(false);
        setDescriptionDraftSnapshot('');
      } else {
        await api.post('/programs', payload);
        message.success('??????');
      }
      setModalVisible(false);
      setCustomFields([]);
      loadData();
    } catch (error: any) {
      console.error('Failed to submit:', error);
      const serverMsg = error?.response?.data?.error;
      message.error(
        serverMsg ? `??????: ${serverMsg}` : '???????????????',
      );
    }
  };

  // 批量导入相关函数
  const handleBatchImport = () => {
    if (!batchImportSupported) {
      message.warning('当前环境未启用批量导入接口');
      return;
    }
    setBatchImportVisible(true);
    setBatchImportStep(0);
    setBatchImportPreview(null);
    setWorkstationMappings([]);
    setBatchImportStatus(null);
  };

  const handleBatchUpload = async (file: File) => {
    setUploadLoading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await api.post('/programs/batch-upload', formData, {
        headers: { 'Content-Type': undefined },
      });

      setBatchImportPreview(response.data);

      // 初始化映射
      const mappings = response.data.workstations.map(
        (ws: WorkstationInfo) => ({
          workstation_name: ws.name,
          production_line_id: null,
          vehicle_model_id: null,
        }),
      );
      setWorkstationMappings(mappings);

      setBatchImportStep(1);
      message.success('文件解析成功');
    } catch (error: any) {
      console.error('Failed to upload:', error);
      if (error?.response?.status === 404 || error?.response?.status === 405) {
        setBatchImportSupported(false);
        message.warning('批量导入接口未启用');
      } else {
        message.error(error.response?.data?.error || '上传失败');
      }
    } finally {
      setUploadLoading(false);
    }
  };

  const handleMappingChange = (
    workstationName: string,
    field: string,
    value: any,
  ) => {
    setWorkstationMappings((prev) =>
      prev.map((m) =>
        m.workstation_name === workstationName ? { ...m, [field]: value } : m,
      ),
    );
  };

  const handleConfirmBatchImport = async () => {
    // 验证所有映射
    const invalidMappings = workstationMappings.filter(
      (m) => !m.production_line_id,
    );
    if (invalidMappings.length > 0) {
      message.error('请为所有工位选择生产线');
      return;
    }

    setBatchImportStep(2);

    try {
      const response = await api.post('/programs/batch-import', {
        preview_id: batchImportPreview?.preview_id,
        mappings: workstationMappings,
      });

      message.success('批量导入已开始');
      startBatchImportPolling(response.data.task_id);
    } catch (error: any) {
      console.error('Failed to start batch import:', error);
      if (error?.response?.status === 404 || error?.response?.status === 405) {
        setBatchImportSupported(false);
        message.warning('批量导入接口未启用');
        setBatchImportStep(0);
      } else {
        message.error(error.response?.data?.error || '启动导入失败');
        setBatchImportStep(1);
      }
    }
  };

  const startBatchImportPolling = (taskId: number) => {
    if (batchImportPolling) clearInterval(batchImportPolling);

    const interval = setInterval(async () => {
      try {
        const response = await api.get(`/tasks/${taskId}/status`);
        setBatchImportStatus(response.data);

        if (
          response.data.status === 'completed' ||
          response.data.status === 'failed'
        ) {
          clearInterval(interval);
          setBatchImportPolling(null);
          if (response.data.status === 'completed') {
            message.success('批量导入完成');
            loadData();
          }
        }
      } catch (error: any) {
        console.error('Failed to poll import status:', error);
        if (
          error?.response?.status === 404 ||
          error?.response?.status === 405
        ) {
          clearInterval(interval);
          setBatchImportPolling(null);
          setBatchImportSupported(false);
          message.warning('任务状态接口未启用');
        }
      }
    }, 2000);

    setBatchImportPolling(interval);
  };

  const handleCloseBatchImport = () => {
    if (batchImportPolling) {
      clearInterval(batchImportPolling);
    }
    setBatchImportVisible(false);
    setBatchImportStep(0);
    setBatchImportPreview(null);
    setWorkstationMappings([]);
    setBatchImportStatus(null);
  };

  const handleUpload = (record: Program) => {
    setCurrentProgram(record);
    uploadForm.resetFields();
    uploadForm.setFieldValue('program_id', record.id);
    setUploadModalVisible(true);
  };

  const handleUploadSubmit = async (values: any) => {
    console.log('上传数据:', values);

    const formData = new FormData();

    // 支持多文件上传 - 检查后端期望的字段名
    if (values.file && values.file.length > 0) {
      values.file.forEach((fileObj: any) => {
        console.log('添加文件:', fileObj.originFileObj.name);
        formData.append('files', fileObj.originFileObj);
      });
    }

    formData.append('program_id', values.program_id);
    formData.append('version', values.version);
    formData.append('description', values.description || '');

    console.log('FormData内容:');
    for (const [key, value] of formData.entries()) {
      console.log(key, value);
    }

    try {
      const response = await api.post('/files/upload', formData, {
        headers: {
          'Content-Type': undefined,
        },
      });

      const { isNewVersion } = response.data;
      if (isNewVersion) {
        message.success('新版本文件上传成功');
      } else {
        message.success('文件重新上传成功');
      }

      setUploadModalVisible(false);
      loadData();

      if (currentProgram && (fileModalVisible || modalVisible)) {
        try {
          await loadVersions(currentProgram.id, versionsPage, versionsPageSize);
        } catch (e) {
          console.error('Failed to reload versions:', e);
        }
      }
    } catch (error: any) {
      console.error('Failed to upload:', error);
      if (error.response?.data?.error) {
        message.error(`上传失败: ${error.response.data.error}`);
      } else {
        message.error('上传失败，请稍后重试');
      }
    }
  };

  // 小眼睛入口: 版本文件查看弹窗，控制 `fileModalVisible`
  const handleViewFiles = async (record: Program) => {
    setCurrentProgram(record);
    setVersions([]);
    setVersionsPage(1);
    setVersionsTotal(0);
    setModalLoading(true);
    try {
      await loadVersions(record.id, 1);
      setFileModalVisible(true);
    } catch (error) {
      console.error('Failed to load files:', error);
      message.error('加载文件列表失败');
    } finally {
      setModalLoading(false);
    }
  };

  const downloadWithAuth = async (url: string, fallbackName: string) => {
    const response = await api.get(url, { responseType: 'blob' });
    const blob = new Blob([response.data]);

    const contentDisposition = response.headers['content-disposition'];
    let filename = fallbackName;
    if (contentDisposition) {
      const match = /filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i.exec(
        contentDisposition,
      );
      const encodedName = match?.[1] || match?.[2];
      if (encodedName) {
        filename = decodeURIComponent(encodedName);
      }
    }

    const urlObject = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = urlObject;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(urlObject);
  };

  const handleDownload = async (record: any) => {
    try {
      const response = await api.get(`/files/program/${record.id}`);
      const versions = response.data?.versions || [];
      if (versions.length === 0) {
        message.warning('该程序暂无上传的文件');
        return;
      }

      // 获取最新版本的所有文件
      const latestVersion =
        versions.find((version: any) => version.is_current) || versions[0];
      const files = latestVersion?.files || [];
      if (files.length > 0) {
        if (files.length === 1) {
          // 如果只有一个文件，直接下载
          const file = files[0];
          await downloadWithAuth(`/files/download/${file.id}`, file.file_name);
        } else {
          // 如果有多个文件，打包下载最新版本
          await downloadWithAuth(
            `/files/download/program/${record.id}/latest`,
            `${record.code || record.id}_${latestVersion.version}.zip`,
          );
          message.success('正在打包下载最新版本的所有文件...');
        }
      } else {
        message.warning('该程序暂无可用文件');
      }
    } catch (error) {
      console.error('Failed to download:', error);
      message.error('下载失败');
    }
  };

  const handleDeleteSingleFile = async (fileId: number) => {
    try {
      await api.delete(`/files/${fileId}`);
      message.success('文件删除成功');
      if (currentProgram) {
        await loadVersions(currentProgram.id, versionsPage, versionsPageSize);
      }
    } catch (error: any) {
      console.error('Failed to delete file:', error);
      message.error(error.response?.data?.error || '删除文件失败');
    }
  };

  const columns = [
    {
      title: '程序名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Program) => {
        const fieldSummaries = getProgramCustomFieldSummaries(record);

        return (
          <Space size={[8, 8]} wrap>
            <span
              style={{
                color: '#2D3335',
                fontSize: '15px',
                fontWeight: 800,
                fontFamily: 'Inter, sans-serif',
                letterSpacing: '-0.01em',
              }}
            >
              {text}
            </span>
            {record.mapping_info && (
              <Tag
                style={{
                  marginInlineEnd: 0,
                  padding: '0 6px',
                  fontSize: '10px',
                  lineHeight: '18px',
                  height: '20px',
                  borderRadius: '9999px',
                  color: '#005BC1',
                  borderColor: 'rgba(0, 91, 193, 0.2)',
                  background: 'rgba(0, 91, 193, 0.08)',
                }}
              >
                与 {record.mapping_info.parent_program_name} 关联
              </Tag>
            )}
            {fieldSummaries.map((field) => (
              <Tooltip
                key={field.field_id}
                title={`${field.field_name}: ${field.value}`}
              >
                <Tag
                  style={{
                    marginInlineEnd: 0,
                    padding: '0 6px',
                    fontSize: '10px',
                    lineHeight: '18px',
                    height: '20px',
                    borderRadius: '9999px',
                    color: '#5A6062',
                    borderColor: '#DEE3E6',
                    background: '#F8F9FA',
                  }}
                >
                  {field.value}
                </Tag>
              </Tooltip>
            ))}
          </Space>
        );
      },
    },
    {
      title: '程序编号',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => (
        <span
          style={{
            color: '#5A6062',
            fontSize: '14px',
            fontWeight: 500,
            fontFamily: 'Inter, sans-serif',
          }}
        >
          {text}
        </span>
      ),
    },
    {
      title: '生产线',
      dataIndex: ['production_line', 'name'],
      key: 'production_line',
    },
    {
      title: '车型',
      dataIndex: ['vehicle_model', 'name'],
      key: 'vehicle_model',
    },
    {
      title: '当前版本',
      dataIndex: 'version',
      key: 'version',
      render: (version: string) =>
        version ? <Tag color="blue">{version}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string, record: Program) => {
        const isCompleted = status === 'completed';
        return (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '6px',
              alignItems: 'flex-start',
            }}
          >
            <div
              style={{
                background: isCompleted
                  ? 'rgba(61, 137, 255, 0.20)'
                  : 'rgba(222, 204, 253, 0.40)',
                borderRadius: '9999px',
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px 10px',
                gap: '6px',
              }}
            >
              <div
                style={{
                  width: '6px',
                  height: '6px',
                  borderRadius: '50%',
                  background: isCompleted ? '#005BC1' : '#50426B',
                }}
              ></div>
              <span
                style={{
                  color: isCompleted ? '#005BC1' : '#50426B',
                  fontSize: '11px',
                  fontWeight: 700,
                }}
              >
                {isCompleted ? '已完成' : '进行中'}
              </span>
            </div>
            {!isCompleted && record.editing_user?.name && (
              <span
                style={{ color: '#5A6062', fontSize: '11px', fontWeight: 500 }}
              >
                {record.editing_by === user?.id
                  ? '我正在编辑'
                  : `${record.editing_user.name} 正在编辑`}
              </span>
            )}
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: Program) => (
        <Space size="small">
          {/* 小眼睛: 打开版本文件查看弹窗，不是编辑弹窗 */}
          <Tooltip title="查看版本">
            <Button
              type="text"
              icon={<EyeOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleViewFiles(record)}
              style={{
                width: '32px',
                height: '32px',
                borderRadius: '4px',
                background: '#F8F9FA',
              }}
            />
          </Tooltip>
          <Tooltip title="上传文件">
            <Button
              type="text"
              icon={<UploadOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleUpload(record)}
              style={{
                width: '32px',
                height: '32px',
                borderRadius: '4px',
                background: '#F8F9FA',
              }}
            />
          </Tooltip>
          {/* 小铅笔: 打开编辑程序弹窗，这是编辑页入口 */}
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleEdit(record)}
              style={{
                width: '32px',
                height: '32px',
                borderRadius: '4px',
                background: '#F8F9FA',
              }}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'download',
                  icon: <DownloadOutlined />,
                  label: '下载',
                  onClick: () => handleDownload(record),
                },
                {
                  key: 'relation',
                  icon: <LinkOutlined />,
                  label: '映射管理',
                  onClick: () => handleViewRelations(record),
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: '删除',
                  danger: true,
                  onClick: () => handleDelete(record.id),
                },
              ],
            }}
          >
            <Tooltip title="更多操作">
              <Button
                type="text"
                icon={<SettingOutlined style={{ color: '#5A6062' }} />}
                style={{
                  width: '32px',
                  height: '32px',
                  borderRadius: '4px',
                  background: '#F8F9FA',
                }}
              />
            </Tooltip>
          </Dropdown>
        </Space>
      ),
    },
  ];

  const fileModalVersions = [...versions].sort(
    (a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );

  const fileModalSelectedVersion =
    fileModalVersions.find(
      (version) => getVersionSelectionKey(version) === selectedVersionKey,
    ) ??
    fileModalVersions.find((version) => version.is_current) ??
    fileModalVersions[0] ??
    null;

  const fileModalVersionFiles = fileModalSelectedVersion?.files || [];
  const fileModalStatusLabel = fileModalSelectedVersion?.is_current
    ? '已部署'
    : '历史版本';
  const fileModalCustomFieldPills = currentProgram
    ? getProgramCustomFieldSummaries(currentProgram).map(
        (field) => `${field.field_name}: ${field.value}`,
      )
    : [];
  const fileModalAttributePills = [
    currentProgram?.production_line?.name
      ? `产线: ${currentProgram.production_line.name}`
      : null,
    currentProgram?.vehicle_model?.name
      ? `车型: ${currentProgram.vehicle_model.name}`
      : null,
    currentProgram?.status
      ? `状态: ${currentProgram.status === 'completed' ? '已完成' : '进行中'}`
      : null,
    ...fileModalCustomFieldPills,
  ].filter((item): item is string => Boolean(item));

  const sortedEditorVersions = [...versions].sort(
    (a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );

  const editorSelectedVersion =
    sortedEditorVersions.find(
      (version) => getVersionSelectionKey(version) === selectedVersionKey,
    ) ??
    sortedEditorVersions.find((version) => version.is_current) ??
    sortedEditorVersions[0] ??
    null;

  useEffect(() => {
    setVersionDescriptionValue(editorSelectedVersion?.change_log || '');
  }, [editorSelectedVersion?.id, editorSelectedVersion?.change_log]);

  const editorVersionFiles = editorSelectedVersion?.files || [];
  const editorDynamicFields = customFields.filter((field) => field.enabled);

  const handleEditorRetransfer = () => {
    if (editorSelectedVersion?.version && currentProgram) {
      uploadForm.resetFields();
      uploadForm.setFieldValue('program_id', currentProgram.id);
      uploadForm.setFieldValue('version', editorSelectedVersion.version);
      setUploadModalVisible(true);
      return;
    }

    form.submit();
  };

  const handleEditorDownloadAll = async () => {
    if (!currentProgram || !editorSelectedVersion?.version) {
      message.warning('当前版本暂无可下载文件');
      return;
    }

    try {
      await downloadWithAuth(
        `/files/download/version/${editorSelectedVersion.version}?program_id=${currentProgram.id}`,
        `${currentProgram.code || currentProgram.id}_${editorSelectedVersion.version}.zip`,
      );
    } catch (error) {
      console.error('Failed to download version files:', error);
      message.error('下载失败');
    }
  };

  const handleFileModalDownloadAll = async () => {
    if (!currentProgram || !fileModalSelectedVersion?.version) {
      message.warning('该版本暂无可下载文件');
      return;
    }

    try {
      await downloadWithAuth(
        `/files/download/version/${fileModalSelectedVersion.version}?program_id=${currentProgram.id}`,
        `${currentProgram.code || currentProgram.id}_${fileModalSelectedVersion.version}.zip`,
      );
    } catch (error) {
      console.error('Failed to download file modal version files:', error);
      message.error('下载失败');
    }
  };

  const handleStartPropertyEdit = () => {
    setPropertyDraftSnapshot(
      buildPropertyDraftSnapshot(form.getFieldsValue(true)),
    );
    setIsEditingProperties(true);
    setTimeout(() => {
      try {
        form.scrollToField('name');
      } catch (error) {
        void error;
      }
      const input = document.querySelector(
        'input#name',
      ) as HTMLInputElement | null;
      input?.focus();
    }, 0);
  };

  const handleCancelPropertyEdit = () => {
    if (propertyDraftSnapshot) {
      form.setFieldsValue(buildPropertyDraftSnapshot(propertyDraftSnapshot));
    }
    setIsEditingProperties(false);
    setPropertyDraftSnapshot(null);
  };

  const handleSavePropertyEdit = async () => {
    try {
      const values = await form.validateFields();
      await handleSubmit(values);
    } catch (error) {
      void error;
    }
  };

  const handleStartDescriptionEdit = () => {
    setDescriptionDraftSnapshot(versionDescriptionValue || '');
    setIsEditingDescription(true);
    setTimeout(() => {
      const textArea = document.querySelector(
        'textarea#version-description',
      ) as HTMLTextAreaElement | null;
      textArea?.focus();
    }, 0);
  };

  const handleCancelDescriptionEdit = () => {
    setVersionDescriptionValue(descriptionDraftSnapshot);
    setIsEditingDescription(false);
    setDescriptionDraftSnapshot('');
  };

  const handleSaveDescriptionEdit = async () => {
    try {
      if (!editorSelectedVersion?.id) {
        message.warning('当前版本暂不支持编辑说明');
        return;
      }

      await api.put(`/versions/${editorSelectedVersion.id}`, {
        change_log: versionDescriptionValue || '',
      });

      if (currentProgram) {
        await loadVersions(currentProgram.id, versionsPage, versionsPageSize);
      }

      setIsEditingDescription(false);
      setDescriptionDraftSnapshot('');
      message.success('版本说明更新成功');
    } catch (error: any) {
      if (error?.response?.status === 404 || error?.response?.status === 405) {
        setVersionChangeLogSupported(false);
        message.warning('版本说明更新接口未启用');
      } else {
        message.error('版本说明更新失败');
      }
    }
  };

  return (
    <div className="management-page">
      {/* 顶部标题区 */}
      <ProgramHeader
        batchImportSupported={batchImportSupported}
        onExportExcel={handleExportExcel}
        onBatchImport={handleBatchImport}
        onAddProgram={handleAdd}
      />

      {/* 筛选区域 */}
      <ProgramFilterPanel
        searchKeyword={searchInputValue}
        filterProductionLine={filterProductionLine}
        filterVehicleModel={filterVehicleModel}
        filterStatus={filterStatus}
        productionLines={productionLines}
        vehicleModels={vehicleModels}
        customFieldFilters={customFieldFilters}
        customFieldFilterValues={customFieldFilterValues}
        onSearchKeywordChange={(value) => {
          setProgramPage(1);
          setSearchInputValue(value);
        }}
        onApplySearch={applySearchKeyword}
        onFilterProductionLineChange={handleFilterProductionLineChange}
        onFilterVehicleModelChange={(value) => {
          setProgramPage(1);
          setFilterVehicleModel(value ?? null);
        }}
        onFilterStatusChange={(value) => {
          setProgramPage(1);
          setFilterStatus(value ?? null);
        }}
        onDateRangeChange={(range) => {
          setProgramPage(1);
          setFilterDateRange(range);
        }}
        onReset={handleResetFilter}
        onCustomFieldFilterChange={handleCustomFieldFilterChange}
      />

      {/* 数据表格 */}
      <div className="management-table-card">
        <Table
          className="custom-table"
          columns={columns}
          dataSource={filteredPrograms}
          rowKey="id"
          loading={tableLoading}
          onChange={(pagination: TablePaginationConfig) => {
            const nextPage = pagination.current || 1;
            const nextPageSize = pagination.pageSize || 20;
            if (nextPage !== programPage) {
              setProgramPage(nextPage);
            }
            if (nextPageSize !== programPageSize) {
              setProgramPageSize(nextPageSize);
              setProgramPage(1);
            }
          }}
          pagination={{
            current: programPage,
            pageSize: programPageSize,
            total: programTotal,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total, range) =>
              `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: {
              padding: '16px 24px',
              margin: 0,
              background: 'rgba(241, 244, 245, 0.50)',
            },
          }}
          locale={{
            emptyText: (
              <div style={{ padding: '40px 0' }}>
                <FileOutlined
                  style={{
                    fontSize: '48px',
                    color: '#d9d9d9',
                    marginBottom: '16px',
                  }}
                />
                <div style={{ color: '#999', marginBottom: '16px' }}>
                  暂无程序数据
                </div>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={handleAdd}
                >
                  创建第一个程序
                </Button>
              </div>
            ),
          }}
        />
      </div>
      {/* 小铅笔对应的编辑程序弹窗 */}
      <Modal
        className="program-editor-modal"
        title={null}
        closable={false}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setCustomFields([]);
          setIsEditingProperties(false);
          setPropertyDraftSnapshot(null);
          setIsEditingDescription(false);
          setDescriptionDraftSnapshot('');
        }}
        footer={null}
        width={1152}
        styles={{ body: { padding: 0 } }}
        centered
      >
        <div
          data-testid="program-management-overlay"
          className="program-editor-shell compact"
        >
          <div className="program-editor-header">
            <div className="program-editor-header-title">
              <AppstoreOutlined
                style={{ fontSize: '20px', color: '#2D3335' }}
              />
              <span>
                {currentProgram?.name
                  ? `${currentProgram.name} - 版本文件管理`
                  : 'ssb2 - 版本文件管理'}
              </span>
            </div>
            <div className="program-editor-header-actions">
              <div
                className="program-editor-close-button"
                onClick={() => {
                  setModalVisible(false);
                  setCustomFields([]);
                  setIsEditingProperties(false);
                  setPropertyDraftSnapshot(null);
                  setIsEditingDescription(false);
                  setDescriptionDraftSnapshot('');
                }}
              >
                <CloseOutlined style={{ color: '#5A6062', fontSize: '14px' }} />
              </div>
            </div>
          </div>

          <Form
            form={form}
            layout="vertical"
            onFinish={handleSubmit}
            requiredMark={false}
            className="program-editor-form"
          >
            <div className="program-editor-property-strip">
              <div className="program-editor-property-strip-content">
                <div className="program-editor-section-header">
                  <div className="program-editor-section-title">属性</div>
                  <div className="program-editor-section-actions">
                    {isEditingProperties ? (
                      <>
                        <Button
                          type="text"
                          className="program-editor-header-button"
                          onClick={handleCancelPropertyEdit}
                        >
                          取消
                        </Button>
                        <Button
                          type="text"
                          className="program-editor-header-button"
                          icon={
                            <EditOutlined
                              style={{ color: '#5A6062', fontSize: '14px' }}
                            />
                          }
                          onClick={() => void handleSavePropertyEdit()}
                        >
                          保存
                        </Button>
                      </>
                    ) : (
                      <Button
                        type="text"
                        className="program-editor-header-button"
                        icon={
                          <EditOutlined
                            style={{ color: '#5A6062', fontSize: '14px' }}
                          />
                        }
                        onClick={handleStartPropertyEdit}
                      >
                        编辑
                      </Button>
                    )}
                  </div>
                </div>
                <div className="program-editor-divider program-editor-divider-tight">
                  <div className="program-editor-property-grid">
                    <div>
                      <div className="program-editor-field-label">程序名称</div>
                      <Form.Item
                        name="name"
                        rules={[{ required: true, message: '请输入程序名称' }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Input
                          disabled={!isEditingProperties}
                          size="large"
                          className="program-editor-input"
                          placeholder="请输入程序名称"
                        />
                      </Form.Item>
                    </div>
                    <div>
                      <div className="program-editor-field-label">程序编号</div>
                      <Form.Item
                        name="code"
                        rules={[{ required: true, message: '请输入程序编号' }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Input
                          disabled={!isEditingProperties}
                          size="large"
                          className="program-editor-input"
                          placeholder="请输入程序编号"
                        />
                      </Form.Item>
                    </div>
                    <div>
                      <div className="program-editor-field-label">状态</div>
                      <Form.Item
                        name="status"
                        initialValue="in_progress"
                        style={{ marginBottom: 0 }}
                      >
                        <Select
                          disabled={!isEditingProperties}
                          size="large"
                          className="program-editor-input"
                        >
                          <Select.Option value="in_progress">
                            进行中
                          </Select.Option>
                          <Select.Option value="completed">
                            已完成
                          </Select.Option>
                        </Select>
                      </Form.Item>
                    </div>
                  </div>

                  <div className="program-editor-property-grid">
                    <div>
                      <div className="program-editor-field-label">产线</div>
                      <Form.Item
                        name="production_line_id"
                        rules={[{ required: true, message: '请选择生产线' }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Select
                          disabled={!isEditingProperties}
                          size="large"
                          className="program-editor-input"
                          onChange={(value) =>
                            void handleModalProductionLineChange(value)
                          }
                          placeholder="请选择生产线"
                        >
                          {productionLines.map((line: any) => (
                            <Select.Option key={line.id} value={line.id}>
                              {line.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </div>
                    <div>
                      <div className="program-editor-field-label">车型</div>
                      <Form.Item
                        name="vehicle_model_id"
                        style={{ marginBottom: 0 }}
                      >
                        <Select
                          disabled={!isEditingProperties}
                          size="large"
                          className="program-editor-input"
                          allowClear
                          placeholder="请选择车型"
                        >
                          {vehicleModels.map((model: any) => (
                            <Select.Option key={model.id} value={model.id}>
                              {model.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </div>
                    {editorDynamicFields.map((field) => {
                      const options = parseCustomFieldOptions(
                        field.options_json,
                      );

                      return (
                        <div key={field.id}>
                          <div className="program-editor-field-label">
                            {field.name}
                          </div>
                          <Form.Item
                            name={['custom_field_values', String(field.id)]}
                            style={{ marginBottom: 0 }}
                          >
                            {field.field_type === 'select' ? (
                              <Select
                                disabled={!isEditingProperties}
                                size="large"
                                allowClear
                                className="program-editor-input"
                                placeholder={`请选择${field.name}`}
                              >
                                {options.map((option) => (
                                  <Select.Option key={option} value={option}>
                                    {option}
                                  </Select.Option>
                                ))}
                              </Select>
                            ) : (
                              <Input
                                disabled={!isEditingProperties}
                                size="large"
                                className="program-editor-input"
                                placeholder={`请输入${field.name}`}
                              />
                            )}
                          </Form.Item>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            </div>

            <div className="program-editor-body">
              <div className="program-editor-sidebar">
                <div className="program-editor-sidebar-title">历史版本</div>
                <div className="program-editor-sidebar-list">
                  {(sortedEditorVersions.length > 0
                    ? sortedEditorVersions
                    : [
                        {
                          id: -1,
                          version: currentProgram?.version || '26.2.8',
                          is_current: true,
                          created_at:
                            currentProgram?.created_at ||
                            new Date().toISOString(),
                        },
                      ]
                  ).map((version, _index, versionList) => {
                    const hasSingleVersion = versionList.length === 1;
                    const isFallbackVersion = version.id === -1;
                    const versionKey = getVersionSelectionKey(version);
                    const selectedKey = getVersionSelectionKey(
                      editorSelectedVersion,
                    );
                    const isActive =
                      hasSingleVersion ||
                      isFallbackVersion ||
                      (versionKey !== null && versionKey === selectedKey);
                    return (
                      <div
                        key={version.id}
                        onClick={() =>
                          setSelectedVersionKey(getVersionSelectionKey(version))
                        }
                        className={`program-editor-version-button${isActive ? ' active' : ''}${version.id > 0 ? ' interactive' : ''}`}
                      >
                        <div className="program-editor-version-button-header">
                          <span className="program-editor-version-name">
                            {version.version}
                          </span>
                          {isActive && (
                            <span className="program-editor-version-badge">
                              当前
                            </span>
                          )}
                        </div>
                        <div className="program-editor-version-meta">
                          <span className="program-editor-version-label">
                            {version.id > 0
                              ? new Date(version.created_at).toLocaleString(
                                  'zh-CN',
                                  { hour12: false },
                                )
                              : '当前编辑版本'}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
                {currentProgram && versionsTotal > versionsPageSize && (
                  <div style={{ marginTop: '8px' }}>
                    <Pagination
                      size="small"
                      current={versionsPage}
                      pageSize={versionsPageSize}
                      total={versionsTotal}
                      showSizeChanger
                      pageSizeOptions={[10, 20, 50]}
                      onChange={(page, pageSize) => {
                        if (!currentProgram) {
                          return;
                        }
                        void loadVersions(currentProgram.id, page, pageSize);
                      }}
                    />
                  </div>
                )}
              </div>

              <div className="program-editor-main">
                <div className="program-editor-card program-editor-hero">
                  <div className="program-editor-hero-top">
                    <div>
                      <div className="program-editor-kicker">当前查看版本</div>
                      <div className="program-editor-version-summary">
                        <span className="program-editor-version-display">
                          {editorSelectedVersion?.version ||
                            currentProgram?.version ||
                            '26.2.8'}
                        </span>
                        <div className="program-editor-status-chip">
                          {(editorSelectedVersion?.is_current ?? true)
                            ? 'STABLE'
                            : 'ARCHIVE'}
                        </div>
                      </div>
                    </div>

                    <div className="program-editor-hero-actions">
                      <Button
                        onClick={handleEditorRetransfer}
                        className="program-editor-neutral-button"
                      >
                        重传此版本
                      </Button>
                      <Button
                        type="primary"
                        onClick={() => void handleEditorDownloadAll()}
                        className="program-editor-primary-button"
                      >
                        全部下载
                      </Button>
                    </div>
                  </div>

                  <div className="program-editor-divider">
                    <div className="program-editor-meta-grid">
                      <div>
                        <div className="program-editor-meta-label">
                          <UserOutlined
                            style={{ color: '#5A6062', fontSize: '12px' }}
                          />
                          <span>上传者</span>
                        </div>
                        <div className="program-editor-meta-content">
                          {editorSelectedVersion?.uploader?.name ||
                            currentProgram?.editing_user?.name ||
                            user?.name ||
                            '系统管理员'}
                        </div>
                      </div>
                      <div>
                        <div className="program-editor-meta-label">
                          <ClockCircleOutlined
                            style={{ color: '#5A6062', fontSize: '12px' }}
                          />
                          <span>创建时间</span>
                        </div>
                        <div className="program-editor-meta-content">
                          {editorSelectedVersion?.created_at ||
                          currentProgram?.created_at
                            ? new Date(
                                editorSelectedVersion?.created_at ||
                                  currentProgram?.created_at ||
                                  '',
                              ).toLocaleString('zh-CN', { hour12: false })
                            : '创建后自动生成'}
                        </div>
                      </div>
                      <div>
                        <div className="program-editor-meta-label program-editor-meta-label-with-action">
                          <div className="program-editor-meta-label-main">
                            <FileTextOutlined
                              style={{ color: '#5A6062', fontSize: '12px' }}
                            />
                            <span>版本说明</span>
                          </div>
                          <div className="program-editor-meta-label-actions">
                            {versionChangeLogSupported ? (
                              isEditingDescription ? (
                                <>
                                  <Button
                                    type="text"
                                    className="program-editor-header-button"
                                    onClick={handleCancelDescriptionEdit}
                                  >
                                    取消
                                  </Button>
                                  <Button
                                    type="text"
                                    className="program-editor-header-button"
                                    icon={
                                      <EditOutlined
                                        style={{
                                          color: '#5A6062',
                                          fontSize: '14px',
                                        }}
                                      />
                                    }
                                    onClick={() =>
                                      void handleSaveDescriptionEdit()
                                    }
                                  >
                                    保存
                                  </Button>
                                </>
                              ) : (
                                <Button
                                  type="text"
                                  className="program-editor-header-button"
                                  icon={
                                    <EditOutlined
                                      style={{
                                        color: '#5A6062',
                                        fontSize: '14px',
                                      }}
                                    />
                                  }
                                  onClick={handleStartDescriptionEdit}
                                >
                                  编辑
                                </Button>
                              )
                            ) : (
                              <Text type="secondary">接口未启用</Text>
                            )}
                          </div>
                        </div>
                        <Input.TextArea
                          id="version-description"
                          value={versionDescriptionValue}
                          onChange={(event) =>
                            setVersionDescriptionValue(event.target.value)
                          }
                          disabled={
                            !isEditingDescription || !versionChangeLogSupported
                          }
                          rows={3}
                          className="program-editor-input"
                          placeholder="请输入当前版本说明"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <div className="program-editor-assets">
                  <div className="program-editor-assets-header">
                    <div>
                      <div className="program-editor-file-head">文件资产</div>
                      <div className="program-editor-file-subhead">
                        {editorSelectedVersion?.version
                          ? `版本 ${editorSelectedVersion.version}`
                          : '当前编辑草稿'}
                      </div>
                    </div>
                    <div className="program-editor-assets-count">
                      共 {editorVersionFiles.length} 个文件
                    </div>
                  </div>

                  <div className="program-editor-file-table">
                    <div className="program-editor-file-table-head">
                      <span>文件</span>
                      <span>文件信息</span>
                    </div>
                    <div className="program-editor-file-table-body">
                      {editorVersionFiles.length > 0 ? (
                        editorVersionFiles.map((file) => (
                          <div
                            key={`version-file-${file.id}`}
                            className="program-editor-file-row"
                          >
                            <div className="program-editor-file-main">
                              <div className="program-editor-file-icon">
                                <FileOutlined
                                  style={{ color: '#005BC1', fontSize: '16px' }}
                                />
                              </div>
                              <div>
                                <div className="program-editor-file-title">
                                  {file.file_name}
                                </div>
                                <div className="program-editor-file-caption">
                                  {file.file_exists === false
                                    ? '文件缺失'
                                    : '已上传文件'}
                                </div>
                              </div>
                            </div>
                            <div className="program-editor-file-side">
                              <div className="program-editor-file-info">
                                <div>{formatFileSize(file.file_size)}</div>
                                <div>
                                  {new Date(file.created_at).toLocaleString(
                                    'zh-CN',
                                    { hour12: false },
                                  )}
                                </div>
                              </div>
                              <div className="program-editor-file-actions">
                                <Tooltip
                                  title={
                                    file.file_exists === false
                                      ? '文件已缺失，无法下载'
                                      : '下载文件'
                                  }
                                >
                                  <div
                                    className={`program-editor-file-action download${file.file_exists === false ? ' disabled' : ''}`}
                                    onClick={() => {
                                      if (file.file_exists === false) {
                                        message.warning(
                                          '该文件已被物理删除，请联系管理员清理记录',
                                        );
                                        return;
                                      }
                                      void downloadWithAuth(
                                        `/files/download/${file.id}`,
                                        file.file_name,
                                      );
                                    }}
                                  >
                                    <DownloadOutlined
                                      style={{
                                        color: '#5A6062',
                                        fontSize: '16px',
                                      }}
                                    />
                                  </div>
                                </Tooltip>
                                <Popconfirm
                                  title="确定删除这个文件吗？"
                                  onConfirm={() =>
                                    handleDeleteSingleFile(file.id)
                                  }
                                >
                                  <Tooltip title="删除文件">
                                    <div className="program-editor-file-action delete">
                                      <DeleteOutlined
                                        style={{
                                          color: '#A83836',
                                          fontSize: '16px',
                                        }}
                                      />
                                    </div>
                                  </Tooltip>
                                </Popconfirm>
                              </div>
                            </div>
                          </div>
                        ))
                      ) : (
                        <div className="program-editor-file-empty">
                          当前版本暂无文件。
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </Form>
        </div>
      </Modal>

      <Modal
        title="上传程序文件"
        open={uploadModalVisible}
        onCancel={() => setUploadModalVisible(false)}
        onOk={() => uploadForm.submit()}
      >
        <Form form={uploadForm} layout="vertical" onFinish={handleUploadSubmit}>
          <Form.Item name="program_id" hidden>
            <Input />
          </Form.Item>
          <Form.Item
            name="version"
            label="版本号"
            rules={[{ required: true }]}
            extra="如需重新上传当前版本，请输入相同版本号"
          >
            <Input placeholder="例如: v1.0.0" />
          </Form.Item>
          <Form.Item
            name="file"
            label="选择文件"
            valuePropName="fileList"
            getValueFromEvent={(e) => (Array.isArray(e) ? e : e?.fileList)}
            rules={[{ required: true, message: '请选择文件' }]}
          >
            <Upload beforeUpload={() => false} multiple>
              <Button icon={<UploadOutlined />}>选择文件（可多选）</Button>
            </Upload>
          </Form.Item>
          <Form.Item name="description" label="变更说明">
            <TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 小眼睛对应的版本文件查看弹窗 */}
      <Modal
        title={null}
        closable={false}
        open={fileModalVisible}
        onCancel={() => setFileModalVisible(false)}
        footer={null}
        width={1280}
        destroyOnHidden
        centered
        className="program-view-modal"
        styles={{
          body: { padding: 0 },
          content: { padding: 0, background: 'transparent', boxShadow: 'none' },
        }}
      >
        <div className="program-view-shell" data-testid="program-view-overlay">
          <div className="program-view-topbar">
            <div className="program-view-topbar-left">
              <div className="program-view-title">
                {currentProgram?.name || '程序详情'} - 程序详情
              </div>
              <div className="program-view-pill-list">
                {fileModalAttributePills.map((pill) => (
                  <div key={pill} className="program-view-pill">
                    {pill}
                  </div>
                ))}
              </div>
            </div>
            <div className="program-view-topbar-actions">
              <Button
                type="text"
                className="program-view-icon-button"
                icon={
                  <LeftOutlined
                    style={{ color: '#005BC1', fontSize: '16px' }}
                  />
                }
                onClick={() => setFileModalVisible(false)}
              />
              <Button
                aria-label="下载全部"
                className="program-view-secondary-button"
                icon={<DownloadOutlined style={{ fontSize: '14px' }} />}
                onClick={() => void handleFileModalDownloadAll()}
              >
                下载全部
              </Button>
              <Button
                aria-label="进入编辑"
                type="primary"
                className="program-view-primary-button"
                onClick={() => {
                  setFileModalVisible(false);
                  if (currentProgram) {
                    void handleEdit(currentProgram);
                  }
                }}
              >
                进入编辑
              </Button>
            </div>
          </div>

          <div className="program-view-body">
            <aside className="program-view-sidebar">
              <div className="program-view-sidebar-header">
                <div className="program-view-sidebar-title">版本记录</div>
                <div className="program-view-sidebar-subtitle">只读模式</div>
              </div>

              {modalLoading ? (
                <div className="program-view-sidebar-empty">加载中...</div>
              ) : fileModalVersions.length === 0 ? (
                <div className="program-view-sidebar-empty">暂无版本信息</div>
              ) : (
                <>
                  <div className="program-view-version-list">
                    {fileModalVersions.map((version) => {
                      const versionKey = getVersionSelectionKey(version);
                      const selectedKey = getVersionSelectionKey(
                        fileModalSelectedVersion,
                      );
                      const isActive =
                        versionKey !== null && versionKey === selectedKey;
                      return (
                        <button
                          key={version.id}
                          type="button"
                          onClick={() =>
                            setSelectedVersionKey(
                              getVersionSelectionKey(version),
                            )
                          }
                          className={`program-view-version-item${isActive ? ' active' : ''}`}
                        >
                          <div className="program-view-version-dot">
                            <ClockCircleFilled />
                          </div>
                          <div className="program-view-version-content">
                            <div className="program-view-version-row">
                              <span className="program-view-version-name">
                                {version.version}
                              </span>
                              {isActive && (
                                <span className="program-view-version-current">
                                  当前
                                </span>
                              )}
                            </div>
                            <div className="program-view-version-date">
                              {new Date(version.created_at).toLocaleString(
                                'zh-CN',
                                { hour12: false },
                              )}
                            </div>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                  {currentProgram && versionsTotal > versionsPageSize && (
                    <div style={{ marginTop: '8px' }}>
                      <Pagination
                        size="small"
                        current={versionsPage}
                        pageSize={versionsPageSize}
                        total={versionsTotal}
                        showSizeChanger
                        pageSizeOptions={[10, 20, 50]}
                        onChange={(page, pageSize) => {
                          if (!currentProgram) {
                            return;
                          }
                          void loadVersions(currentProgram.id, page, pageSize);
                        }}
                      />
                    </div>
                  )}
                </>
              )}
            </aside>

            <main className="program-view-main">
              {modalLoading ? (
                <div className="program-view-empty-state">加载中...</div>
              ) : !fileModalSelectedVersion ? (
                <div className="program-view-empty-state">
                  请选择左侧版本查看详情
                </div>
              ) : (
                <div className="program-view-content">
                  <section className="program-view-card program-view-overview-card">
                    <div className="program-view-overview-header">
                      <div className="program-view-overview-version">
                        {fileModalSelectedVersion.version}
                      </div>
                      <div className="program-view-overview-heading">
                        当前版本概览
                      </div>
                    </div>
                    <div className="program-view-overview-grid">
                      <div className="program-view-meta-block">
                        <div className="program-view-meta-label">上传者</div>
                        <div className="program-view-meta-value">
                          {fileModalSelectedVersion.uploader?.name ||
                            '系统管理员'}
                        </div>
                      </div>
                      <div className="program-view-meta-block">
                        <div className="program-view-meta-label">创建时间</div>
                        <div className="program-view-meta-value">
                          {new Date(
                            fileModalSelectedVersion.created_at,
                          ).toLocaleString('zh-CN', { hour12: false })}
                        </div>
                      </div>
                      <div className="program-view-meta-block program-view-meta-block-wide">
                        <div className="program-view-meta-label">运行状态</div>
                        <div className="program-view-status-row">
                          <span className="program-view-status-dot" />
                          <span className="program-view-meta-value">
                            {fileModalStatusLabel}
                          </span>
                        </div>
                      </div>
                    </div>
                  </section>

                  <section className="program-view-card">
                    <div className="program-view-card-title">版本说明</div>
                    <div className="program-view-description">
                      {fileModalSelectedVersion.change_log || '暂无说明'}
                    </div>
                  </section>

                  <section className="program-view-card">
                    <div className="program-view-assets-header">
                      <div className="program-view-card-title">文件资产</div>
                      <div className="program-view-assets-count">
                        共 {fileModalVersionFiles.length} 个文件
                      </div>
                    </div>
                    <div className="program-view-file-list">
                      {fileModalVersionFiles.length > 0 ? (
                        fileModalVersionFiles.map((file) => {
                          const isTextFile =
                            file.file_name.toLowerCase().endsWith('.txt') ||
                            file.file_name.toLowerCase().endsWith('.log');
                          return (
                            <div
                              key={file.id}
                              className="program-view-file-row"
                            >
                              <div className="program-view-file-main">
                                <div className="program-view-file-icon">
                                  {isTextFile ? (
                                    <FileTextOutlined
                                      style={{
                                        color: '#5A6062',
                                        fontSize: '18px',
                                      }}
                                    />
                                  ) : (
                                    <FileOutlined
                                      style={{
                                        color: '#5A6062',
                                        fontSize: '18px',
                                      }}
                                    />
                                  )}
                                </div>
                                <div className="program-view-file-texts">
                                  <div className="program-view-file-name">
                                    {file.file_name}
                                  </div>
                                  <div className="program-view-file-meta">
                                    {formatFileSize(file.file_size)} ·{' '}
                                    {getFileTypeLabel(file.file_name)}
                                  </div>
                                </div>
                              </div>
                              <Button
                                type="text"
                                className="program-view-file-action"
                                icon={
                                  <DownloadOutlined
                                    style={{
                                      color: '#5A6062',
                                      fontSize: '14px',
                                    }}
                                  />
                                }
                                disabled={file.file_exists === false}
                                onClick={() => {
                                  if (file.file_exists === false) {
                                    message.warning(
                                      '该文件已被物理删除，请联系管理员清理记录',
                                    );
                                    return;
                                  }
                                  void downloadWithAuth(
                                    `/files/download/${file.id}`,
                                    file.file_name,
                                  );
                                }}
                              />
                            </div>
                          );
                        })
                      ) : (
                        <div className="program-view-empty-inline">
                          当前版本暂无文件
                        </div>
                      )}
                    </div>
                  </section>
                </div>
              )}
            </main>
          </div>
        </div>
      </Modal>

      {/* 批量导入模态框 */}
      <Modal
        title="批量导入程序"
        open={batchImportVisible}
        onCancel={handleCloseBatchImport}
        width={800}
        footer={null}
      >
        <Steps current={batchImportStep} style={{ marginBottom: 24 }}>
          <Step title="上传文件" description="上传zip压缩包" />
          <Step title="配置映射" description="工位映射到生产线" />
          <Step title="导入进度" description="查看导入结果" />
        </Steps>

        {batchImportStep === 0 && (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Alert
              message="文件结构要求"
              description={
                <div style={{ textAlign: 'left' }}>
                  <p>请将程序文件按以下结构组织并压缩为zip文件：</p>
                  <pre
                    style={{
                      background: '#f5f5f5',
                      padding: '12px',
                      borderRadius: '4px',
                    }}
                  >
                    {`工位名称/
  程序名称/
    文件1.nc
    文件2.nc
  另一个程序/
    文件1.nc
另一个工位/
  程序名称/
    文件1.nc`}
                  </pre>
                </div>
              }
              type="info"
              showIcon
              style={{ marginBottom: 24, textAlign: 'left' }}
            />
            <Upload.Dragger
              accept=".zip"
              showUploadList={false}
              beforeUpload={(file) => {
                handleBatchUpload(file);
                return false;
              }}
              disabled={uploadLoading}
            >
              <p className="ant-upload-drag-icon">
                {uploadLoading ? <LoadingOutlined /> : <FolderAddOutlined />}
              </p>
              <p className="ant-upload-text">点击或拖拽zip文件到此区域</p>
              <p className="ant-upload-hint">
                支持批量导入，文件夹结构会被自动解析
              </p>
            </Upload.Dragger>
          </div>
        )}

        {batchImportStep === 1 && batchImportPreview && (
          <div>
            <Alert
              message={`解析完成：共 ${batchImportPreview.total_programs} 个程序，${batchImportPreview.total_files} 个文件`}
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <div style={{ maxHeight: 400, overflow: 'auto' }}>
              {batchImportPreview.workstations.map((ws, wsIndex) => (
                <Card
                  key={wsIndex}
                  title={
                    <Space>
                      <Tag color="blue">工位: {ws.name}</Tag>
                      <span>({ws.programs.length} 个程序)</span>
                    </Space>
                  }
                  style={{ marginBottom: 16 }}
                  size="small"
                >
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <div style={{ display: 'flex', gap: 16 }}>
                      <div style={{ flex: 1 }}>
                        <Text strong>选择生产线 *</Text>
                        <Select
                          style={{ width: '100%', marginTop: 8 }}
                          placeholder="请选择生产线"
                          value={
                            workstationMappings[wsIndex]?.production_line_id
                          }
                          onChange={(value) =>
                            handleMappingChange(
                              ws.name,
                              'production_line_id',
                              value,
                            )
                          }
                        >
                          {productionLines.map((line) => (
                            <Select.Option key={line.id} value={line.id}>
                              {line.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </div>
                      <div style={{ flex: 1 }}>
                        <Text strong>选择车型</Text>
                        <Select
                          style={{ width: '100%', marginTop: 8 }}
                          placeholder="请选择车型（可选）"
                          allowClear
                          value={workstationMappings[wsIndex]?.vehicle_model_id}
                          onChange={(value) =>
                            handleMappingChange(
                              ws.name,
                              'vehicle_model_id',
                              value,
                            )
                          }
                        >
                          {vehicleModels.map((model) => (
                            <Select.Option key={model.id} value={model.id}>
                              {model.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </div>
                    </div>
                    <div>
                      <Text type="secondary">包含程序：</Text>
                      <div style={{ marginTop: 8 }}>
                        {ws.programs.map((prog, progIndex) => (
                          <Tag key={progIndex}>
                            {prog.name} ({prog.files.length}个文件)
                          </Tag>
                        ))}
                      </div>
                    </div>
                  </Space>
                </Card>
              ))}
            </div>

            <Divider />

            <div style={{ textAlign: 'right' }}>
              <Space>
                <Button onClick={() => setBatchImportStep(0)}>上一步</Button>
                <Button type="primary" onClick={handleConfirmBatchImport}>
                  开始导入
                </Button>
              </Space>
            </div>
          </div>
        )}

        {batchImportStep === 2 && batchImportStatus && (
          <div style={{ textAlign: 'center', padding: '20px 0' }}>
            {batchImportStatus.status === 'processing' && (
              <>
                <Progress
                  type="circle"
                  percent={Math.round(batchImportStatus.progress)}
                  status="active"
                />
                <div style={{ marginTop: 16 }}>
                  <Text>正在导入: {batchImportStatus.current_item}</Text>
                </div>
                <div style={{ marginTop: 8 }}>
                  <Text type="secondary">
                    成功: {batchImportStatus.success} /{' '}
                    {batchImportStatus.total}
                  </Text>
                </div>
              </>
            )}

            {batchImportStatus.status === 'completed' && (
              <>
                <CheckCircleOutlined
                  style={{ fontSize: 64, color: '#52c41a' }}
                />
                <Title level={4} style={{ marginTop: 16 }}>
                  导入完成
                </Title>
                <div style={{ marginTop: 8 }}>
                  <Space size="large">
                    <Statistic title="总数" value={batchImportStatus.total} />
                    <Statistic
                      title="成功"
                      value={batchImportStatus.success}
                      valueStyle={{ color: '#52c41a' }}
                    />
                    <Statistic
                      title="失败"
                      value={batchImportStatus.failed}
                      valueStyle={{ color: '#ff4d4f' }}
                    />
                  </Space>
                </div>
                <Button
                  type="primary"
                  style={{ marginTop: 24 }}
                  onClick={handleCloseBatchImport}
                >
                  完成
                </Button>
              </>
            )}

            {batchImportStatus.status === 'failed' && (
              <>
                <Alert
                  message="导入失败"
                  description={batchImportStatus.error_message}
                  type="error"
                  showIcon
                />
                <Button
                  style={{ marginTop: 16 }}
                  onClick={handleCloseBatchImport}
                >
                  关闭
                </Button>
              </>
            )}
          </div>
        )}
      </Modal>

      {/* 映射管理抽屉 */}
      <Drawer
        title={`${currentProgram?.name} - 映射子程序`}
        placement="right"
        width={640}
        onClose={() => setRelationDrawerVisible(false)}
        open={relationDrawerVisible}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setAddRelationModalVisible(true)}
          >
            添加映射
          </Button>
        }
      >
        <Alert
          message="映射说明"
          description="子程序映射后将展示并沿用当前父程序的版本、说明、文件、状态等业务数据。被映射程序必须没有任何版本或文件。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
            gap: '16px',
          }}
        >
          {relatedPrograms?.map((mapping) => {
            const program = mapping.child_program;
            return (
              <div
                key={mapping.id}
                className="program-card"
                style={{
                  width: '100%',
                  background: 'white',
                  borderRadius: '12px',
                  borderLeft: '4px solid #005BC1',
                  boxShadow: '0px 4px 12px rgba(30, 58, 138, 0.05)',
                  display: 'flex',
                  flexDirection: 'column',
                  position: 'relative',
                  overflow: 'hidden',
                  border: '1px solid #F1F4F5',
                  borderLeftWidth: '4px',
                }}
              >
                <div
                  style={{
                    padding: '20px 24px 16px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                  }}
                >
                  <div style={{ display: 'flex', gap: '12px' }}>
                    <div
                      style={{
                        width: '40px',
                        height: '40px',
                        background: 'rgba(0, 91, 193, 0.05)',
                        borderRadius: '8px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <AppstoreOutlined
                        style={{ color: '#005BC1', fontSize: '20px' }}
                      />
                    </div>
                    <div
                      style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: '4px',
                      }}
                    >
                      <div
                        style={{
                          color: '#2D3335',
                          fontSize: '16px',
                          fontWeight: 700,
                          fontFamily: 'Inter, sans-serif',
                        }}
                      >
                        {program.name}
                      </div>
                      <div
                        style={{
                          color: '#5A6062',
                          fontSize: '12px',
                          fontFamily: 'Liberation Mono, monospace',
                        }}
                      >
                        编号: {program.code}
                      </div>
                      <div
                        style={{
                          color: '#005BC1',
                          fontSize: '12px',
                          fontWeight: 600,
                        }}
                      >
                        与 {currentProgram?.name} 关联
                      </div>
                    </div>
                  </div>
                </div>

                <div
                  style={{
                    padding: '12px 24px',
                    borderTop: '1px solid #F1F4F5',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '8px',
                  }}
                >
                  <div style={{ color: '#5A6062', fontSize: '12px' }}>
                    显示父程序数据：{currentProgram?.version || '暂无版本'}
                  </div>
                  <div style={{ color: '#5A6062', fontSize: '12px' }}>
                    子程序自身版本数：{program.own_version_count || 0}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Popconfirm
                      title="确定取消映射?"
                      onConfirm={() => handleDeleteRelation(mapping.id)}
                    >
                      <span
                        style={{
                          color: '#A83836',
                          fontSize: '12px',
                          fontWeight: 700,
                          cursor: 'pointer',
                          fontFamily: 'WenQuanYi Zen Hei, sans-serif',
                        }}
                      >
                        取消映射
                      </span>
                    </Popconfirm>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
        {(!relatedPrograms || relatedPrograms.length === 0) &&
          !modalLoading && (
            <div
              style={{ textAlign: 'center', padding: '40px', color: '#999' }}
            >
              暂无映射子程序
            </div>
          )}
      </Drawer>

      {/* 添加映射模态框 */}
      <Modal
        title="添加映射子程序"
        open={addRelationModalVisible}
        onCancel={() => {
          setAddRelationModalVisible(false);
          relationForm.resetFields();
          setMappingSearchKeyword('');
          setDebouncedMappingSearchKeyword('');
          setMappingFilterProductionLine(null);
          setMappingFilterVehicleModel(null);
          setMappingFilterStatus(null);
        }}
        onOk={() => relationForm.submit()}
      >
        <Form
          form={relationForm}
          layout="vertical"
          onFinish={handleAddRelation}
        >
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '1.4fr 1fr 1fr 1fr',
              gap: '12px',
              marginBottom: '16px',
            }}
          >
            <Input
              value={mappingSearchKeyword}
              onChange={(event) => setMappingSearchKeyword(event.target.value)}
              placeholder="搜索程序名称或编号"
              allowClear
            />
            <Select
              value={mappingFilterProductionLine ?? undefined}
              onChange={(value) =>
                setMappingFilterProductionLine(value ?? null)
              }
              placeholder="筛选产线"
              allowClear
              options={productionLines.map((line) => ({
                value: line.id,
                label: line.name,
              }))}
            />
            <Select
              value={mappingFilterVehicleModel ?? undefined}
              onChange={(value) => setMappingFilterVehicleModel(value ?? null)}
              placeholder="筛选车型"
              allowClear
              options={vehicleModels.map((model) => ({
                value: model.id,
                label: model.name,
              }))}
            />
            <Select
              value={mappingFilterStatus ?? undefined}
              onChange={(value) => setMappingFilterStatus(value ?? null)}
              placeholder="筛选状态"
              allowClear
              options={[
                { value: 'in_progress', label: '进行中' },
                { value: 'completed', label: '已完成' },
              ]}
            />
          </div>
          <Form.Item
            name="child_program_ids"
            label={`选择要映射的子程序（共 ${availableMappingCandidatePrograms.length} 个候选）`}
            rules={[{ required: true, message: '请选择至少一个程序' }]}
          >
            <Select
              mode="multiple"
              showSearch
              placeholder="从筛选结果中选择多个��程序"
              filterOption={false}
              options={availableMappingCandidatePrograms.map((p) => ({
                value: p.id,
                disabled:
                  (p.own_version_count || 0) > 0 || (p.own_file_count || 0) > 0,
                label: `${p.name} (${p.code}) · ${p.production_line?.name || '-'} · ${p.vehicle_model?.name || '-'}${(p.own_version_count || 0) > 0 || (p.own_file_count || 0) > 0 ? ' · 已有版本/文件，不能映射' : ''}`,
              }))}
              maxTagCount="responsive"
            />
          </Form.Item>
          <Alert
            type="warning"
            showIcon
            message="被映射的子程序必须没有任何版本或文件，否则后端会拒绝创建映射。"
          />
        </Form>
      </Modal>
      {/* 注入表格自定义样式 */}
    </div>
  );
};

export default ProgramManagement;
