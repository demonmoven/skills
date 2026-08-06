import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload, IconCozDownload, IconCozEye, IconCozTrashCan } from '@coze-arch/coze-design/icons';

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
        }
    ];
    const renderFileOperation = (fileItem) => (
        <div style={{ display: 'flex', columnGap: 8, padding: '0 8px' }}>
            <Button icon={<IconCozEye></IconCozEye>} type="tertiary" theme="borderless" size="small"></Button>
            <Button icon={<IconCozDownload></IconCozDownload>} type="tertiary" theme="borderless" size="small"></Button>
            <Button onClick={e=>fileItem.onRemove()} icon={<IconCozTrashCan></IconCozTrashCan>} type="tertiary" theme="borderless" size="small"></Button>
        </div>
    );
    return <Upload action={action} defaultFileList={defaultFileList} itemStyle={{ width: 300 }} renderFileOperation={renderFileOperation}>
        <Button icon={<IconCozUpload />} theme="light">点击上传</Button>
    </Upload>;
};

export default Demo;
