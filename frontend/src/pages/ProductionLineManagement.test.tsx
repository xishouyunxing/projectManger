import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ProductionLineManagement from './ProductionLineManagement'
import ProductionLineCustomFieldManager from '../components/production-line/ProductionLineCustomFieldManager'
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

const mockApiGet = api.get as Mock
const mockApiPost = api.post as Mock
const mockApiPut = api.put as Mock
const mockApiDelete = api.delete as Mock

const renderPage = () =>
  render(
    <MemoryRouter>
      <ProductionLineManagement />
    </MemoryRouter>
  )

const lineFields = [
  {
    id: 11,
    name: '状态',
    field_type: 'select',
    options_json: '["试产","量产"]',
    sort_order: 1,
    enabled: true,
  },
  {
    id: 12,
    name: '备注',
    field_type: 'text',
    options_json: '',
    sort_order: 2,
    enabled: true,
  },
]

describe('ProductionLineManagement custom field integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mockApiGet.mockImplementation((url: string, config?: { params?: { page?: number; page_size?: number } }) => {
      if (url === '/production-lines') {
        const page = config?.params?.page ?? 1
        const pageSize = config?.params?.page_size ?? 20
        return Promise.resolve({
          data: {
            items: [
              {
                id: page === 1 ? 1 : 2,
                name: page === 1 ? 'Line A' : 'Line B',
                code: page === 1 ? 'LA' : 'LB',
                type: 'upper',
                status: 'active',
                description: 'Line desc',
                created_at: '2026-01-01T08:00:00.000Z',
                updated_at: '2026-01-01T08:00:00.000Z',
              },
            ],
            total: 40,
            page,
            page_size: pageSize,
          },
        })
      }

      if (url === '/production-lines/1/custom-fields') {
        return Promise.resolve({ data: lineFields })
      }

      return Promise.reject(new Error(`Unhandled GET ${url}`))
    })

    mockApiPost.mockResolvedValue({ data: {} })
    mockApiPut.mockResolvedValue({ data: {} })
    mockApiDelete.mockResolvedValue({ data: { message: '删除成功' } })
  })

  it('requests paged list data on first load and opens custom field manager from actions', async () => {
    renderPage()

    await screen.findByText('生产线管理')
    expect(screen.getByText('Line A')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /新建生产线/ })).toBeInTheDocument()

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/production-lines', {
        params: { page: 1, page_size: 20 },
      })
    })

    fireEvent.click(screen.getByRole('button', { name: '管理字段' }))

    const dialog = await screen.findByRole('dialog', { name: 'Line A 字段管理' })
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/production-lines/1/custom-fields')
    })
    await waitFor(() => {
      expect(within(dialog).getByDisplayValue('状态')).toBeInTheDocument()
      expect(within(dialog).getByDisplayValue('备注')).toBeInTheDocument()
    })
  })

  it('triggers page query params when changing table page', async () => {
    renderPage()

    await screen.findByText('Line A')

    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/production-lines', {
        params: { page: 2, page_size: 20 },
      })
    })

    await screen.findByText('Line B')
  })

  it('resets pagination to first page when clicking reset', async () => {
    renderPage()

    await screen.findByText('Line A')
    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/production-lines', {
        params: { page: 2, page_size: 20 },
      })
    })

    const callsBeforeReset = mockApiGet.mock.calls.length
    fireEvent.click(screen.getByRole('button', { name: '重置' }))

    await waitFor(() => {
      expect(mockApiGet.mock.calls.length).toBeGreaterThan(callsBeforeReset)
    })

    expect(mockApiGet.mock.calls.slice(callsBeforeReset)).toContainEqual([
      '/production-lines',
      { params: { page: 1, page_size: 20 } },
    ])
  })

  it('adds a new custom field row in the manager', async () => {
    render(
      <ProductionLineCustomFieldManager
        open
        productionLine={{ id: 1, name: 'Line A' }}
        onClose={vi.fn()}
      />
    )

    const dialog = await screen.findByRole('dialog', { name: 'Line A 字段管理' })
    await within(dialog).findByDisplayValue('状态')

    fireEvent.click(within(dialog).getByRole('button', { name: 'plus新增字段' }))

    await waitFor(() => {
      const emptyInputs = within(dialog)
        .getAllByPlaceholderText('字段名称')
        .filter((input) => (input as HTMLInputElement).value === '')
      expect(emptyInputs.length).toBeGreaterThan(0)
    })
  })

  it('updates an existing select field with normalized options', async () => {
    renderPage()

    fireEvent.click(await screen.findByRole('button', { name: '管理字段' }))

    const dialog = await screen.findByRole('dialog', { name: 'Line A 字段管理' })
    const textarea = await within(dialog).findByPlaceholderText('选项列表，每行一个选项')
    fireEvent.change(textarea, { target: { value: ' 试产 \n量产\n暂停 ' } })

    fireEvent.click(within(dialog).getByRole('button', { name: '保存字段' }))

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/production-lines/1/custom-fields/11', {
        name: '状态',
        field_type: 'select',
        options_json: '["试产","量产","暂停"]',
        sort_order: 1,
        enabled: true,
      })
    })
  })
})
