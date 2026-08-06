import { EmptyState } from '@coze-arch/coze-design';
import {
  IconCozIllusAdd,
  IconCozIllusAddDark,
} from '@coze-arch/coze-design/illustrations';

const Demo = () => (
  <EmptyState
    icon={<IconCozIllusAdd />}
    darkModeIcon={<IconCozIllusAddDark />}
    title="开始创建"
    description="点击下方按钮开始创建内容"
    buttonText="创建"
  />
);

export default Demo;