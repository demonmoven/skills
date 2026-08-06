import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload, IconCozPlus } from '@coze-arch/coze-design/icons';

class ManulUploadDemo extends React.Component {
    constructor() {
        super();
        this.manulUpload = this.manulUpload.bind(this);
        this.uploadRef = React.createRef();
    }

    manulUpload() {
        this.uploadRef.current.upload();
    }

    render() {
        let action = 'https://api.semi.design/upload';
        return (
            <div>
                <Upload
                    accept="image/gif, image/png, image/jpeg, image/bmp, image/webp"
                    action={action}
                    uploadTrigger="custom"
                    ref={this.uploadRef}
                    onSuccess={(...v) => console.log(...v)}
                    onError={(...v) => console.log(...v)}
                >
                    <Button icon={<IconCozPlus />} theme="light" style={{ marginRight: 8 }}>
                        选择文件
                    </Button>
                </Upload>
                <Button icon={<IconCozUpload />} theme="light" onClick={this.manulUpload}>
                    开始上传
                </Button>
            </div>
        );
    }
}

const Demo = () => <ManulUploadDemo />;

export default Demo;
