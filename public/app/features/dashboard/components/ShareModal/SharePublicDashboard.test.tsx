import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import React from 'react';

import { BackendSrv } from '@grafana/runtime';
import config from 'app/core/config';
import { DashboardModel, PanelModel } from 'app/features/dashboard/state';

import { ShareModal } from './ShareModal';
import { PublicDashboardConfig } from './SharePublicDashboardUtils';

// Mock api request
const publicDashboardconfigResp: PublicDashboardConfig = {
  isPublic: true,
  publicDashboard: { uid: '', dashboardUid: '' },
};

const backendSrv = {
  get: () => publicDashboardconfigResp,
} as unknown as BackendSrv;

jest.mock('@grafana/runtime', () => ({
  ...(jest.requireActual('@grafana/runtime') as unknown as object),
  getBackendSrv: () => backendSrv,
}));

jest.mock('app/core/core', () => {
  return {
    contextSrv: {
      hasPermission: () => true,
    },
    appEvents: {
      subscribe: () => {
        return {
          unsubscribe: () => {},
        };
      },
      emit: () => {},
    },
  };
});

describe('SharePublic', () => {
  let originalBootData: any;

  beforeAll(() => {
    originalBootData = config.bootData;
    config.appUrl = 'http://dashboards.grafana.com/';

    config.bootData = {
      user: {
        orgId: 1,
      },
    } as any;
  });

  afterAll(() => {
    config.bootData = originalBootData;
  });

  it('does not render share panel when public dashboards feature is disabled', () => {
    const mockDashboard = new DashboardModel({
      uid: 'mockDashboardUid',
    });
    const mockPanel = new PanelModel({
      id: 'mockPanelId',
    });

    render(<ShareModal panel={mockPanel} dashboard={mockDashboard} onDismiss={() => {}} />);

    expect(screen.getByRole('tablist')).toHaveTextContent('Link');
    expect(screen.getByRole('tablist')).not.toHaveTextContent('Public Dashboard');
  });

  it('renders share panel when public dashboards feature is enabled', async () => {
    config.featureToggles.publicDashboards = true;
    const mockDashboard = new DashboardModel({
      uid: 'mockDashboardUid',
    });
    const mockPanel = new PanelModel({
      id: 'mockPanelId',
    });

    render(<ShareModal panel={mockPanel} dashboard={mockDashboard} onDismiss={() => {}} />);

    await waitFor(() => screen.getByText('Link'));
    expect(screen.getByRole('tablist')).toHaveTextContent('Link');
    expect(screen.getByRole('tablist')).toHaveTextContent('Public Dashboard');

    fireEvent.click(screen.getByText('Public Dashboard'));

    await waitFor(() => screen.getByText('Enabled'));
  });
});
