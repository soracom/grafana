import { useFormContext } from 'react-hook-form';

import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceJsonData } from '@grafana/schema';
import { RadioButtonGroup, Stack, Text } from '@grafana/ui';
import { contextSrv } from 'app/core/core';
import { ExpressionDatasourceUID } from 'app/features/expressions/types';
import { AccessControlAction } from 'app/types';
import { AlertQuery } from 'app/types/unified-alerting-dto';

import { RuleFormType, RuleFormValues } from '../../../types/rule-form';
import { NeedHelpInfo } from '../NeedHelpInfo';

function getAvailableRuleTypes() {
  const canCreateGrafanaRules = contextSrv.hasPermission(AccessControlAction.AlertingRuleCreate);
  const canCreateCloudRules = contextSrv.hasPermission(AccessControlAction.AlertingRuleExternalWrite);
  const defaultRuleType = canCreateGrafanaRules ? RuleFormType.grafana : RuleFormType.cloudAlerting;

  const enabledRuleTypes: RuleFormType[] = [];
  if (canCreateGrafanaRules) {
    enabledRuleTypes.push(RuleFormType.grafana);
  }
  if (canCreateCloudRules) {
    enabledRuleTypes.push(RuleFormType.cloudAlerting, RuleFormType.cloudRecording);
  }

  return { enabledRuleTypes, defaultRuleType };
}

const onlyOneDSInQueries = (queries: AlertQuery[]) => {
  return queries.filter((q) => q.datasourceUid !== ExpressionDatasourceUID).length === 1;
};
const getCanSwitch = ({
  queries,
  ruleFormType,
  rulesSourcesWithRuler,
}: {
  rulesSourcesWithRuler: Array<DataSourceInstanceSettings<DataSourceJsonData>>;
  queries: AlertQuery[];
  ruleFormType: RuleFormType | undefined;
}) => {
  // get available rule types
  const availableRuleTypes = getAvailableRuleTypes();

  // check if we have only one query in queries and if it's a cloud datasource
  const onlyOneDS = onlyOneDSInQueries(queries);
  const dataSourceIdFromQueries = queries[0]?.datasourceUid ?? '';
  const isRecordingRuleType = ruleFormType === RuleFormType.cloudRecording;

  //let's check if we switch to cloud type
  const canSwitchToCloudRule =
    !isRecordingRuleType &&
    onlyOneDS &&
    rulesSourcesWithRuler.some((dsJsonData) => dsJsonData.uid === dataSourceIdFromQueries);

  const canSwitchToGrafanaRule = !isRecordingRuleType;
  // check for enabled types
  const grafanaTypeEnabled = availableRuleTypes.enabledRuleTypes.includes(RuleFormType.grafana);
  const cloudTypeEnabled = availableRuleTypes.enabledRuleTypes.includes(RuleFormType.cloudAlerting);

  // can we switch to the other type? (cloud or grafana)
  const canSwitchFromCloudToGrafana =
    ruleFormType === RuleFormType.cloudAlerting && grafanaTypeEnabled && canSwitchToGrafanaRule;
  const canSwitchFromGrafanaToCloud =
    ruleFormType === RuleFormType.grafana && canSwitchToCloudRule && cloudTypeEnabled && canSwitchToCloudRule;

  return canSwitchFromCloudToGrafana || canSwitchFromGrafanaToCloud;
};

export interface SmartAlertTypeDetectorProps {
  editingExistingRule: boolean;
  rulesSourcesWithRuler: Array<DataSourceInstanceSettings<DataSourceJsonData>>;
  queries: AlertQuery[];
  onClickSwitch: () => void;
}

export function SmartAlertTypeDetector({
  editingExistingRule,
  rulesSourcesWithRuler,
  queries,
  onClickSwitch,
}: SmartAlertTypeDetectorProps) {
  const { getValues } = useFormContext<RuleFormValues>();
  const [ruleFormType] = getValues(['type']);
  const canSwitch = getCanSwitch({ queries, ruleFormType, rulesSourcesWithRuler });

  const options = [
    { label: 'Lagoon-managed', value: RuleFormType.grafana },
  ];

  // if we can't switch to data-source managed, disable it
  // TODO figure out how to show a popover to the user to indicate _why_ it's disabled
  const disabledOptions = canSwitch ? [] : [RuleFormType.cloudAlerting];

  return (
    <Stack direction="column" gap={1} alignItems="flex-start">
      <Stack direction="column" gap={0}>
        <Text variant="h5">Rule type</Text>
      </Stack>
      <RadioButtonGroup
        options={options}
        disabled={editingExistingRule}
        disabledOptions={disabledOptions}
        value={ruleFormType}
        onChange={onClickSwitch}
        data-testid="rule-type-radio-group"
      />
    </Stack>
  );
}
