import { Tag } from '@coze-arch/coze-design';
import { IconCozFace } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>标准尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag
          size="small"
          onClick={() => {
            console.log('clicked');
          }}
        >
          可点击
        </Tag>
        <Tag
          size="small"
          closable
          onClose={() => {
            console.log('closed');
          }}
        >
          可关闭
        </Tag>
      </div>
    </div>
    <div>
      <h4>迷你尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag
          size="mini"
          onClick={() => {
            console.log('clicked');
          }}
        >
          可点击
        </Tag>
        <Tag
          size="mini"
          closable
          onClose={() => {
            console.log('closed');
          }}
        >
          可关闭
        </Tag>
      </div>
    </div>
  </div>
);

export default Demo;