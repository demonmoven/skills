import React, { useState } from 'react';
import { Upload, Avatar, Toast } from '@coze-arch/coze-design';
import { IconCozImage } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const [url, setUrl] = useState('https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png');
    const onSuccess = (response, file) => {
        Toast.success('头像更新成功');
        setUrl('https://sf6-cdn-tos.douyinstatic.com/obj/ttfe/ies/semi/ttmoment.jpeg');
    };

    const style = {
        backgroundColor: 'var(--semi-color-overlay-bg)',
        height: '100%',
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--semi-color-white)',
    };
    
    const hoverMask = (<div style={style}>
        <IconCozImage />
    </div>);

    const api = 'https://api.semi.design/upload';
    let imageOnly = 'image/*';

    return (
        <Upload
            className="avatar-upload"
            action={api}
            onSuccess={onSuccess}
            accept={imageOnly}
            showUploadList={false}
            onError={() => Toast.error('上传失败')}
        >
            <Avatar src={url} style={{ margin: 4 }} hoverMask={hoverMask} />
        </Upload>
    );
};

export default Demo;
