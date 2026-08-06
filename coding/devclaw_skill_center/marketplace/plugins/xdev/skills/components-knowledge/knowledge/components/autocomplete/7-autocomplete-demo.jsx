import { AutoComplete } from '@coze-arch/coze-design';

const Demo = () => (
    <>
        <AutoComplete defaultValue="ies" validateStatus="warning"></AutoComplete>
        <br />
        <br />
        <AutoComplete defaultValue="ies" validateStatus="error"></AutoComplete>
        <br />
        <br />
        <AutoComplete defaultValue="ies"></AutoComplete>
    </>
);

export default Demo;
