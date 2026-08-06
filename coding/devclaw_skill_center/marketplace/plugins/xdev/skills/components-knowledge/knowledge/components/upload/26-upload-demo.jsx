import React from 'react';
import { Upload } from '@coze-arch/coze-design';

const Demo = () => (
    <Upload
        action="https://api.semi.design/upload"
        draggable={true}
        dragMainText={'点击上传文件或拖拽文件到这里'}
        dragSubText="支持任意类型文件"
    ></Upload>
);

export default Demo;
