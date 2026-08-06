<!-- BLOCK_1 | doxcnbI8dSla9z8IdFuN9T8h7Kc -->
## 监控报警（2026-03-19～2026-03-24）<!-- 标题序号: 1 --><!-- END_BLOCK_1 -->

<!-- BLOCK_2 | doxcnGBL4DN7hBg5tfegI2E2vme -->
本周期内“追光结算计费产研报警群”的重点监控报警及原因如下。
<!-- END_BLOCK_2 -->

<!-- BLOCK_3 | doxcndlVMCPQpc0pYs5PebxESoh -->
<table header-row="true" col-widths="500,500">
    <tr>
        <td>告警内容+链接</td>
        <td>报警原因</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184911487310?zone=CN&check_vregion=China-Pay&psm=caijing.bytepay.settle_faas&argos_from_source=argos_alarm_card)[warning] faas-timer函数执行报错5分钟超过1次](https://cloud.bytedance.net/argos/alarm/detail/0/35184911487310?zone=CN&check_vregion=China-Pay&psm=caijing.bytepay.settle_faas&argos_from_source=argos_alarm_card)</td>
        <td>BatchTaskGenerateExec 方法调用失败。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184948373220?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.work_order_generate&argos_from_source=argos_alarm_card)[已恢复][](https://cloud.bytedance.net/argos/alarm/detail/0/35184948373220?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.work_order_generate&argos_from_source=argos_alarm_card)[warning] cronjob-执行failed告警](https://cloud.bytedance.net/argos/alarm/detail/0/35184948373220?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.work_order_generate&argos_from_source=argos_alarm_card)</td>
        <td>settle_center_job 执行失败。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/1101732351/17592698753619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center_job&argos_from_source=argos_alarm_card)[已恢复][](https://cloud.bytedance.net/argos/alarm/detail/1101732351/17592698753619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center_job&argos_from_source=argos_alarm_card)[warning] CronJob EXEC failed alarm](https://cloud.bytedance.net/argos/alarm/detail/1101732351/17592698753619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center_job&argos_from_source=argos_alarm_card)</td>
        <td>settle_center_job 执行失败。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184947143051?zone=China-Pay&check_vregion=China-Pay&psm=toutiao.mysql.settle_union_db_write&argos_from_source=argos_alarm_card)[warning] 【MySQL】Proxy-慢查询- P99](https://cloud.bytedance.net/argos/alarm/detail/0/35184947143051?zone=China-Pay&check_vregion=China-Pay&psm=toutiao.mysql.settle_union_db_write&argos_from_source=argos_alarm_card)</td>
        <td>settle_union_db 数据库代理实例存在问题，导致慢查询。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/1101729098/17592712754678?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.bytepay_charge&argos_from_source=argos_alarm_card)[已恢复][](https://cloud.bytedance.net/argos/alarm/detail/1101729098/17592712754678?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.bytepay_charge&argos_from_source=argos_alarm_card)[warning][](https://cloud.bytedance.net/argos/alarm/detail/1101729098/17592712754678?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.bytepay_charge&argos_from_source=argos_alarm_card)[已确认] BytePay_](https://cloud.bytedance.net/argos/alarm/detail/1101729098/17592712754678?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.bytepay_charge&argos_from_source=argos_alarm_card)[Charge请求下游接口失败QPS过多](https://cloud.bytedance.net/argos/alarm/detail/1101729098/17592712754678?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.bytepay_charge&argos_from_source=argos_alarm_card)</td>
        <td>请求下游 cmp.ecom.ecom_merchant_rpc 服务的 QueryOpenAccountList 接口时，因限流被拒绝。</td>
    </tr>
    <tr>
        <td>[首笔发现告警：异常上升](https://metrics-fe.byted.org/web/plot/metrics#1773879390(-1d),1773883590,,,,,,,China-Pay%7CChina-East%7CChina-North%7CChina-Pay2,true,,;sum:store:caijing.bytepay.settle.request.throughput%7B_region=China-Pay,method_name=Settle,ret_code=SE012048%7D%7B%7D;0)</td>
        <td>商户结算账户被限流，导致统一支付返回 PROCESSING，settle 服务感知后抛出 SE012048 错误。</td>
    </tr>
</table>
<!-- END_BLOCK_3 -->

<!-- BLOCK_4 | doxcnWoYQDpRLS6nt5rEvAzdsJb -->
本周期内“【P0】追光计费结算报警群”的重点报警及讨论结论如下。
<!-- END_BLOCK_4 -->

<!-- BLOCK_5 | doxcnCuWfs5xwX9YZuEMedcudbc -->
<table header-row="true" col-widths="500,500">
    <tr>
        <td>告警内容+链接</td>
        <td>报警原因</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/1101196237/17592700742195?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_mq&argos_from_source=argos_alarm_card)[critical] 交易消息全机房流入跌0](https://cloud.bytedance.net/argos/alarm/detail/1101196237/17592700742195?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_mq&argos_from_source=argos_alarm_card)</td>
        <td>Metrics 基础组件故障，导致上报延迟，监控数据出现“跌0”假象。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/1101196237/17592700742113?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_mq&argos_from_source=argos_alarm_card)[critical] 支付消息全机房跌0](https://cloud.bytedance.net/argos/alarm/detail/1101196237/17592700742113?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_mq&argos_from_source=argos_alarm_card)</td>
        <td>同上，Metrics 基础组件故障。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184950668059?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.account_trans&argos_from_source=argos_alarm_card)[warning] 【追光账务】端外—自动迁移NDB](https://cloud.bytedance.net/argos/alarm/detail/0/35184950668059?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.account_trans&argos_from_source=argos_alarm_card)</td>
        <td>配置了热点账户（6120260303821610），触发了自动迁移 NDB 相关的监控。</td>
    </tr>
    <tr>
        <td>[【财经】【已恢复】3月23日D0结算问题沟通](https://g.bytedance.net/coordination/incident-admin/review?id=19279&picked_detail=19279)</td>
        <td>D0 结算任务执行出现问题，触发 GOC 事故流程。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184943854619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)[critical] 结算架构归一之成功率告警-Settle](https://cloud.bytedance.net/argos/alarm/detail/0/35184943854619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)</td>
        <td>机房演练叠加业务高峰（退款结算咨询量从 700tps 增长至 2700tps），导致服务触发限流。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184943854619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)[critical] 结算架构归一之成功率告警-RefundAccountDecision](https://cloud.bytedance.net/argos/alarm/detail/0/35184943854619?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)</td>
        <td>机房演练与业务高峰共同导致服务限流。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184931151258?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle&argos_from_source=argos_alarm_card)[critical] 财经支付触发限流告警](https://cloud.bytedance.net/argos/alarm/detail/0/35184931151258?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle&argos_from_source=argos_alarm_card)</td>
        <td>上游服务（caijing.bytepay.settle_center）的 RefundAccountDecision 方法触发限流，导致下游 settle 服务报错。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184937752681?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)[critical] 结算中心卡单报警](https://cloud.bytedance.net/argos/alarm/detail/0/35184937752681?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_center&argos_from_source=argos_alarm_card)</td>
        <td>OPPO 商户（6020230828335647）的现金户因限流导致卡单。</td>
    </tr>
    <tr>
        <td>[[](https://cloud.bytedance.net/argos/alarm/detail/0/35184931156207?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_clean_mq&argos_from_source=argos_alarm_card)[critical] 财经支付触发限流告警](https://cloud.bytedance.net/argos/alarm/detail/0/35184931156207?zone=China-Pay&check_vregion=China-Pay&psm=caijing.bytepay.settle_clean_mq&argos_from_source=argos_alarm_card)</td>
        <td>settle_faas 服务的 ChargeArchive 方法出现偶发性流量尖刺，触发了 settle_clean_mq 的限流。</td>
    </tr>
</table>
<!-- END_BLOCK_5 -->

<!-- BLOCK_6 | doxcn85jSJ075uqxxqzK8tNUS1b -->
## 核对差异（2026-03-19～2026-03-24）<!-- 标题序号: 2 --><!-- END_BLOCK_6 -->

<!-- BLOCK_7 | doxcnIzTsS0oCRtPQjcSki9gE9e -->
本周期内“结算内部对账报警群”的重点核对差异及原因如下。
<!-- END_BLOCK_7 -->

<!-- BLOCK_8 | doxcnv8r2xvWvt4Wffp2UX9voyc -->
<table header-row="true" col-widths="500,500">
    <tr>
        <td>核对差异内容+链接</td>
        <td>差异原因</td>
    </tr>
    <tr>
        <td>[【追光清结算】2.0链路计费基线 核对](https://boss.bytedance.net/check-core/data-board/cmc/add?type=edit&SceneId=Scene250819211020173651820545)</td>
        <td>历史数据显示，原因为核对配置问题，导致部分产品码出现 0 费率。当前排查中。</td>
    </tr>
    <tr>
        <td>[【追光清结算】架构归一-交易vs结算请求幂等](https://boss.bytedance.net/check-core/dda/config-new/add?mode=checkDetail&PageNo=1&PageSize=10&id=EQN250811153847901376342537)</td>
        <td>历史数据显示，主要原因为 2 月 24 日数据丢失或数据过期。当前排查中。</td>
    </tr>
    <tr>
        <td>[【重要】OC-账龄监控](https://boss.bytedance.net/check-core/check/hsql/list?TaskStatus=ALL&ScheduleType=Hsql&PageNo=1&PageSize=10&TaskCode=250709164914050954686719)</td>
        <td>历史数据显示，原因为苹果商户跨境业务量高、跨天交易多导致。当前排查中。</td>
    </tr>
    <tr>
        <td>[【P0】结算请求vs端内交易-架构归一](https://boss.bytedance.net/check-core/check/task-manage/task/list?TaskStatus=ALL&PageNo=1&PageSize=10&TaskCode=250811141217892737992700)</td>
        <td>历史数据显示，存在退款关闭不兼容、异常机器退出、核对平台数据丢失、交易核心冷库查询失败等问题。当前排查中。</td>
    </tr>
    <tr>
        <td>[【追光清结算】带贴息的结算请求和贴息资金服务比对](https://boss.bytedance.net/check-core/check/task-manage/task/list?TaskStatus=ALL&PageNo=1&PageSize=10&TaskCode=250311155724679844751392)</td>
        <td>排查中</td>
    </tr>
    <tr>
        <td>[【追光计收费-观测】商户&费项维度入金大于出金](https://boss.bytedance.net/check-core/check/hsql/list?TaskStatus=ALL&ScheduleType=Hsql&PageNo=1&PageSize=10&TaskCode=250118105111168671612292)</td>
        <td>观察性核对，排查中。</td>
    </tr>
    <tr>
        <td>[【p0】资金操作vs计收费票据](https://boss.bytedance.net/check-core/check/task-manage/task/list?TaskStatus=ALL&PageNo=1&PageSize=10&TaskCode=240817172830886910242592)</td>
        <td>排查中</td>
    </tr>
    <tr>
        <td>[【追光清结算】月付息费电商结算单和计费明细单状态一致性](https://boss.bytedance.net/check-core/check/hsql/list?TaskStatus=ALL&ScheduleType=Hsql&PageNo=1&PageSize=10&TaskCode=251009195051010651208342)</td>
        <td>历史数据显示，原因为核对配置问题。当前排查中。</td>
    </tr>
</table>
<!-- END_BLOCK_8 -->

