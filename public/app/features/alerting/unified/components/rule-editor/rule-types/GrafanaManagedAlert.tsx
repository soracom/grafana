import { RuleFormType } from '../../../types/rule-form';

import { RuleType, SharedProps } from './RuleType';

const GrafanaManagedRuleType = ({ selected = false, disabled, onClick }: SharedProps) => {
  return (
    <RuleType
      name="Lagoon managed alert"
      description={
        <span>
          Supports SORACOM Harvest.
          <br />
          Transform data with expressions.
        </span>
      }
      image="public/img/lagoon-logo-cl.svg"
      selected={selected}
      disabled={disabled}
      value={RuleFormType.grafana}
      onClick={onClick}
    />
  );
};

export { GrafanaManagedRuleType };
