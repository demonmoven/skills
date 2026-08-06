import { LoadingButton } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <LoadingButton loadingToast="正在加载...">加载按钮</LoadingButton>
    <LoadingButton loadingToast={{ content: '自定义加载提示', duration: 0 }}>
      自定义提示
    </LoadingButton>
  </div>
);

export default Demo;