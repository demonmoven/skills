import React from 'react';
import { Transfer, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const data = Array.from({ length: 30 }, (v, i) => {
        return {
            label: `选项名称 ${i}`,
            value: i,
            disabled: false,
            key: i,
        };
    });

    const renderSourceHeader = (props) => {
        const { num, showButton, allChecked, onAllClick } = props;
        return <div style={{ margin: '10px 0 0 10px', height: 24, display: 'flex', alignItems: 'center' }}>
            <span>共 {num} 项</span>
            {showButton && <Button
                theme="borderless"
                type="tertiary"
                size="small" 
                onClick={onAllClick}>{ allChecked ? '取消全选' : '全选' }</Button>}
        </div>;
    };

    const renderSelectedHeader = (props) => {
        const { num, showButton, onClear } = props;
        return <div style={{ margin: '10px 0 0 10px', height: 24, display: 'flex', alignItems: 'center' }}>
            <span>{num} 项已选</span>
            {showButton && <Button
                theme="borderless"
                type="tertiary"
                size="small"
                onClick={onClear}>清空</Button>}
        </div>;
    };

    return (
        <Transfer
            style={{ width: 568, height: 416 }}
            dataSource={data}
            renderSourceHeader={renderSourceHeader}
            renderSelectedHeader={renderSelectedHeader}
        />
    );
};

export default Demo;
