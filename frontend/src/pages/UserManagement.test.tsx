import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import UserManagement from './UserManagement'
import api from '../services/api'

vi.mock('../services/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../services/api')>()
  return {
    ...actual,
    default: {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
    },
  }
})

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    isAdmin: true,
  }),
}))

const mockApiGet = api.get as Mock
const mockApiDelete = api.delete as Mock

const renderPage = () =>
  render(
    <MemoryRouter>
      <UserManagement />
    </MemoryRouter>
  )

describe('UserManagement pagination', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    let deletedOnPage2 = false
    mockApiGet.mockImplementation((url: string, config?: { params?: { page?: number; page_size?: number } }) => {
      if (url === '/users') {
        const page = config?.params?.page ?? 1
        const pageSize = config?.params?.page_size ?? 20
        return Promise.resolve({
          data: {
            items: deletedOnPage2 && page === 2
              ? []
              : [
                  {
                    id: page === 1 ? 1 : 2,
                    employee_id: page === 1 ? 'U001' : 'U002',
                    name: page === 1 ? 'Alice' : 'Bob',
                    role: 'user',
                    status: 'active',
                    department_id: 1,
                    department: { id: 1, name: '制造部' },
                    created_at: '2026-01-01T08:00:00.000Z',
                  },
                ],
            total: deletedOnPage2 ? 20 : 21,
            page,
            page_size: pageSize,
          },
        })
      }

      if (url === '/departments') {
        return Promise.resolve({ data: [{ id: 1, name: '制造部' }] })
      }

      return Promise.reject(new Error(`Unhandled GET ${url}`))
    })

    mockApiDelete.mockImplementation(() => {
      deletedOnPage2 = true
      return Promise.resolve({ data: { message: '删除成功' } })
    })
  })

  it('requests paged users on first load and keeps backend total text', async () => {
    renderPage()

    await screen.findByRole('heading', { name: '用户管理' })
    await screen.findByText('Alice')

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 1, page_size: 20 },
      })
    })

    expect(screen.getByText('显示第 1 至 20 条，共 21 条记录')).toBeInTheDocument()
  })

  it('requests next page params when table page changes', async () => {
    renderPage()

    await screen.findByText('Alice')
    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 2, page_size: 20 },
      })
    })

    await screen.findByText('Bob')
  })

  it('resets pagination to first page when clicking reset', async () => {
    renderPage()

    await screen.findByText('Alice')
    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 2, page_size: 20 },
      })
    })

    const callsBeforeReset = mockApiGet.mock.calls.length
    fireEvent.click(screen.getByRole('button', { name: '重置' }))

    await waitFor(() => {
      expect(mockApiGet.mock.calls.length).toBeGreaterThan(callsBeforeReset)
    })

    expect(mockApiGet.mock.calls.slice(callsBeforeReset)).toContainEqual([
      '/users',
      { params: { page: 1, page_size: 20 } },
    ])
  })

  it('keeps current page when deleting a user', async () => {
    renderPage()

    await screen.findByText('Alice')
    fireEvent.click(screen.getByTitle('2'))
    await screen.findByText('Bob')

    const deleteButtons = screen.getAllByRole('button', { name: 'delete' })
    fireEvent.click(deleteButtons[0])
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockApiDelete).toHaveBeenCalledWith('/users/2')
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 2, page_size: 20 },
      })
    })
  })

  it('falls back to previous page when current page becomes empty after delete', async () => {
    renderPage()

    await screen.findByText('Alice')
    fireEvent.click(screen.getByTitle('2'))
    await screen.findByText('Bob')

    const deleteButtons = screen.getAllByRole('button', { name: 'delete' })
    fireEvent.click(deleteButtons[0])
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 2, page_size: 20 },
      })
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/users', {
        params: { page: 1, page_size: 20 },
      })
    })
  })
})
