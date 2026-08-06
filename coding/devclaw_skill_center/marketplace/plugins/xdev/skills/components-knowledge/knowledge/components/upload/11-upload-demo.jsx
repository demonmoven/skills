import React from 'react';
import { Upload, Toast } from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    const defaultFileList = [
        {
            uid: '1',
            name: 'dyBag.png',
            status: 'success',
            size: '130KB',
            preview: true,
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/edit-bag.jpeg',
        },
        {
            uid: '2',
            name: 'dyBag2.jpeg',
            status: 'success',
            size: '222KB',
            preview: true,
            fileInstance: new File([new ArrayBuffer(2048)], 'dyBag2.jpeg', { type: 'image/png' }),
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/edit-bag.jpeg',
        },
    ];
    return (
        <Upload
            action={action}
            limit={2}
            listType="picture"
            accept="image/*"
            defaultFileList={defaultFileList}
            onExceed={() => Toast.warning('最多只允许上传2个文件')}
        >
            <IconCozPlus size="extra-large" />
        </Upload>
    );
};

export default Demo;
