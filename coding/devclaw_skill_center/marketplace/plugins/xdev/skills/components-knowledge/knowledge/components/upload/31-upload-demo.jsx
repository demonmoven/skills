import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const afterUpload = ({ response, file }) => {
        // 可以根据业务接口返回，决定当次上传是否成功
        if (response.status_code === 200) {
            return {
                autoRemove: false,
                status: 'uploadFail',
                validateMessage: '内容不合法',
                name: 'RenameByServer.jpg',
                url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/edit-bag.jpeg'
            };
        } else {
            return {};
        }
    };

    return (
        <Upload action="https://api.semi.design/upload" afterUpload={afterUpload}>
            <Button icon={<IconCozUpload />} theme="light">
                点击上传
            </Button>
        </Upload>
    )
}

export default Demo;
