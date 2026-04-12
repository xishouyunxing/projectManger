import { useState, useEffect, useCallback } from 'react';
import { Modal, Input, List, Typography, Tag, Empty, Spin, Tabs } from 'antd';
import {
  SearchOutlined,
  FileTextOutlined,
  UserOutlined,
  ApartmentOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';

const { Text } = Typography;

interface SearchResult {
  programs: {
    id: number;
    name: string;
    code: string;
    line_name: string;
  }[];
  users: {
    id: number;
    name: string;
    employee_id: string;
    department: string;
  }[];
  lines: {
    id: number;
    name: string;
    code: string;
  }[];
}

interface GlobalSearchProps {
  open: boolean;
  onClose: () => void;
}

const GlobalSearch = ({ open, onClose }: GlobalSearchProps) => {
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<SearchResult | null>(null);
  const [activeTab, setActiveTab] = useState('programs');
  const navigate = useNavigate();

  const handleSearch = useCallback(async (value: string) => {
    if (!value.trim()) {
      setResult(null);
      return;
    }

    setLoading(true);
    try {
      const response = await api.get('/search', { params: { keyword: value } });
      setResult(response.data);

      // 自动切换到有结果的标签
      if (response.data.programs.length > 0) {
        setActiveTab('programs');
      } else if (response.data.users.length > 0) {
        setActiveTab('users');
      } else if (response.data.lines.length > 0) {
        setActiveTab('lines');
      }
    } catch (error) {
      console.error('Search failed:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => {
      handleSearch(keyword);
    }, 300);

    return () => clearTimeout(timer);
  }, [keyword, handleSearch]);

  const handleItemClick = (type: string, item: any) => {
    onClose();
    setKeyword('');
    setResult(null);

    switch (type) {
      case 'program':
        navigate(`/programs?keyword=${encodeURIComponent(item.name || item.code || '')}&id=${item.id}`);
        break;
      case 'user':
        navigate(`/users?keyword=${encodeURIComponent(item.name || item.employee_id || '')}&id=${item.id}`);
        break;
      case 'line':
        navigate(`/production-lines?keyword=${encodeURIComponent(item.name || item.code || '')}&id=${item.id}`);
        break;
    }
  };

  const getTotalCount = () => {
    if (!result) return 0;
    return result.programs.length + result.users.length + result.lines.length;
  };

  return (
    <Modal
      open={open}
      onCancel={() => {
        onClose();
        setKeyword('');
        setResult(null);
      }}
      footer={null}
      width={600}
      title={
        <Input
          prefix={<SearchOutlined />}
          placeholder="搜索程序、用户、生产线..."
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          size="large"
          autoFocus
          allowClear
        />
      }
      closable={false}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <Spin />
        </div>
      ) : !result || getTotalCount() === 0 ? (
        keyword ? (
          <Empty description="未找到相关结果" />
        ) : (
          <Empty description="输入关键词开始搜索" />
        )
      ) : (
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'programs',
              label: (
                <span>
                  <FileTextOutlined /> 程序 ({result.programs.length})
                </span>
              ),
              children: (
                <List
                  dataSource={result.programs}
                  renderItem={(item) => (
                    <List.Item
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleItemClick('program', item)}
                    >
                      <List.Item.Meta
                        avatar={<FileTextOutlined style={{ fontSize: '20px', color: '#1890ff' }} />}
                        title={item.name}
                        description={
                          <>
                            <Text type="secondary">{item.code}</Text>
                            {item.line_name && (
                              <Tag style={{ marginLeft: 8 }}>{item.line_name}</Tag>
                            )}
                          </>
                        }
                      />
                    </List.Item>
                  )}
                />
              ),
            },
            {
              key: 'users',
              label: (
                <span>
                  <UserOutlined /> 用户 ({result.users.length})
                </span>
              ),
              children: (
                <List
                  dataSource={result.users}
                  renderItem={(item) => (
                    <List.Item
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleItemClick('user', item)}
                    >
                      <List.Item.Meta
                        avatar={<UserOutlined style={{ fontSize: '20px', color: '#52c41a' }} />}
                        title={item.name}
                        description={
                          <>
                            <Text type="secondary">{item.employee_id}</Text>
                            {item.department && (
                              <Tag style={{ marginLeft: 8 }}>{item.department}</Tag>
                            )}
                          </>
                        }
                      />
                    </List.Item>
                  )}
                />
              ),
            },
            {
              key: 'lines',
              label: (
                <span>
                  <ApartmentOutlined /> 生产线 ({result.lines.length})
                </span>
              ),
              children: (
                <List
                  dataSource={result.lines}
                  renderItem={(item) => (
                    <List.Item
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleItemClick('line', item)}
                    >
                      <List.Item.Meta
                        avatar={<ApartmentOutlined style={{ fontSize: '20px', color: '#faad14' }} />}
                        title={item.name}
                        description={<Text type="secondary">{item.code}</Text>}
                      />
                    </List.Item>
                  )}
                />
              ),
            },
          ]}
        />
      )}
    </Modal>
  );
};

export default GlobalSearch;
