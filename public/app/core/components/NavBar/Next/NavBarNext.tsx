import React, { useState } from 'react';
import { useLocation } from 'react-router-dom';
import CSSTransition from 'react-transition-group/CSSTransition';
import { css, cx } from '@emotion/css';
import { cloneDeep } from 'lodash';
import { GrafanaTheme2, NavModelItem, NavSection } from '@grafana/data';
import { Icon, IconButton, IconName, useStyles2, useTheme2 } from '@grafana/ui';
import { config, locationService } from '@grafana/runtime';
import { getKioskMode } from 'app/core/navigation/kiosk';
import { KioskMode, StoreState } from 'app/types';
import { enrichConfigItems, getActiveItem, isMatchOrChildMatch, isSearchActive, SEARCH_ITEM_ID } from '../utils';
import { OrgSwitcher } from '../../OrgSwitcher';
import { NavBarMenu } from './NavBarMenu';
import NavBarItem from './NavBarItem';
import { useSelector } from 'react-redux';
import { NavBarItemWithoutMenu } from './NavBarItemWithoutMenu';
import { FocusScope } from '@react-aria/focus';
import { NavBarContext } from '../context';

const onOpenSearch = () => {
  locationService.partial({ search: 'open' });
};

const searchItem: NavModelItem = {
  id: SEARCH_ITEM_ID,
  onClick: onOpenSearch,
  text: 'Search dashboards',
  icon: 'search',
};

// Here we need to hack in a "home" NavModelItem since this is constructed in the frontend
const homeItem: NavModelItem = {
  id: 'home',
  text: 'Home',
  url: config.appSubUrl || '/',
  icon: 'grafana',
};

export const NavBarNext = React.memo(() => {
  const navBarTree = useSelector((state: StoreState) => state.navBarTree);
  const theme = useTheme2();
  const styles = getStyles(theme);
  const animStyles = useStyles2(getAnimStyles);
  const location = useLocation();
  const kiosk = getKioskMode();
  const [showSwitcherModal, setShowSwitcherModal] = useState(false);
  const toggleSwitcherModal = () => {
    setShowSwitcherModal(!showSwitcherModal);
  };
  const navTree = cloneDeep(navBarTree);
  navTree.unshift(homeItem);

  const coreItems = navTree.filter((item) => item.section === NavSection.Core);
  const pluginItems = navTree.filter((item) => item.section === NavSection.Plugin);
  const configItems = enrichConfigItems(
    navTree.filter((item) => item.section === NavSection.Config),
    location,
    toggleSwitcherModal
  );
  const activeItem = isSearchActive(location) ? searchItem : getActiveItem(navTree, location.pathname);
  const [menuOpen, setMenuOpen] = useState(false);
  const [menuIdOpen, setMenuIdOpen] = useState<string | null>(null);

  if (kiosk !== KioskMode.Off) {
    return null;
  }

  return (
    <div className={styles.navWrapper}>
      <nav className={cx(styles.sidemenu, 'sidemenu')} data-testid="sidemenu" aria-label="Main menu">
        <NavBarContext.Provider
          value={{
            menuIdOpen: menuIdOpen,
            setMenuIdOpen: setMenuIdOpen,
          }}
        >
          <FocusScope>
            <div className={styles.mobileSidemenuLogo} onClick={() => setMenuOpen(!menuOpen)} key="hamburger">
              <Icon name="bars" size="xl" />
            </div>

            <ul className={styles.itemList}>
              <NavBarItemWithoutMenu
                isActive={isMatchOrChildMatch(homeItem, activeItem)}
                label="Home"
                className={styles.grafanaLogo}
                url={homeItem.url}
              >
                <Icon name="grafana" size="xl" />
              </NavBarItemWithoutMenu>
              <NavBarItem className={styles.search} isActive={activeItem === searchItem} link={searchItem}>
                <Icon name="search" size="xl" />
              </NavBarItem>

              {coreItems.map((link, index) => (
                <NavBarItem
                  key={`${link.id}-${index}`}
                  isActive={isMatchOrChildMatch(link, activeItem)}
                  link={{ ...link, subTitle: undefined, onClick: undefined }}
                >
                  {link.icon && <Icon name={link.icon as IconName} size="xl" />}
                  {link.img && <img src={link.img} alt={`${link.text} logo`} />}
                </NavBarItem>
              ))}

              {pluginItems.length > 0 &&
                pluginItems.map((link, index) => (
                  <NavBarItem key={`${link.id}-${index}`} isActive={isMatchOrChildMatch(link, activeItem)} link={link}>
                    {link.icon && <Icon name={link.icon as IconName} size="xl" />}
                    {link.img && <img src={link.img} alt={`${link.text} logo`} />}
                  </NavBarItem>
                ))}

              {configItems.map((link, index) => (
                <NavBarItem
                  key={`${link.id}-${index}`}
                  isActive={isMatchOrChildMatch(link, activeItem)}
                  reverseMenuDirection
                  link={link}
                  className={cx({ [styles.verticalSpacer]: index === 0 })}
                >
                  {link.icon && <Icon name={link.icon as IconName} size="xl" />}
                  {link.img && <img src={link.img} alt={`${link.text} logo`} />}
                </NavBarItem>
              ))}
            </ul>
          </FocusScope>
        </NavBarContext.Provider>
      </nav>
      {showSwitcherModal && <OrgSwitcher onDismiss={toggleSwitcherModal} />}
      <div className={styles.menuWrapper}>
        <CSSTransition in={menuOpen} classNames={animStyles} timeout={150} unmountOnExit>
          <NavBarMenu
            activeItem={activeItem}
            navItems={[homeItem, searchItem, ...coreItems, ...pluginItems, ...configItems]}
            onClose={() => setMenuOpen(false)}
          />
        </CSSTransition>
        <IconButton
          name={menuOpen ? 'angle-left' : 'angle-right'}
          className={styles.menuToggle}
          size="xl"
          onClick={() => setMenuOpen(!menuOpen)}
        />
      </div>
    </div>
  );
});

NavBarNext.displayName = 'NavBarNext';

const getStyles = (theme: GrafanaTheme2) => ({
  navWrapper: css({
    position: 'relative',
    display: 'flex',

    '.sidemenu-hidden &': {
      display: 'none',
    },
  }),
  sidemenu: css({
    label: 'sidemenu',
    display: 'flex',
    flexDirection: 'column',
    backgroundColor: theme.colors.background.primary,
    zIndex: theme.zIndex.sidemenu,
    padding: `${theme.spacing(1)} 0`,
    position: 'relative',
    width: theme.spacing(7),
    borderRight: `1px solid ${theme.colors.border.weak}`,

    [theme.breakpoints.down('md')]: {
      position: 'fixed',
      paddingTop: '0px',
      backgroundColor: 'inherit',
      borderRight: 0,
    },
  }),
  mobileSidemenuLogo: css({
    alignItems: 'center',
    cursor: 'pointer',
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'space-between',
    padding: theme.spacing(2),

    [theme.breakpoints.up('md')]: {
      display: 'none',
    },
  }),
  itemList: css({
    backgroundColor: 'inherit',
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
    '> *': {
      height: theme.spacing(6),
    },

    [theme.breakpoints.down('md')]: {
      visibility: 'hidden',
    },
  }),
  grafanaLogo: css({
    alignItems: 'center',
    display: 'flex',
    img: {
      height: theme.spacing(3),
      width: theme.spacing(3),
    },
    justifyContent: 'center',
  }),
  search: css({
    display: 'none',
    marginTop: 0,

    [theme.breakpoints.up('md')]: {
      display: 'grid',
    },
  }),
  verticalSpacer: css({
    marginTop: 'auto',
  }),
  hideFromMobile: css({
    [theme.breakpoints.down('md')]: {
      display: 'none',
    },
  }),
  menuWrapper: css({
    position: 'fixed',
    display: 'grid',
    gridAutoFlow: 'column',
    height: '100%',
    zIndex: theme.zIndex.sidemenu,
  }),
  menuToggle: css({
    backgroundColor: theme.colors.background.secondary,
    border: `1px solid ${theme.colors.border.weak}`,
    position: 'absolute',
    marginRight: 0,
    top: '43px',
    right: '0px',
    zIndex: theme.zIndex.sidemenu,
    transform: `translateX(calc(${theme.spacing(7)} + 50%))`,
    borderRadius: '50%',

    [theme.breakpoints.down('md')]: {
      display: 'none',
    },
  }),
});

const getAnimStyles = (theme: GrafanaTheme2) => {
  const transitionProps = {
    transitionProperty: 'width, background-color',
    transitionDuration: '150ms',
    transitionTimingFunction: 'ease-in-out',
  };

  const openStyles = {
    backgroundColor: theme.colors.background.canvas,
    width: '300px',
  };

  const closedStyles = {
    backgroundColor: theme.colors.background.primary,
    width: theme.spacing(7),
  };

  const buttonShift = {
    '& + button': {
      transform: 'translateX(0%)',
    },
  };

  return {
    enter: css({
      ...closedStyles,
      ...buttonShift,
    }),
    enterActive: css({
      ...transitionProps,
      ...openStyles,
      ...buttonShift,
    }),
    enterDone: css({
      ...openStyles,
      ...buttonShift,
    }),
    exit: css({
      ...openStyles,
      ...buttonShift,
    }),
    exitActive: css({
      ...transitionProps,
      ...closedStyles,
      ...buttonShift,
    }),
  };
};
