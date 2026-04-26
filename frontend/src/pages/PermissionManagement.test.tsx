import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mock } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import PermissionManagement from './PermissionManagement';
import api from '../services/api';

vi.mock('../services/api', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
  },
}));

const mockApiGet = api.get as Mock;
const mockApiPut = api.put as Mock;

const users = [
  {
    id: 1,
    name: 'Alice',
    employee_id: 'U001',
    role: 'user',
    department: { name: '制造部' },
  },
  {
    id: 2,
    name: 'Bob',
    employee_id: 'U002',
    role: 'admin',
    department: { name: '工艺部' },
  },
];

const departments = [
  { id: 1, name: '制造部' },
  { id: 2, name: '工艺部' },
];

const productionLines = [
  { id: 10, name: '总装线' },
  { id: 11, name: '调试线' },
];

const matrixRows = [
  {
    production_line_id: 10,
    production_line_name: '总装线',
    can_view: true,
    can_download: false,
    can_upload: false,
    can_manage: false,
    source: 'user',
  },
  {
    production_line_id: 11,
    production_line_name: '调试线',
    can_view: false,
    can_download: false,
    can_upload: true,
    can_manage: false,
    source: 'none',
  },
];

const renderPage = () =>
  render(
    <MemoryRouter>
      <PermissionManagement />
    </MemoryRouter>,
  );

const openSelect = (testID: string) => {
  const select = screen.getByTestId(testID);
  fireEvent.mouseDown(within(select).getByRole('combobox'));
};

const clickLastVisibleText = async (text: string) => {
  await waitFor(() => {
    expect(screen.queryAllByText(text).length).toBeGreaterThan(0);
  });
  const matches = screen.queryAllByText(text);
  fireEvent.click(matches[matches.length - 1]);
};

describe('PermissionManagement permission matrices', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockApiGet.mockImplementation((url: string) => {
      if (url === '/users') {
        return Promise.resolve({ data: users });
      }
      if (url === '/departments') {
        return Promise.resolve({ data: departments });
      }
      if (url === '/production-lines') {
        return Promise.resolve({ data: productionLines });
      }
      if (
        url === '/permissions/user/1/matrix' ||
        url === '/permissions/user/2/matrix' ||
        url === '/department-permissions/department/1/matrix' ||
        url === '/department-permissions/department/2/matrix' ||
        url === '/permission-defaults/roles/user/matrix' ||
        url === '/permission-defaults/roles/admin/matrix' ||
        url === '/permission-defaults/departments/1/matrix' ||
        url === '/permission-defaults/departments/2/matrix'
      ) {
        return Promise.resolve({ data: { items: matrixRows } });
      }
      return Promise.reject(new Error(`Unhandled GET ${url}`));
    });

    mockApiPut.mockResolvedValue({ data: { message: 'ok' } });
  });

  it('loads users, departments, and production lines on initial render', async () => {
    renderPage();

    await screen.findByRole('heading', { name: '权限管理' });

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users');
      expect(mockApiGet).toHaveBeenCalledWith('/departments');
      expect(mockApiGet).toHaveBeenCalledWith('/production-lines');
    });
  });

  it('loads a selected user matrix and saves exact permission booleans', async () => {
    renderPage();

    await screen.findByText('总装线');
    openSelect('user-matrix-select');
    fireEvent.click(await screen.findByText(/Bob/));

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/permissions/user/2/matrix');
    });

    fireEvent.click(screen.getByLabelText('总装线 下载'));
    fireEvent.click(screen.getAllByRole('button', { name: /保存/ })[0]);

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/permissions/user/2/matrix', {
        permissions: [
          {
            production_line_id: 10,
            can_view: true,
            can_download: true,
            can_upload: false,
            can_manage: false,
          },
          {
            production_line_id: 11,
            can_view: false,
            can_download: false,
            can_upload: true,
            can_manage: false,
          },
        ],
      });
    });
  });

  it('loads department, role default, and department default matrices by owner selection', async () => {
    renderPage();

    await screen.findByText('用户权限矩阵');
    fireEvent.click(screen.getByText('部门权限矩阵'));
    openSelect('department-matrix-select');
    await clickLastVisibleText('工艺部');

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/department-permissions/department/2/matrix',
      );
    });

    fireEvent.click(screen.getByText('角色默认权限'));

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permission-defaults/roles/user/matrix',
      );
    });

    fireEvent.click(screen.getByText('部门默认权限'));
    openSelect('department-default-select');
    await clickLastVisibleText('工艺部');

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith(
        '/permission-defaults/departments/2/matrix',
      );
    });
  });
});
