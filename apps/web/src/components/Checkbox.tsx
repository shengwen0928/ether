import { useId } from 'react';
import type { InputHTMLAttributes, ReactElement } from 'react';
import { CheckIcon } from './icons';

export interface CheckboxProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'className'> {
  label: string;
  description?: string;
}

export function Checkbox({ label, description, id, ...rest }: CheckboxProps): ReactElement {
  const generated = useId();
  const fieldId = id ?? generated;
  return (
    <div className="et-choice">
      <input id={fieldId} type="checkbox" className="et-choice__input" {...rest} />
      <span className="et-choice__box et-choice__box--check" aria-hidden="true">
        <CheckIcon size={14} />
      </span>
      <label className="et-choice__text" htmlFor={fieldId}>
        <span className="et-choice__label">{label}</span>
        {description !== undefined ? (
          <span className="et-choice__description">{description}</span>
        ) : null}
      </label>
    </div>
  );
}

export interface RadioOption {
  value: string;
  label: string;
  description?: string;
  disabled?: boolean;
}

export interface RadioGroupProps {
  legend: string;
  name: string;
  options: RadioOption[];
  value: string;
  onChange: (value: string) => void;
}

export function RadioGroup({
  legend,
  name,
  options,
  value,
  onChange,
}: RadioGroupProps): ReactElement {
  return (
    <fieldset className="et-fieldset">
      <legend className="et-fieldset__legend">{legend}</legend>
      <div className="et-choice-list">
        {options.map((option) => {
          const optionId = `${name}-${option.value}`;
          return (
            <div className="et-choice" key={option.value}>
              <input
                id={optionId}
                name={name}
                type="radio"
                className="et-choice__input"
                value={option.value}
                checked={value === option.value}
                disabled={option.disabled === true}
                onChange={() => {
                  onChange(option.value);
                }}
              />
              <span className="et-choice__box et-choice__box--radio" aria-hidden="true" />
              <label className="et-choice__text" htmlFor={optionId}>
                <span className="et-choice__label">{option.label}</span>
                {option.description !== undefined ? (
                  <span className="et-choice__description">{option.description}</span>
                ) : null}
              </label>
            </div>
          );
        })}
      </div>
    </fieldset>
  );
}
