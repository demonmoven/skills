import { SingleSelect } from '@coze-arch/coze-design';
import {
  IconCozThumbsup,
  IconCozThumbsupFill,
  IconCozThumbdown,
  IconCozThumbdownFill,
} from '@coze-arch/coze-design/icons';

const { SingleSelectLabel } = SingleSelect;

const Demo = () => (
  <SingleSelect
    defaultValue="1"
    options={[
      {
        value: '1',
        label: (
          <SingleSelectLabel
            icon={<IconCozThumbsup />}
            activeIcon={<IconCozThumbsupFill />}
            text="1,024"
          />
        ),
      },
      {
        value: '2',
        label: (
          <SingleSelectLabel
            icon={<IconCozThumbdown />}
            activeIcon={<IconCozThumbdownFill />}
            text="206"
          />
        ),
      },
    ]}
  />
);

export default Demo;