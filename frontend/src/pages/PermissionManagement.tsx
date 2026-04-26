import { useCallback, useEffect, useMemo, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import {
  Button,
  ConfigProvider,
  Select,
  Space,
  Switch,
  Tabs,
  Table,
  Typography,
  message,
} from 'antd';
import type { TableColumnsType } from 'antd';
import {
  ApartmentOutlined,
  ReloadOutlined,
  SaveOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;

type PermissionBit = 'can_view' | 'can_download' | 'can_upload' | 'can_manage';

type PermissionMatrixItem = {
  production_line_id: number;
  production_line_name: string;
  can_view: boolean;
  can_download: boolean;
  can_upload: boolean;
  can_manage: boolean;
  source?: string;
};

type User = {
  id: number;
  name: string;
  employee_id?: string;
  role?: string;
  department?: { name?: string };
};

type Department = {
  id: number;
  name: string;
};

type ProductionLine = {
  id: number;
  name: string;
};

const permissionBits: Array<{ key: PermissionBit; label: string }> = [
  { key: 'can_view', label: '查看' },
  { key: 'can_download', label: '下载' },
  { key: 'can_upload', label: '上传' },
  { key: 'can_manage', label: '管理' },
];

const sourceLabels: Record<string, string> = {
  user: '用户',
  department: '部门',
  role_default: '角色默认',
  department_default: '部门默认',
  none: '无',
};

const toMatrixPayload = (rows: PermissionMatrixItem[]) => ({
  permissions: rows.map((row) => ({
    production_line_id: row.production_line_id,
    can_view: row.can_view,
    can_download: row.can_download,
    can_upload: row.can_upload,
    can_manage: row.can_manage,
  })),
});

const PermissionManagement = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [productionLines, setProductionLines] = useState<ProductionLine[]>([]);

  const [selectedUserID, setSelectedUserID] = useState<number>();
  const [selectedDepartmentID, setSelectedDepartmentID] = useState<number>();
  const [selectedRole, setSelectedRole] = useState<string>();
  const [selectedDefaultDepartmentID, setSelectedDefaultDepartmentID] =
    useState<number>();

  const [userMatrix, setUserMatrix] = useState<PermissionMatrixItem[]>([]);
  const [departmentMatrix, setDepartmentMatrix] = useState<
    PermissionMatrixItem[]
  >([]);
  const [roleDefaultMatrix, setRoleDefaultMatrix] = useState<
    PermissionMatrixItem[]
  >([]);
  const [departmentDefaultMatrix, setDepartmentDefaultMatrix] = useState<
    PermissionMatrixItem[]
  >([]);

  const [baseLoading, setBaseLoading] = useState(false);
  const [userLoading, setUserLoading] = useState(false);
  const [departmentLoading, setDepartmentLoading] = useState(false);
  const [roleDefaultLoading, setRoleDefaultLoading] = useState(false);
  const [departmentDefaultLoading, setDepartmentDefaultLoading] =
    useState(false);
  const [userSaving, setUserSaving] = useState(false);
  const [departmentSaving, setDepartmentSaving] = useState(false);
  const [roleDefaultSaving, setRoleDefaultSaving] = useState(false);
  const [departmentDefaultSaving, setDepartmentDefaultSaving] = useState(false);

  const roleOptions = useMemo(() => {
    const roles = users
      .map((user) => user.role?.trim())
      .filter((role): role is string => Boolean(role));
    return Array.from(new Set(['admin', 'user', ...roles]));
  }, [users]);

  const loadBaseData = useCallback(async () => {
    setBaseLoading(true);
    try {
      const [usersRes, departmentsRes, linesRes] = await Promise.all([
        api.get('/users'),
        api.get('/departments'),
        api.get('/production-lines'),
      ]);

      const loadedUsers = Array.isArray(usersRes.data) ? usersRes.data : [];
      const loadedDepartments = Array.isArray(departmentsRes.data)
        ? departmentsRes.data
        : [];
      const loadedLines = Array.isArray(linesRes.data) ? linesRes.data : [];

      setUsers(loadedUsers);
      setDepartments(loadedDepartments);
      setProductionLines(loadedLines);
      setSelectedUserID((current) => current ?? loadedUsers[0]?.id);
      setSelectedDepartmentID((current) => current ?? loadedDepartments[0]?.id);
      setSelectedDefaultDepartmentID(
        (current) => current ?? loadedDepartments[0]?.id,
      );
      setSelectedRole((current) => current ?? loadedUsers[0]?.role ?? 'admin');
    } catch (error) {
      console.error('Failed to load permission base data:', error);
      message.error('基础数据加载失败');
    } finally {
      setBaseLoading(false);
    }
  }, []);

  const loadMatrix = useCallback(
    async (
      url: string,
      setRows: (rows: PermissionMatrixItem[]) => void,
      setLoading: (loading: boolean) => void,
    ) => {
      setLoading(true);
      try {
        const response = await api.get(url);
        setRows(Array.isArray(response.data?.items) ? response.data.items : []);
      } catch (error) {
        console.error(`Failed to load permission matrix ${url}:`, error);
        message.error('权限矩阵加载失败');
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    loadBaseData();
  }, [loadBaseData]);

  useEffect(() => {
    if (selectedUserID) {
      loadMatrix(
        `/permissions/user/${selectedUserID}/matrix`,
        setUserMatrix,
        setUserLoading,
      );
    }
  }, [loadMatrix, selectedUserID]);

  useEffect(() => {
    if (selectedDepartmentID) {
      loadMatrix(
        `/department-permissions/department/${selectedDepartmentID}/matrix`,
        setDepartmentMatrix,
        setDepartmentLoading,
      );
    }
  }, [loadMatrix, selectedDepartmentID]);

  useEffect(() => {
    if (selectedRole) {
      loadMatrix(
        `/permission-defaults/roles/${encodeURIComponent(selectedRole)}/matrix`,
        setRoleDefaultMatrix,
        setRoleDefaultLoading,
      );
    }
  }, [loadMatrix, selectedRole]);

  useEffect(() => {
    if (selectedDefaultDepartmentID) {
      loadMatrix(
        `/permission-defaults/departments/${selectedDefaultDepartmentID}/matrix`,
        setDepartmentDefaultMatrix,
        setDepartmentDefaultLoading,
      );
    }
  }, [loadMatrix, selectedDefaultDepartmentID]);

  const updateMatrixBit = (
    setRows: Dispatch<SetStateAction<PermissionMatrixItem[]>>,
    productionLineID: number,
    bit: PermissionBit,
    checked: boolean,
  ) => {
    setRows((rows) =>
      rows.map((row) =>
        row.production_line_id === productionLineID
          ? { ...row, [bit]: checked }
          : row,
      ),
    );
  };

  const saveMatrix = async (
    url: string,
    rows: PermissionMatrixItem[],
    setSaving: (saving: boolean) => void,
    reload: () => void,
  ) => {
    setSaving(true);
    try {
      await api.put(url, toMatrixPayload(rows));
      message.success('权限矩阵已保存');
      reload();
    } catch (error) {
      console.error(`Failed to save permission matrix ${url}:`, error);
      message.error('权限矩阵保存失败');
    } finally {
      setSaving(false);
    }
  };

  const buildColumns = (
    setRows: Dispatch<SetStateAction<PermissionMatrixItem[]>>,
  ): TableColumnsType<PermissionMatrixItem> => [
    {
      title: '生产线',
      dataIndex: 'production_line_name',
      key: 'production_line_name',
      width: 220,
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 120,
      render: (source?: string) => sourceLabels[source || 'none'] || source,
    },
    ...permissionBits.map((bit) => ({
      title: bit.label,
      dataIndex: bit.key,
      key: bit.key,
      width: 110,
      render: (value: boolean, record: PermissionMatrixItem) => (
        <Switch
          aria-label={`${record.production_line_name} ${bit.label}`}
          checked={value}
          onChange={(checked) =>
            updateMatrixBit(setRows, record.production_line_id, bit.key, checked)
          }
        />
      ),
    })),
  ];

  const renderToolbar = (
    selector: React.ReactNode,
    rows: PermissionMatrixItem[],
    loading: boolean,
    saving: boolean,
    onReload: () => void,
    onSave: () => void,
  ) => (
    <Space
      wrap
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        marginBottom: 16,
        width: '100%',
      }}
    >
      <Space wrap>
        {selector}
        <span style={{ color: '#5A6062' }}>
          共 {productionLines.length} 条产线
        </span>
      </Space>
      <Space>
        <Button icon={<ReloadOutlined />} onClick={onReload} loading={loading}>
          重载
        </Button>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          onClick={onSave}
          loading={saving}
          disabled={rows.length === 0}
        >
          保存
        </Button>
      </Space>
    </Space>
  );

  const renderMatrix = (
    rows: PermissionMatrixItem[],
    setRows: Dispatch<SetStateAction<PermissionMatrixItem[]>>,
    loading: boolean,
  ) => (
    <Table
      columns={buildColumns(setRows)}
      dataSource={rows}
      rowKey="production_line_id"
      loading={loading || baseLoading}
      pagination={false}
      className="custom-table"
      scroll={{ x: 760 }}
    />
  );

  return (
    <div className="management-page">
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>系统</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>
              /
            </span>
            <span className="active">权限管理</span>
          </div>
          <Title level={2} className="management-page-title">
            权限管理
          </Title>
        </div>
      </div>

      <ConfigProvider
        theme={{
          components: {
            Select: {
              controlHeight: 36,
              borderRadius: 6,
            },
          },
        }}
      >
        <div className="management-table-card">
          <Tabs
            items={[
              {
                key: 'user_matrix',
                label: (
                  <span>
                    <UserOutlined />
                    用户权限矩阵
                  </span>
                ),
                children: (
                  <>
                    {renderToolbar(
                      <Select
                        aria-label="选择用户"
                        data-testid="user-matrix-select"
                        style={{ width: 280 }}
                        placeholder="选择用户"
                        value={selectedUserID}
                        onChange={setSelectedUserID}
                        options={users.map((user) => ({
                          value: user.id,
                          label: `${user.name}${
                            user.employee_id ? ` (${user.employee_id})` : ''
                          }${
                            user.department?.name
                              ? ` - ${user.department.name}`
                              : ''
                          }`,
                        }))}
                      />,
                      userMatrix,
                      userLoading,
                      userSaving,
                      () => {
                        if (selectedUserID) {
                          loadMatrix(
                            `/permissions/user/${selectedUserID}/matrix`,
                            setUserMatrix,
                            setUserLoading,
                          );
                        }
                      },
                      () => {
                        if (selectedUserID) {
                          saveMatrix(
                            `/permissions/user/${selectedUserID}/matrix`,
                            userMatrix,
                            setUserSaving,
                            () =>
                              loadMatrix(
                                `/permissions/user/${selectedUserID}/matrix`,
                                setUserMatrix,
                                setUserLoading,
                              ),
                          );
                        }
                      },
                    )}
                    {renderMatrix(userMatrix, setUserMatrix, userLoading)}
                  </>
                ),
              },
              {
                key: 'department_matrix',
                label: (
                  <span>
                    <TeamOutlined />
                    部门权限矩阵
                  </span>
                ),
                children: (
                  <>
                    {renderToolbar(
                      <Select
                        aria-label="选择部门"
                        data-testid="department-matrix-select"
                        style={{ width: 260 }}
                        placeholder="选择部门"
                        value={selectedDepartmentID}
                        onChange={setSelectedDepartmentID}
                        options={departments.map((department) => ({
                          value: department.id,
                          label: department.name,
                        }))}
                      />,
                      departmentMatrix,
                      departmentLoading,
                      departmentSaving,
                      () => {
                        if (selectedDepartmentID) {
                          loadMatrix(
                            `/department-permissions/department/${selectedDepartmentID}/matrix`,
                            setDepartmentMatrix,
                            setDepartmentLoading,
                          );
                        }
                      },
                      () => {
                        if (selectedDepartmentID) {
                          saveMatrix(
                            `/department-permissions/department/${selectedDepartmentID}/matrix`,
                            departmentMatrix,
                            setDepartmentSaving,
                            () =>
                              loadMatrix(
                                `/department-permissions/department/${selectedDepartmentID}/matrix`,
                                setDepartmentMatrix,
                                setDepartmentLoading,
                              ),
                          );
                        }
                      },
                    )}
                    {renderMatrix(
                      departmentMatrix,
                      setDepartmentMatrix,
                      departmentLoading,
                    )}
                  </>
                ),
              },
              {
                key: 'role_defaults',
                label: (
                  <span>
                    <SafetyCertificateOutlined />
                    角色默认权限
                  </span>
                ),
                children: (
                  <>
                    {renderToolbar(
                      <Select
                        aria-label="选择角色"
                        data-testid="role-default-select"
                        style={{ width: 220 }}
                        placeholder="选择角色"
                        value={selectedRole}
                        onChange={setSelectedRole}
                        options={roleOptions.map((role) => ({
                          value: role,
                          label: role,
                        }))}
                      />,
                      roleDefaultMatrix,
                      roleDefaultLoading,
                      roleDefaultSaving,
                      () => {
                        if (selectedRole) {
                          loadMatrix(
                            `/permission-defaults/roles/${encodeURIComponent(
                              selectedRole,
                            )}/matrix`,
                            setRoleDefaultMatrix,
                            setRoleDefaultLoading,
                          );
                        }
                      },
                      () => {
                        if (selectedRole) {
                          saveMatrix(
                            `/permission-defaults/roles/${encodeURIComponent(
                              selectedRole,
                            )}/matrix`,
                            roleDefaultMatrix,
                            setRoleDefaultSaving,
                            () =>
                              loadMatrix(
                                `/permission-defaults/roles/${encodeURIComponent(
                                  selectedRole,
                                )}/matrix`,
                                setRoleDefaultMatrix,
                                setRoleDefaultLoading,
                              ),
                          );
                        }
                      },
                    )}
                    {renderMatrix(
                      roleDefaultMatrix,
                      setRoleDefaultMatrix,
                      roleDefaultLoading,
                    )}
                  </>
                ),
              },
              {
                key: 'department_defaults',
                label: (
                  <span>
                    <ApartmentOutlined />
                    部门默认权限
                  </span>
                ),
                children: (
                  <>
                    {renderToolbar(
                      <Select
                        aria-label="选择默认部门"
                        data-testid="department-default-select"
                        style={{ width: 260 }}
                        placeholder="选择部门"
                        value={selectedDefaultDepartmentID}
                        onChange={setSelectedDefaultDepartmentID}
                        options={departments.map((department) => ({
                          value: department.id,
                          label: department.name,
                        }))}
                      />,
                      departmentDefaultMatrix,
                      departmentDefaultLoading,
                      departmentDefaultSaving,
                      () => {
                        if (selectedDefaultDepartmentID) {
                          loadMatrix(
                            `/permission-defaults/departments/${selectedDefaultDepartmentID}/matrix`,
                            setDepartmentDefaultMatrix,
                            setDepartmentDefaultLoading,
                          );
                        }
                      },
                      () => {
                        if (selectedDefaultDepartmentID) {
                          saveMatrix(
                            `/permission-defaults/departments/${selectedDefaultDepartmentID}/matrix`,
                            departmentDefaultMatrix,
                            setDepartmentDefaultSaving,
                            () =>
                              loadMatrix(
                                `/permission-defaults/departments/${selectedDefaultDepartmentID}/matrix`,
                                setDepartmentDefaultMatrix,
                                setDepartmentDefaultLoading,
                              ),
                          );
                        }
                      },
                    )}
                    {renderMatrix(
                      departmentDefaultMatrix,
                      setDepartmentDefaultMatrix,
                      departmentDefaultLoading,
                    )}
                  </>
                ),
              },
            ]}
          />
        </div>
      </ConfigProvider>
    </div>
  );
};

export default PermissionManagement;
