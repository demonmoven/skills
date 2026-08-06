import { useState } from 'react';
import { AutoComplete } from '@coze-arch/coze-design';
import { IconCozMagnifier, IconCozCheckMark, IconCozDocument, IconCozTable, IconCozInputNumber, IconCozButton } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let initList = [
        { value: 'select', label: '选择器', icon: <IconCozCheckMark/> },
        { value: 'input', label: '输入框', icon: <IconCozInputNumber/> },
        { value: 'form', label: '表单', icon: <IconCozDocument /> },
        { value: 'button', label: '按钮', icon: <IconCozButton /> },
        { value: 'table', label: '表格', icon: <IconCozTable /> },
    ];

    const [loading, setLoading] = useState(false);
    const [list, setList] = useState(initList);

    const handleSearch = (inputValue) => {
        setLoading(true);
        let newList = initList;
        if (inputValue) {
            newList = list.filter(item => item.value.includes(inputValue));
        }
        setTimeout(() => {
            setList(newList);
            setLoading(false);
        }, 1000);
    };

    const debounce = (fn, delay) => {
        let timer;
        return function(...args) {
            clearTimeout(timer);
            timer = setTimeout(() => fn.apply(this, args), delay);
        };
    };

    const search = debounce(handleSearch, 200);

    const handleSelect = (value) => {
        console.log(value);
    };

    const renderItem = (item) => {
        return (
            <div style={{ display: 'flex', alignItems: 'center' }}>
                <div style={{ fontSize: 32 }}>{item.icon}</div>
                <div style={{ marginLeft: 12 }}>
                    <p>{item.value}</p>
                    <p>{item.label}</p>
                </div>
            </div>
        );
    };

    const renderSelectedItem = (item) => {
        // 注意:与其他组件如Select不同,此处只能返回String类型的值,不能返回ReactNode
        return item.value;
    };

    return (
        <AutoComplete
            data={list}
            style={{ width: 250 }}
            prefix={<IconCozMagnifier />}
            onSearch={search}
            loading={loading}
            renderItem={renderItem}
            renderSelectedItem={renderSelectedItem}
            onSelect={handleSelect}
        ></AutoComplete>
    );
};

export default Demo;
