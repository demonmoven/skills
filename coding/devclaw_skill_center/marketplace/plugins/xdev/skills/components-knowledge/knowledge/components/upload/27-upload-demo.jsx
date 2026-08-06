import React from 'react';
import { Upload } from '@coze-arch/coze-design';
import { IconCozLightning } from '@coze-arch/coze-design/icons';

const Demo = () => <Upload
    action="https://api.semi.design/upload"
    dragIcon={<IconCozLightning />}
    draggable={true}
    accept="application/pdf,.jpeg"
    dragMainText={'点击上传文件或拖拽文件到这里'}
    dragSubText="仅支持jpeg、pdf"
    style={{ marginTop: 10 }}
></Upload>;

export default Demo;
