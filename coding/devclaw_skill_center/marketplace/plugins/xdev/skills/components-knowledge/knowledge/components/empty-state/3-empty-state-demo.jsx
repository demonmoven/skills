import { EmptyState, Toast } from '@coze-arch/coze-design';
import { IconCozWarningCircle } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-16">
    <div>
      <h4>带按钮的空状态</h4>
      <EmptyState
        title="加载失败"
        buttonText="重试"
        icon={<IconCozWarningCircle />}
        description="数据加载失败，请重试"
        onButtonClick={() => {
          Toast.info({ content: '点击重试' });
        }}
      />
    </div>
    <div>
      <h4>带额外内容的空状态</h4>
      <EmptyState
        buttonText="重试"
        extra={
          <div className="py-8px coz-fg-primary text-lg">您可以稍后再试</div>
        }
        icon={<IconCozWarningCircle />}
        title="加载失败"
        description="数据加载失败，请重试"
        onButtonClick={() => {
          Toast.info({ content: '点击重试' });
        }}
      />
    </div>
  </div>
);

export default Demo;