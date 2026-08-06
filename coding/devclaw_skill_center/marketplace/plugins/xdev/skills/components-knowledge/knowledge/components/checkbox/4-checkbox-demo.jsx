import { Checkbox } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-2">
      <Checkbox disabled>禁用状态</Checkbox>
      <Checkbox disabled defaultChecked>
        已选禁用状态
      </Checkbox>
      <Checkbox disabled indeterminate>
        半选禁用状态
      </Checkbox>
    </div>
  );
};

export default Demo;