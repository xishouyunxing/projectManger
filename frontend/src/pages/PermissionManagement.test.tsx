import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
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
    isLineAdmin: false,
    permissions: { codes: [], lines: {}, managed_line_ids: [] },
    hasPermission: () => true,
    hasLinePermission: () => true,
    isLineManager: () => true,
  }),
}));

const mockApiGet = api.get as Mock;
const mockApiPut = api.put as Mock;

const users = [
  {
    id: 1,
    name: 'Alice',
    employee_id: 'U001',
    role: 'engineer',
    department: { id: 1, name: '制造部' },
  },
];

const departments = [
  { id: 1, name: '制造部' },
  { id: 2, name: '工艺部' },
];

const roles = [
  { id: 2, name: 'engineer', display_name: '工程师' },
  { id: 5, name: 'viewer', display_name: '查看者' },
];

const makeCell = (
  action: string,
  setting: 'unset' | 'allow' | 'deny',
  effective: 'allow' | 'deny',
  source = 'department',
) => ({
  action,
  setting,
  setting_label:
    setting === 'unset' ? '按规则' : setting === 'allow' ? '允许' : '拒绝',
  effective,
  effective_label: effective === 'allow' ? '允许' : '拒绝',
  source,
  source_label:
    source === 'department'
      ? '部门规则'
      : source === 'user'
        ? '单独设置'
        : '系统默认',
});

const matrixItems = [
  {
    resource_type: 'production_line',
    resource_id: 10,
    resource_name: '总装线',
    actions: {
      view: makeCell('view', 'unset', 'allow'),
      download: makeCell('download', 'deny', 'deny', 'user'),
      upload: makeCell('upload', 'unset', 'deny', 'system_default'),
      manage: makeCell('manage', 'unset', 'deny', 'system_default'),
    },
  },
  {
    resource_type: 'production_line',
    resource_id: 11,
    resource_name: '调试线',
    actions: {
      view: makeCell('view', 'unset', 'deny', 'system_default'),
      download: makeCell('download', 'unset', 'deny', 'system_default'),
      upload: makeCell('upload', 'unset', 'deny', 'system_default'),
      manage: makeCell('manage', 'unset', 'deny', 'system_default'),
    },
  },
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
      if (url === '/users') return Promise.resolve({ data: users });
      if (url === '/departments') return Promise.resolve({ data: departments });
      if (url === '/roles') return Promise.resolve({ data: roles });
      if (url.includes('effective-matrix') || url.includes('default-matrix')) {
        return Promise.resolve({ data: { items: matrixItems } });
      }
      return Promise.reject(new Error(`Unhandled GET ${url}`));
    });

    mockApiPut.mockResolvedValue({ data: { items: matrixItems } });
  });

  it('renders the four permission rule tabs without technical wording', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: '权限管理' })).toBeTruthy();
    expect(screen.getByText('用户权限')).toBeTruthy();
    expect(screen.getByText('部门规则')).toBeTruthy();
    expect(screen.getByText('角色规则')).toBeTruthy();
    expect(screen.getByText('部门默认规则')).toBeTruthy();
    expect(screen.queryByText(/继承|allow|deny|unset|policy/)).toBeNull();
  });

  it('loads the user permission matrix from the rule API', async () => {
    renderPage();

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permissions/users/1/effective-matrix',
      );
    });
    expect(await screen.findByText('总装线')).toBeTruthy();
    expect(screen.getAllByText('跟随').length).toBeGreaterThan(0);
    expect(screen.getAllByText(/部门覆盖/).length).toBeGreaterThan(0);
  });

  it('submits only the changed cell when setting one permission to refuse', async () => {
    renderPage();

    const selector = await screen.findByLabelText('总装线-查看程序列表-设置');
    fireEvent.click(within(selector).getByText('拒绝'));
    fireEvent.click(screen.getByRole('button', { name: /保存 1 项/ }));

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/permissions/users/1/rules', {
        changes: [
          {
            resource_type: 'production_line',
            resource_id: 10,
            action: 'view',
            decision: 'deny',
          },
        ],
      });
    });
  });

  it('submits unset when changing a separate setting back to rule based mode', async () => {
    renderPage();

    const selector = await screen.findByLabelText('总装线-下载程序文件-设置');
    fireEvent.click(within(selector).getByText('跟随'));
    fireEvent.click(screen.getByRole('button', { name: /保存 1 项/ }));

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/permissions/users/1/rules', {
        changes: [
          {
            resource_type: 'production_line',
            resource_id: 10,
            action: 'download',
            decision: 'unset',
          },
        ],
      });
    });
  });

  it('loads department, role and department default matrices from their rule APIs', async () => {
    renderPage();

    fireEvent.click(await screen.findByText('部门规则'));
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permissions/departments/1/effective-matrix',
      );
    });

    fireEvent.click(screen.getByText('角色规则'));
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permissions/roles/2/effective-matrix',
      );
    });

    fireEvent.click(screen.getByText('部门默认规则'));
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permissions/departments/1/default-matrix',
      );
    });
  });

  it('shows the matching rule source label when editing non-user rule tabs', async () => {
    const { container } = renderPage();

    fireEvent.click(await screen.findByRole('tab', { name: /\u89d2\u8272\u89c4\u5219/ }));
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permissions/roles/2/effective-matrix',
      );
    });

    const selector = await waitFor(() => {
      const activePanel = container.querySelector('.ant-tabs-tabpane-active');
      const segmented = activePanel?.querySelector('.ant-segmented');
      expect(segmented).toBeTruthy();
      return segmented as HTMLElement;
    });
    fireEvent.click(within(selector).getByText('\u5141\u8bb8'));

    expect(screen.getAllByText(/\u89d2\u8272\u8986\u76d6/).length).toBeGreaterThan(0);
  });
});
