import { AutoComplete } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <AutoComplete
            data={[1, 2, 3, 4]}
            size="small"
            placeholder={'small'}
            style={{ width: 200 }}
        ></AutoComplete>
        <br />
        <br />
        <AutoComplete
            data={[1, 2, 3, 4]}
            size="default"
            placeholder={'default'}
            style={{ width: 200 }}
        ></AutoComplete>
        <br />
        <br />
        <AutoComplete
            data={[1, 2, 3, 4]}
            size="large"
            placeholder={'large'}
            style={{ width: 200 }}
        ></AutoComplete>
    </div>
);

export default Demo;
