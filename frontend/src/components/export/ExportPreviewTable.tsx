import { Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface ColumnHeader {
  key: string;
  label: string;
}

interface Props {
  columns: ColumnHeader[];
  dataSource: Record<string, any>[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
}

const statusColors: Record<string, { color: string; bg: string; dot: string }> = {
  已完成: { color: '#005BC1', bg: 'rgba(61, 137, 255, 0.20)', dot: '#005BC1' },
  进行中: { color: '#50426B', bg: 'rgba(222, 204, 253, 0.40)', dot: '#50426B' },
};

const ExportPreviewTable = ({
  columns,
  dataSource,
  loading,
  total,
  page,
  pageSize,
  onPageChange,
}: Props) => {
  const tableColumns: ColumnsType<Record<string, any>> = columns.map((col) => ({
    title: col.label,
    dataIndex: col.key,
    key: col.key,
    ellipsis: true,
    width: col.key === 'name' ? 200 : col.key === 'description' ? 250 : 140,
    render: (value: any) => {
      if (col.key === 'status' && typeof value === 'string') {
        const style = statusColors[value];
        if (style) {
          return (
            <div
              style={{
                background: style.bg,
                borderRadius: '9999px',
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px 10px',
                gap: 6,
              }}
            >
              <div
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: style.dot,
                }}
              />
              <span
                style={{
                  color: style.color,
                  fontSize: 11,
                  fontWeight: 700,
                }}
              >
                {value}
              </span>
            </div>
          );
        }
      }

      if (col.key === 'file_count' || col.key === 'version_count') {
        const num = Number(value) || 0;
        return (
          <Tag color={num > 0 ? 'blue' : 'default'}>{num}</Tag>
        );
      }

      if (value === '' || value === null || value === undefined) {
        return <Text type="secondary">-</Text>;
      }

      return <span>{String(value)}</span>;
    },
  }));

  return (
    <Table
      className="custom-table"
      columns={tableColumns}
      dataSource={dataSource}
      rowKey={(_, index) => String(index)}
      loading={loading}
      pagination={{
        current: page,
        pageSize,
        total,
        showSizeChanger: true,
        pageSizeOptions: ['10', '20', '50', '100'],
        showTotal: (t, range) =>
          `显示第 ${range[0]} 至 ${range[1]} 条，共 ${t} 条`,
        onChange: onPageChange,
      }}
      scroll={{ x: 'max-content' }}
      size="small"
      locale={{
        emptyText: (
          <div style={{ padding: '40px 0', color: '#999' }}>
            暂无数据，请调整筛选条件或选择列
          </div>
        ),
      }}
    />
  );
};

export default ExportPreviewTable;
