import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  Modal,
  Select,
  Space,
  Switch,
  Tabs,
  Table,
  Tag,
  Typography,
  message,
  Popconfirm,
  Input,
  Form,
} from 'antd';
import type { TableColumnsType } from 'antd';
import {
  DeleteOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const { Title, Text } = Typography;

// ---------- 类型定义 ----------

type Role = {
  id: number;
  name: string;
  description: string;
  is_preset: boolean;
  is_system: boolean;
  status: string;
  sort_order: number;
};

type Permission = {
  id: number;
  code: string;
  name: string;
  type: string;
  resource: string;
};

type RoleLinePermItem = {
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
  employee_id: string;
  role: string;
  role_id?: number;
  department?: { id: number; name: string };
};

type ProductionLine = {
  id: number;
  name: string;
};

type LineAdminAssignment = {
  id: number;
  user_id: number;
  production_line_id: number;
  user?: User;
  production_line?: ProductionLine;
};

type PermissionBit = 'can_view' | 'can_download' | 'can_upload' | 'can_manage';

const permBitConfig: Array<{ key: PermissionBit; label: string }> = [
  { key: 'can_view', label: '查看' },
  { key: 'can_download', label: '下载' },
  { key: 'can_upload', label: '上传' },
  { key: 'can_manage', label: '管理' },
];

const roleLabels: Record<string, string> = {
  system_admin: '系统管理员',
  line_admin: '产线管理员',
  engineer: '工程师',
  operator: '操作员',
  viewer: '访客',
};

// ---------- 主组件 ----------

const PermissionManagement = () => {
  const { isAdmin } = useAuth();

  return (
    <div style={{ padding: 0 }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        <LockOutlined style={{ marginRight: 8 }} />
        权限管理
      </Title>
      <Tabs
        defaultActiveKey="roles"
        items={[
          {
            key: 'roles',
            label: (
              <span>
                <TeamOutlined /> 角色管理
              </span>
            ),
            children: <RoleTab />,
          },
          {
            key: 'users',
            label: (
              <span>
                <UserOutlined /> 用户权限
              </span>
            ),
            children: <UserPermTab />,
          },
          {
            key: 'line-admin',
            label: (
              <span>
                <LockOutlined /> 产线管理员
              </span>
            ),
            children: <LineAdminTab />,
            disabled: !isAdmin,
          },
        ]}
      />
    </div>
  );
};

// ========== Tab 1: 角色管理 ==========

const RoleTab = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [allPermissions, setAllPermissions] = useState<Permission[]>([]);
  const [rolePermIDs, setRolePermIDs] = useState<Set<number>>(new Set());
  const [lineMatrix, setLineMatrix] = useState<RoleLinePermItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();

  const loadRoles = useCallback(async () => {
    try {
      const res = await api.get('/roles');
      setRoles(Array.isArray(res.data) ? res.data : []);
    } catch {
      message.error('加载角色失败');
    }
  }, []);

  const loadPermissions = useCallback(async () => {
    try {
      const res = await api.get('/permission-definitions');
      setAllPermissions(Array.isArray(res.data) ? res.data : []);
    } catch {
      // ignore
    }
  }, []);

  const loadRoleDetail = useCallback(async (roleId: number) => {
    setLoading(true);
    try {
      const [detailRes, matrixRes] = await Promise.all([
        api.get(`/roles/${roleId}`),
        api.get(`/roles/${roleId}/permissions`),
      ]);

      const perms: Permission[] = detailRes.data?.permissions || [];
      setRolePermIDs(new Set(perms.map((p: Permission) => p.id)));

      const items: RoleLinePermItem[] = matrixRes.data?.permissions || [];
      setLineMatrix(items);
    } catch {
      message.error('加载角色详情失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadRoles();
    loadPermissions();
  }, [loadRoles, loadPermissions]);

  useEffect(() => {
    if (selectedRole) {
      loadRoleDetail(selectedRole.id);
    }
  }, [selectedRole, loadRoleDetail]);

  const handleCreateRole = async (values: { name: string; description: string }) => {
    try {
      await api.post('/roles', values);
      message.success('角色创建成功');
      setCreateModalOpen(false);
      createForm.resetFields();
      loadRoles();
    } catch (error: any) {
      message.error(error.response?.data?.error || '创建失败');
    }
  };

  const handleDeleteRole = async (roleId: number) => {
    try {
      await api.delete(`/roles/${roleId}`);
      message.success('角色已删除');
      if (selectedRole?.id === roleId) {
        setSelectedRole(null);
      }
      loadRoles();
    } catch (error: any) {
      message.error(error.response?.data?.error || '删除失败');
    }
  };

  const handleTogglePerm = (permId: number, checked: boolean) => {
    setRolePermIDs((prev) => {
      const next = new Set(prev);
      if (checked) next.add(permId);
      else next.delete(permId);
      return next;
    });
  };

  const handleSaveFunctionPerms = async () => {
    if (!selectedRole) return;
    setSaving(true);
    try {
      await api.put(`/roles/${selectedRole.id}/function-permissions`, {
        permission_ids: Array.from(rolePermIDs),
      });
      message.success('功能权限已保存');
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleToggleLineBit = (lineId: number, bit: PermissionBit, checked: boolean) => {
    setLineMatrix((rows) =>
      rows.map((row) =>
        row.production_line_id === lineId ? { ...row, [bit]: checked } : row,
      ),
    );
  };

  const handleSaveLinePerms = async () => {
    if (!selectedRole) return;
    setSaving(true);
    try {
      await api.put(`/roles/${selectedRole.id}/permissions`, {
        permissions: lineMatrix.map((row) => ({
          production_line_id: row.production_line_id,
          can_view: row.can_view,
          can_download: row.can_download,
          can_upload: row.can_upload,
          can_manage: row.can_manage,
        })),
      });
      message.success('产线权限已保存');
      loadRoleDetail(selectedRole.id);
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  // 按 resource 分组功能权限
  const groupedPerms = allPermissions.reduce<Record<string, Permission[]>>((acc, p) => {
    const key = p.type === 'page' ? '页面权限' : '操作权限';
    if (!acc[key]) acc[key] = [];
    acc[key].push(p);
    return acc;
  }, {});

  const lineColumns: TableColumnsType<RoleLinePermItem> = [
    { title: '产线', dataIndex: 'production_line_name', width: 200 },
    ...permBitConfig.map((bit) => ({
      title: bit.label,
      dataIndex: bit.key,
      width: 80,
      align: 'center' as const,
      render: (_: boolean, record: RoleLinePermItem) => (
        <Checkbox
          checked={record[bit.key]}
          onChange={(e) => handleToggleLineBit(record.production_line_id, bit.key, e.target.checked)}
        />
      ),
    })),
  ];

  return (
    <div style={{ display: 'flex', gap: 24, minHeight: 500 }}>
      {/* 左侧：角色列表 */}
      <div style={{ width: 240, flexShrink: 0 }}>
        <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text strong>角色列表</Text>
          <Button size="small" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            新建
          </Button>
        </div>
        <div style={{ border: '1px solid #f0f0f0', borderRadius: 8, overflow: 'hidden' }}>
          {roles.map((role) => (
            <div
              key={role.id}
              onClick={() => setSelectedRole(role)}
              style={{
                padding: '10px 12px',
                cursor: 'pointer',
                background: selectedRole?.id === role.id ? '#e6f4ff' : '#fff',
                borderBottom: '1px solid #f0f0f0',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <div>
                <div style={{ fontWeight: selectedRole?.id === role.id ? 600 : 400 }}>
                  {roleLabels[role.name] || role.name}
                </div>
                {role.description && (
                  <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>{role.description}</div>
                )}
              </div>
              {role.is_system ? (
                <Tag color="blue">系统</Tag>
              ) : role.is_preset ? (
                <Tag>预设</Tag>
              ) : (
                <Popconfirm title="确认删除该角色？" onConfirm={() => handleDeleteRole(role.id)}>
                  <DeleteOutlined style={{ color: '#ff4d4f', fontSize: 12 }} />
                </Popconfirm>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* 右侧：权限配置 */}
      <div style={{ flex: 1 }}>
        {!selectedRole ? (
          <div style={{ textAlign: 'center', padding: 80, color: '#999' }}>请从左侧选择一个角色</div>
        ) : (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Text strong style={{ fontSize: 16 }}>
                {roleLabels[selectedRole.name] || selectedRole.name}
              </Text>
              {selectedRole.is_system && <Tag color="blue" style={{ marginLeft: 8 }}>系统角色</Tag>}
            </div>

            {/* 功能权限 */}
            <div style={{ marginBottom: 24, padding: 16, background: '#fafafa', borderRadius: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                <Text strong>功能权限</Text>
                <Button
                  size="small"
                  type="primary"
                  icon={<SaveOutlined />}
                  loading={saving}
                  onClick={handleSaveFunctionPerms}
                >
                  保存功能权限
                </Button>
              </div>
              {Object.entries(groupedPerms).map(([group, perms]) => (
                <div key={group} style={{ marginBottom: 12 }}>
                  <Text type="secondary" style={{ fontSize: 12 }}>{group}</Text>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px 16px', marginTop: 4 }}>
                    {perms.map((perm) => (
                      <Checkbox
                        key={perm.id}
                        checked={rolePermIDs.has(perm.id)}
                        onChange={(e) => handleTogglePerm(perm.id, e.target.checked)}
                      >
                        {perm.name}
                      </Checkbox>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            {/* 产线权限 */}
            <div style={{ padding: 16, background: '#fafafa', borderRadius: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                <Text strong>产线权限</Text>
                <Space>
                  <Button size="small" icon={<ReloadOutlined />} onClick={() => loadRoleDetail(selectedRole.id)}>
                    刷新
                  </Button>
                  <Button
                    size="small"
                    type="primary"
                    icon={<SaveOutlined />}
                    loading={saving}
                    onClick={handleSaveLinePerms}
                  >
                    保存产线权限
                  </Button>
                </Space>
              </div>
              <Table
                dataSource={lineMatrix}
                columns={lineColumns}
                rowKey="production_line_id"
                loading={loading}
                size="small"
                pagination={false}
                scroll={{ y: 400 }}
              />
            </div>
          </div>
        )}
      </div>

      {/* 新建角色弹窗 */}
      <Modal
        title="新建角色"
        open={createModalOpen}
        onCancel={() => {
          setCreateModalOpen(false);
          createForm.resetFields();
        }}
        onOk={() => createForm.submit()}
      >
        <Form form={createForm} onFinish={handleCreateRole} layout="vertical">
          <Form.Item name="name" label="角色标识" rules={[{ required: true, message: '请输入角色标识' }]}>
            <Input placeholder="如 custom_role_1" />
          </Form.Item>
          <Form.Item name="description" label="角色描述">
            <Input.TextArea placeholder="角色用途说明" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// ========== Tab 2: 用户权限 ==========

const UserPermTab = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<number>();
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [lineMatrix, setLineMatrix] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.get('/users').then((res) => {
      const list = Array.isArray(res.data) ? res.data : [];
      setUsers(list);
      if (list.length > 0 && !selectedUserId) {
        setSelectedUserId(list[0].id);
      }
    });
  }, []);

  const loadUserMatrix = useCallback(async (userId: number) => {
    setLoading(true);
    try {
      const res = await api.get(`/permissions/user/${userId}/matrix`);
      const items = res.data?.items || [];
      // 加载角色产线权限作为对比
      const user = users.find((u) => u.id === userId);
      setSelectedUser(user || null);

      // 标记来源
      const marked = items.map((item: any) => ({
        ...item,
        override: item.source === 'user' || item.source === 'user_override',
        dirty: false,
        original: {
          can_view: item.can_view,
          can_download: item.can_download,
          can_upload: item.can_upload,
          can_manage: item.can_manage,
          override: item.source === 'user' || item.source === 'user_override',
        },
      }));
      setLineMatrix(marked);
    } catch {
      message.error('加载用户权限失败');
    } finally {
      setLoading(false);
    }
  }, [users]);

  useEffect(() => {
    if (selectedUserId) {
      loadUserMatrix(selectedUserId);
    }
  }, [selectedUserId, loadUserMatrix]);

  const handleToggleBit = (lineId: number, bit: PermissionBit, checked: boolean) => {
    setLineMatrix((rows) =>
      rows.map((row) => {
        if (row.production_line_id !== lineId) return row;
        const next = { ...row, [bit]: checked, override: true };
        const dirty =
          !next.original ||
          next.override !== next.original.override ||
          permBitConfig.some((b) => next[b.key] !== next.original?.[b.key]);
        return { ...next, dirty };
      }),
    );
  };

  const handleToggleOverride = (lineId: number, checked: boolean) => {
    setLineMatrix((rows) =>
      rows.map((row) => {
        if (row.production_line_id !== lineId) return row;
        if (checked) {
          return { ...row, override: true, dirty: true };
        }
        // 回退到继承
        const next = {
          ...row,
          override: false,
          can_view: false,
          can_download: false,
          can_upload: false,
          can_manage: false,
        };
        return { ...next, dirty: true };
      }),
    );
  };

  const handleSave = async () => {
    if (!selectedUserId) return;
    setSaving(true);
    try {
      const dirtyRows = lineMatrix.filter((r) => r.dirty);
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
      message.success('用户权限已保存');
      loadUserMatrix(selectedUserId);
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const sourceTag = (source: string) => {
    const colors: Record<string, string> = {
      user: 'blue',
      user_override: 'blue',
      role: 'green',
      role_default: 'green',
      department: 'orange',
      department_default: 'orange',
      none: 'default',
    };
    const labels: Record<string, string> = {
      user: '用户覆盖',
      user_override: '用户覆盖',
      role: '角色',
      role_default: '角色默认',
      department: '部门',
      department_default: '部门默认',
      none: '无',
    };
    return <Tag color={colors[source] || 'default'}>{labels[source] || source}</Tag>;
  };

  const columns: TableColumnsType<any> = [
    { title: '产线', dataIndex: 'production_line_name', width: 180 },
    {
      title: '来源',
      dataIndex: 'source',
      width: 100,
      render: (source: string) => sourceTag(source),
    },
    {
      title: '覆盖',
      dataIndex: 'override',
      width: 80,
      align: 'center',
      render: (override: boolean, record: any) => (
        <Switch
          size="small"
          checked={override}
          onChange={(checked) => handleToggleOverride(record.production_line_id, checked)}
        />
      ),
    },
    ...permBitConfig.map((bit) => ({
      title: bit.label,
      dataIndex: bit.key,
      width: 80,
      align: 'center' as const,
      render: (val: boolean, record: any) => (
        <Checkbox
          checked={val}
          disabled={!record.override}
          onChange={(e) => handleToggleBit(record.production_line_id, bit.key, e.target.checked)}
        />
      ),
    })),
    {
      title: '状态',
      width: 80,
      render: (_: any, record: any) =>
        record.dirty ? <Tag color="orange">已修改</Tag> : <Tag color="green">正常</Tag>,
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          <Text strong>选择用户：</Text>
          <Select
            style={{ width: 240 }}
            showSearch
            optionFilterProp="label"
            value={selectedUserId}
            onChange={setSelectedUserId}
            options={users.map((u) => ({
              value: u.id,
              label: `${u.name} (${u.employee_id})`,
            }))}
          />
          {selectedUser && (
            <Text type="secondary">
              角色: {roleLabels[selectedUser.role] || selectedUser.role}
              {selectedUser.department?.name ? ` | 部门: ${selectedUser.department.name}` : ''}
            </Text>
          )}
        </Space>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => selectedUserId && loadUserMatrix(selectedUserId)}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            onClick={handleSave}
          >
            保存
          </Button>
        </Space>
      </div>

      <Table
        dataSource={lineMatrix}
        columns={columns}
        rowKey="production_line_id"
        loading={loading}
        size="small"
        pagination={false}
        scroll={{ y: 500 }}
      />
    </div>
  );
};

// ========== Tab 3: 产线管理员 ==========

const LineAdminTab = () => {
  const [assignments, setAssignments] = useState<LineAdminAssignment[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [lines, setLines] = useState<ProductionLine[]>([]);
  const [loading, setLoading] = useState(false);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [addForm] = Form.useForm();

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [assignRes, usersRes, linesRes] = await Promise.all([
        api.get('/line-admin/assignments'),
        api.get('/users'),
        api.get('/production-lines'),
      ]);
      setAssignments(Array.isArray(assignRes.data) ? assignRes.data : []);
      setUsers(Array.isArray(usersRes.data) ? usersRes.data : []);
      setLines(Array.isArray(linesRes.data) ? linesRes.data : []);
    } catch {
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleAdd = async (values: { user_id: number; production_line_id: number }) => {
    try {
      await api.post('/line-admin/assignments', values);
      message.success('分配成功');
      setAddModalOpen(false);
      addForm.resetFields();
      loadData();
    } catch (error: any) {
      message.error(error.response?.data?.error || '分配失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/line-admin/assignments/${id}`);
      message.success('已取消分配');
      loadData();
    } catch (error: any) {
      message.error(error.response?.data?.error || '操作失败');
    }
  };

  // 按产线分组
  const lineGroups = lines.map((line) => {
    const lineAssignments = assignments.filter((a) => a.production_line_id === line.id);
    return {
      line,
      assignments: lineAssignments,
    };
  });

  const columns: TableColumnsType<any> = [
    { title: '产线', dataIndex: ['line', 'name'], width: 200 },
    {
      title: '管理员',
      render: (_: any, record: any) => (
        <Space wrap>
          {record.assignments.map((a: LineAdminAssignment) => (
            <Tag
              key={a.id}
              closable
              onClose={() => handleDelete(a.id)}
              color="blue"
            >
              {a.user?.name || `用户${a.user_id}`}
            </Tag>
          ))}
          {record.assignments.length === 0 && <Text type="secondary">无管理员</Text>}
        </Space>
      ),
    },
    {
      title: '操作',
      width: 100,
      render: (_: any, record: any) => (
        <Button
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            addForm.setFieldsValue({ production_line_id: record.line.id });
            setAddModalOpen(true);
          }}
        >
          添加
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Text type="secondary">产线管理员可以管理指定产线的内容，并为其他用户分配该产线的查看/下载/上传权限。</Text>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadData}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              addForm.resetFields();
              setAddModalOpen(true);
            }}
          >
            新增分配
          </Button>
        </Space>
      </div>

      <Table
        dataSource={lineGroups}
        columns={columns}
        rowKey={(record) => record.line.id}
        loading={loading}
        size="small"
        pagination={false}
      />

      <Modal
        title="分配产线管理员"
        open={addModalOpen}
        onCancel={() => {
          setAddModalOpen(false);
          addForm.resetFields();
        }}
        onOk={() => addForm.submit()}
      >
        <Form form={addForm} onFinish={handleAdd} layout="vertical">
          <Form.Item name="user_id" label="用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择用户"
              options={users.map((u) => ({
                value: u.id,
                label: `${u.name} (${u.employee_id}) - ${roleLabels[u.role] || u.role}`,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="production_line_id"
            label="产线"
            rules={[{ required: true, message: '请选择产线' }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择产线"
              options={lines.map((l) => ({ value: l.id, label: l.name }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PermissionManagement;
