import React, { useState } from 'react';
import { Upload, Button, Toast } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    let [disabled, setDisabled] = useState(false);
    let limit = 2;
    let onChange = props => {
        let length = props.fileList.length;
        if (length === limit) {
            setDisabled(true);
        } else {
            setDisabled(false);
        }
    };
    return (
        <Upload
            action={action}
            limit={limit}
            onExceed={() => Toast.warning(`最多只允许上传${limit}个文件`)}
            onChange={onChange}
        >
            <Button icon={<IconCozUpload />} theme="light" disabled={disabled}>
                点击上传（最多{limit}项）
            </Button>
        </Upload>
    );
};

export default Demo;
