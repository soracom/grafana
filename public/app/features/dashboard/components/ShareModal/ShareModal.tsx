import React from 'react';

import { reportInteraction } from '@grafana/runtime/src';
import { Modal, ModalTabsHeader, TabContent } from '@grafana/ui';
import { config } from 'app/core/config';
import { contextSrv } from 'app/core/core';
import { t } from 'app/core/internationalization';
import { SharePublicDashboard } from 'app/features/dashboard/components/ShareModal/SharePublicDashboard/SharePublicDashboard';
import { DashboardModel, PanelModel } from 'app/features/dashboard/state';
import { isPanelModelLibraryPanel } from 'app/features/library-panels/guard';

import { ShareEmbed } from './ShareEmbed';
import { ShareExport } from './ShareExport';
import { ShareLibraryPanel } from './ShareLibraryPanel';
import { ShareLink } from './ShareLink';
import { ShareSnapshot } from './ShareSnapshot';
import { ShareModalTabModel } from './types';

const customDashboardTabs: ShareModalTabModel[] = [];
const customPanelTabs: ShareModalTabModel[] = [];

export function addDashboardShareTab(tab: ShareModalTabModel) {
  customDashboardTabs.push(tab);
}

export function addPanelShareTab(tab: ShareModalTabModel) {
  customPanelTabs.push(tab);
}

function getInitialState(props: Props): State {
  const { tabs, activeTab } = getTabs(props);

  return {
    tabs,
    activeTab,
  };
}

function getTabs(props: Props) {
  const { panel, activeTab } = props;

  const linkLabel = t('share-modal.tab-title.link', 'Link');
  const tabs: ShareModalTabModel[] = [{ label: linkLabel, value: 'link', component: ShareLink }];

  const operatorId = config.bootData.user.orgName || '';
  if (contextSrv.isSignedIn && operatorId.endsWith('-PRO')) {
    const snapshotLabel = t('share-modal.tab-title.snapshot', 'Snapshot');
    tabs.push({ label: snapshotLabel, value: 'snapshot', component: ShareSnapshot });
  }

  if (panel) {
    const embedLabel = t('share-modal.tab-title.embed', 'Embed');
    tabs.push({ label: embedLabel, value: 'embed', component: ShareEmbed });

    if (!isPanelModelLibraryPanel(panel)) {
      const libraryPanelLabel = t('share-modal.tab-title.library-panel', 'Library panel');
      tabs.push({ label: libraryPanelLabel, value: 'library_panel', component: ShareLibraryPanel });
    }
    tabs.push(...customPanelTabs);
  } else {
    const exportLabel = t('share-modal.tab-title.export', 'Export');
    tabs.push({ label: exportLabel, value: 'export', component: ShareExport });
    tabs.push(...customDashboardTabs);
  }

  if (Boolean(config.featureToggles['publicDashboards'])) {
    tabs.push({ label: 'Public dashboard', value: 'share', component: SharePublicDashboard });
  }

  const at = tabs.find((t) => t.value === activeTab);

  return {
    tabs,
    activeTab: at?.value ?? tabs[0].value,
  };
}

interface Props {
  dashboard: DashboardModel;
  panel?: PanelModel;
  activeTab?: string;

  onDismiss(): void;
}

interface State {
  tabs: ShareModalTabModel[];
  activeTab: string;
}

export class ShareModal extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = getInitialState(props);
  }

  componentDidMount() {
    reportInteraction('grafana_dashboards_share_modal_viewed');
  }

  onSelectTab = (t: any) => {
    this.setState({ activeTab: t.value });
  };

  getTabs() {
    return getTabs(this.props).tabs;
  }

  getActiveTab() {
    const { tabs, activeTab } = this.state;
    return tabs.find((t) => t.value === activeTab)!;
  }

  renderTitle() {
    const { panel } = this.props;
    const { activeTab } = this.state;
    const title = panel ? t('share-modal.panel.title', 'Share Panel') : t('share-modal.dashboard.title', 'Share');

    return (
      <ModalTabsHeader
        title={title}
        icon="share-alt"
        tabs={this.getTabs()}
        activeTab={activeTab}
        onChangeTab={this.onSelectTab}
      />
    );
  }

  render() {
    const { dashboard, panel } = this.props;
    const activeTabModel = this.getActiveTab();
    const ActiveTab = activeTabModel.component;

    return (
      <Modal isOpen={true} title={this.renderTitle()} onDismiss={this.props.onDismiss}>
        <TabContent>
          <ActiveTab dashboard={dashboard} panel={panel} onDismiss={this.props.onDismiss} />
        </TabContent>
      </Modal>
    );
  }
}
