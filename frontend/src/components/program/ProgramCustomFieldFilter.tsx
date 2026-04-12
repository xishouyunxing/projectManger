import { Input, Select } from 'antd'

type ProgramCustomFieldDefinition = {
  id: number
  name: string
  field_type: 'text' | 'select'
  options_json: string
  sort_order: number
  enabled: boolean
}

type ProgramCustomFieldFilterProps = {
  fields: ProgramCustomFieldDefinition[]
  values: Record<string, string>
  onChange: (fieldId: string, value: string) => void
}

const parseOptions = (optionsJson: string) => {
  if (!optionsJson) {
    return []
  }

  try {
    const parsed = JSON.parse(optionsJson)
    return Array.isArray(parsed) ? parsed.filter((option): option is string => typeof option === 'string') : []
  } catch {
    return []
  }
}

const ProgramCustomFieldFilter = ({ fields, values, onChange }: ProgramCustomFieldFilterProps) => {
  if (fields.length === 0) {
    return null
  }

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(192px, 192px))',
        gap: '14px 16px',
      }}
    >
      {fields.map((field) => (
        <div key={field.id} style={{ display: 'flex', flexDirection: 'column', gap: '5px', width: '192px' }}>
          <label
            style={{
              color: '#94A3B8',
              fontSize: '10px',
              fontWeight: 700,
              letterSpacing: '0.12em',
              textTransform: 'uppercase',
              paddingLeft: '4px',
              lineHeight: 1.4,
            }}
          >
            {field.name}
          </label>
          {field.field_type === 'select' ? (
            <Select
              allowClear
              value={values[String(field.id)] || undefined}
              onChange={(value) => onChange(String(field.id), value ?? '')}
              style={{ width: '100%' }}
            >
              {parseOptions(field.options_json).map((option) => (
                <Select.Option key={option} value={option}>
                  {option}
                </Select.Option>
              ))}
            </Select>
          ) : (
            <Input
              value={values[String(field.id)] || ''}
              onChange={(event) => onChange(String(field.id), event.target.value)}
              style={{ width: '100%' }}
            />
          )}
        </div>
      ))}
    </div>
  )
}

export default ProgramCustomFieldFilter
