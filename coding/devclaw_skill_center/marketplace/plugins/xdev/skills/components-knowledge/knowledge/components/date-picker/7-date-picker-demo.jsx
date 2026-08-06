import { DatePicker, Tooltip } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <DatePicker
      type="dateRange"
      placeholder="选择日期范围"
      disabledDate={date => {
        if (!date) {
          return false;
        } // 确保日期不是undefined
        const today = new Date();
        const oneWeekAhead = new Date(today);
        oneWeekAhead.setDate(today.getDate() + 7);
        return date > oneWeekAhead;
      }}
      renderDate={(dayNumber, fullDate) => {
        if (fullDate) {
          // fullDate格式为 xxxx-xx-xx
          const date = new Date(fullDate);
          const today = new Date();
          const oneWeekAhead = new Date(today);
          oneWeekAhead.setDate(today.getDate() + 7);

          // 如果日期超过一周，添加提示
          if (date > oneWeekAhead) {
            return (
              <Tooltip content="超过一周">
                <span>{dayNumber}</span>
              </Tooltip>
            );
          }
        }
        return <>{dayNumber}</>;
      }}
    />
  );
};

export default Demo;