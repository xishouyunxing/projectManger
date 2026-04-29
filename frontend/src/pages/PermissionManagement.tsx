import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  message,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { TableColumnsType } from 'antd';
import {
  ApartmentOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;

// ---------- Types ----------

type Role = {
  id: number;
  name: string;
  display_name?: string;
  description?: string;
  is_preset: boolean;
  is_system: boolean;
  sort_order: number;
};

type PermissionDef = {
  id: number;
  code: string;
  name: string;
  type: string;
  resource?: string;
};

type LinePermRow = {
  production_line_id: number;
  production_line_name: string;
  can_view: boolean;
  can_download: boolean;
  can_upload: boolean;
  can_manage: boolean;
};

type User = {
  id: number;
  name: string;
  employee_id?: string;
  role?: string;
  department?: { name?: string };
};

type UserMatrixRow = {
  production_line_id: number;
  production_line_name: string;
  can_view: boolean;
  can_download: boolean;
  can_upload: boolean;
  can_manage: boolean;
  source?: string;
  override?: boolean;
  dirty?: boolean;
};

type LineAdminAssignment = {
  id: number;
  user_id: number;
  user_name?: string;
  user_employee_id?: string;
  production_line_id: number;
  production_line_name?: string;
};

// ---------- Tab 1: Role Management ----------

const RoleManagementTab = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [allPermissions, setAllPermissions] = useState<PermissionDef[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [rolePermIds, setRolePermIds] = useState<Set<number>>(new Set());
  const [linePermRows, setLinePermRows] = useState<LinePermRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [savingFunc, setSavingFunc] = useState(false);
  const [savingLine, setSavingLine] = useState(false);

  const loadRoles = useCallback(async () => {
    try {
      const res = await api.get('/roles');
      setRoles(Array.isArray(res.data) ? res.data : []);
    } catch {
      message.error('加载角色列表失败');
    }
  }, []);

  const loadAllPermissions = useCallback(async () => {
    try {
      const res = await api.get('/permission-definitions');
      setAllPermissions(Array.isArray(res.data) ? res.data : []);
    } catch {
      // silent
    }
  }, []);

  useEffect(() => {
    loadRoles();
    loadAllPermissions();
  }, [loadRoles, loadAllPermissions]);

  const loadRoleDetail = useCallback(async (roleId: number) => {
    setLoading(true);
    try {
      const [detailRes, linePermRes] = await Promise.all([
        api.get(`/roles/${roleId}`),
        api.get(`/roles/${roleId}/permissions`),
      ]);
      const perms = detailRes.data?.permissions || [];
      setRolePermIds(new Set(perms.map((p: PermissionDef) => p.id)));

      const lines = linePermRes.data?.permissions || [];
      setLinePermRows(lines);
    } catch {
      message.error('加载角色详情失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleRoleClick = (role: Role) => {
    setSelectedRole(role);
    loadRoleDetail(role.id);
  };

  const handleFuncPermToggle = (permId: number, checked: boolean) => {
    setRolePermIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(permId);
      else next.delete(permId);
      return next;
    });
  };

  const handleSaveFuncPerms = async () => {
    if (!selectedRole) return;
    setSavingFunc(true);
    try {
      await api.put(`/roles/${selectedRole.id}/function-permissions`, {
        permission_ids: Array.from(rolePermIds),
      });
      message.success('功能权限保存成功');
    } catch {
      message.error('功能权限保存失败');
    } finally {
      setSavingFunc(false);
    }
  };

  const handleLinePermChange = (
    lineId: number,
    field: keyof LinePermRow,
    value: boolean,
  ) => {
    setLinePermRows((rows) =>
      rows.map((row) =>
        row.production_line_id === lineId ? { ...row, [field]: value } : row,
      ),
    );
  };

  const handleSaveLinePerms = async () => {
    if (!selectedRole) return;
    setSavingLine(true);
    try {
      await api.put(`/roles/${selectedRole.id}/permissions`, {
        permissions: linePermRows.map((row) => ({
          production_line_id: row.production_line_id,
          can_view: row.can_view,
          can_download: row.can_download,
          can_upload: row.can_upload,
          can_manage: row.can_manage,
        })),
      });
      message.success('产线权限保存成功');
    } catch {
      message.error('产线权限保存失败');
    } finally {
      setSavingLine(false);
    }
  };

  const pagePermissions = allPermissions.filter((p) => p.type === 'page');
  const opPermissions = allPermissions.filter((p) => p.type === 'operation');

  const lineColumns: TableColumnsType<LinePermRow> = [
    {
      title: '生产线',
      dataIndex: 'production_line_name',
      key: 'name',
      width: 180,
      render: (text: string) => <strong>{text}</strong>,
    },
    ...(['can_view', 'can_download', 'can_upload', 'can_manage'] as const).map(
      (field) => ({
        title: { can_view: '查看', can_download: '下载', can_upload: '上传', can_manage: '管理' }[field],
        dataIndex: field,
        key: field,
        width: 80,
        render: (val: boolean, record: LinePermRow) => (
          <Switch
            size="small"
            checked={val}
            onChange={(checked) =>
              handleLinePermChange(record.production_line_id, field, checked)
            }
          />
        ),
      }),
    ),
  ];

  return (
    <div style={{ display: 'flex', gap: 24, minHeight: 500 }}>
      {/* Left: Role List */}
      <div
        style={{
          width: 240,
          borderRight: '1px solid #f0f0f0',
          paddingRight: 16,
        }}
      >
        <Title level={5} style={{ marginBottom: 12 }}>
          角色列表
        </Title>
        {roles.map((role) => (
          <div
            key={role.id}
            onClick={() => handleRoleClick(role)}
            style={{
              padding: '8px 12px',
              cursor: 'pointer',
              borderRadius: 6,
              marginBottom: 4,
              background:
                selectedRole?.id === role.id ? '#e8f3ff' : 'transparent',
              color: selectedRole?.id === role.id ? '#1677ff' : '#1d2129',
              fontWeight: selectedRole?.id === role.id ? 600 : 400,
            }}
          >
            {role.display_name || role.name}
            {role.is_preset && (
              <Tag color="blue" style={{ marginLeft: 8, fontSize: 11 }}>
                预设
              </Tag>
            )}
          </div>
        ))}
      </div>

      {/* Right: Permission Config */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {!selectedRole ? (
          <div style={{ color: '#999', padding: 40, textAlign: 'center' }}>
            请从左侧选择角色
          </div>
        ) : (
          <>
            {/* Function Permissions */}
            <div style={{ marginBottom: 24 }}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 12,
                }}
              >
                <Title level={5} style={{ margin: 0 }}>
                  功能权限 — {selectedRole.display_name || selectedRole.name}
                </Title>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  size="small"
                  onClick={handleSaveFuncPerms}
                  loading={savingFunc}
                >
                  保存功能权限
                </Button>
              </div>

              {pagePermissions.length > 0 && (
                <div style={{ marginBottom: 12 }}>
                  <strong style={{ fontSize: 13, color: '#666' }}>
                    页面权限
                  </strong>
                  <div
                    style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 8,
                      marginTop: 6,
                    }}
                  >
                    {pagePermissions.map((perm) => (
                      <Checkbox
                        key={perm.id}
                        checked={rolePermIds.has(perm.id)}
                        onChange={(e) =>
                          handleFuncPermToggle(perm.id, e.target.checked)
                        }
                      >
                        {perm.name}
                      </Checkbox>
                    ))}
                  </div>
                </div>
              )}

              {opPermissions.length > 0 && (
                <div>
                  <strong style={{ fontSize: 13, color: '#666' }}>
                    操作权限
                  </strong>
                  <div
                    style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 8,
                      marginTop: 6,
                    }}
                  >
                    {opPermissions.map((perm) => (
                      <Checkbox
                        key={perm.id}
                        checked={rolePermIds.has(perm.id)}
                        onChange={(e) =>
                          handleFuncPermToggle(perm.id, e.target.checked)
                        }
                      >
                        {perm.name}
                      </Checkbox>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Line Permissions Matrix */}
            <div>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 12,
                }}
              >
                <Title level={5} style={{ margin: 0 }}>
                  产线权限
                </Title>
                <Space>
                  <Button
                    icon={<ReloadOutlined />}
                    size="small"
                    onClick={() => loadRoleDetail(selectedRole.id)}
                    loading={loading}
                  >
                    重载
                  </Button>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    size="small"
                    onClick={handleSaveLinePerms}
                    loading={savingLine}
                  >
                    保存产线权限
                  </Button>
                </Space>
              </div>
              <Table
                columns={lineColumns}
                dataSource={linePermRows}
                rowKey="production_line_id"
                size="small"
                pagination={false}
                loading={loading}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

// ---------- Tab 2: User Permissions ----------

const UserPermissionsTab = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [productionLines, setProductionLines] = useState<
    { id: number; name: string }[]
  >([]);
  const [selectedUserId, setSelectedUserId] = useState<number>();
  const [matrixRows, setMatrixRows] = useState<UserMatrixRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadBaseData = useCallback(async () => {
    try {
      const [usersRes, linesRes] = await Promise.all([
        api.get('/users'),
        api.get('/production-lines'),
      ]);
      setUsers(Array.isArray(usersRes.data) ? usersRes.data : []);
      setProductionLines(Array.isArray(linesRes.data) ? linesRes.data : []);
    } catch {
      message.error('加载基础数据失败');
    }
  }, []);

  useEffect(() => {
    loadBaseData();
  }, [loadBaseData]);

  const loadMatrix = useCallback(async (userId: number) => {
    setLoading(true);
    try {
      const res = await api.get(`/permissions/user/${userId}/matrix`);
      const items = Array.isArray(res.data?.items) ? res.data.items : [];
      setMatrixRows(
        items.map((row: UserMatrixRow) => ({
          ...row,
          override: row.override ?? row.source !== 'none',
          dirty: false,
        })),
      );
    } catch {
      message.error('加载用户权限矩阵失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (selectedUserId) loadMatrix(selectedUserId);
  }, [selectedUserId, loadMatrix]);

  const handleBitChange = (
    lineId: number,
    field: 'can_view' | 'can_download' | 'can_upload' | 'can_manage',
    value: boolean,
  ) => {
    setMatrixRows((rows) =>
      rows.map((row) =>
        row.production_line_id === lineId
          ? { ...row, [field]: value, override: true, dirty: true }
          : row,
      ),
    );
  };

  const handleOverrideChange = (lineId: number, checked: boolean) => {
    setMatrixRows((rows) =>
      rows.map((row) => {
        if (row.production_line_id !== lineId) return row;
        if (checked) {
          return { ...row, override: true, dirty: true };
        }
        return {
          ...row,
          override: false,
          can_view: false,
          can_download: false,
          can_upload: false,
          can_manage: false,
          dirty: true,
        };
      }),
    );
  };

  const handleSave = async () => {
    if (!selectedUserId) return;
    setSaving(true);
    try {
      const dirtyRows = matrixRows.filter((r) => r.dirty);
      await api.put(`/permissions/user/${selectedUserId}/matrix`, {
        permissions: dirtyRows.map((row) => ({
          production_line_id: row.production_line_id,
          inherit: !row.override,
          can_view: row.can_view,
          can_download: row.can_download,
          can_upload: row.can_upload,
          can_manage: row.can_manage,
        })),
      });
      message.success('权限保存成功');
      loadMatrix(selectedUserId);
    } catch {
      message.error('权限保存失败');
    } finally {
      setSaving(false);
    }
  };

  const sourceTag = (row: UserMatrixRow) => {
    if (row.dirty) {
      return row.override ? (
        <Tag color="orange">显式覆盖</Tag>
      ) : (
        <Tag color="green">继承</Tag>
      );
    }
    const map: Record<string, { color: string; text: string }> = {
      user: { color: 'blue', text: '用户' },
      department: { color: 'cyan', text: '部门' },
      role_default: { color: 'purple', text: '角色默认' },
      department_default: { color: 'geekblue', text: '部门默认' },
      none: { color: 'default', text: '无' },
    };
    const info = map[row.source || 'none'] || { color: 'default', text: row.source || '无' };
    return <Tag color={info.color}>{info.text}</Tag>;
  };

  const columns: TableColumnsType<UserMatrixRow> = [
    {
      title: '生产线',
      dataIndex: 'production_line_name',
      key: 'name',
      width: 180,
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '来源',
      key: 'source',
      width: 100,
      render: (_: unknown, record: UserMatrixRow) => sourceTag(record),
    },
    {
      title: '模式',
      key: 'override',
      width: 100,
      render: (_: unknown, record: UserMatrixRow) => (
        <Switch
          size="small"
          checked={record.override}
          checkedChildren="覆盖"
          unCheckedChildren="继承"
          onChange={(checked) =>
            handleOverrideChange(record.production_line_id, checked)
          }
        />
      ),
    },
    ...([
      { field: 'can_view' as const, label: '查看' },
      { field: 'can_download' as const, label: '下载' },
      { field: 'can_upload' as const, label: '上传' },
      { field: 'can_manage' as const, label: '管理' },
    ].map(({ field, label }) => ({
      title: label,
      dataIndex: field,
      key: field,
      width: 80,
      render: (val: boolean, record: UserMatrixRow) => (
        <Switch
          size="small"
          checked={val}
          disabled={!record.override}
          onChange={(checked) =>
            handleBitChange(record.production_line_id, field, checked)
          }
        />
      ),
    }))),
  ];

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Space>
          <Select
            aria-label="选择用户"
            style={{ width: 280 }}
            placeholder="选择用户"
            value={selectedUserId}
            onChange={setSelectedUserId}
            options={users.map((u) => ({
              value: u.id,
              label: `${u.name}${u.employee_id ? ` (${u.employee_id})` : ''}${u.department?.name ? ` - ${u.department.name}` : ''}`,
            }))}
            showSearch
            filterOption={(input, option) =>
              (option?.label as string)?.includes(input) ?? false
            }
          />
          <span style={{ color: '#999' }}>
            共 {productionLines.length} 条产线
          </span>
        </Space>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => selectedUserId && loadMatrix(selectedUserId)}
            loading={loading}
          >
            重载
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={handleSave}
            loading={saving}
            disabled={!matrixRows.some((r) => r.dirty)}
          >
            保存
          </Button>
        </Space>
      </div>
      <Table
        columns={columns}
        dataSource={matrixRows}
        rowKey="production_line_id"
        size="small"
        pagination={false}
        loading={loading}
      />
    </div>
  );
};

// ---------- Tab 3: Line Admin ----------

const LineAdminTab = () => {
  const [assignments, setAssignments] = useState<LineAdminAssignment[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [productionLines, setProductionLines] = useState<
    { id: number; name: string }[]
  >([]);
  const [loading, setLoading] = useState(false);
  const [addUserId, setAddUserId] = useState<number>();
  const [addLineId, setAddLineId] = useState<number>();

  const loadAssignments = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/line-admin/assignments');
      const data = Array.isArray(res.data) ? res.data : [];
      setAssignments(
        data.map((a: any) => ({
          id: a.id,
          user_id: a.user_id,
          user_name: a.user?.name || a.user_name,
          user_employee_id: a.user?.employee_id || a.user_employee_id,
          production_line_id: a.production_line_id,
          production_line_name:
            a.production_line?.name || a.production_line_name,
        })),
      );
    } catch {
      message.error('加载分配列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const loadBaseData = useCallback(async () => {
    try {
      const [usersRes, linesRes] = await Promise.all([
        api.get('/users'),
        api.get('/production-lines'),
      ]);
      setUsers(
        (Array.isArray(usersRes.data) ? usersRes.data : []).filter(
          (u: User) => u.role === 'line_admin',
        ),
      );
      setProductionLines(Array.isArray(linesRes.data) ? linesRes.data : []);
    } catch {
      // silent
    }
  }, []);

  useEffect(() => {
    loadAssignments();
    loadBaseData();
  }, [loadAssignments, loadBaseData]);

  const handleAdd = async () => {
    if (!addUserId || !addLineId) {
      message.warning('请选择用户和产线');
      return;
    }
    try {
      await api.post('/line-admin/assignments', {
        user_id: addUserId,
        production_line_id: addLineId,
      });
      message.success('分配成功');
      setAddUserId(undefined);
      setAddLineId(undefined);
      loadAssignments();
    } catch (error: any) {
      message.error(error.response?.data?.error || '分配失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/line-admin/assignments/${id}`);
      message.success('已取消分配');
      loadAssignments();
    } catch {
      message.error('取消失败');
    }
  };

  const columns: TableColumnsType<LineAdminAssignment> = [
    {
      title: '用户',
      key: 'user',
      render: (_: unknown, record: LineAdminAssignment) =>
        `${record.user_name || '-'}${record.user_employee_id ? ` (${record.user_employee_id})` : ''}`,
    },
    {
      title: '产线',
      dataIndex: 'production_line_name',
      key: 'line',
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: LineAdminAssignment) => (
        <Button
          type="link"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => handleDelete(record.id)}
        >
          取消
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div
        style={{
          display: 'flex',
          gap: 12,
          marginBottom: 16,
          alignItems: 'center',
        }}
      >
        <Select
          style={{ width: 200 }}
          placeholder="选择产线管理员"
          value={addUserId}
          onChange={setAddUserId}
          options={users.map((u) => ({
            value: u.id,
            label: `${u.name}${u.employee_id ? ` (${u.employee_id})` : ''}`,
          }))}
          showSearch
          filterOption={(input, option) =>
            (option?.label as string)?.includes(input) ?? false
          }
        />
        <Select
          style={{ width: 200 }}
          placeholder="选择产线"
          value={addLineId}
          onChange={setAddLineId}
          options={productionLines.map((l) => ({
            value: l.id,
            label: l.name,
          }))}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加分配
        </Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={loadAssignments}
          loading={loading}
        >
          刷新
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={assignments}
        rowKey="id"
        size="small"
        pagination={false}
        loading={loading}
      />
    </div>
  );
};

// ---------- Main Component ----------

const PermissionManagement = () => {
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

      <div className="management-table-card">
        <Tabs
          items={[
            {
              key: 'roles',
              label: (
                <span>
                  <SafetyCertificateOutlined />
                  角色管理
                </span>
              ),
              children: <RoleManagementTab />,
            },
            {
              key: 'users',
              label: (
                <span>
                  <UserOutlined />
                  用户权限
                </span>
              ),
              children: <UserPermissionsTab />,
            },
            {
              key: 'line_admins',
              label: (
                <span>
                  <ApartmentOutlined />
                  产线管理员
                </span>
              ),
              children: <LineAdminTab />,
            },
          ]}
        />
      </div>
    </div>
  );
};

export default PermissionManagement;
