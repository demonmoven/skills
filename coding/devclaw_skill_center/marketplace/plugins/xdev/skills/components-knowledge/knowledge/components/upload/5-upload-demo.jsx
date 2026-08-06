import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    let data = {
        role: 'ies',
        time: new Date().getTime(),
    };
    let headers = {
        'x-tt-semi': 'semi-upload',
    };
    return (
        <Upload action={action} data={data} headers={headers}>
            <Button icon={<IconCozUpload />} theme="light">
                点击上传
            </Button>
        </Upload>
    );
};

export default Demo;
