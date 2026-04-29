import {
  render,
  screen,
  fireEvent,
  waitFor,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mock } from 'vitest';
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
    isAdmin: true,
    hasPermission: () => true,
  }),
}));

const mockApiGet = api.get as Mock;
const mockApiPut = api.put as Mock;
const mockApiPost = api.post as Mock;

const roles = [
  { id: 1, name: 'system_admin', description: '系统管理员', is_preset: true, is_system: true, status: 'active', sort_order: 1 },
  { id: 2, name: 'engineer', description: '工程师', is_preset: true, is_system: false, status: 'active', sort_order: 3 },
];

const allPermissions = [
  { id: 1, code: 'page:dashboard', name: '仪表盘', type: 'page', resource: 'dashboard' },
  { id: 2, code: 'page:programs', name: '程序管理', type: 'page', resource: 'program' },
  { id: 3, code: 'op:program_create', name: '创建程序', type: 'operation', resource: 'program' },
  { id: 4, code: 'op:file_upload', name: '上传文件', type: 'operation', resource: 'file' },
];

const users = [
  { id: 1, name: 'Alice', employee_id: 'U001', role: 'engineer', role_id: 2, department: { id: 1, name: '制造部' } },
  { id: 2, name: 'Bob', employee_id: 'U002', role: 'viewer', role_id: 5, department: { id: 2, name: '工艺部' } },
];

const productionLines = [
  { id: 10, name: '总装线' },
  { id: 11, name: '调试线' },
];

const roleDetail = {
  role: roles[1],
  permissions: [allPermissions[0], allPermissions[1], allPermissions[2]],
  line_permissions: [
    { production_line_id: 10, production_line_name: '总装线', can_view: true, can_download: true, can_upload: true, can_manage: false },
  ],
};

const roleLineMatrix = {
  role: roles[1],
  lines: productionLines,
  permissions: [
    { production_line_id: 10, production_line_name: '总装线', can_view: true, can_download: true, can_upload: true, can_manage: false },
    { production_line_id: 11, production_line_name: '调试线', can_view: false, can_download: false, can_upload: false, can_manage: false },
  ],
};

const userMatrixItems = [
  { production_line_id: 10, production_line_name: '总装线', can_view: true, can_download: false, can_upload: false, can_manage: false, source: 'user' },
  { production_line_id: 11, production_line_name: '调试线', can_view: true, can_download: false, can_upload: false, can_manage: false, source: 'role' },
];

const lineAssignments = [
  { id: 1, user_id: 1, production_line_id: 10, user: users[0], production_line: productionLines[0] },
];

const renderPage = () =>
  render(
    <MemoryRouter>
      <PermissionManagement />
    </MemoryRouter>,
  );

describe('PermissionManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockApiGet.mockImplementation((url: string) => {
      if (url === '/roles') return Promise.resolve({ data: roles });
      if (url === '/permission-definitions') return Promise.resolve({ data: allPermissions });
      if (url === '/roles/2') return Promise.resolve({ data: roleDetail });
      if (url === '/roles/2/permissions') return Promise.resolve({ data: roleLineMatrix });
      if (url === '/users') return Promise.resolve({ data: users });
      if (url === '/production-lines') return Promise.resolve({ data: productionLines });
      if (url.startsWith('/permissions/user/') && url.endsWith('/matrix')) {
        return Promise.resolve({ data: { items: userMatrixItems } });
      }
      if (url === '/line-admin/assignments') return Promise.resolve({ data: lineAssignments });
      return Promise.reject(new Error(`Unhandled GET ${url}`));
    });

    mockApiPut.mockResolvedValue({ data: { message: 'ok' } });
    mockApiPost.mockResolvedValue({ data: { message: 'ok' } });
  });

  it('renders the page heading and three tabs', async () => {
    renderPage();
    expect(await screen.findByText('权限管理')).toBeTruthy();
    expect(screen.getByText(/角色管理/)).toBeTruthy();
    expect(screen.getByText(/用户权限/)).toBeTruthy();
    expect(screen.getByText(/产线管理员/)).toBeTruthy();
  });

  it('loads roles on the role tab', async () => {
    renderPage();
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/roles');
      expect(mockApiGet).toHaveBeenCalledWith('/permission-definitions');
    });
    // 角色名称出现多次（主标签+描述），用 getAllByText 验证
    await waitFor(() => {
      expect(screen.getAllByText(/系统管理员/).length).toBeGreaterThan(0);
      expect(screen.getAllByText(/工程师/).length).toBeGreaterThan(0);
    });
  });

  it('loads role detail when selecting a role', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText(/工程师/).length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText(/工程师/)[0]);

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/roles/2');
      expect(mockApiGet).toHaveBeenCalledWith('/roles/2/permissions');
    });
  });

  it('saves function permissions for a role', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText(/工程师/).length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText(/工程师/)[0]);

    await waitFor(() => {
      expect(screen.getByText('保存功能权限')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('保存功能权限'));

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith(
        '/roles/2/function-permissions',
        expect.objectContaining({ permission_ids: expect.any(Array) }),
      );
    });
  });

  it('saves line permissions for a role', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText(/工程师/).length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText(/工程师/)[0]);

    await waitFor(() => {
      expect(screen.getByText('保存产线权限')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('保存产线权限'));

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith(
        '/roles/2/permissions',
        expect.objectContaining({ permissions: expect.any(Array) }),
      );
    });
  });

  it('loads user permission matrix on the user tab', async () => {
    renderPage();
    await screen.findByText(/用户权限/);
    fireEvent.click(screen.getByText(/用户权限/));

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users');
    });

    expect(await screen.findByText('总装线')).toBeTruthy();
    expect(screen.getByText('调试线')).toBeTruthy();
  });

  it('loads line admin assignments on the line admin tab', async () => {
    renderPage();
    await screen.findByText(/产线管理员/);
    fireEvent.click(screen.getByText(/产线管理员/));

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/line-admin/assignments');
    });
    expect(await screen.findByText('总装线')).toBeTruthy();
  });
});
