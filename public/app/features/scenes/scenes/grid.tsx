import {
  VizPanel,
  SceneTimePicker,
  SceneFlexLayout,
  SceneGridLayout,
  SceneTimeRange,
  EmbeddedScene,
} from '@grafana/scenes';

import { Scene } from '../components/Scene';
import { SceneEditManager } from '../editor/SceneEditManager';

import { getQueryRunnerWithRandomWalkQuery } from './queries';

export function getGridLayoutTest(standalone: boolean): Scene | EmbeddedScene {
  const state = {
    title: 'Grid layout test',
    body: new SceneGridLayout({
      children: [
        new VizPanel({
          pluginId: 'timeseries',
          title: 'Draggable and resizable',
          placement: {
            x: 0,
            y: 0,
            width: 12,
            height: 10,
            isResizable: true,
            isDraggable: true,
          },
        }),

        new VizPanel({
          pluginId: 'timeseries',
          title: 'No drag and no resize',
          placement: { x: 12, y: 0, width: 12, height: 10, isResizable: false, isDraggable: false },
        }),

        new SceneFlexLayout({
          direction: 'column',
          placement: { x: 6, y: 11, width: 12, height: 10, isDraggable: true, isResizable: true },
          children: [
            new VizPanel({
              placement: { ySizing: 'fill' },
              pluginId: 'timeseries',
              title: 'Child of flex layout',
            }),
            new VizPanel({
              placement: { ySizing: 'fill' },
              pluginId: 'timeseries',
              title: 'Child of flex layout',
            }),
          ],
        }),
      ],
    }),
    $editor: new SceneEditManager({}),
    $timeRange: new SceneTimeRange(),
    $data: getQueryRunnerWithRandomWalkQuery(),
    actions: [new SceneTimePicker({})],
  };

  return standalone ? new Scene(state) : new EmbeddedScene(state);
}
