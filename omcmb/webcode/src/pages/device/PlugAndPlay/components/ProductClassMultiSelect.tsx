import { Select, type SelectProps } from 'antd';
import type { ProductClassOption } from '../productClassOptions';

const PRODUCT_CLASS_POPUP_WIDTH = 420;

type Props = Omit<
  SelectProps<string[], ProductClassOption>,
  'mode' | 'options' | 'optionFilterProp' | 'optionRender' | 'popupMatchSelectWidth' | 'showSearch'
> & {
  options: ProductClassOption[];
};

export default function ProductClassMultiSelect({ style, ...props }: Props) {
  return (
    <Select<string[], ProductClassOption>
      {...props}
      mode="multiple"
      showSearch
      optionFilterProp="label"
      popupMatchSelectWidth={PRODUCT_CLASS_POPUP_WIDTH}
      options={props.options}
      maxTagCount="responsive"
      style={{ width: '100%', maxWidth: PRODUCT_CLASS_POPUP_WIDTH, ...style }}
      optionRender={(option) => {
        const fullLabel = String(option.label ?? option.value);
        return (
          <span title={fullLabel} style={{ display: 'block' }}>
            {option.label}
          </span>
        );
      }}
    />
  );
}
