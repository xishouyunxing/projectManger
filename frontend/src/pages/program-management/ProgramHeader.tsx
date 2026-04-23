import { Button, Space, Typography } from 'antd';
import { FileExcelOutlined, FolderAddOutlined, PlusOutlined } from '@ant-design/icons';

const { Title } = Typography;

interface ProgramHeaderProps {
  batchImportSupported: boolean;
  onExportExcel: () => void;
  onBatchImport: () => void;
  onAddProgram: () => void;
}

const ProgramHeader = ({
  batchImportSupported,
  onExportExcel,
  onBatchImport,
  onAddProgram,
}: ProgramHeaderProps) => {
  return (
    <div className="management-page-header">
      <div>
        <div className="management-page-breadcrumb">
          <span>制造</span>
          <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
          <span className="active">程序管理</span>
        </div>
        <Title level={2} className="management-page-title">
          程序管理
        </Title>
      </div>
      <Space>
        <Button
          icon={<FileExcelOutlined />}
          onClick={onExportExcel}
          style={{ height: '44px', borderRadius: '8px', fontWeight: 600, padding: '0 16px' }}
        >
          导出Excel
        </Button>
        <Button
          icon={<FolderAddOutlined />}
          onClick={onBatchImport}
          disabled={!batchImportSupported}
          style={{ height: '44px', borderRadius: '8px', fontWeight: 600, padding: '0 16px' }}
        >
          批量导入
        </Button>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={onAddProgram}
          style={{
            background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
            border: 'none',
            boxShadow: '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)',
            borderRadius: '8px',
            height: '44px',
            padding: '0 24px',
            fontWeight: 600,
            fontSize: '16px',
          }}
        >
          新建程序
        </Button>
      </Space>
    </div>
  );
};

export default ProgramHeader;
