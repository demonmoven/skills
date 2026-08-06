import React from 'react';
import { Calendar } from '@coze-arch/coze-design';

const Demo = () => {
    const importantDate = {
        position: 'absolute',
        left: '0',
        right: '0',
        top: '0',
        bottom: '0',
        backgroundColor: 'var(--semi-color-danger-light-default)',
    };
    const displayValue = new Date(2019, 6, 23, 8, 32, 0);
    const importDates = [new Date(2019, 6, 2), new Date(2019, 6, 8), new Date(2019, 6, 19), new Date(2019, 6, 23)];
    const dateRender = dateString => {
        if (importDates.filter(date => date.toString() === dateString).length) {
            return <div style={importantDate} />;
        }
        return null;
    };
    return <Calendar height={700} mode="month" displayValue={displayValue} dateGridRender={dateRender} />;
};

export default Demo;
