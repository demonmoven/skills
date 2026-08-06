import { useState } from 'react';
import { AutoComplete } from '@coze-arch/coze-design';

const Demo = () => {
    const [data, setData] = useState([]);

    const change = (input) => {
        let newData = ['gmail.com', '163.com', 'qq.com'].map(domain => `${input}@${domain}`);
        if (!input) {
            newData = [];
        }
        setData(newData);
    };
    return (
        <div>
            <AutoComplete
                data={data}
                position="top"
                onSearch={change}
                placeholder="选项菜单在上方显示"
                style={{ width: 200, margin: 10 }}
            ></AutoComplete>
            <AutoComplete
                data={data}
                position="rightTop"
                onSearch={change}
                placeholder="选项菜单在右侧显示"
                style={{ width: 200, margin: 10 }}
            ></AutoComplete>
        </div>
    );
};

export default Demo;
