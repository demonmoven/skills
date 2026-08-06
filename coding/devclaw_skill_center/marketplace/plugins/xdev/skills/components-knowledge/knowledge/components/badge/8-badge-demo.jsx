import { Badge, TabBar, SegmentTab } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-6 p-4">
    <div>
      <h4 className="mb-2 text-sm">与TabBar结合：</h4>
      <TabBar mode="tab">
        <TabBar.TabPanel tab="我的" itemKey="tab1">
          我的
        </TabBar.TabPanel>
        <TabBar.TabPanel
          tab={
            <div className="flex items-center">
              <span>插件</span> <Badge cozLayout count={99} />
            </div>
          }
          itemKey="tab2"
        >
          插件
        </TabBar.TabPanel>
        <TabBar.TabPanel
          tab={
            <div className="flex items-center">
              <span>消息</span> <Badge type="mini" cozLayout />
            </div>
          }
          itemKey="tab3"
        >
          消息
        </TabBar.TabPanel>
      </TabBar>
    </div>

    <div>
      <h4 className="mb-2 text-sm">与SegmentTab结合：</h4>
      <SegmentTab defaultValue={1} style={{ width: 300 }}>
        <SegmentTab.Tab value={1}>全部</SegmentTab.Tab>
        <SegmentTab.Tab value={2}>
          <div className="flex items-center justify-center">
            未读
            <Badge type="default" cozLayout count={99}></Badge>
          </div>
        </SegmentTab.Tab>
        <SegmentTab.Tab value={3}>
          <div className="flex items-center justify-center">
            已读
            <Badge type="mini" cozLayout></Badge>
          </div>
        </SegmentTab.Tab>
      </SegmentTab>
    </div>
  </div>
);

export default Demo;