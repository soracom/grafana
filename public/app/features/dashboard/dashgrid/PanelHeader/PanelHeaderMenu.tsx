import React, { PureComponent } from 'react';

import { PanelMenuItem } from '@grafana/data';

import { PanelHeaderMenuItem } from './PanelHeaderMenuItem';

export interface Props {
  items: PanelMenuItem[];
  style?: React.CSSProperties;
}

export class PanelHeaderMenu extends PureComponent<Props> {
  renderItems = (menu: PanelMenuItem[], isSubMenu = false) => {
    return (
      <ul
        className="dropdown-menu dropdown-menu--menu panel-menu"
        style={this.props.style}
        role={isSubMenu ? '' : 'menu'}
      >
        {menu.map((menuItem, idx: number) => {
          return (
            <PanelHeaderMenuItem
              key={`${menuItem.text}${idx}`}
              type={menuItem.type}
              text={menuItem.text}
              iconClassName={menuItem.iconClassName}
              onClick={menuItem.onClick}
              shortcut={menuItem.shortcut}
            >
              {menuItem.subMenu && this.renderItems(menuItem.subMenu, true)}
            </PanelHeaderMenuItem>
          );
        })}
      </ul>
    );
  };

  render() {
    return <div className="panel-menu-container dropdown open">{this.renderItems(this.props.items)}</div>;
  }
}
