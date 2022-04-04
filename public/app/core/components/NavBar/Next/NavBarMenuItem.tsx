import React from 'react';
import { GrafanaTheme2 } from '@grafana/data';
import { Icon, IconName, Link, useTheme2 } from '@grafana/ui';
import { css, cx } from '@emotion/css';

export interface Props {
  icon?: IconName;
  isActive?: boolean;
  isDivider?: boolean;
  onClick?: () => void;
  styleOverrides?: string;
  target?: HTMLAnchorElement['target'];
  text: React.ReactNode;
  url?: string;
  adjustHeightForBorder?: boolean;
  isMobile?: boolean;
}

export function NavBarMenuItem({
  icon,
  isActive,
  isDivider,
  onClick,
  styleOverrides,
  target,
  text,
  url,
  isMobile = false,
}: Props) {
  const theme = useTheme2();
  const styles = getStyles(theme, isActive);
  const elStyle = cx(styles.element, styleOverrides);

  const linkContent = (
    <div className={styles.linkContent}>
      <span>{text}</span>
      {target === '_blank' && (
        <Icon data-testid="external-link-icon" name="external-link-alt" className={styles.externalLinkIcon} />
      )}
    </div>
  );

  let element = (
    <button className={elStyle} onClick={onClick} tabIndex={-1}>
      {linkContent}
    </button>
  );

  if (url) {
    element =
      !target && url.startsWith('/') ? (
        <Link className={elStyle} href={url} target={target} onClick={onClick} tabIndex={!isMobile ? -1 : 0}>
          {linkContent}
        </Link>
      ) : (
        <a href={url} target={target} className={elStyle} onClick={onClick} tabIndex={!isMobile ? -1 : 0}>
          {linkContent}
        </a>
      );
  }

  if (isMobile) {
    return isDivider ? (
      <li data-testid="dropdown-child-divider" className={styles.divider} tabIndex={-1} aria-disabled />
    ) : (
      <li className={styles.listItem}>{element}</li>
    );
  }

  return isDivider ? (
    <div data-testid="dropdown-child-divider" className={styles.divider} tabIndex={-1} aria-disabled />
  ) : (
    <div style={{ position: 'relative' }}>{element}</div>
  );
}

NavBarMenuItem.displayName = 'NavBarMenuItem';

const getStyles = (theme: GrafanaTheme2, isActive: Props['isActive']) => ({
  linkContent: css({
    display: 'grid',
    placeItems: 'center',
    gridAutoFlow: 'column',
    gap: '0.5rem',
  }),
  externalLinkIcon: css({
    color: theme.colors.text.secondary,
    gridColumnStart: 3,
  }),
  element: css({
    alignItems: 'center',
    background: 'none',
    border: 'none',
    color: isActive ? theme.colors.text.primary : theme.colors.text.secondary,
    display: 'flex',
    fontSize: 'inherit',
    height: '100%',
    padding: '5px 12px 5px 10px',
    textAlign: 'left',
    whiteSpace: 'nowrap',

    '&:focus-visible + .pin-button': {
      opacity: '100%',
    },

    '&:focus-visible': {
      outline: 'none',
      boxShadow: 'none',

      '&::after': {
        boxShadow: 'none',
        outline: `${theme.shape.borderRadius} solid ${theme.colors.primary.main}`,
        outlineOffset: `-${theme.shape.borderRadius(1)}`,
        transition: 'none',
      },
    },

    '&::before': {
      display: isActive ? 'block' : 'none',
      content: '" "',
      position: 'absolute',
      left: 0,
      top: 0,
      bottom: 0,
      width: theme.spacing(0.5),
      borderRadius: theme.shape.borderRadius(1),
      backgroundImage: theme.colors.gradients.brandVertical,
    },

    '&::after': {
      position: 'absolute',
      content: '" "',
      left: 0,
      top: 0,
      bottom: 0,
      right: 0,
    },
  }),
  listItem: css({
    position: 'relative',
    display: 'flex',
    alignItems: 'center',

    '&:hover, &:focus-within': {
      color: theme.colors.text.primary,

      '> *:first-child::after': {
        backgroundColor: theme.colors.action.hover,
      },
    },

    '> .pin-button': {
      opacity: 0,
    },

    '&:hover > .pin-button, &:focusVisible > .pin-button': {
      opacity: '100%',
    },
  }),
  divider: css({
    borderBottom: `1px solid ${theme.colors.border.weak}`,
    height: '1px',
    margin: `${theme.spacing(1)} 0`,
    overflow: 'hidden',
  }),
});
