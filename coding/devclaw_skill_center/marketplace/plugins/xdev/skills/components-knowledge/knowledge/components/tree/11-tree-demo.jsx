import React from 'react';
import { Tree } from '@coze-arch/coze-design';

class Demo extends React.Component {
    constructor() {
        super();
        this.state = {
            value: 'Shanghai'
        };
    }
    onChange(value) {
        this.setState({ value });
    }
    render() {
        const treeData = [
            {
                label: '亚洲',
                value: 'Asia',
                key: '0',
                children: [
                    {
                        label: '中国',
                        value: 'China',
                        key: '0-0',
                        children: [
                            {
                                label: '北京',
                                value: 'Beijing',
                                key: '0-0-0',
                            },
                            {
                                label: '上海',
                                value: 'Shanghai',
                                key: '0-0-1',
                            },
                        ],
                    },
                ],
            },
            {
                label: '北美洲',
                value: 'North America',
                key: '1',
            }
        ];
        const style = {
            width: 260,
            height: 420,
            border: '1px solid var(--semi-color-border)'
        };
        return (
            <Tree
                treeData={treeData}
                value={this.state.value}
                onChange={value => this.onChange(value)}
                style={style}
            />
        );
    }
}

export default Demo;
