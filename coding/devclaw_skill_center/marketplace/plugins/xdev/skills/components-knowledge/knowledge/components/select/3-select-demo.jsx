import { Select } from '@coze-arch/coze-design';

const Demo = () => (
  <Select placeholder="请选择" style={{ width: '200px' }} filter>
    <Select.OptGroup label="亚洲">
      <Select.Option value="cn">中国</Select.Option>
      <Select.Option value="jp">日本</Select.Option>
      <Select.Option value="kr">韩国</Select.Option>
    </Select.OptGroup>
    <Select.OptGroup label="欧洲">
      <Select.Option value="uk">英国</Select.Option>
      <Select.Option value="fr">法国</Select.Option>
      <Select.Option value="de">德国</Select.Option>
    </Select.OptGroup>
  </Select>
);

export default Demo;