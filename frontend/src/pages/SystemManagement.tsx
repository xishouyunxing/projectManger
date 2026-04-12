import { useEffect, useState } from 'react';
import {
  Card,
  Button,
  Table,
  Space,
  Modal,
  message,
  Typography,
  Tag,
  Progress,
  Divider,
  Popconfirm,
  Alert,
  Tabs,
  Form,
  Input,
  Select,
  Row,
  Col,
  Statistic,
  Collapse,
} from 'antd';
import {
  DatabaseOutlined,
  FileOutlined,
  CloudDownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  ExclamationCircleOutlined,
  SwapOutlined,
  LoadingOutlined,
  PlusOutlined,
  TeamOutlined,
  EyeOutlined,
  InfoCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const { Title, Text, Paragraph } = Typography;
const { Panel } = Collapse;

interface BackupInfo {
  name: string;
  path: string;
  size: number;
  created_at: string;
  type: 'database' | 'files' | 'full';
}

interface MigrationStatus {
  status: 'not_started' | 'running' | 'completed' | 'failed';
  total_files?: number;
  migrated_files?: number;
  failed_files?: number;
  skipped_files?: number;
  progress?: number;
  current_file?: string;
  start_time?: string;
  end_time?: string;
  error_msg?: string;
  failed_list?: string[];
}

interface MigrationPreview {
  total_files: number;
  need_migrate: number;
  already_migrated: number;
  files: {
    id: number;
    file_name: string;
    old_path: string;
    new_path: string;
    status: string;
    program_name: string;
    line_name: string;
    model_name: string;
  }[];
}

interface IntegrityCheckResult {
  missing_files: {
    id: number;
    file_name: string;
    file_path: string;
    program_id: number;
    version: string;
  }[];
  missing_count: number;
  checked_count: number;
}

const SystemManagement = () => {
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [departments, setDepartments] = useState([]);
  const [productionLines, setProductionLines] = useState([]);
  const [vehicleModels, setVehicleModels] = useState([]);
  const [loading, setLoading] = useState(false);
  const [departmentLoading, setDepartmentLoading] = useState(false);
  const [operationLoading, setOperationLoading] = useState('');
  const [confirmModalVisible, setConfirmModalVisible] = useState(false);
  const [departmentModalVisible, setDepartmentModalVisible] = useState(false);
  const [previewModalVisible, setPreviewModalVisible] = useState(false);
  const [currentDepartment, setCurrentDepartment] = useState<any>(null);
  const [currentOperation, setCurrentOperation] = useState<any>(null);
  const [migrationStatus, setMigrationStatus] = useState<MigrationStatus>({
    status: 'not_started',
  });
  const [migrationPreview, setMigrationPreview] =
    useState<MigrationPreview | null>(null);
  const [integrityResult, setIntegrityResult] =
    useState<IntegrityCheckResult | null>(null);
  const [selectedVehicleModel, setSelectedVehicleModel] = useState<
    number | null
  >(null);
  const [selectedProductionLine, setSelectedProductionLine] = useState<
    number | null
  >(null);
  const [migrationPolling, setMigrationPolling] = useState<ReturnType<
    typeof setInterval
  > | null>(null);
  const [form] = Form.useForm();

  const { isAdmin } = useAuth();

  useEffect(() => {
    loadBackups();
    loadMigrationStatus();
    loadDepartments();
    loadFilterData();
    return () => {
      if (migrationPolling) {
        clearInterval(migrationPolling);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadFilterData = async () => {
    try {
      const [linesRes, modelsRes] = await Promise.all([
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ]);
      setProductionLines(linesRes.data);
      setVehicleModels(modelsRes.data);
    } catch (error) {
      console.error('Failed to load filter data:', error);
    }
  };

  const loadDepartments = async () => {
    setDepartmentLoading(true);
    try {
      const response = await api.get('/departments');
      setDepartments(response.data);
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error('加载部门列表失败');
    } finally {
      setDepartmentLoading(false);
    }
  };

  const handleAddDepartment = () => {
    setCurrentDepartment(null);
    form.resetFields();
    setDepartmentModalVisible(true);
  };

  const handleEditDepartment = (record: any) => {
    setCurrentDepartment(record);
    form.setFieldsValue(record);
    setDepartmentModalVisible(true);
  };

  const handleDeleteDepartment = async (id: number) => {
    try {
      await api.delete(`/departments/${id}`);
      message.success('删除成功');
      loadDepartments();
    } catch (error) {
      console.error('Failed to delete department:', error);
      message.error('删除失败');
    }
  };

  const handleDepartmentSubmit = async (values: any) => {
    try {
      if (currentDepartment) {
        await api.put(`/departments/${currentDepartment.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/departments', values);
        message.success('创建成功');
      }
      setDepartmentModalVisible(false);
      loadDepartments();
    } catch (error) {
      console.error('Failed to submit department:', error);
      message.error('操作失败');
    }
  };

  const loadBackups = async () => {
    setLoading(true);
    try {
      const response = await api.get('/backup');
      setBackups(response.data.backups || []);
    } catch (error) {
      console.error('Failed to load backups:', error);
      message.error('加载备份列表失败');
    } finally {
      setLoading(false);
    }
  };

  const loadMigrationStatus = async () => {
    try {
      const response = await api.get('/migration/status');
      setMigrationStatus(response.data);

      // 如果正在运行，启动轮询
      if (response.data.status === 'running') {
        startMigrationPolling();
      }
    } catch (error) {
      console.error('Failed to load migration status:', error);
    }
  };

  const loadMigrationPreview = async () => {
    try {
      const params: any = {};
      if (selectedVehicleModel) params.vehicle_model_id = selectedVehicleModel;
      if (selectedProductionLine)
        params.production_line_id = selectedProductionLine;

      const response = await api.get('/migration/preview', { params });
      setMigrationPreview(response.data);
      setPreviewModalVisible(true);
    } catch (error) {
      console.error('Failed to load migration preview:', error);
      message.error('加载迁移预览失败');
    }
  };

  const startMigrationPolling = () => {
    if (migrationPolling) clearInterval(migrationPolling);
    const interval = setInterval(async () => {
      try {
        const response = await api.get('/migration/status');
        setMigrationStatus(response.data);
        if (response.data.status !== 'running') {
          clearInterval(interval);
          setMigrationPolling(null);
        }
      } catch (error) {
        clearInterval(interval);
        setMigrationPolling(null);
      }
    }, 2000);
    setMigrationPolling(interval);
  };

  const checkFileIntegrity = async () => {
    try {
      const response = await api.get('/files/integrity-check');
      setIntegrityResult(response.data);
      if (response.data.missing_count > 0) {
        message.warning(`发现 ${response.data.missing_count} 个缺失文件记录`);
      } else {
        message.success('文件完整性检查通过，未发现缺失文件');
      }
    } catch (error: any) {
      console.error('Failed to check file integrity:', error);
      message.error(error.response?.data?.error || '文件完整性检查失败');
    }
  };

  const cleanupMissingFiles = async () => {
    Modal.confirm({
      title: '确认清理缺失文件记录',
      content: '该操作会删除数据库中所有已缺失物理文件的记录，删除后无法恢复。是否继续？',
      okText: '确认清理',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          const response = await api.delete('/files/cleanup-missing');
          message.success(response.data.message || '缺失文件记录清理完成');
          await checkFileIntegrity();
        } catch (error: any) {
          console.error('Failed to cleanup missing files:', error);
          message.error(error.response?.data?.error || '清理缺失文件记录失败');
        }
      },
    });
  };

  const startMigration = async () => {
    try {
      const data: any = {};
      if (selectedVehicleModel) data.vehicle_model_id = selectedVehicleModel;
      if (selectedProductionLine)
        data.production_line_id = selectedProductionLine;

      await api.post('/migration/start', data);
      message.success('文件迁移已开始');
      await loadMigrationStatus();
      startMigrationPolling();
    } catch (error: any) {
      console.error('Failed to start migration:', error);
      message.error(error.response?.data?.error || '启动迁移失败');
    }
  };

  const rollbackMigration = async () => {
    try {
      await api.post('/migration/rollback');
      message.success('迁移回滚已开始');
      await loadMigrationStatus();
      startMigrationPolling();
    } catch (error: any) {
      console.error('Failed to rollback migration:', error);
      message.error(error.response?.data?.error || '回滚失败');
    }
  };

  const createBackup = async (type: string) => {
    setOperationLoading(type);
    try {
      const endpoint =
        type === 'database'
          ? '/backup/database'
          : type === 'files'
            ? '/backup/files'
            : '/backup/full';

      const response = await api.post(endpoint);
      message.success(response.data.message);

      // 刷新备份列表
      await loadBackups();
    } catch (error: any) {
      console.error(`Failed to create ${type} backup:`, error);
      message.error(error.response?.data?.error || `创建${type}备份失败`);
    } finally {
      setOperationLoading('');
    }
  };

  const deleteBackup = async (backupName: string) => {
    try {
      await api.delete(`/backup/${backupName}`);
      message.success('备份删除成功');
      await loadBackups();
    } catch (error: any) {
      console.error('Failed to delete backup:', error);
      message.error(error.response?.data?.error || '删除备份失败');
    }
  };

  const downloadBackup = async (backupName: string) => {
    try {
      const response = await api.get(`/backup/download/${backupName}`, {
        responseType: 'blob',
      });

      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', backupName);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (error: any) {
      console.error('Failed to download backup:', error);
      message.error('下载备份失败');
    }
  };

  const confirmRestore = (backupName: string, type: 'database' | 'files') => {
    setCurrentOperation({ backupName, type });
    setConfirmModalVisible(true);
  };

  const executeRestore = async () => {
    if (!currentOperation) return;

    setOperationLoading(`restore-${currentOperation.type}`);
    try {
      const endpoint =
        currentOperation.type === 'database'
          ? `/backup/restore/database/${currentOperation.backupName}`
          : `/backup/restore/files/${currentOperation.backupName}`;

      const response = await api.post(endpoint);

      Modal.success({
        title: '恢复成功',
        content: (
          <div>
            <p>{response.data.message}</p>
            {response.data.rollback_backup && (
              <p>回滚备份已创建：{response.data.rollback_backup}</p>
            )}
          </div>
        ),
      });

      setConfirmModalVisible(false);
      await loadBackups();
    } catch (error: any) {
      console.error('Failed to restore:', error);
      message.error(error.response?.data?.error || '恢复失败');
    } finally {
      setOperationLoading('');
      setCurrentOperation(null);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN');
  };

  const getBackupTypeTag = (type: string) => {
    const config = {
      database: { color: 'blue', text: '数据库' },
      files: { color: 'green', text: '文件系统' },
      full: { color: 'purple', text: '完整备份' },
    };
    const { color, text } = config[type as keyof typeof config] || {
      color: 'default',
      text: '未知',
    };
    return <Tag color={color}>{text}</Tag>;
  };

  const columns = [
    {
      title: '备份名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => getBackupTypeTag(type),
    },
    {
      title: '文件大小',
      dataIndex: 'size',
      key: 'size',
      render: (size: number) => formatFileSize(size),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: BackupInfo) => (
        <Space>
          <Button
            type="link"
            icon={<CloudDownloadOutlined />}
            onClick={() => downloadBackup(record.name)}
            style={{ padding: 0 }}
          >
            下载
          </Button>
          <Popconfirm
            title="确定要删除这个备份吗？"
            onConfirm={() => deleteBackup(record.name)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />} style={{ padding: 0 }}>
              删除
            </Button>
          </Popconfirm>
          {record.type === 'database' || record.type === 'full' ? (
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => confirmRestore(record.name, 'database')}
              loading={operationLoading === 'restore-database'}
              style={{ padding: 0 }}
            >
              恢复数据库
            </Button>
          ) : null}
          {record.type === 'files' || record.type === 'full' ? (
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => confirmRestore(record.name, 'files')}
              loading={operationLoading === 'restore-files'}
            >
              恢复文件
            </Button>
          ) : null}
        </Space>
      ),
    },
  ];

  const departmentColumns = [
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => {
        const dateObj = new Date(text);
        const dateFmt = dateObj.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).replace(/\//g, '-');
        const timeFmt = dateObj.toLocaleTimeString('zh-CN', { hour12: false });
        return (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>{dateFmt}</span>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 400, fontFamily: 'Inter, sans-serif', opacity: 0.6 }}>{timeFmt}</span>
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space size="middle">
          <Button
            type="link"
            size="small"
            onClick={() => handleEditDepartment(record)}
            style={{ padding: 0 }}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDeleteDepartment(record.id)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small" style={{ padding: 0 }}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const previewColumns = [
    {
      title: '文件名',
      dataIndex: 'file_name',
      key: 'file_name',
    },
    {
      title: '程序',
      dataIndex: 'program_name',
      key: 'program_name',
    },
    {
      title: '生产线',
      dataIndex: 'line_name',
      key: 'line_name',
    },
    {
      title: '车型',
      dataIndex: 'model_name',
      key: 'model_name',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const config = {
          need_migrate: { color: 'orange', text: '待迁移' },
          already_migrated: { color: 'green', text: '已迁移' },
          source_missing: { color: 'red', text: '源文件缺失' },
        };
        const { color, text } = config[status as keyof typeof config] || {
          color: 'default',
          text: '未知',
        };
        return <Tag color={color}>{text}</Tag>;
      },
    },
  ];

  if (!isAdmin) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="权限不足"
          description="只有系统管理员才能访问系统管理功能。"
          type="error"
          showIcon
        />
      </div>
    );
  }

  return (
    <div className="management-page">
      <div className="management-page-header" style={{ marginBottom: '24px' }}>
        <div>
          <div className="management-page-breadcrumb">
            <span>系统</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">系统管理</span>
          </div>
          <Title level={2} className="management-page-title">
            系统管理
          </Title>
        </div>
      </div>

      <Tabs
        items={[
          {
            key: 'migration',
            label: (
              <span>
                <SwapOutlined />
                文件迁移
              </span>
            ),
            children: (
              <>
                {/* 功能说明 */}
                <Alert
                  message="文件迁移功能说明"
                  description={
                    <div>
                      <Paragraph>
                        文件迁移功能用于将旧的扁平文件结构迁移到新的按
                        <strong>车型/生产线/程序/版本</strong>组织的层级结构。
                      </Paragraph>
                      <Paragraph>
                        <strong>迁移前：</strong>uploads/file1.nc（扁平结构）
                        <br />
                        <strong>迁移后：</strong>
                        uploads/25吨汽车起重机/吊臂主臂生产线/PROG001_程序名/v1.0/file1.nc
                      </Paragraph>
                      <Paragraph type="secondary">
                        <InfoCircleOutlined />{' '}
                        迁移前会自动创建完整备份，支持回滚操作。
                      </Paragraph>
                    </div>
                  }
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                {/* 迁移状态卡片 */}
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={6}>
                    <Card>
                      <Statistic
                        title="迁移状态"
                        value={
                          migrationStatus.status === 'running'
                            ? '进行中'
                            : migrationStatus.status === 'completed'
                              ? '已完成'
                              : migrationStatus.status === 'failed'
                                ? '失败'
                                : '待机'
                        }
                        valueStyle={{
                          color:
                            migrationStatus.status === 'running'
                              ? '#1890ff'
                              : migrationStatus.status === 'completed'
                                ? '#52c41a'
                                : migrationStatus.status === 'failed'
                                  ? '#ff4d4f'
                                  : '#999',
                        }}
                        prefix={
                          migrationStatus.status === 'running' ? (
                            <LoadingOutlined />
                          ) : migrationStatus.status === 'completed' ? (
                            <CheckCircleOutlined />
                          ) : migrationStatus.status === 'failed' ? (
                            <CloseCircleOutlined />
                          ) : null
                        }
                      />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card>
                      <Statistic
                        title="总文件数"
                        value={migrationStatus.total_files || 0}
                        prefix={<FileOutlined />}
                      />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card>
                      <Statistic
                        title="已迁移"
                        value={migrationStatus.migrated_files || 0}
                        valueStyle={{ color: '#52c41a' }}
                        prefix={<CheckCircleOutlined />}
                      />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card>
                      <Statistic
                        title="失败/跳过"
                        value={
                          (migrationStatus.failed_files || 0) +
                          (migrationStatus.skipped_files || 0)
                        }
                        valueStyle={{
                          color: migrationStatus.failed_files
                            ? '#ff4d4f'
                            : '#999',
                        }}
                        prefix={<CloseCircleOutlined />}
                      />
                    </Card>
                  </Col>
                </Row>

                {/* 进度条 */}
                {migrationStatus.status === 'running' && (
                  <Card style={{ marginBottom: 16 }}>
                    <Progress
                      percent={Math.round(migrationStatus.progress || 0)}
                      status="active"
                      format={(percent) => `${percent}%`}
                    />
                    <Text type="secondary">
                      当前处理: {migrationStatus.current_file}
                    </Text>
                  </Card>
                )}

                {/* 筛选条件 */}
                <Card title="迁移选项" style={{ marginBottom: 16 }}>
                  <Space size="large">
                    <div>
                      <Text>按车型筛选：</Text>
                      <Select
                        style={{ width: 200 }}
                        placeholder="全部车型"
                        allowClear
                        value={selectedVehicleModel}
                        onChange={setSelectedVehicleModel}
                      >
                        {vehicleModels.map((model: any) => (
                          <Select.Option key={model.id} value={model.id}>
                            {model.name}
                          </Select.Option>
                        ))}
                      </Select>
                    </div>
                    <div>
                      <Text>按生产线筛选：</Text>
                      <Select
                        style={{ width: 200 }}
                        placeholder="全部生产线"
                        allowClear
                        value={selectedProductionLine}
                        onChange={setSelectedProductionLine}
                      >
                        {productionLines.map((line: any) => (
                          <Select.Option key={line.id} value={line.id}>
                            {line.name}
                          </Select.Option>
                        ))}
                      </Select>
                    </div>
                  </Space>
                </Card>

                {/* 操作按钮 */}
                <Card title="迁移操作">
                  <Space size="large">
                    <Button
                      icon={<EyeOutlined />}
                      onClick={loadMigrationPreview}
                    >
                      预览迁移
                    </Button>
                    <Button
                      type="primary"
                      icon={<SwapOutlined />}
                      onClick={startMigration}
                      disabled={migrationStatus.status === 'running'}
                      loading={operationLoading === 'migration-start'}
                    >
                      开始迁移
                    </Button>
                    <Button
                      danger
                      icon={<ReloadOutlined />}
                      onClick={rollbackMigration}
                      disabled={
                        migrationStatus.status === 'running' ||
                        migrationStatus.status === 'not_started'
                      }
                      loading={operationLoading === 'migration-rollback'}
                    >
                      回滚迁移
                    </Button>
                  </Space>

                  {/* 失败列表 */}
                  {migrationStatus.failed_list &&
                    migrationStatus.failed_list.length > 0 && (
                      <Collapse style={{ marginTop: 16 }}>
                        <Panel
                          header={`失败文件列表 (${migrationStatus.failed_list.length})`}
                          key="1"
                        >
                          <ul>
                            {migrationStatus.failed_list.map((item, index) => (
                              <li key={index}>
                                <Text type="danger">{item}</Text>
                              </li>
                            ))}
                          </ul>
                        </Panel>
                      </Collapse>
                    )}
                </Card>

                <Card title="文件完整性检查" style={{ marginTop: 16 }}>
                  <Space style={{ marginBottom: 16 }}>
                    <Button icon={<EyeOutlined />} onClick={checkFileIntegrity}>
                      检查完整性
                    </Button>
                    <Button danger onClick={cleanupMissingFiles} disabled={!integrityResult || integrityResult.missing_count === 0}>
                      清理缺失记录
                    </Button>
                  </Space>
                  {integrityResult && (
                    <>
                      <Alert
                        type={integrityResult.missing_count > 0 ? 'warning' : 'success'}
                        showIcon
                        message={`共检查 ${integrityResult.checked_count} 条文件记录，缺失 ${integrityResult.missing_count} 条`}
                        style={{ marginBottom: 16 }}
                      />
                      {integrityResult.missing_count > 0 && (
                        <Table
                          className="custom-table"
                          rowKey="id"
                          pagination={false}
                          dataSource={integrityResult.missing_files}
                          columns={[
                            { title: '文件名', dataIndex: 'file_name', key: 'file_name' },
                            { title: '版本', dataIndex: 'version', key: 'version' },
                            { title: '文件路径', dataIndex: 'file_path', key: 'file_path' },
                            { title: '程序ID', dataIndex: 'program_id', key: 'program_id' },
                          ]}
                        />
                      )}
                    </>
                  )}
                </Card>
              </>
            ),
          },
          {
            key: 'backup',
            label: (
              <span>
                <DatabaseOutlined />
                备份恢复
              </span>
            ),
            children: (
              <>
                {/* 备份操作区域 */}
                <Card
                  title="备份操作"
                  extra={
                    <Button onClick={loadBackups} loading={loading}>
                      刷新列表
                    </Button>
                  }
                >
                  <Space size="large" wrap>
                    <Button
                      type="primary"
                      icon={<DatabaseOutlined />}
                      onClick={() => createBackup('database')}
                      loading={operationLoading === 'database'}
                    >
                      备份数据库
                    </Button>
                    <Button
                      type="primary"
                      icon={<FileOutlined />}
                      onClick={() => createBackup('files')}
                      loading={operationLoading === 'files'}
                    >
                      备份文件系统
                    </Button>
                    <Button
                      type="primary"
                      danger
                      icon={<CloudDownloadOutlined />}
                      onClick={() => createBackup('full')}
                      loading={operationLoading === 'full'}
                    >
                      创建完整备份
                    </Button>
                  </Space>

                  <Divider />

                  <Alert
                    message="备份说明"
                    description={
                      <ul>
                        <li>数据库备份：导出MySQL数据库</li>
                        <li>文件系统备份：压缩uploads目录下的所有文件</li>
                        <li>完整备份：包含数据库和文件系统的完整备份</li>
                        <li>恢复操作会自动创建回滚点，请谨慎操作</li>
                      </ul>
                    }
                    type="info"
                    showIcon
                  />
                </Card>

                {/* 备份列表 */}
                <Card
                  title="备份列表"
                  extra={
                    <Text type="secondary">共 {backups.length} 个备份文件</Text>
                  }
                  style={{ marginTop: '16px' }}
                  className="management-table-card"
                >
                  <Table
                    className="custom-table"
                    columns={columns}
                    dataSource={backups}
                    rowKey="name"
                    loading={loading}
                    pagination={{
                      pageSize: 10,
                      showSizeChanger: true,
                      showQuickJumper: true,
                      showTotal: (total) => `共 ${total} 条记录`,
                    }}
                  />
                </Card>
              </>
            ),
          },
          {
            key: 'departments',
            label: (
              <span>
                <TeamOutlined />
                部门管理
              </span>
            ),
            children: (
              <Card
                title="部门管理"
                extra={
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleAddDepartment}
                  >
                    新建部门
                  </Button>
                }
                className="management-table-card"
              >
                <Table
                  className="custom-table"
                  columns={departmentColumns}
                  dataSource={departments}
                  rowKey="id"
                  loading={departmentLoading}
                />
              </Card>
            ),
          },
        ]}
      />

      {/* 恢复确认对话框 */}
      <Modal
        title={
          <span>
            <ExclamationCircleOutlined
              style={{ color: '#faad14', marginRight: 8 }}
            />
            确认恢复操作
          </span>
        }
        open={confirmModalVisible}
        onOk={executeRestore}
        onCancel={() => {
          setConfirmModalVisible(false);
          setCurrentOperation(null);
        }}
        confirmLoading={operationLoading?.startsWith('restore-')}
        okText="确认恢复"
        cancelText="取消"
      >
        <div>
          <Alert
            message="警告"
            description={
              currentOperation && (
                <div>
                  <p>您即将恢复备份文件：{currentOperation.backupName}</p>
                  <p>
                    恢复类型：
                    {currentOperation.type === 'database'
                      ? '数据库'
                      : '文件系统'}
                  </p>
                  <p>此操作会覆盖当前数据，系统会自动创建回滚备份。</p>
                  <p style={{ color: '#ff4d4f', fontWeight: 'bold' }}>
                    请确认您已保存当前工作，并了解此操作的风险！
                  </p>
                </div>
              )
            }
            type="warning"
            showIcon
          />
        </div>
      </Modal>

      {/* 部门管理对话框 */}
      <Modal
        title={currentDepartment ? '编辑部门' : '新建部门'}
        open={departmentModalVisible}
        onCancel={() => setDepartmentModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleDepartmentSubmit}>
          <Form.Item
            name="name"
            label="部门名称"
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select>
              <Select.Option value="active">启用</Select.Option>
              <Select.Option value="inactive">禁用</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 迁移预览对话框 */}
      <Modal
        title="迁移预览"
        open={previewModalVisible}
        onCancel={() => setPreviewModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setPreviewModalVisible(false)}>
            关闭
          </Button>,
          <Button
            key="start"
            type="primary"
            onClick={() => {
              setPreviewModalVisible(false);
              startMigration();
            }}
            disabled={
              migrationStatus.status === 'running' ||
              !migrationPreview?.need_migrate
            }
          >
            开始迁移
          </Button>,
        ]}
        width={800}
      >
        {migrationPreview && (
          <>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Statistic
                  title="总文件数"
                  value={migrationPreview.total_files}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="待迁移"
                  value={migrationPreview.need_migrate}
                  valueStyle={{ color: '#faad14' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="已迁移"
                  value={migrationPreview.already_migrated}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Col>
            </Row>
            <Table
              className="custom-table"
              columns={previewColumns}
              dataSource={migrationPreview.files}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 5 }}
              scroll={{ y: 300 }}
            />
          </>
        )}
      </Modal>
    </div>
  );
};

export default SystemManagement;
