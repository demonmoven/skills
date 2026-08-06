import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const defaultFileList = [
        {
            uid: '1',
            name: 'first.png',
            status: 'success',
            size: '130KB',
            preview: true,
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png',
        },
        {
            uid: '2',
            name: 'second.png',
            status: 'validateFail',
            size: '222KB',
            preview: true,
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png',
        },
    ];
    let action = 'https://api.semi.design/upload';
    return (
        <>
            <Upload action={action} disabled defaultFileList={defaultFileList}>
                <Button icon={<IconCozUpload />} theme="light" disabled>
                    点击上传
                </Button>
            </Upload>
        </>
    );
};

export default Demo;
