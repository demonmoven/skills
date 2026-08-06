import { Select } from '@coze-arch/coze-design';

const Demo = () => {
  const options = [
    'brand',
    'primary',
    'green',
    'yellow',
    'red',
    'cyan',
    'blue',
    'purple',
    'magenta',
  ].map(color => ({
    label: color,
    value: color,
    chipColor: color,
  }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div>
        <h4>单选 - selectedItem 模式</h4>
        <Select
          chipRender="selectedItem"
          dropdownStyle={{ width: 200 }}
          optionList={options}
          defaultValue="brand"
        />
      </div>
      <div>
        <h4>多选 - selectedItem 模式</h4>
        <Select
          multiple
          chipRender="selectedItem"
          dropdownStyle={{ width: 200 }}
          optionList={options}
          defaultValue={['primary', 'green']}
        />
      </div>
      <div>
        <h4>trigger 模式</h4>
        <Select
          chipRender="trigger"
          optionList={options}
          dropdownStyle={{ minWidth: 200 }}
          defaultValue="brand"
        />
      </div>
    </div>
  );
};

export default Demo;