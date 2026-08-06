import { EmptyState } from '@coze-arch/coze-design';
import { IconCozIllusAdd } from '@coze-arch/coze-design/illustrations';

const Demo = () => (
  <EmptyState
    size="full_screen"
    icon={<IconCozIllusAdd />}
    title="开始创建"
    description="点击下方按钮开始创建内容"
    buttonText="创建"
    onButtonClick={() => {}}
  />
);

export default Demo;