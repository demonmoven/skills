import { Checkbox } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <h4>水平排列</h4>
        <Checkbox.Group direction="horizontal">
          <Checkbox value={1}>选项A</Checkbox>
          <Checkbox value={2}>选项B</Checkbox>
          <Checkbox value={3}>选项C</Checkbox>
        </Checkbox.Group>
      </div>

      <div>
        <h4>垂直排列</h4>
        <Checkbox.Group direction="vertical">
          <Checkbox value={1}>选项A</Checkbox>
          <Checkbox value={2}>选项B</Checkbox>
          <Checkbox value={3}>选项C</Checkbox>
        </Checkbox.Group>
      </div>
    </div>
  );
};

export default Demo;