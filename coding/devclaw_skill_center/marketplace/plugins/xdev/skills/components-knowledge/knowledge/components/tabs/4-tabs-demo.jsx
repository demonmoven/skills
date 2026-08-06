import React from 'react';
import { Tabs } from '@coze-arch/coze-design';

class Demo extends React.Component {
    constructor() {
        super();
        this.state = { key: '1' };
        this.onTabClick = this.onTabClick.bind(this);
    }

    onTabClick(key, type) {
        this.setState({ [type]: key });
    }

    render() {
        // eslint-disable-next-line react/jsx-key
        const contentList = [<div>文档</div>, <div>快速起步</div>, <div>帮助</div>];
        const tabList = [
            { tab: '文档', itemKey: '1' },
            { tab: '快速起步', itemKey: '2' },
            { tab: '帮助', itemKey: '3' },
        ];
        return (
            <Tabs
                type="slash"
                tabList={tabList}
                onChange={key => {
                    this.onTabClick(key, 'key');
                }}
            >
                {contentList[this.state.key - 1]}
            </Tabs>
        );
    }
}

export default Demo;
