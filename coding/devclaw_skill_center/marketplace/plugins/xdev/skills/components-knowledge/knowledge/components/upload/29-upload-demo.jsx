import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

class ValidateDemo extends React.Component {
    constructor(props) {
        super(props);
        this.state = {};
        this.beforeUpload = this.beforeUpload.bind(this);
        this.transformFile = this.transformFile.bind(this);
        this.count = 0;
    }

    transformFile(fileInstance) {
        if (this.count === 0) {
            let newFile = new File([fileInstance], 'newFileName', { type: 'image/png' });
            return newFile;
        } else {
            return fileInstance;
        }
    }

    beforeUpload({ file, fileList }) {
        let result;
        if (this.count > 0) {
            result = {
                autoRemove: false,
                fileInstance: file.fileInstance,
                shouldUpload: true,
            };
        } else {
            result = {
                autoRemove: false,
                fileInstance: file.fileInstance,
                status: 'validateFail',
                shouldUpload: false,
            };
        }
        this.count = this.count + 1;
        return result;
    }

    render() {
        return (
            <Upload
                action="https://api.semi.design/upload"
                transformFile={this.transformFile}
                beforeUpload={this.beforeUpload}
            >
                <Button icon={<IconCozUpload />} theme="light">
                    点击上传（上传前同步校验）
                </Button>
            </Upload>
        );
    }
}

const Demo = () => <ValidateDemo />;

export default Demo;
