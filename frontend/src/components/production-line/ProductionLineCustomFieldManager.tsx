import { useEffect, useRef, useState } from 'react'
import {
  Button,
  Divider,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  message,
} from 'antd'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import api from '../../services/api'

type ProductionLineCustomField = {
  id?: number
  name: string
  field_type: 'text' | 'select'
  options_json: string
  sort_order: number
  enabled: boolean
}

type ProductionLineRecord = {
  id: number
  name: string
}

type Props = {
  open: boolean
  productionLine: ProductionLineRecord | null
  onClose: () => void
}

const emptyField = (sortOrder: number): ProductionLineCustomField => ({
  name: '',
  field_type: 'text',
  options_json: '',
  sort_order: sortOrder,
  enabled: true,
})

const normalizeOptions = (value: string) => {
  const options = value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)

  return options.length > 0 ? JSON.stringify(options) : ''
}

const optionsToTextarea = (value: string) => {
  if (!value) {
    return ''
  }

  try {
    const parsed = JSON.parse(value)
    if (Array.isArray(parsed)) {
      return parsed.join('\n')
    }
  } catch {
    return value
  }

  return value
}

const ProductionLineCustomFieldManager = ({ open, productionLine, onClose }: Props) => {
  const [fields, setFields] = useState<ProductionLineCustomField[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const requestIdRef = useRef(0)

  useEffect(() => {
    if (!open || !productionLine) {
      setFields([])
      setLoading(false)
      return
    }

    const requestId = requestIdRef.current + 1
    requestIdRef.current = requestId
    setFields([])

    const loadFields = async () => {
      setLoading(true)
      try {
        const response = await api.get(`/production-lines/${productionLine.id}/custom-fields`)
        if (requestIdRef.current !== requestId) {
          return
        }
        setFields(response.data)
      } catch (error) {
        if (requestIdRef.current !== requestId) {
          return
        }
        console.error('Failed to load custom fields:', error)
        message.error('加载自定义字段失败')
      } finally {
        if (requestIdRef.current === requestId) {
          setLoading(false)
        }
      }
    }

    loadFields()
  }, [open, productionLine])

  const updateField = (index: number, patch: Partial<ProductionLineCustomField>) => {
    setFields((current) =>
      current.map((field, fieldIndex) =>
        fieldIndex === index
          ? {
              ...field,
              ...patch,
            }
          : field
      )
    )
  }

  const addField = () => {
    if (loading) {
      return
    }

    setFields((current) => [...current, emptyField(current.length + 1)])
  }

  const deleteField = async (index: number) => {
    const field = fields[index]

    if (!field) {
      return
    }

    if (loading) {
      return
    }

    if (!field.id) {
      setFields((current) => current.filter((_, fieldIndex) => fieldIndex !== index))
      return
    }

    try {
      await api.delete(`/production-lines/${productionLine?.id}/custom-fields/${field.id}`)
      setFields((current) => current.filter((_, fieldIndex) => fieldIndex !== index))
      message.success('删除成功')
    } catch (error) {
      console.error('Failed to delete custom field:', error)
      message.error('删除自定义字段失败')
    }
  }

  const saveFields = async () => {
    if (!productionLine) {
      return
    }

    if (loading) {
      return
    }

    setSaving(true)

    try {
      const savedFields: ProductionLineCustomField[] = []

      for (let index = 0; index < fields.length; index += 1) {
        const field = fields[index]
        const payload = {
          name: field.name.trim(),
          field_type: field.field_type,
          options_json:
            field.field_type === 'select' ? normalizeOptions(field.options_json) : '',
          sort_order: index + 1,
          enabled: field.enabled,
        }

        if (field.id) {
          const response = await api.put(
            `/production-lines/${productionLine.id}/custom-fields/${field.id}`,
            payload
          )
          savedFields.push(response.data)
        } else {
          const response = await api.post(`/production-lines/${productionLine.id}/custom-fields`, payload)
          savedFields.push(response.data)
        }
      }

      setFields(savedFields)
      message.success('字段保存成功')
    } catch (error) {
      console.error('Failed to save custom fields:', error)
      message.error('保存自定义字段失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      destroyOnHidden
      title={productionLine ? `${productionLine.name} 字段管理` : '字段管理'}
      open={open}
      onCancel={onClose}
      onOk={saveFields}
      okText="保存字段"
      cancelText="关闭"
      width={720}
      confirmLoading={saving}
    >
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Button icon={<PlusOutlined />} onClick={addField} disabled={loading || saving}>
          新增字段
        </Button>
        <Divider style={{ margin: 0 }} />
        {loading ? <div>加载中...</div> : null}
        {fields.map((field, index) => (
          <div
            key={field.id ?? `new-${index}`}
            style={{ border: '1px solid #f0f0f0', borderRadius: 8, padding: 16 }}
          >
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <Space align="start" style={{ width: '100%', justifyContent: 'space-between' }}>
                <Input
                  placeholder="字段名称"
                  value={field.name}
                  onChange={(event) => updateField(index, { name: event.target.value })}
                  disabled={loading || saving}
                />
                <Select
                  value={field.field_type}
                  style={{ width: 140 }}
                  disabled={loading || saving}
                  onChange={(value) =>
                    updateField(index, {
                      field_type: value,
                      options_json: value === 'select' ? field.options_json : '',
                    })
                  }
                  options={[
                    { value: 'text', label: '文本' },
                    { value: 'select', label: '下拉选项' },
                  ]}
                />
                <Switch
                  checked={field.enabled}
                  checkedChildren="启用"
                  unCheckedChildren="停用"
                  disabled={loading || saving}
                  onChange={(checked) => updateField(index, { enabled: checked })}
                />
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  aria-label="删除字段"
                  disabled={loading || saving}
                  onClick={() => deleteField(index)}
                />
              </Space>
              {field.field_type === 'text' ? (
                <Input placeholder="示例输入" readOnly />
              ) : (
                <Input.TextArea
                  placeholder="选项列表，每行一个选项"
                  rows={4}
                  value={optionsToTextarea(field.options_json)}
                  disabled={loading || saving}
                  onChange={(event) =>
                    updateField(index, { options_json: event.target.value })
                  }
                />
              )}
            </Space>
          </div>
        ))}
        {!loading && fields.length === 0 ? <div>暂无字段，点击“新增字段”开始配置。</div> : null}
      </Space>
    </Modal>
  )
}

export default ProductionLineCustomFieldManager
