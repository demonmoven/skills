import { Table } from '@coze-arch/coze-design';
import { useState, useRef } from 'react';

const Demo = () => {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const pageRef = useRef(1);

  const tableRef = useRef(null);

  const columns = [
    {
      title: '序号',
      dataIndex: 'id',
    },
    {
      title: '名称',
      dataIndex: 'name',
    },
    {
      title: '内容',
      dataIndex: 'content',
    },
  ];

  // 模拟加载数据
  const loadData = () => {
    if (loading) return;

    setLoading(true);

    // 模拟异步请求
    setTimeout(() => {
      const newPage = pageRef.current;
      const newData = Array(10)
        .fill(0)
        .map((_, index) => ({
          key: `${newPage}-${index}`,
          id: (newPage - 1) * 10 + index + 1,
          name: `项目 ${(newPage - 1) * 10 + index + 1}`,
          content: `这是第 ${newPage} 页的第 ${index + 1} 条数据`,
        }));

      setData(prev => [...prev, ...newData]);
      setLoading(false);
      pageRef.current += 1;

      // 模拟数据加载完毕
      if (pageRef.current > 3) {
        setHasMore(false);
      }
    }, 1000);
  };

  // 初始加载
  if (data.length === 0 && !loading && hasMore) {
    loadData();
  }

  return (
    <div style={{ height: 300 }}>
      <Table
        ref={tableRef}
        tableProps={{
          columns,
          dataSource: data,
          loading,
        }}
        enableLoad={true}
        loadMode="cursor"
        hasMore={hasMore}
        onLoad={loadData}
        offsetY={0}
      />
    </div>
  );
};

export default Demo;