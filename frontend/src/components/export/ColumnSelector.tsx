import { useMemo } from 'react';
import { Checkbox, Collapse, Typography, Tag, Space } from 'antd';

const { Text } = Typography;

export interface ColumnDef {
  key: string;
  label: string;
  group: string;
  field_type?: string;
  production_line_id?: number;
}

interface Props {
  builtinFields: ColumnDef[];
  customFields: ColumnDef[];
  selectedKeys: string[];
  onChange: (keys: string[]) => void;
  productionLineNames?: Record<number, string>;
}

const ColumnSelector = ({
  builtinFields,
  customFields,
  selectedKeys,
  onChange,
  productionLineNames = {},
}: Props) => {
  const selectedSet = useMemo(() => new Set(selectedKeys), [selectedKeys]);

  // 按分组组织内置字段
  const builtinGroups = useMemo(() => {
    const groups = new Map<string, ColumnDef[]>();
    builtinFields.forEach((field) => {
      const arr = groups.get(field.group) || [];
      arr.push(field);
      groups.set(field.group, arr);
    });
    return groups;
  }, [builtinFields]);

  // 按产线分组自定义字段
  const customFieldGroups = useMemo(() => {
    const groups = new Map<string, ColumnDef[]>();
    customFields.forEach((field) => {
      const lineName =
        productionLineNames[field.production_line_id || 0] ||
        `产线 ${field.production_line_id}`;
      const arr = groups.get(lineName) || [];
      arr.push(field);
      groups.set(lineName, arr);
    });
    return groups;
  }, [customFields, productionLineNames]);

  const handleToggle = (key: string, checked: boolean) => {
    if (checked) {
      onChange([...selectedKeys, key]);
    } else {
      onChange(selectedKeys.filter((k) => k !== key));
    }
  };

  const handleGroupToggle = (groupKeys: string[], checked: boolean) => {
    if (checked) {
      const newKeys = [...selectedKeys];
      groupKeys.forEach((k) => {
        if (!newKeys.includes(k)) newKeys.push(k);
      });
      onChange(newKeys);
    } else {
      const groupSet = new Set(groupKeys);
      onChange(selectedKeys.filter((k) => !groupSet.has(k)));
    }
  };

  const handleSelectAll = () => {
    const allKeys = [
      ...builtinFields.map((f) => f.key),
      ...customFields.map((f) => f.key),
    ];
    onChange(allKeys);
  };

  const handleDeselectAll = () => {
    onChange([]);
  };

  const allKeys = [
    ...builtinFields.map((f) => f.key),
    ...customFields.map((f) => f.key),
  ];
  const allSelected =
    allKeys.length > 0 && allKeys.every((k) => selectedSet.has(k));

  const renderGroup = (groupName: string, fields: ColumnDef[]) => {
    const groupKeys = fields.map((f) => f.key);
    const allGroupSelected = groupKeys.every((k) => selectedSet.has(k));
    const someGroupSelected =
      groupKeys.some((k) => selectedSet.has(k)) && !allGroupSelected;

    return (
      <div key={groupName} style={{ marginBottom: 8 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '4px 0',
          }}
        >
          <Checkbox
            indeterminate={someGroupSelected}
            checked={allGroupSelected}
            onChange={(e) => handleGroupToggle(groupKeys, e.target.checked)}
          >
            <Text strong style={{ fontSize: 13 }}>
              {groupName}
            </Text>
          </Checkbox>
          <Text type="secondary" style={{ fontSize: 11 }}>
            {groupKeys.filter((k) => selectedSet.has(k)).length}/
            {groupKeys.length}
          </Text>
        </div>
        <div style={{ paddingLeft: 24, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {fields.map((field) => (
            <div
              key={field.key}
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '2px 0',
                width: '100%',
              }}
            >
              <Checkbox
                checked={selectedSet.has(field.key)}
                onChange={(e) => handleToggle(field.key, e.target.checked)}
              >
                <Space size={4}>
                  <span style={{ fontSize: 13 }}>{field.label}</span>
                  {field.field_type && (
                    <Tag
                      style={{
                        fontSize: 10,
                        lineHeight: '16px',
                        padding: '0 4px',
                        margin: 0,
                      }}
                    >
                      {field.field_type}
                    </Tag>
                  )}
                </Space>
              </Checkbox>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const collapseItems = [
    {
      key: 'builtin',
      label: (
        <Space>
          <Text strong>内置字段</Text>
          <Tag color="blue">{builtinFields.length}</Tag>
        </Space>
      ),
      children: (
        <div>
          {[...builtinGroups.entries()].map(([group, fields]) =>
            renderGroup(group, fields),
          )}
        </div>
      ),
    },
    ...(customFields.length > 0
      ? [
          {
            key: 'custom',
            label: (
              <Space>
                <Text strong>自定义字段</Text>
                <Tag color="purple">{customFields.length}</Tag>
              </Space>
            ),
            children: (
              <div>
                {[...customFieldGroups.entries()].map(([group, fields]) =>
                  renderGroup(group, fields),
                )}
              </div>
            ),
          },
        ]
      : []),
  ];

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 12,
        }}
      >
        <Text type="secondary" style={{ fontSize: 12 }}>
          已选 {selectedKeys.length} / {allKeys.length} 列
        </Text>
        <Space size={4}>
          <a
            onClick={handleSelectAll}
            style={{ fontSize: 12, color: allSelected ? '#d9d9d9' : '#1890ff' }}
          >
            全选
          </a>
          <span style={{ color: '#d9d9d9' }}>|</span>
          <a
            onClick={handleDeselectAll}
            style={{
              fontSize: 12,
              color: selectedKeys.length === 0 ? '#d9d9d9' : '#1890ff',
            }}
          >
            清空
          </a>
        </Space>
      </div>

      <Collapse
        items={collapseItems}
        defaultActiveKey={['builtin', 'custom']}
        size="small"
        bordered={false}
        style={{ background: 'transparent' }}
      />
    </div>
  );
};

export default ColumnSelector;
