import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import PermissionManagement from './PermissionManagement';
import api from '../services/api';

vi.mock('../services/api', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, name: 'Admin', role: 'system_admin', employee_id: 'A001' },
    token: 'test-token',
    isAdmin: true,
    hasPermission: () => true,
    isLineAdmin: false,
    isLineManager: () => false,
    hasLinePermission: () => true,
  }),
}));

const mockApiGet = api.get as unknown as ReturnType<typeof vi.fn>;
const mockApiPut = api.put as unknown as ReturnType<typeof vi.fn>;

const roles = [
  { id: 1, name: 'system_admin', display_name: '系统管理员', description: '系统最高权限', is_preset: true, is_system: true, sort_order: 0 },
  { id: 2, name: 'line_admin', display_name: '产线管理员', description: '管理指定产线', is_preset: true, is_system: false, sort_order: 1 },
  { id: 3, name: 'engineer', display_name: '工程师', description: '产线操作', is_preset: true, is_system: false, sort_order: 2 },
];

const allPermissions = [
  { id: 1, code: 'page:user_management', name: '用户管理页面', type: 'page', resource: '/users' },
  { id: 2, code: 'page:permission_management', name: '权限管理页面', type: 'page', resource: '/permissions' },
  { id: 3, code: 'op:create_program', name: '创建程序', type: 'operation', resource: 'program' },
];

const users = [
  { id: 1, name: 'Admin', employee_id: 'A001', role: 'system_admin', department: { name: '制造部' } },
  { id: 2, name: 'Alice', employee_id: 'U001', role: 'line_admin', department: { name: '制造部' } },
];

const productionLines = [
  { id: 10, name: '总装线' },
  { id: 11, name: '调试线' },
];

const rolePermissions = [
  { id: 1, code: 'page:user_management', name: '用户管理页面', type: 'page' },
];

const roleLinePermissions = [
  { production_line_id: 10, production_line_name: '总装线', can_view: true, can_download: true, can_upload: true, can_manage: false },
];

const userMatrixRows = [
  { production_line_id: 10, production_line_name: '总装线', can_view: true, can_download: false, can_upload: false, can_manage: false, source: 'user' },
  { production_line_id: 11, production_line_name: '调试线', can_view: false, can_download: false, can_upload: false, can_manage: false, source: 'none' },
];

const lineAdmins = [
  { id: 1, user_id: 2, user: { name: 'Alice', employee_id: 'U001' }, production_line_id: 10, production_line: { name: '总装线' } },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockApiGet.mockImplementation((url: string) => {
    if (url === '/roles') return Promise.resolve({ data: roles });
    if (url === '/permission-definitions') return Promise.resolve({ data: allPermissions });
    if (url === '/roles/1') return Promise.resolve({ data: { role: roles[0], permissions: rolePermissions, line_permissions: roleLinePermissions } });
    if (url === '/roles/1/permissions') return Promise.resolve({ data: { permissions: roleLinePermissions } });
    if (url === '/users') return Promise.resolve({ data: users });
    if (url === '/production-lines') return Promise.resolve({ data: productionLines });
    if (url === '/permissions/user/1/matrix') return Promise.resolve({ data: { items: userMatrixRows } });
    if (url === '/permissions/user/2/matrix') return Promise.resolve({ data: { items: userMatrixRows } });
    if (url === '/line-admin/assignments') return Promise.resolve({ data: lineAdmins });
    return Promise.reject(new Error(`Unhandled GET ${url}`));
  });
  mockApiPut.mockResolvedValue({ data: { message: 'ok' } });
});

describe('PermissionManagement', () => {
  it('renders three tabs', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    // Tab labels are rendered by Ant Design Tabs component
    await screen.findByText('角色管理');
    // "用户权限" and "产线管理员" are tab labels - use getAllByText since they may also appear as role names
    expect(screen.getAllByText('用户权限').length).toBeGreaterThan(0);
    expect(screen.getAllByText('产线管理员').length).toBeGreaterThan(0);
  });

  it('loads and displays roles in role management tab', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('系统管理员');
    // "产线管理员" and "工程师" appear both as tab labels and role names
    expect(screen.getAllByText('产线管理员').length).toBeGreaterThan(1);
    expect(screen.getAllByText('工程师').length).toBeGreaterThan(0);
  });

  it('loads role detail on role click', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('系统管理员');
    fireEvent.click(screen.getByText('系统管理员'));
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/roles/1');
      expect(mockApiGet).toHaveBeenCalledWith('/roles/1/permissions');
    });
  });

  it('saves function permissions', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('系统管理员');
    fireEvent.click(screen.getByText('系统管理员'));
    await screen.findByText('用户管理页面');
    const saveBtn = screen.getByRole('button', { name: /保存功能权限/ });
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalled();
    });
  });

  it('saves line permissions', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('系统管理员');
    fireEvent.click(screen.getByText('系统管理员'));
    await screen.findByText('总装线');
    const saveBtn = screen.getByRole('button', { name: /保存产线权限/ });
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalled();
    });
  });

  it('loads user permission matrix in user permissions tab', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('角色管理');
    // Click the "用户权限" tab
    const tabList = screen.getByRole('tablist');
    const userTab = tabList.querySelector('[data-node-key="users"]');
    expect(userTab).toBeTruthy();
    fireEvent.click(userTab!);
    // The UserPermissionsTab has a Select with aria-label="选择用户"
    // There may be multiple if tab content is lazy-rendered; use the last one
    await waitFor(() => {
      expect(screen.getAllByLabelText('选择用户').length).toBeGreaterThan(0);
    });
    const allSelects = screen.getAllByLabelText('选择用户');
    const selectContainer = allSelects[allSelects.length - 1];
    const selectEl = selectContainer.closest('.ant-select')?.querySelector('.ant-select-selector');
    expect(selectEl).toBeTruthy();
    fireEvent.mouseDown(selectEl!);
    // Option text format: "Alice (U001) - 制造部"
    const option = await screen.findByText(/Alice/);
    fireEvent.click(option);
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/permissions/user/2/matrix');
    });
  });

  it('loads line admin assignments in line admin tab', async () => {
    render(<MemoryRouter><PermissionManagement /></MemoryRouter>);
    await screen.findByText('角色管理');
    // Click the "产线管理员" tab
    const tabList = screen.getByRole('tablist');
    const lineAdminTab = tabList.querySelector('[data-node-key="line_admins"]');
    expect(lineAdminTab).toBeTruthy();
    fireEvent.click(lineAdminTab!);
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/line-admin/assignments');
    });
  });
});
