import React, { PureComponent } from 'react';
import { Button, ClipboardButton, Field, Icon, Input, LinkButton, Modal, Select, Spinner, Switch } from '@grafana/ui';
import { AppEvents, dateMath, SelectableValue } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { DashboardModel, PanelModel } from 'app/features/dashboard/state';
import { getTimeSrv } from 'app/features/dashboard/services/TimeSrv';
import { appEvents } from 'app/core/core';
import { VariableRefresh } from '../../../variables/types';
import { ShareModalTabProps } from './types';

const snapshotApiUrl = '/api/snapshots';

const expireOptions: Array<SelectableValue<number>> = [
  { label: 'Never', value: 0 },
  { label: '1 Hour', value: 60 * 60 },
  { label: '1 Day', value: 60 * 60 * 24 },
  { label: '7 Days', value: 60 * 60 * 24 * 7 },
];

interface Props extends ShareModalTabProps {}

interface State {
  isLoading: boolean;
  step: number;
  snapshotName: string;
  selectedExpireOption: SelectableValue<number>;
  snapshotExpires?: number;
  snapshotUrl: string;
  deleteUrl: string;
  timeoutSeconds: number;
  externalEnabled: boolean;
  sharingButtonText: string;
  snapshotAutoUpdate: boolean;
}

export class ShareSnapshot extends PureComponent<Props, State> {
  private dashboard: DashboardModel;

  constructor(props: Props) {
    super(props);
    this.dashboard = props.dashboard;
    this.state = {
      isLoading: false,
      step: 1,
      selectedExpireOption: expireOptions[0],
      snapshotExpires: expireOptions[0].value,
      snapshotName: props.dashboard.title,
      timeoutSeconds: 4,
      snapshotUrl: '',
      deleteUrl: '',
      externalEnabled: false,
      sharingButtonText: '',
      snapshotAutoUpdate: false,
    };
  }

  componentDidMount() {
    this.getSnaphotShareOptions();
  }

  async getSnaphotShareOptions() {
    const shareOptions = await getBackendSrv().get('/api/snapshot/shared-options');
    this.setState({
      sharingButtonText: shareOptions['externalSnapshotName'],
      externalEnabled: shareOptions['externalEnabled'],
    });
  }

  createSnapshot = (external?: boolean) => () => {
    let { snapshotAutoUpdate, timeoutSeconds } = this.state;
    this.dashboard.snapshot = {
      timestamp: new Date(),
    };

    if (!external) {
      this.dashboard.snapshot.originalUrl = window.location.href;
    }

    // For live snapshots any time ranges not relative or greater than 24 hours use the default last 6 hours range.
    if (snapshotAutoUpdate) {
      // we want to make sure we have enough time if being run as lambda
      timeoutSeconds = 10;
      let nextRange;
      const fromMoment = dateMath.parse(this.dashboard.time.from);
      const toMoment = dateMath.parse(this.dashboard.time.to);
      if (this.dashboard.time.to !== 'now') {
        nextRange = {
          from: 'now-6h',
          to: 'now',
        };
      } else if (!toMoment || !fromMoment || toMoment?.diff(fromMoment, 'hours') > 24) {
        nextRange = {
          from: 'now-24h',
          to: 'now',
        };
      }

      if (nextRange) {
        getTimeSrv().setTime(nextRange);
      }
    }

    this.setState({ isLoading: true });
    this.dashboard.startRefresh();

    setTimeout(() => {
      this.saveSnapshot(this.dashboard, external, snapshotAutoUpdate);
    }, timeoutSeconds * 1000);
  };

  saveSnapshot = async (dashboard: DashboardModel, external?: boolean, live?: boolean) => {
    const { snapshotExpires } = this.state;
    const dash = this.dashboard.getSaveModelClone();
    this.scrubDashboard(dash);

    const cmdData = {
      dashboard: dash,
      name: dash.title,
      expires: snapshotExpires,
      external: external,
      key: '',
    };
    if (live) {
      cmdData.key = this.dashboard.uid + '-live';
    }

    try {
      const results: { deleteUrl: any; url: any } = await getBackendSrv().post(snapshotApiUrl, cmdData);
      this.setState({
        deleteUrl: results.deleteUrl,
        snapshotUrl: results.url,
        step: 2,
      });
    } finally {
      this.setState({ isLoading: false });
    }
  };

  scrubDashboard = (dash: DashboardModel) => {
    const { panel } = this.props;
    const { snapshotName } = this.state;
    // change title
    dash.title = snapshotName;

    // make relative times absolute
    dash.time = getTimeSrv().timeRange();

    // Remove links
    dash.links = [];

    // remove panel queries & links
    dash.panels.forEach((panel) => {
      panel.targets = [];
      panel.links = [];
      panel.datasource = null;
    });

    // remove annotation queries
    const annotations = dash.annotations.list.filter((annotation) => annotation.enable);
    dash.annotations.list = annotations.map((annotation) => {
      return {
        name: annotation.name,
        enable: annotation.enable,
        iconColor: annotation.iconColor,
        snapshotData: annotation.snapshotData,
        type: annotation.type,
        builtIn: annotation.builtIn,
        hide: annotation.hide,
      };
    });

    // remove template queries
    dash.getVariables().forEach((variable: any) => {
      variable.query = '';
      variable.options = variable.current ? [variable.current] : [];
      variable.refresh = VariableRefresh.never;
    });

    // snapshot single panel
    if (panel) {
      const singlePanel = panel.getSaveModel();
      singlePanel.gridPos.w = 24;
      singlePanel.gridPos.x = 0;
      singlePanel.gridPos.y = 0;
      singlePanel.gridPos.h = 20;
      dash.panels = [singlePanel];
    }

    // cleanup snapshotData
    delete this.dashboard.snapshot;
    this.dashboard.forEachPanel((panel: PanelModel) => {
      delete panel.snapshotData;
    });
    this.dashboard.annotations.list.forEach((annotation) => {
      delete annotation.snapshotData;
    });
  };

  deleteSnapshot = async () => {
    const { deleteUrl } = this.state;
    await getBackendSrv().get(deleteUrl);
    this.setState({ step: 3 });
  };

  getSnapshotUrl = () => {
    return this.state.snapshotUrl;
  };

  onSnapshotNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ snapshotName: event.target.value });
  };

  onTimeoutChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ timeoutSeconds: Number(event.target.value) });
  };

  onExpireChange = (option: SelectableValue<number>) => {
    this.setState({
      selectedExpireOption: option,
      snapshotExpires: option.value,
    });
  };

  onSnapshotAutoUpdateChange = () => {
    this.setState({ snapshotAutoUpdate: !this.state.snapshotAutoUpdate });
  };

  onSnapshotUrlCopy = () => {
    appEvents.emit(AppEvents.alertSuccess, ['Content copied to clipboard']);
  };

  renderStep1() {
    const { onDismiss } = this.props;
    const {
      snapshotName,
      snapshotAutoUpdate,
      selectedExpireOption,
      timeoutSeconds,
      isLoading,
      sharingButtonText,
      externalEnabled,
    } = this.state;

    return (
      <>
        <div>
          <p className="share-modal-info-text">
            A snapshot is an instant way to share an interactive dashboard publicly. When created, we strip sensitive
            data like queries (metric, template, and annotation) and panel links, leaving only the visible metric data
            and series names embedded in your dashboard.
          </p>
          <p className="share-modal-info-text">
            Keep in mind, your snapshot <em>can be viewed by anyone</em> that has the link and can access the URL. Share
            wisely.
          </p>
        </div>
        <Field label="Snapshot name">
          <Input id="snapshot-name-input" width={30} value={snapshotName} onChange={this.onSnapshotNameChange} />
        </Field>
        <Field label="Expire">
          <Select
            inputId="expire-select-input"
            menuShouldPortal
            width={30}
            options={expireOptions}
            value={selectedExpireOption}
            onChange={this.onExpireChange}
          />
        </Field>
        <Field
          label="Automatically Update (Beta)"
          description="Enabling this will automatically update the snapshot to most recent data whenever possible."
        >
          <Switch id="auto-update-snapshot" value={snapshotAutoUpdate} onChange={this.onSnapshotAutoUpdateChange} />
        </Field>
        {snapshotAutoUpdate && (
          <div id="auto-update-snapshot-message">
            Updating Snapshot Notes:
            <ul className="share-modal-info-text" style={{ marginLeft: '20px' }}>
              <li>
                Selected <strong>timerange may change</strong> if greater than 24 hours or not relative to the current
                time.
              </li>
              <li>
                Snapshots will <strong>fail to update</strong> if the original dashboard is deleted.
              </li>
              <li>
                High data queries <strong>may time out</strong> and result in an error. If this occurs please reduce the
                time range and try again.
              </li>
            </ul>
          </div>
        )}
        {!snapshotAutoUpdate && (
          <Field
            label="Timeout (seconds)"
            description="You may need to configure the timeout value if it takes a long time to collect your dashboard's
            metrics."
          >
            <Input type="number" width={21} value={timeoutSeconds} onChange={this.onTimeoutChange} />
          </Field>
        )}

        <Modal.ButtonRow>
          <Button variant="secondary" onClick={onDismiss} fill="outline">
            Cancel
          </Button>
          {externalEnabled && (
            <Button variant="secondary" disabled={isLoading} onClick={this.createSnapshot(true)}>
              {sharingButtonText}
            </Button>
          )}
          <Button variant="primary" disabled={isLoading} onClick={this.createSnapshot()}>
            Snapshot
          </Button>
        </Modal.ButtonRow>
      </>
    );
  }

  renderStep2() {
    const { snapshotUrl } = this.state;

    return (
      <>
        <div className="gf-form" style={{ marginTop: '40px' }}>
          <div className="gf-form-row">
            <a href={snapshotUrl} className="large share-modal-link" target="_blank" rel="noreferrer">
              <Icon name="external-link-alt" /> {snapshotUrl}
            </a>
            <br />
            <ClipboardButton variant="secondary" getText={this.getSnapshotUrl} onClipboardCopy={this.onSnapshotUrlCopy}>
              Copy Link
            </ClipboardButton>
          </div>
        </div>

        <div className="pull-right" style={{ padding: '5px' }}>
          Did you make a mistake?{' '}
          <LinkButton fill="text" target="_blank" onClick={this.deleteSnapshot}>
            Delete snapshot.
          </LinkButton>
        </div>
      </>
    );
  }

  renderStep3() {
    return (
      <div className="share-modal-header">
        <p className="share-modal-info-text">
          The snapshot has been deleted. If you have already accessed it once, then it might take up to an hour before
          before it is removed from browser caches or CDN caches.
        </p>
      </div>
    );
  }

  render() {
    const { isLoading, step } = this.state;

    return (
      <>
        {step === 1 && this.renderStep1()}
        {step === 2 && this.renderStep2()}
        {step === 3 && this.renderStep3()}
        {isLoading && <Spinner inline={true} />}
      </>
    );
  }
}
