import { css } from '@emotion/css';
import React from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

export interface PublicDashboardFooterCfg {
  hide: boolean;
  text: string;
  logo: string;
  link: string;
}

export const PublicDashboardFooter = function () {
  const styles = useStyles2(getStyles);
  const conf = getPublicDashboardFooterConfig();

  return conf.hide ? null : (
    <div className={styles.footer}>
      <span className={styles.logoText}>
        <a href={conf.link} target="_blank" rel="noreferrer noopener">
          {conf.text} <img className={styles.logoImg} alt="" src={conf.logo}></img>
        </a>
      </span>
    </div>
  );
};

export function setPublicDashboardFooterConfigFn(fn: typeof getPublicDashboardFooterConfig) {
  getPublicDashboardFooterConfig = fn;
}
export let getPublicDashboardFooterConfig = (): PublicDashboardFooterCfg => ({
  hide: false,
  text: 'powered by Grafana',
  logo: 'public/img/grafana_icon.svg',
  link: 'https://grafana.com/',
});

const getStyles = (theme: GrafanaTheme2) => ({
  footer: css`
    display: flex;
    justify-content: end;
    height: 30px;
    padding: ${theme.spacing(0, 1, 0, 1)};
  `,
  logoText: css`
    margin-right: ${theme.spacing(1)};
  `,
  logoImg: css`
    height: 100%;
    padding: ${theme.spacing(0.25, 0, 0.5, 0)};
  `,
});
