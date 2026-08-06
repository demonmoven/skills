import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    return (
        <Upload action="https://api.semi.design/upload">
            <Button icon={<IconCozUpload />} theme="light">
                点击上传
            </Button>
        </Upload>
    );
};

export default Demo;
