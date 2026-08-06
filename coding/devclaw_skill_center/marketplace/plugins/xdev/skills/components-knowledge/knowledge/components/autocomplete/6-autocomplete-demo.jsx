import { AutoComplete } from '@coze-arch/coze-design';

const Demo = () => (
    <AutoComplete data={[1, 2, 3, 4]} placeholder={'禁用下拉菜单'} disabled style={{ width: 200 }}></AutoComplete>
);

export default Demo;
