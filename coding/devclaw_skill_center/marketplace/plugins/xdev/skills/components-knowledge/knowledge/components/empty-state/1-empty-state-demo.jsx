import { EmptyState } from '@coze-arch/coze-design';
import { IconCozWarningCircle } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-16">
    <div>
      <h4>默认尺寸（32x32）</h4>
      <EmptyState
        size="default"
        icon={<IconCozWarningCircle className="coz-fg-dim text-32px" />}
        title="暂无数据"
        description="请添加数据后查看"
      />
    </div>
    <div>
      <h4>大尺寸（48x48）</h4>
      <EmptyState
        size="large"
        icon={<IconCozWarningCircle className="coz-fg-dim text-48px" />}
        title="暂无数据"
        description="请添加数据后查看"
      />
    </div>
  </div>
);

export default Demo;