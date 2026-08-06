import { EmptyState } from '@coze-arch/coze-design';
import {
  IconCozIllusAdd,
  IconCozIllusAddDark,
} from '@coze-arch/coze-design/illustrations';

const Demo = () => (
  <EmptyState
    size="full_screen"
    icon={<IconCozIllusAdd />}
    darkModeIcon={<IconCozIllusAddDark />}
    title="出现了一个错误"
    description="请稍后再试"
  />
);

export default Demo;