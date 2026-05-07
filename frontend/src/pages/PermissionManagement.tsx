import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  ConfigProvider,
  Segmented,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import type { TableColumnsType } from 'antd';
import {
  ApartmentOutlined,
  ReloadOutlined,
  SaveOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title, Text } = Typography;

type Decision = 'unset' | 'allow' | 'deny';
type ActionKey = 'view' | 'download' | 'upload' | 'manage';

type User = {
  id: number;
  name: string;
  employee_id?: string;
  role?: string;
  department?: { id?: number; name?: string } | null;
};

type Department = {
  id: number;
  name: string;
};

type Role = {
  id: number;
  name: string;
  display_name?: string;
  description?: string;
};

type PermissionCell = {
  action: ActionKey;
  setting: Decision;
  setting_label?: string;
  effective: 'allow' | 'deny';
  effective_label?: string;
  source: string;
  source_label?: string;
};

type PermissionMatrixLine = {
  resource_type: string;
  resource_id: number;
  resource_name: string;
  actions: Record<ActionKey, PermissionCell>;
};

type PermissionRuleChange = {
  resource_type: string;
  resource_id: number;
  action: ActionKey;
  decision: Decision;
};

type MatrixMode = 'user' | 'department' | 'role' | 'departmentDefault';

type MatrixEditorProps = {
  mode: MatrixMode;
  targetId?: number | string;
  emptyText: string;
};

/* ────────────────── 统一 Select 主题 ────────────────── */

const selectTheme = {
  components: {
    Select: {
      controlHeight: 36,
      borderRadius: 8,
      colorBorder: 'transparent',
      colorPrimaryHover: 'transparent',
      controlOutline: 'none',
    },
  },
};

/* ────────────────── 常量 ────────────────── */

const actionConfigs: Array<{ key: ActionKey; label: string }> = [
  { key: 'view', label: '查看' },
  { key: 'download', label: '下载' },
  { key: 'upload', label: '上传' },
  { key: 'manage', label: '管理' },
];

const decisionOptions: Array<{ label: string; value: Decision }> = [
  { label: '按规则', value: 'unset' },
  { label: '允许', value: 'allow' },
  { label: '拒绝', value: 'deny' },
];

const roleLabels: Record<string, string> = {
  admin: '管理员',
  system_admin: '系统管理员',
  line_admin: '产线管理员',
  engineer: '工程师',
  operator: '操作员',
  viewer: '查看者',
};

const sourceLabels: Record<string, string> = {
  user: '单独设置',
  department: '部门规则',
  role: '角色规则',
  role_default: '角色规则',
  department_default: '部门默认规则',
  system_default: '系统默认',
  none: '系统默认',
};

const modeSourceLabels: Record<MatrixMode, string> = {
  user: '单独设置',
  department: '部门规则',
  role: '角色规则',
  departmentDefault: '部门默认规则',
};

const decisionLabels: Record<Decision | 'allow' | 'deny', string> = {
  unset: '按规则',
  allow: '允许',
  deny: '拒绝',
};

/* ────────────────── 工具函数 ────────────────── */

const makeCellKey = (lineId: number, action: ActionKey) => `${lineId}:${action}`;

const getMatrixEndpoints = (mode: MatrixMode, targetId: number | string) => {
  switch (mode) {
    case 'user':
      return {
        load: `/permissions/users/${targetId}/effective-matrix`,
        save: `/permissions/users/${targetId}/rules`,
      };
    case 'department':
      return {
        load: `/permissions/departments/${targetId}/effective-matrix`,
        save: `/permissions/departments/${targetId}/rules`,
      };
    case 'role':
      return {
        load: `/permissions/roles/${targetId}/effective-matrix`,
        save: `/permissions/roles/${targetId}/rules`,
      };
    case 'departmentDefault':
      return {
        load: `/permissions/departments/${targetId}/default-matrix`,
        save: `/permissions/departments/${targetId}/default-rules`,
      };
  }
};

const getCellSetting = (
  row: PermissionMatrixLine,
  action: ActionKey,
  changes: Record<string, Decision>,
) => changes[makeCellKey(row.resource_id, action)] ?? row.actions[action]?.setting ?? 'unset';

const getVisibleResult = (
  cell: PermissionCell,
  setting: Decision,
  mode: MatrixMode,
): { result: 'allow' | 'deny'; label: string; sourceLabel: string } => {
  if (setting === 'allow' || setting === 'deny') {
    return {
      result: setting,
      label: decisionLabels[setting],
      sourceLabel: modeSourceLabels[mode],
    };
  }

  const effective = cell.effective === 'allow' ? 'allow' : 'deny';
  return {
    result: effective,
    label: decisionLabels[effective],
    sourceLabel: sourceLabels[cell.source] || cell.source_label || '系统默认',
  };
};

/* ────────────────── 权限矩阵编辑器 ────────────────── */

const MatrixEditor = ({ mode, targetId, emptyText }: MatrixEditorProps) => {
  const [rows, setRows] = useState<PermissionMatrixLine[]>([]);
  const [changes, setChanges] = useState<Record<string, Decision>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const changedCount = Object.keys(changes).length;

  const loadMatrix = useCallback(async () => {
    if (!targetId) {
      setRows([]);
      setChanges({});
      return;
    }

    setLoading(true);
    try {
      const endpoints = getMatrixEndpoints(mode, targetId);
      const res = await api.get(endpoints.load);
      setRows(Array.isArray(res.data?.items) ? res.data.items : []);
      setChanges({});
    } catch (error: any) {
      message.error(error.response?.data?.error || '加载权限矩阵失败');
    } finally {
      setLoading(false);
    }
  }, [mode, targetId]);

  useEffect(() => {
    loadMatrix();
  }, [loadMatrix]);

  const handleDecisionChange = (
    row: PermissionMatrixLine,
    action: ActionKey,
    decision: Decision,
  ) => {
    const key = makeCellKey(row.resource_id, action);
    const original = row.actions[action]?.setting ?? 'unset';

    setChanges((prev) => {
      const next = { ...prev };
      if (decision === original) {
        delete next[key];
      } else {
        next[key] = decision;
      }
      return next;
    });
  };

  const handleSave = async () => {
    if (!targetId || changedCount === 0) return;

    const payload: { changes: PermissionRuleChange[] } = {
      changes: rows.flatMap((row) =>
        actionConfigs.flatMap(({ key: action }) => {
          const decision = changes[makeCellKey(row.resource_id, action)];
          if (!decision) return [];
          return [
            {
              resource_type: row.resource_type || 'production_line',
              resource_id: row.resource_id,
              action,
              decision,
            },
          ];
        }),
      ),
    };

    setSaving(true);
    try {
      const endpoints = getMatrixEndpoints(mode, targetId);
      const res = await api.put(endpoints.save, payload);
      setRows(Array.isArray(res.data?.items) ? res.data.items : rows);
      setChanges({});
      message.success('权限规则已保存');
    } catch (error: any) {
      message.error(error.response?.data?.error || '保存权限规则失败');
    } finally {
      setSaving(false);
    }
  };

  const columns: TableColumnsType<PermissionMatrixLine> = useMemo(
    () => [
      {
        title: '产线',
        dataIndex: 'resource_name',
        key: 'resource_name',
        width: 180,
        fixed: 'left',
        render: (text: string) => <Text strong>{text}</Text>,
      },
      ...actionConfigs.map(({ key: action, label }) => ({
        title: label,
        dataIndex: ['actions', action],
        key: action,
        width: 180,
        render: (_: PermissionCell, row: PermissionMatrixLine) => {
          const cell = row.actions[action];
          const setting = getCellSetting(row, action, changes);
          const visible = getVisibleResult(cell, setting, mode);
          const changed = Boolean(changes[makeCellKey(row.resource_id, action)]);

          return (
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <Space size={6}>
                <Tag
                  color={visible.result === 'allow' ? 'green' : 'red'}
                  style={{ borderRadius: 4, fontWeight: 600, fontSize: 12 }}
                >
                  {visible.label}
                </Tag>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {visible.sourceLabel}
                </Text>
              </Space>
              <Segmented
                aria-label={`${row.resource_name}-${label}-设置`}
                size="small"
                value={setting}
                options={decisionOptions}
                onChange={(value) =>
                  handleDecisionChange(row, action, value as Decision)
                }
              />
              {changed ? (
                <Text type="warning" style={{ fontSize: 12 }}>
                  待保存
                </Text>
              ) : null}
            </Space>
          );
        },
      })),
    ],
    [changes],
  );

  if (!targetId) {
    return <Alert type="info" showIcon message={emptyText} />;
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text type="secondary" style={{ fontSize: 13 }}>
          每个单元格都可以设置为按规则、允许或拒绝。保存时只提交改动过的单元格。
        </Text>
        <Space size={10}>
          <Button
            icon={<ReloadOutlined />}
            onClick={loadMatrix}
            loading={loading}
            style={{
              height: 36,
              borderRadius: 8,
              background: '#dee3e6',
              color: '#2d3335',
              fontWeight: 700,
              border: 'none',
            }}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            disabled={changedCount === 0}
            loading={saving}
            onClick={handleSave}
            style={{
              height: 36,
              borderRadius: 8,
              background: changedCount > 0
                ? 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)'
                : undefined,
              border: 'none',
              fontWeight: 700,
              boxShadow: changedCount > 0
                ? '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)'
                : undefined,
            }}
          >
            保存{changedCount > 0 ? ` ${changedCount} 项` : ''}
          </Button>
        </Space>
      </div>
      <Table
        className="custom-table"
        columns={columns}
        dataSource={rows}
        rowKey="resource_id"
        loading={loading}
        pagination={false}
        size="middle"
        scroll={{ x: 900 }}
      />
    </Space>
  );
};

/* ────────────────── Tab: 用户权限 ────────────────── */

const UserPermissionTab = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<number>();
  const [loading, setLoading] = useState(false);

  const selectedUser = users.find((user) => user.id === selectedUserId);

  useEffect(() => {
    setLoading(true);
    api
      .get('/users')
      .then((res) => {
        const list = Array.isArray(res.data) ? res.data : [];
        setUsers(list);
        setSelectedUserId((current) => current ?? list[0]?.id);
      })
      .catch(() => message.error('加载用户列表失败'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <ConfigProvider theme={selectTheme}>
        <div className="management-filter-panel">
          <div className="management-filter-field" style={{ width: 300 }}>
            <div className="management-filter-label">用户</div>
            <Select
              aria-label="选择用户"
              style={{ width: '100%' }}
              loading={loading}
              showSearch
              optionFilterProp="label"
              value={selectedUserId}
              onChange={setSelectedUserId}
              options={users.map((user) => ({
                value: user.id,
                label: `${user.name}${user.employee_id ? ` (${user.employee_id})` : ''}`,
              }))}
            />
          </div>
          {selectedUser && (
            <div style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 2 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>
                {selectedUser.department?.name || '未分配部门'} ·{' '}
                {roleLabels[selectedUser.role || ''] || selectedUser.role || '未分配角色'}
              </Text>
            </div>
          )}
        </div>
      </ConfigProvider>
      <MatrixEditor
        mode="user"
        targetId={selectedUserId}
        emptyText="请选择用户"
      />
    </Space>
  );
};

/* ────────────────── Tab: 部门规则 ────────────────── */

const DepartmentRuleTab = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<number>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    api
      .get('/departments')
      .then((res) => {
        const list = Array.isArray(res.data) ? res.data : [];
        setDepartments(list);
        setSelectedDepartmentId((current) => current ?? list[0]?.id);
      })
      .catch(() => message.error('加载部门列表失败'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <ConfigProvider theme={selectTheme}>
        <div className="management-filter-panel">
          <div className="management-filter-field" style={{ width: 260 }}>
            <div className="management-filter-label">部门</div>
            <Select
              aria-label="选择部门"
              style={{ width: '100%' }}
              loading={loading}
              showSearch
              optionFilterProp="label"
              value={selectedDepartmentId}
              onChange={setSelectedDepartmentId}
              options={departments.map((d) => ({ value: d.id, label: d.name }))}
            />
          </div>
        </div>
      </ConfigProvider>
      <MatrixEditor
        mode="department"
        targetId={selectedDepartmentId}
        emptyText="请选择部门"
      />
    </Space>
  );
};

/* ────────────────── Tab: 角色规则 ────────────────── */

const RoleRuleTab = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    api
      .get('/roles')
      .then((res) => {
        const list = Array.isArray(res.data) ? res.data : [];
        setRoles(list);
        setSelectedRoleId((current) => current ?? list[0]?.id);
      })
      .catch(() => message.error('加载角色列表失败'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <ConfigProvider theme={selectTheme}>
        <div className="management-filter-panel">
          <div className="management-filter-field" style={{ width: 260 }}>
            <div className="management-filter-label">角色</div>
            <Select
              aria-label="选择角色"
              style={{ width: '100%' }}
              loading={loading}
              showSearch
              optionFilterProp="label"
              value={selectedRoleId}
              onChange={setSelectedRoleId}
              options={roles.map((role) => ({
                value: role.id,
                label: role.display_name || roleLabels[role.name] || role.name,
              }))}
            />
          </div>
        </div>
      </ConfigProvider>
      <MatrixEditor
        mode="role"
        targetId={selectedRoleId}
        emptyText="请选择角色"
      />
    </Space>
  );
};

/* ────────────────── Tab: 部门默认规则 ────────────────── */

const DepartmentDefaultRuleTab = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<number>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    api
      .get('/departments')
      .then((res) => {
        const list = Array.isArray(res.data) ? res.data : [];
        setDepartments(list);
        setSelectedDepartmentId((current) => current ?? list[0]?.id);
      })
      .catch(() => message.error('加载部门列表失败'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <ConfigProvider theme={selectTheme}>
        <div className="management-filter-panel">
          <div className="management-filter-field" style={{ width: 260 }}>
            <div className="management-filter-label">部门</div>
            <Select
              aria-label="选择默认规则部门"
              style={{ width: '100%' }}
              loading={loading}
              showSearch
              optionFilterProp="label"
              value={selectedDepartmentId}
              onChange={setSelectedDepartmentId}
              options={departments.map((d) => ({ value: d.id, label: d.name }))}
            />
          </div>
        </div>
      </ConfigProvider>
      <MatrixEditor
        mode="departmentDefault"
        targetId={selectedDepartmentId}
        emptyText="请选择部门"
      />
    </Space>
  );
};

/* ────────────────── 主页面 ────────────────── */

const PermissionManagement = () => (
  <div className="management-page">
    <div className="management-page-header">
      <div>
        <div className="management-page-breadcrumb">
          <span>系统</span>
          <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
          <span className="active">权限管理</span>
        </div>
        <Title level={2} className="management-page-title">
          权限管理
        </Title>
      </div>
    </div>
    <div className="management-table-card" style={{ padding: '24px' }}>
      <Tabs
        items={[
          {
            key: 'users',
            label: (
              <span>
                <UserOutlined />
                用户权限
              </span>
            ),
            children: <UserPermissionTab />,
          },
          {
            key: 'departments',
            label: (
              <span>
                <ApartmentOutlined />
                部门规则
              </span>
            ),
            children: <DepartmentRuleTab />,
          },
          {
            key: 'roles',
            label: (
              <span>
                <TeamOutlined />
                角色规则
              </span>
            ),
            children: <RoleRuleTab />,
          },
          {
            key: 'department-defaults',
            label: (
              <span>
                <ApartmentOutlined />
                部门默认规则
              </span>
            ),
            children: <DepartmentDefaultRuleTab />,
          },
        ]}
      />
    </div>
  </div>
);

export default PermissionManagement;
