import { css } from '@emotion/css';
import React, { FC } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import { Trans } from 'app/core/internationalization';

const helpOptions = [
  { value: 0, label: 'Documentation (English)', href: 'https://developers.soracom.io/en/docs/lagoon' },
  { value: 1, label: 'ドキュメント (日本語)', href: 'https://users.soracom.io/ja-jp/docs/lagoon-v3' },
  { value: 2, label: 'SORACOM ユーザーコンソール', href: 'https://console.soracom.io' },
];

export const WelcomeBanner: FC = () => {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.container}>
      <h1 className={styles.title}>
        <Trans i18nKey="welcome.title">Welcome to Lagoon</Trans>
      </h1>
      <div className={styles.help}>
        <h3 className={styles.helpText}>
          <Trans i18nKey="welcome.need-help">Need help?</Trans>
        </h3>
        <div className={styles.helpLinks}>
          {helpOptions.map((option, index) => {
            return (
              <a
                key={`${option.label}-${index}`}
                className={styles.helpLink}
                href={`${option.href}?utm_source=lagoon_gettingstarted`}
              >
                {option.label}
              </a>
            );
          })}
        </div>
      </div>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    container: css`
      display: flex;
      /// background: url(public/img/g8_home_v2.svg) no-repeat;
      background-size: cover;
      height: 100%;
      align-items: center;
      padding: 0 16px;
      justify-content: space-between;
      padding: 0 ${theme.spacing(3)};

      ${theme.breakpoints.down('lg')} {
        background-position: 0px;
        flex-direction: column;
        align-items: flex-start;
        justify-content: center;
      }

      ${theme.breakpoints.down('sm')} {
        padding: 0 ${theme.spacing(1)};
      }
    `,
    title: css`
      margin-bottom: 0;

      ${theme.breakpoints.down('lg')} {
        margin-bottom: ${theme.spacing(1)};
      }

      ${theme.breakpoints.down('md')} {
        font-size: ${theme.typography.h2.fontSize};
      }
      ${theme.breakpoints.down('sm')} {
        font-size: ${theme.typography.h3.fontSize};
      }
    `,
    help: css`
      display: flex;
      align-items: baseline;
    `,
    helpText: css`
      margin-right: ${theme.spacing(2)};
      margin-bottom: 0;

      ${theme.breakpoints.down('md')} {
        font-size: ${theme.typography.h4.fontSize};
      }

      ${theme.breakpoints.down('sm')} {
        display: none;
      }
    `,
    helpLinks: css`
      display: flex;
      flex-wrap: wrap;
    `,
    helpLink: css`
      margin-right: ${theme.spacing(2)};
      text-decoration: underline;
      text-wrap: no-wrap;

      ${theme.breakpoints.down('sm')} {
        margin-right: 8px;
      }
    `,
  };
};
