import React from 'react';
import { ScrollList, ScrollItem, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [selectIndex1, setSelectIndex1] = React.useState(1);
    const [selectIndex2, setSelectIndex2] = React.useState(1);
    const [selectIndex3, setSelectIndex3] = React.useState(1);

    const ampms = [
        {
            value: '上午',
        },
        {
            value: '下午',
        },
    ];

    const hours = new Array(12).fill(0).map((itm, index) => {
        return {
            value: index + 1,
        };
    });

    const minutes = new Array(60).fill(0).map((itm, index) => {
        return {
            value: index,
            disabled: Math.random() > 0.5 ? true : false,
        };
    });

    const onSelectAP = (data) => {
        setSelectIndex1(data.index);
    };

    const onSelectHour = (data) => {
        console.log('You have choose the hour for: ', data.value);
        setSelectIndex2(data.index);
    };

    const onSelectMinute = (data) => {
        console.log('You have choose the minute for: ', data.value);
        setSelectIndex3(data.index);
    };

    const handleClose = () => {
        console.log('close');
    };

    const renderFooter = () => {
        return (
            <Button size="small" type="primary" onClick={handleClose}>
                Ok
            </Button>
        );
    };

    const scrollStyle = {
        border: 'unset',
        boxShadow: 'unset',
    };

    return (
        <ScrollList style={scrollStyle} header={'无限滚动列表'} footer={renderFooter()}>
            <ScrollItem
                mode="wheel"
                cycled={false}
                list={ampms}
                type={1}
                selectedIndex={selectIndex1}
                onSelect={onSelectAP}
            />
            <ScrollItem
                mode="wheel"
                cycled={true}
                list={hours}
                type={2}
                selectedIndex={selectIndex2}
                onSelect={onSelectHour}
            />
            <ScrollItem
                mode="wheel"
                cycled={true}
                list={minutes}
                type={3}
                selectedIndex={selectIndex3}
                onSelect={onSelectMinute}
            />
        </ScrollList>
    );
};

export default Demo;
