import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { isNumber, sortBy, toLower, uniqBy } from 'lodash';

import { MetricFindValue, stringToJsRegex } from '@grafana/data';

import { ALL_VARIABLE_TEXT, ALL_VARIABLE_VALUE, NONE_VARIABLE_TEXT, NONE_VARIABLE_VALUE } from '../constants';
import { getInstanceState } from '../state/selectors';
import { initialVariablesState, VariablePayload, VariablesState } from '../state/types';
import { initialVariableModelState, QueryVariableModel, VariableOption, VariableRefresh, VariableSort } from '../types';

enum MetaFlag {
  text = 'text',
  value = 'value',
  none = '',
}

// I don't particularly like this function, but keyof typeof causes betterer eslint to get angry
const stringToMetaFlag = (value: string): MetaFlag => {
  switch (value) {
    case MetaFlag.text:
      return MetaFlag.text;
    case MetaFlag.value:
      return MetaFlag.value;
    case MetaFlag.none:
      return MetaFlag.none;
    default:
      throw new Error(`${value} is not a valid meta flag.`);
  }
};

interface VariableOptionsUpdate {
  templatedRegex: string;
  results: MetricFindValue[];
}

export const initialQueryVariableModelState: QueryVariableModel = {
  ...initialVariableModelState,
  type: 'query',
  datasource: null,
  query: '',
  regex: '',
  sort: VariableSort.disabled,
  refresh: VariableRefresh.onDashboardLoad,
  multi: false,
  includeAll: false,
  allValue: null,
  options: [],
  current: {} as VariableOption,
  definition: '',
};

export const sortVariableValues = (options: any[], sortOrder: VariableSort) => {
  if (sortOrder === VariableSort.disabled) {
    return options;
  }

  const sortType = Math.ceil(sortOrder / 2);
  const reverseSort = sortOrder % 2 === 0;

  if (sortType === 1) {
    options = sortBy(options, 'text');
  } else if (sortType === 2) {
    options = sortBy(options, (opt) => {
      if (!opt.text) {
        return -1;
      }

      const matches = opt.text.match(/.*?(\d+).*/);
      if (!matches || matches.length < 2) {
        return -1;
      } else {
        return parseInt(matches[1], 10);
      }
    });
  } else if (sortType === 3) {
    options = sortBy(options, (opt) => {
      return toLower(opt.text);
    });
  }

  if (reverseSort) {
    options = options.reverse();
  }

  return options;
};

const getAllMatches = (str: string, regex: RegExp): RegExpExecArray[] => {
  const results: RegExpExecArray[] = [];
  let matches = null;

  regex.lastIndex = 0;

  do {
    matches = regex.exec(str);
    if (matches) {
      results.push(matches);
    }
  } while (regex.global && matches && matches[0] !== '' && matches[0] !== undefined);

  return results;
};

const stringToMetaFlagAndRegex = (variableRegEx: string): [MetaFlag, RegExp] => {
  const REGEX_CHAR = '/';

  // variableRegEx has no metafield and is only a regex string
  if (variableRegEx[0] === REGEX_CHAR) {
    return [MetaFlag.none, stringToJsRegex(variableRegEx)];
  }

  // variableRegEx doesn't include / and is maybe just a basic string
  if (!variableRegEx.includes(REGEX_CHAR)) {
    return [MetaFlag.none, stringToJsRegex(variableRegEx)];
  }

  const regexStartIdx = variableRegEx.indexOf(REGEX_CHAR);

  const metaString = variableRegEx.slice(0, regexStartIdx).toLowerCase();
  const metaFlag = stringToMetaFlag(metaString);

  const extractedRegexString = variableRegEx.slice(regexStartIdx);

  return [metaFlag, stringToJsRegex(extractedRegexString)];
};

export const metricNamesToVariableValues = (variableRegEx: string, sort: VariableSort, metricNames: any[]) => {
  let regex;
  let metaFlag = MetaFlag.none;
  let options: VariableOption[] = [];

  if (variableRegEx) {
    [metaFlag, regex] = stringToMetaFlagAndRegex(variableRegEx);
  }

  for (let i = 0; i < metricNames.length; i++) {
    const item = metricNames[i];
    let text = item.text === undefined || item.text === null ? item.value : item.text;
    let value = item.value === undefined || item.value === null ? item.text : item.value;

    if (isNumber(value)) {
      value = value.toString();
    }

    if (isNumber(text)) {
      text = text.toString();
    }

    if (regex) {
      let matches;
      switch (metaFlag) {
        case MetaFlag.text:
          matches = getAllMatches(text, regex);
          break;
        case MetaFlag.value: //Fallthrough - default grafana behavior is matching against value
        case MetaFlag.none:
        default:
          matches = getAllMatches(value, regex);
          break;
      }

      if (!matches.length) {
        continue;
      }

      const valueGroup = matches.find((m) => m.groups && m.groups.value);
      const textGroup = matches.find((m) => m.groups && m.groups.text);
      const firstMatch = matches.find((m) => m.length > 1);
      const manyMatches = matches.length > 1 && firstMatch;

      if (valueGroup || textGroup) {
        value = valueGroup?.groups?.value ?? textGroup?.groups?.text;
        text = textGroup?.groups?.text ?? valueGroup?.groups?.value;
      } else if (manyMatches) {
        for (let j = 0; j < matches.length; j++) {
          const match = matches[j];
          options.push({ text: match[1], value: match[1], selected: false });
        }
        continue;
      } else if (firstMatch) {
        text = firstMatch[1];
        value = firstMatch[1];
      }
    }

    options.push({ text: text, value: value, selected: false });
  }

  options = uniqBy(options, 'value');
  return sortVariableValues(options, sort);
};

export const queryVariableSlice = createSlice({
  name: 'templating/query',
  initialState: initialVariablesState,
  reducers: {
    updateVariableOptions: (state: VariablesState, action: PayloadAction<VariablePayload<VariableOptionsUpdate>>) => {
      const { results, templatedRegex } = action.payload.data;
      const instanceState = getInstanceState(state, action.payload.id);
      if (instanceState.type !== 'query') {
        return;
      }

      const { includeAll, sort } = instanceState;
      const options = metricNamesToVariableValues(templatedRegex, sort, results);

      if (includeAll) {
        options.unshift({ text: ALL_VARIABLE_TEXT, value: ALL_VARIABLE_VALUE, selected: false });
      }

      if (!options.length) {
        options.push({ text: NONE_VARIABLE_TEXT, value: NONE_VARIABLE_VALUE, isNone: true, selected: false });
      }

      instanceState.options = options;
    },
  },
});

export const queryVariableReducer = queryVariableSlice.reducer;

export const { updateVariableOptions } = queryVariableSlice.actions;
