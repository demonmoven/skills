import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    let limit = 1;
    let onChange = props => {
        console.log(props.fileList);
    };
    return (
        <Upload
            action={action}
            limit={limit}
            onChange={onChange}
        >
            <Button icon={<IconCozUpload />} theme="light">
                点击上传（最多{limit}项）
            </Button>
        </Upload>
    );
};

export default Demo;
