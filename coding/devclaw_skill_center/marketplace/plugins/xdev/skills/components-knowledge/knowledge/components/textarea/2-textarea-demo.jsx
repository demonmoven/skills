import { TextArea } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>自动调整高度</h4>
      <TextArea autosize placeholder="输入内容会自动调整高度" />
    </div>
    <div>
      <h4>限制最小/最大行数</h4>
      <TextArea
        autosize={{ minRows: 2, maxRows: 6 }}
        placeholder="最小 2 行，最大 6 行"
      />
    </div>
  </div>
);

export default Demo;