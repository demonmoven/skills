import { Checkbox } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [checkedList, setCheckedList] = useState([]);
  const options = ['选项A', '选项B', '选项C'];
  const indeterminate =
    checkedList.length > 0 && checkedList.length < options.length;
  const checkAll = checkedList.length === options.length;

  const onCheckAllChange = e => {
    setCheckedList(e.target.checked ? options : []);
  };

  return (
    <div className="flex flex-col gap-2">
      <Checkbox
        indeterminate={indeterminate}
        checked={checkAll}
        onChange={onCheckAllChange}
      >
        全选
      </Checkbox>
      <Checkbox.Group value={checkedList} onChange={setCheckedList}>
        {options.map(option => (
          <Checkbox key={option} value={option}>
            {option}
          </Checkbox>
        ))}
      </Checkbox.Group>
    </div>
  );
};

export default Demo;