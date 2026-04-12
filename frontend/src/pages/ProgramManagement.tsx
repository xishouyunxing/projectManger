import { useEffect, useState } from 'react';
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
  DatePicker,
  ConfigProvider,
} from 'antd';
import {
  PlusOutlined,
  UploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileOutlined,
  ClockCircleOutlined,
  FolderAddOutlined,
  CheckCircleOutlined,
  LoadingOutlined,
  EditOutlined,
  UserOutlined,
  FileExcelOutlined,
  LinkOutlined,
  SettingOutlined,
  SearchOutlined,
  SlidersOutlined,
  AppstoreOutlined,
  CloseOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import ProgramCustomFieldInputs from '../components/program/ProgramCustomFieldInputs';
import ProgramCustomFieldFilter from '../components/program/ProgramCustomFieldFilter';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Step } = Steps;

// ---- TypeScript 接口定义 ----
interface ProductionLine {
  id: number;
  name: string;
}

interface VehicleModel {
  id: number;
  name: string;
}

interface ProgramFile {
  id: number;
  file_name: string;
  file_size: number;
  created_at: string;
  file_exists?: boolean;
  uploader?: { name: string };
}

interface ProgramVersion {
  id: number;
  version: string;
  is_current: boolean;
  change_log?: string;
  created_at: string;
  file_count?: number;
  files?: ProgramFile[];
  uploader?: { name: string };
}

interface Program {
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
}

interface ProgramCustomFieldDefinition {
  id: number;
  name: string;
  field_type: 'text' | 'select';
  options_json: string;
  sort_order: number;
  enabled: boolean;
}

interface ProgramListCustomFieldValue {
  field_id: number;
  field_name: string;
  field_type: 'text' | 'select';
  sort_order: number;
  value: string;
}

// 批量导入相关接口
interface WorkstationInfo {
  name: string;
  programs: {
    name: string;
    files: { name: string; size: number; path: string }[];
  }[];
}

interface BatchImportPreview {
  workstations: WorkstationInfo[];
  total_programs: number;
  total_files: number;
  temp_dir: string;
}

interface WorkstationMapping {
  workstation_name: string;
  production_line_id: number | null;
  vehicle_model_id: number | null;
}

interface BatchImportStatus {
  status: 'idle' | 'processing' | 'completed' | 'failed';
  total: number;
  processed: number;
  success: number;
  failed: number;
  progress: number;
  current_item: string;
  error_message: string;
}

const buildEnabledCustomFields = (data: unknown): ProgramCustomFieldDefinition[] => {
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

const normalizeCustomFieldValues = (
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

const buildBaseProgramPayload = (values: any) => {
  const { custom_field_values, ...baseValues } = values;
  return baseValues;
};

const buildCustomFieldValuesPayload = (values: any) => {
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

const getProgramCustomFieldValue = (program: Program, fieldId: number) => {
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

const getProgramCustomFieldSummaries = (
  program: Program,
): ProgramListCustomFieldValue[] => {
  if (Array.isArray(program.custom_field_values)) {
    return [...program.custom_field_values].sort((a, b) => a.sort_order - b.sort_order);
  }

  return [];
};

const ProgramManagement = () => {
  const [searchParams] = useSearchParams();
  const { user } = useAuth();
  const selectedProgramId = Number(searchParams.get('id') || 0);
  const [programs, setPrograms] = useState<Program[]>([]);
  const [productionLines, setProductionLines] = useState<ProductionLine[]>([]);
  const [vehicleModels, setVehicleModels] = useState<VehicleModel[]>([]);
  const [tableLoading, setTableLoading] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [fileModalVisible, setFileModalVisible] = useState(false);
  const [currentProgram, setCurrentProgram] = useState<Program | null>(null);
  const [versions, setVersions] = useState<ProgramVersion[]>([]);
  const [customFields, setCustomFields] = useState<ProgramCustomFieldDefinition[]>([]);
  const [form] = Form.useForm();
  const [uploadForm] = Form.useForm();

  // 筛选相关状态
  const [filterProductionLine, setFilterProductionLine] = useState<number | null>(null);
  const [filterVehicleModel, setFilterVehicleModel] = useState<number | null>(null);
  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterDateRange, setFilterDateRange] = useState<[string | null, string | null]>([null, null]);
  const [customFieldFilters, setCustomFieldFilters] = useState<ProgramCustomFieldDefinition[]>([]);
  const [customFieldFilterValues, setCustomFieldFilterValues] = useState<Record<string, string>>({});

  // 关联管理相关状态
  const [relationDrawerVisible, setRelationDrawerVisible] = useState(false);
  const [relatedPrograms, setRelatedPrograms] = useState<Program[]>([]);
  const [addRelationModalVisible, setAddRelationModalVisible] = useState(false);
  const [relationForm] = Form.useForm();

  // 筛选后的程序列表
  const filteredPrograms = programs.filter((program) => {
    // 关键词搜索（模糊匹配名称和编号）
    if (searchKeyword) {
      const keyword = searchKeyword.toLowerCase();
      const nameMatch = program.name?.toLowerCase().includes(keyword);
      const codeMatch = program.code?.toLowerCase().includes(keyword);
      if (!nameMatch && !codeMatch) {
        return false;
      }
    }
    if (filterProductionLine && program.production_line_id !== filterProductionLine) {
      return false;
    }
    if (filterVehicleModel && program.vehicle_model_id !== filterVehicleModel) {
      return false;
    }
    if (filterStatus && program.status !== filterStatus) {
      return false;
    }
    for (const field of customFieldFilters) {
      const filterValue = customFieldFilterValues[String(field.id)]?.trim();
      if (!filterValue) {
        continue;
      }

      const programValue = getProgramCustomFieldValue(program, field.id);
      if (field.field_type === 'select') {
        if (programValue !== filterValue) {
          return false;
        }
        continue;
      }

      if (!programValue.toLowerCase().includes(filterValue.toLowerCase())) {
        return false;
      }
    }
    // 时间筛选
    if (filterDateRange[0] || filterDateRange[1]) {
      const programDate = new Date(program.created_at);
      if (filterDateRange[0]) {
        const startDate = new Date(filterDateRange[0]);
        startDate.setHours(0, 0, 0, 0);
        if (programDate < startDate) return false;
      }
      if (filterDateRange[1]) {
        const endDate = new Date(filterDateRange[1]);
        endDate.setHours(23, 59, 59, 999);
        if (programDate > endDate) return false;
      }
    }
    return true;
  }).sort((a, b) => {
    if (selectedProgramId) {
      if (a.id === selectedProgramId) return -1;
      if (b.id === selectedProgramId) return 1;
    }

    const aEditingByMe = a.status === 'in_progress' && a.editing_by === user?.id;
    const bEditingByMe = b.status === 'in_progress' && b.editing_by === user?.id;
    if (aEditingByMe && !bEditingByMe) return -1;
    if (!aEditingByMe && bEditingByMe) return 1;

    return 0;
  });

  // 重置筛选
  const handleResetFilter = () => {
    setFilterProductionLine(null);
    setFilterVehicleModel(null);
    setFilterStatus(null);
    setSearchKeyword('');
    setFilterDateRange([null, null]);
    setCustomFieldFilters([]);
    setCustomFieldFilterValues({});
  };

  const loadCustomFields = async (productionLineId: number) => {
    const response = await api.get(`/production-lines/${productionLineId}/custom-fields`);
    return buildEnabledCustomFields(response.data);
  };

  const handleFilterProductionLineChange = async (value?: number) => {
    const nextValue = value ?? null;
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
      if (filterProductionLine) params.production_line_id = filterProductionLine;
      if (filterVehicleModel) params.vehicle_model_id = filterVehicleModel;
      if (filterStatus) params.status = filterStatus;

      const response = await api.get('/programs/export/excel', {
        params,
        responseType: 'blob',
      });

      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      
      // 从响应头获取文件名，或使用默认文件名
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

  // 关联管理函数
  const handleViewRelations = async (record: Program) => {
    setCurrentProgram(record);
    setModalLoading(true);
    try {
      const response = await api.get(`/relations/related/${record.id}`);
      setRelatedPrograms(response.data || []);
      setRelationDrawerVisible(true);
    } catch (error) {
      console.error('Failed to load relations:', error);
      message.error('加载关联程序失败');
      setRelatedPrograms([]);
    } finally {
      setModalLoading(false);
    }
  };

  const handleAddRelation = async (values: any) => {
    try {
      await api.post('/relations', {
        source_program_id: currentProgram?.id,
        related_program_id: values.related_program_id,
        relation_type: 'same_program',
        description: values.description || '相同程序',
      });
      message.success('关联成功');
      setAddRelationModalVisible(false);
      relationForm.resetFields();
      // 重新加载关联列表
      if (currentProgram) {
        const response = await api.get(`/relations/related/${currentProgram.id}`);
        setRelatedPrograms(response.data);
      }
    } catch (error: any) {
      console.error('Failed to add relation:', error);
      message.error(error.response?.data?.error || '关联失败');
    }
  };

  const handleDeleteRelation = async (relatedProgramId: number) => {
    try {
      // 首先获取关联关系ID
      const response = await api.get(`/relations/program/${currentProgram?.id}`);
      const relation = response.data.find(
        (r: any) =>
          (r.source_program_id === currentProgram?.id && r.related_program_id === relatedProgramId) ||
          (r.related_program_id === currentProgram?.id && r.source_program_id === relatedProgramId)
      );
      if (relation) {
        await api.delete(`/relations/${relation.id}`);
        message.success('取消关联成功');
        // 重新加载关联列表
        if (currentProgram) {
          const res = await api.get(`/relations/related/${currentProgram.id}`);
          setRelatedPrograms(res.data);
        }
      }
    } catch (error) {
      console.error('Failed to delete relation:', error);
      message.error('取消关联失败');
    }
  };

  // 版本描述编辑相关状态
  const [editingVersionId, setEditingVersionId] = useState<number | null>(null);
  const [editingChangeLog, setEditingChangeLog] = useState('');

  // 批量导入相关状态
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
    loadData();
  }, []);

  useEffect(() => {
    const keyword = searchParams.get('keyword');
    if (keyword) {
      setSearchKeyword(keyword);
    }
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

  const loadData = async () => {
    setTableLoading(true);
    try {
      const [programsRes, linesRes, modelsRes] = await Promise.all([
        api.get('/programs'),
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ]);
      setPrograms(programsRes.data);
      setProductionLines(linesRes.data);
      setVehicleModels(modelsRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
      message.error('加载数据失败，请刷新重试');
    } finally {
      setTableLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentProgram(null);
    setCustomFields([]);
    form.resetFields();
    form.setFieldValue('status', 'in_progress');
    setModalVisible(true);
  };

  const handleEdit = async (record: Program) => {
    setCurrentProgram(record);
    setCustomFields([]);
    form.setFieldsValue({
      ...record,
      custom_field_values: normalizeCustomFieldValues(record.custom_field_values),
    });
    setModalVisible(true);

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
    const payload = buildBaseProgramPayload(values);
    const customFieldPayload = buildCustomFieldValuesPayload(values);

    try {
      if (currentProgram) {
        await api.put(`/programs/${currentProgram.id}`, payload);
        await api.put(`/programs/${currentProgram.id}/custom-field-values`, customFieldPayload);
        message.success('更新成功');
      } else {
        const response = await api.post('/programs', payload);
        await api.put(`/programs/${response.data.id}/custom-field-values`, customFieldPayload);
        message.success('创建成功');
      }
      setModalVisible(false);
      setCustomFields([]);
      loadData();
    } catch (error: any) {
      console.error('Failed to submit:', error);
      const serverMsg = error?.response?.data?.error;
      message.error(
        serverMsg ? `操作失败: ${serverMsg}` : '操作失败，请稍后重试',
      );
    }
  };

  // 批量导入相关函数
  const handleBatchImport = () => {
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
      message.error(error.response?.data?.error || '上传失败');
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
        temp_dir: batchImportPreview?.temp_dir,
        mappings: workstationMappings,
      });

      message.success('批量导入已开始');
      startBatchImportPolling(response.data.task_id);
    } catch (error: any) {
      console.error('Failed to start batch import:', error);
      message.error(error.response?.data?.error || '启动导入失败');
      setBatchImportStep(1);
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
      } catch (error) {
        console.error('Failed to poll import status:', error);
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
      
      // 如果当前正在查看程序版本详情，则刷新版本列表
      if (currentProgram && fileModalVisible) {
        try {
          const res = await api.get(`/files/program/${currentProgram.id}`);
          setVersions(res.data.versions || []);
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

  const handleViewFiles = async (record: Program) => {
    setCurrentProgram(record);
    setModalLoading(true);
    try {
      const response = await api.get(`/files/program/${record.id}`);
      setVersions(response.data.versions || []);
      setFileModalVisible(true);
    } catch (error) {
      console.error('Failed to load files:', error);
      message.error('加载文件列表失败');
    } finally {
      setModalLoading(false);
    }
  };

  const handleSaveVersionChangeLog = async (versionId: number) => {
    try {
      await api.put(`/versions/${versionId}`, { change_log: editingChangeLog });
      message.success('版本说明更新成功');
      setEditingVersionId(null);
      setEditingChangeLog('');
      // 刷新版本列表
      if (currentProgram) {
        const response = await api.get(`/files/program/${currentProgram.id}`);
        setVersions(response.data.versions || []);
      }
    } catch (error) {
      console.error('Failed to update change log:', error);
      message.error('更新失败');
    }
  };

  const handleCancelEditChangeLog = () => {
    setEditingVersionId(null);
    setEditingChangeLog('');
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
      const latestVersion = versions[0];
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
        const response = await api.get(`/files/program/${currentProgram.id}`);
        setVersions(response.data.versions || []);
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
            <span style={{ color: '#2D3335', fontSize: '15px', fontWeight: 800, fontFamily: 'Inter, sans-serif', letterSpacing: '-0.01em' }}>
              {text}
            </span>
            {fieldSummaries.map((field) => (
              <Tooltip key={field.field_id} title={`${field.field_name}: ${field.value}`}>
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
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
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
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', alignItems: 'flex-start' }}>
            <div style={{ 
              background: isCompleted ? 'rgba(61, 137, 255, 0.20)' : 'rgba(222, 204, 253, 0.40)', 
              borderRadius: '9999px', 
              display: 'inline-flex', 
              alignItems: 'center',
              padding: '2px 10px',
              gap: '6px'
            }}>
              <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: isCompleted ? '#005BC1' : '#50426B' }}></div>
              <span style={{ color: isCompleted ? '#005BC1' : '#50426B', fontSize: '11px', fontWeight: 700 }}>
                {isCompleted ? '已完成' : '进行中'}
              </span>
            </div>
            {!isCompleted && record.editing_user?.name && (
              <span style={{ color: '#5A6062', fontSize: '11px', fontWeight: 500 }}>
                {record.editing_by === user?.id ? '我正在编辑' : `${record.editing_user.name} 正在编辑`}
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
          <Tooltip title="查看版本">
            <Button
              type="text"
              icon={<EyeOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleViewFiles(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Tooltip title="上传文件">
            <Button
              type="text"
              icon={<UploadOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleUpload(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleEdit(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
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
                  label: '关联管理',
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
                style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
              />
            </Tooltip>
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <div className="management-page">
      {/* 顶部标题区 */}
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>制造</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">程序管理</span>
          </div>
          <Title level={2} className="management-page-title">
            程序管理
          </Title>
        </div>
        <Space>
          <Button 
            icon={<FileExcelOutlined />} 
            onClick={handleExportExcel}
            style={{ height: '44px', borderRadius: '8px', fontWeight: 600, padding: '0 16px' }}
          >
            导出Excel
          </Button>
          <Button 
            icon={<FolderAddOutlined />} 
            onClick={handleBatchImport}
            style={{ height: '44px', borderRadius: '8px', fontWeight: 600, padding: '0 16px' }}
          >
            批量导入
          </Button>
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={handleAdd}
            style={{
              background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
              border: 'none',
              boxShadow: '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)',
              borderRadius: '8px',
              height: '44px',
              padding: '0 24px',
              fontWeight: 600,
              fontSize: '16px'
            }}
          >
            新建程序
          </Button>
        </Space>
      </div>

      {/* 筛选区域 */}
      <ConfigProvider
        theme={{
          components: {
            Input: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            Select: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            DatePicker: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            }
          }
        }}
      >
      <div className="management-filter-panel">
        <div className="management-filter-field flex">
          <div className="management-filter-label">程序名称/编号</div>
            <Input 
              style={{ width: '192px', maxWidth: '100%' }}
              placeholder="搜索参数..." 
              value={searchKeyword} 
              onChange={(e) => setSearchKeyword(e.target.value)}
            />
          </div>
        <div className="management-filter-field">
          <div className="management-filter-label">生产线</div>
            <Select 
              placeholder="所有生产线" 
              value={filterProductionLine} 
              onChange={handleFilterProductionLineChange}
              allowClear
              style={{ width: '168px', maxWidth: '100%' }}
            >
              {productionLines.map((line: any) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
          </div>
        <div className="management-filter-field">
          <div className="management-filter-label">车型</div>
            <Select 
              placeholder="所有车型" 
              value={filterVehicleModel} 
              onChange={setFilterVehicleModel}
              allowClear
              style={{ width: '168px', maxWidth: '100%' }}
            >
              {vehicleModels.map((model: any) => (
                <Select.Option key={model.id} value={model.id}>
                  {model.name}
                </Select.Option>
              ))}
            </Select>
          </div>
        <div className="management-filter-field">
          <div className="management-filter-label">状态</div>
            <Select 
              placeholder="所有状态" 
              value={filterStatus} 
              onChange={setFilterStatus}
              allowClear
              style={{ width: '148px', maxWidth: '100%' }}
            >
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="in_progress">进行中</Select.Option>
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">创建日期</div>
            <DatePicker.RangePicker 
              style={{ width: '176px', maxWidth: '100%' }}
              onChange={(_, dateStrings) => {
                setFilterDateRange([dateStrings[0] || null, dateStrings[1] || null]);
              }}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Button 
              onClick={() => {}}
              icon={<SearchOutlined />}
              style={{ height: '40px', width: '104px', borderRadius: '8px', background: '#DEE3E6', color: '#2D3335', fontWeight: 700, border: 'none' }}
            >
              查询
            </Button>
            <Button 
              type="text"
              onClick={handleResetFilter} 
              style={{ height: '40px', color: '#005BC1', fontWeight: 700, letterSpacing: '1.2px', border: 'none', background: 'transparent' }}
            >
              重置
            </Button>
          </div>

          {filterProductionLine && customFieldFilters.length > 0 && (
            <div
              style={{
                gridColumn: '1 / -1',
                marginTop: '4px',
                paddingTop: '20px',
                borderTop: '1px solid rgba(173, 179, 181, 0.18)',
              }}
            >
              <div
                style={{
                  padding: '0',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: '14px',
                    flexWrap: 'wrap',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      gap: '6px',
                      minWidth: '84px',
                      minHeight: '36px',
                      alignSelf: 'center',
                      flexShrink: 0,
                    }}
                  >
                    <SlidersOutlined style={{ color: '#94A3B8', fontSize: '21px' }} />
                    <span
                      style={{
                        color: '#64748B',
                        fontSize: '11px',
                        fontWeight: 700,
                        letterSpacing: '0.14em',
                        textTransform: 'uppercase',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      附加项
                    </span>
                  </div>
                  <div
                    style={{
                      width: '1px',
                      alignSelf: 'stretch',
                      margin: '4px 0',
                      background: 'rgba(173, 179, 181, 0.18)',
                    }}
                  />
                  <div style={{ flex: 1, minWidth: '260px' }}>
                    <ProgramCustomFieldFilter
                      fields={customFieldFilters}
                      values={customFieldFilterValues}
                      onChange={handleCustomFieldFilterChange}
                    />
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </ConfigProvider>

      {/* 数据表格 */}
      <div className="management-table-card">
        <Table
          className="custom-table"
          columns={columns}
          dataSource={filteredPrograms}
          rowKey="id"
          loading={tableLoading}
          pagination={{
            showTotal: (total, range) => `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
          locale={{
            emptyText: (
              <div style={{ padding: '40px 0' }}>
                <FileOutlined style={{ fontSize: '48px', color: '#d9d9d9', marginBottom: '16px' }} />
                <div style={{ color: '#999', marginBottom: '16px' }}>
                  暂无程序数据
                </div>
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                  创建第一个程序
                </Button>
              </div>
            ),
          }}
        />
      </div>
      <Modal
        title={currentProgram ? '编辑程序' : '新建程序'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setCustomFields([]);
        }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="程序名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label="程序编号" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="production_line_id"
            label="生产线"
            rules={[{ required: true }]}
          >
            <Select onChange={(value) => void handleModalProductionLineChange(value)}>
              {productionLines.map((line: any) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <ProgramCustomFieldInputs fields={customFields} />
          <Form.Item name="vehicle_model_id" label="车型">
            <Select allowClear>
              {vehicleModels.map((model: any) => (
                <Select.Option key={model.id} value={model.id}>
                  {model.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={4} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="in_progress">
            <Select>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="in_progress">进行中</Select.Option>
            </Select>
          </Form.Item>
        </Form>
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

      {/* 文件版本管理模态框 */}
      <Modal
        title={null}
        closable={false}
        open={fileModalVisible}
        onCancel={() => setFileModalVisible(false)}
        footer={null}
        width={1024}
        styles={{ body: { padding: 0 } }}
        style={{ top: 20 }}
      >
        <div style={{ background: '#F1F4F5', width: '100%', height: '751px', display: 'flex', flexDirection: 'column', borderRadius: '16px', overflow: 'hidden' }}>
          {/* Header */}
          <div style={{ height: '70px', flexShrink: 0, background: 'rgba(255, 255, 255, 0.70)', backdropFilter: 'blur(10px)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 32px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
               <AppstoreOutlined style={{ fontSize: '20px', color: '#2D3335' }} />
               <span style={{ color: '#2D3335', fontSize: '20px', fontWeight: 700 }}>
                 {currentProgram?.name} - 版本文件管理
               </span>
            </div>
            <div 
              onClick={() => setFileModalVisible(false)}
              style={{ width: '30px', height: '30px', borderRadius: '50%', background: '#F8F9FA', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}
            >
              <CloseOutlined style={{ color: '#5A6062', fontSize: '14px' }} />
            </div>
          </div>
          
          {/* Content */}
          <div style={{ flex: 1, overflowY: 'auto', padding: '32px' }}>
            {modalLoading ? (
              <div style={{ textAlign: 'center', padding: '40px' }}>加载中...</div>
            ) : versions.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '40px' }}>
                <p>暂无版本信息</p>
                <Button
                  type="primary"
                  icon={<UploadOutlined />}
                  onClick={() => {
                    setFileModalVisible(false);
                    if (currentProgram) handleUpload(currentProgram);
                  }}
                >
                  上传第一个版本
                </Button>
              </div>
            ) : (
              <div>
                {versions.map((version: ProgramVersion) => (
                  <div key={version.id} style={{ marginBottom: '40px' }}>
                    {/* Version Info Card */}
                    <div style={{ background: 'white', borderRadius: '16px', padding: '32px', marginBottom: '32px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '24px' }}>
                        <div>
                          <div style={{ color: '#005BC1', fontSize: '12px', fontWeight: 700, letterSpacing: '0.6px', marginBottom: '4px' }}>
                            {version.is_current ? 'Current Version' : 'History Version'}
                          </div>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            <span style={{ color: '#2D3335', fontSize: '36px', fontWeight: 800 }}>
                              {version.version}
                            </span>
                            <div style={{ background: version.is_current ? 'rgba(61, 137, 255, 0.20)' : '#EBEEF0', borderRadius: '9999px', padding: '4px 12px' }}>
                              <span style={{ color: version.is_current ? '#005BC1' : '#5A6062', fontSize: '12px', fontWeight: 700 }}>
                                {version.is_current ? 'STABLE' : 'ARCHIVED'}
                              </span>
                            </div>
                          </div>
                        </div>
                        
                        <div style={{ display: 'flex', gap: '12px' }}>
                          <div 
                             onClick={(e) => {
                                e.stopPropagation();
                                if (currentProgram) {
                                  uploadForm.resetFields();
                                  uploadForm.setFieldValue('program_id', currentProgram.id);
                                  uploadForm.setFieldValue('version', version.version);
                                  setUploadModalVisible(true);
                                }
                             }}
                             style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 24px', background: '#DEE3E6', borderRadius: '12px', cursor: 'pointer' }}
                          >
                             <UploadOutlined style={{ color: '#2D3335', fontSize: '16px' }} />
                             <span style={{ color: '#2D3335', fontSize: '16px', fontWeight: 600 }}>重传此版本</span>
                          </div>
                          <div 
                             onClick={async (e) => {
                                e.stopPropagation();
                                if (version.files && version.files.length > 0) {
                                  try {
                                    if (version.files.length === 1) {
                                      const file = version.files[0];
                                      await downloadWithAuth(
                                        `/files/download/${file.id}`,
                                        file.file_name,
                                      );
                                    } else {
                                      await downloadWithAuth(
                                        `/files/download/version/${version.version}?program_id=${currentProgram?.id}`,
                                        `${currentProgram?.code || currentProgram?.id}_${version.version}.zip`,
                                      );
                                      message.success(
                                        `正在打包下载版本 ${version.version} 的所有文件...`,
                                      );
                                    }
                                  } catch (error) {
                                    console.error('Failed to download version files:', error);
                                    message.error('下载失败');
                                  }
                                } else {
                                  message.warning('该版本暂无文件');
                                }
                             }}
                             style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 24px', background: 'linear-gradient(162deg, #005BC1 0%, #3D89FF 100%)', borderRadius: '12px', cursor: 'pointer' }}
                          >
                             <DownloadOutlined style={{ color: '#F9F8FF', fontSize: '16px' }} />
                             <span style={{ color: '#F9F8FF', fontSize: '16px', fontWeight: 600 }}>批量下载</span>
                          </div>
                          {!version.is_current && (
                             <div 
                               onClick={(e) => {
                                  e.stopPropagation();
                                  Modal.confirm({
                                    title: '确认激活版本',
                                    content: `确定要激活版本 ${version.version} 吗？这将设为当前版本。`,
                                    onOk: async () => {
                                      try {
                                        await api.put(`/versions/${version.id}/activate`);
                                        message.success('版本激活成功');
                                        if (currentProgram) handleViewFiles(currentProgram);
                                      } catch (error) {
                                        message.error('激活失败');
                                      }
                                    },
                                  });
                               }}
                               style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 24px', background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: '12px', cursor: 'pointer' }}
                             >
                               <CheckCircleOutlined style={{ color: '#1890ff', fontSize: '16px' }} />
                               <span style={{ color: '#1890ff', fontSize: '16px', fontWeight: 600 }}>激活</span>
                             </div>
                          )}
                        </div>
                      </div>

                      <div style={{ borderTop: '1px solid rgba(173, 179, 181, 0.10)', borderBottom: '1px solid rgba(173, 179, 181, 0.10)', padding: '24px 0' }}>
                        <div style={{ display: 'flex' }}>
                          <div style={{ flex: 1 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                              <UserOutlined style={{ color: '#5A6062', fontSize: '12px' }} />
                              <span style={{ color: '#5A6062', fontSize: '10px', fontWeight: 700 }}>上传者</span>
                            </div>
                            <div style={{ color: '#2D3335', fontSize: '16px', fontWeight: 500 }}>
                              {version.uploader?.name || '系统管理员'}
                            </div>
                          </div>
                          <div style={{ flex: 1 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                              <ClockCircleOutlined style={{ color: '#5A6062', fontSize: '12px' }} />
                              <span style={{ color: '#5A6062', fontSize: '10px', fontWeight: 700 }}>创建时间</span>
                            </div>
                            <div style={{ color: '#2D3335', fontSize: '16px', fontWeight: 500 }}>
                              {new Date(version.created_at).toLocaleString('zh-CN', { hour12: false })}
                            </div>
                          </div>
                          <div style={{ flex: 2 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <FileTextOutlined style={{ color: '#5A6062', fontSize: '12px' }} />
                                <span style={{ color: '#5A6062', fontSize: '10px', fontWeight: 700 }}>版本说明</span>
                              </div>
                              {editingVersionId !== version.id && (
                                <span 
                                  style={{ color: '#005BC1', fontSize: '10px', fontWeight: 700, cursor: 'pointer' }}
                                  onClick={() => {
                                    setEditingVersionId(version.id);
                                    setEditingChangeLog(version.change_log || '');
                                  }}
                                >
                                  编辑
                                </span>
                              )}
                            </div>
                            <div style={{ color: '#5A6062', fontSize: '14px', fontWeight: 400, lineHeight: 1.5 }}>
                              {editingVersionId === version.id ? (
                                <div>
                                  <Input.TextArea
                                    rows={2}
                                    value={editingChangeLog}
                                    onChange={(e) => setEditingChangeLog(e.target.value)}
                                    placeholder="请输入版本说明..."
                                    style={{ marginBottom: '8px' }}
                                  />
                                  <div style={{ textAlign: 'right' }}>
                                    <Space>
                                      <Button size="small" onClick={handleCancelEditChangeLog}>取消</Button>
                                      <Button size="small" type="primary" onClick={() => handleSaveVersionChangeLog(version.id)}>保存</Button>
                                    </Space>
                                  </div>
                                </div>
                              ) : (
                                version.change_log || '暂无说明'
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* File Assets Card */}
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                        <span style={{ color: '#2D3335', fontSize: '18px', fontWeight: 700 }}>文件资产</span>
                        <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 400 }}>共 {version.files?.length || 0} 个文件</span>
                      </div>
                      
                      <div style={{ background: 'white', borderRadius: '16px', overflow: 'hidden', marginBottom: '24px' }}>
                        <Table
                          dataSource={version.files || []}
                          rowKey="id"
                          pagination={false}
                          showHeader={true}
                          columns={[
                            {
                              title: '文件名',
                              dataIndex: 'file_name',
                              key: 'file_name',
                              render: (text, record) => (
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                  <div style={{ width: '40px', height: '40px', background: 'rgba(61, 137, 255, 0.1)', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                    {text.endsWith('.txt') || text.endsWith('.log') ? (
                                      <FileTextOutlined style={{ color: '#005BC1', fontSize: '16px' }} />
                                    ) : (
                                      <FileOutlined style={{ color: '#005BC1', fontSize: '16px' }} />
                                    )}
                                  </div>
                                  <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                                    <span style={{ color: '#2D3335', fontSize: '16px', fontWeight: 500 }}>{text}</span>
                                    {record.file_exists === false && (
                                      <Tag color="red" style={{ width: 'fit-content', margin: 0 }}>文件缺失</Tag>
                                    )}
                                  </div>
                                </div>
                              ),
                            },
                            {
                              title: '大小',
                              dataIndex: 'file_size',
                              key: 'file_size',
                              render: (size) => <span style={{ color: '#5A6062', fontSize: '14px' }}>{(size / 1024).toFixed(1)} KB</span>,
                            },
                            {
                              title: '上传时间',
                              dataIndex: 'created_at',
                              key: 'created_at',
                              render: (time) => <span style={{ color: '#5A6062', fontSize: '14px' }}>{new Date(time).toLocaleString('zh-CN', { hour12: false })}</span>,
                            },
                            {
                              title: '上传者',
                              key: 'uploader',
                              render: () => <span style={{ color: '#5A6062', fontSize: '14px' }}>{version.uploader?.name || '系统管理员'}</span>,
                            },
                            {
                              title: '操作',
                              key: 'action',
                              align: 'right',
                              render: (_, record) => (
                                <Space size="small">
                                  <Tooltip title={record.file_exists === false ? '文件已缺失，无法下载' : '下载文件'}>
                                    <div 
                                      style={{
                                        display: 'inline-flex',
                                        width: '32px',
                                        height: '32px',
                                        background: record.file_exists === false ? '#F1F4F5' : '#F8F9FA',
                                        borderRadius: '12px',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                        cursor: record.file_exists === false ? 'not-allowed' : 'pointer',
                                        opacity: record.file_exists === false ? 0.5 : 1,
                                      }}
                                      onClick={() => {
                                        if (record.file_exists === false) {
                                          message.warning('该文件已被物理删除，请联系管理员清理记录');
                                          return;
                                        }
                                        downloadWithAuth(`/files/download/${record.id}`, record.file_name);
                                      }}
                                    >
                                      <DownloadOutlined style={{ color: '#5A6062', fontSize: '16px' }} />
                                    </div>
                                  </Tooltip>
                                  <Popconfirm
                                    title="确定删除这个文件吗？"
                                    onConfirm={() => handleDeleteSingleFile(record.id)}
                                  >
                                    <Tooltip title="删除文件">
                                      <div
                                        style={{
                                          display: 'inline-flex',
                                          width: '32px',
                                          height: '32px',
                                          background: 'rgba(168, 56, 54, 0.05)',
                                          borderRadius: '12px',
                                          alignItems: 'center',
                                          justifyContent: 'center',
                                          cursor: 'pointer',
                                        }}
                                      >
                                        <DeleteOutlined style={{ color: '#A83836', fontSize: '16px' }} />
                                      </div>
                                    </Tooltip>
                                  </Popconfirm>
                                </Space>
                              ),
                            },
                          ]}
                          components={{
                            header: {
                              cell: (props: any) => (
                                <th {...props} style={{ background: 'rgba(229, 233, 235, 0.50)', color: '#5A6062', fontSize: '10px', fontWeight: 700, letterSpacing: '1px', borderBottom: 'none', padding: '16px 24px' }}>
                                  {props.children}
                                </th>
                              )
                            },
                            body: {
                              cell: (props: any) => (
                                <td {...props} style={{ padding: '20px 24px', borderBottom: '1px solid #EBEEF0' }}>
                                  {props.children}
                                </td>
                              )
                            }
                          }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
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

      {/* 关联管理抽屉 */}
      <Drawer
        title={`${currentProgram?.name} - 关联程序`}
        placement="right"
        width={600}
        onClose={() => setRelationDrawerVisible(false)}
        open={relationDrawerVisible}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setAddRelationModalVisible(true)}
          >
            添加关联
          </Button>
        }
      >
        <Alert
          message="关联说明"
          description="关联的程序将共享'有程序'状态。当您查看矩阵视图时，关联的程序也会显示为已有程序。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '16px' }}>
          {relatedPrograms?.map((program: any) => (
            <div
              key={program.id}
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
                transition: 'all 0.3s'
              }}
            >
              <div style={{ padding: '20px 24px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ display: 'flex', gap: '12px' }}>
                  <div style={{
                    width: '40px',
                    height: '40px',
                    background: 'rgba(0, 91, 193, 0.05)',
                    borderRadius: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}>
                    <AppstoreOutlined style={{ color: '#005BC1', fontSize: '20px' }} />
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                    <div style={{ color: '#2D3335', fontSize: '16px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
                      {program.name}
                    </div>
                    <div style={{ color: '#5A6062', fontSize: '12px', fontFamily: 'Liberation Mono, monospace' }}>
                      ID: {program.code}
                    </div>
                  </div>
                </div>
                <div style={{
                  background: program.status === 'completed' ? 'rgba(61, 137, 255, 0.20)' : 'rgba(222, 204, 253, 0.40)',
                  borderRadius: '9999px',
                  display: 'inline-flex',
                  alignItems: 'center',
                  padding: '2px 10px',
                  gap: '6px'
                }}>
                  <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: program.status === 'completed' ? '#005BC1' : '#50426B' }}></div>
                  <span style={{
                    color: program.status === 'completed' ? '#005BC1' : '#50426B',
                    fontSize: '11px',
                    fontWeight: 700,
                    fontFamily: 'WenQuanYi Zen Hei, sans-serif',
                    letterSpacing: '0.50px'
                  }}>
                    {program.status === 'completed' ? '已完成' : '进行中'}
                  </span>
                </div>
              </div>
              
              <div style={{
                padding: '12px 24px',
                borderTop: '1px solid #F1F4F5',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span style={{ color: '#5A6062', fontSize: '11px', fontWeight: 700, fontFamily: 'WenQuanYi Zen Hei, sans-serif' }}>
                    当前版本：
                  </span>
                  <div style={{ background: 'rgba(0, 91, 193, 0.05)', padding: '2px 8px', borderRadius: '8px' }}>
                    <span style={{ color: '#005BC1', fontSize: '11px', fontWeight: 700, fontFamily: 'WenQuanYi Zen Hei, sans-serif' }}>
                      {program.version || '暂无版本'}
                    </span>
                  </div>
                </div>
                <div>
                  <Popconfirm
                    title="确定取消关联?"
                    onConfirm={() => handleDeleteRelation(program.id)}
                  >
                    <span style={{ color: '#A83836', fontSize: '12px', fontWeight: 700, cursor: 'pointer', fontFamily: 'WenQuanYi Zen Hei, sans-serif' }}>
                      取消关联
                    </span>
                  </Popconfirm>
                </div>
              </div>
            </div>
          ))}
        </div>
        {(!relatedPrograms || relatedPrograms.length === 0) && !modalLoading && (
          <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>
            暂无关联程序
          </div>
        )}
      </Drawer>

      {/* 添加关联模态框 */}
      <Modal
        title="添加关联程序"
        open={addRelationModalVisible}
        onCancel={() => {
          setAddRelationModalVisible(false);
          relationForm.resetFields();
        }}
        onOk={() => relationForm.submit()}
      >
        <Form form={relationForm} layout="vertical" onFinish={handleAddRelation}>
          <Form.Item
            name="related_program_id"
            label="选择要关联的程序"
            rules={[{ required: true, message: '请选择程序' }]}
          >
            <Select
              showSearch
              placeholder="搜索程序名称或编号"
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              options={programs
                .filter((p) => p.id !== currentProgram?.id)
                .filter((p) => !relatedPrograms.some((rp) => rp.id === p.id))
                .map((p) => ({
                  value: p.id,
                  label: `${p.name} (${p.code}) - ${p.production_line?.name || '-'}`,
                }))}
            />
          </Form.Item>
          <Form.Item name="description" label="关联说明">
            <Input placeholder="例如：相同程序" />
          </Form.Item>
        </Form>
      </Modal>
      {/* 注入表格自定义样式 */}
    </div>
  );
};

export default ProgramManagement;
