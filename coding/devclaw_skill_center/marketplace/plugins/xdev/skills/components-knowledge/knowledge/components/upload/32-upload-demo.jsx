import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const mockRequest = ({ file, onProgress, onError, onSuccess }) => {
        let count = 0;
        let interval = setInterval(() => {
            if (count === 100) {
                clearInterval(interval);
                onSuccess();
                return;
            }
            onProgress({ total: 100, loaded: count });
            count += 20;
        }, 500);
    };

    return (
        <Upload action="https://api.semi.design/upload" customRequest={mockRequest}>
            <Button icon={<IconCozUpload />} theme="light">
                点击上传
            </Button>
        </Upload>
    );
};

export default Demo;
