---
aliases:
  - /docs/grafana/latest/alerting/
  - /docs/grafana/latest/alerting/unified-alerting/alerting/
title: Alerting
weight: 114
---

# Grafana Alerting

Grafana Alerting allows you to learn about problems in your systems moments after they occur. Robust and actionable alerts help you identify and resolve issues quickly, minimizing disruption to your services. It centralizes alerting information in a single, searchable view that allows you to:

- Create and manage Grafana alerts
- Create and manage Grafana Mimir and Loki managed alerts
- View alerting information from Prometheus and Alertmanager compatible data sources

For new installations or existing installs without alerting configured, Grafana Alerting is enabled by default.

| Release     | Cloud         | Enterprise    | OSS           |
| ----------- | ------------- | ------------- | ------------- |
| Grafana 9.0 | On by default | On by default | On by default |

Existing installations that upgrade to v9.0 will have Grafana Alerting enabled by default. For more information on migrating from legacy or the cloud alerting plugin, see [Migrating to Grafana Alerting]({{< relref "migrating-alerts/" >}}).

Before you begin, we recommend that you familiarize yourself with some of the [fundamental concepts]({{< relref "fundamentals/" >}}) of Grafana Alerting. Refer to [Role-based access control]({{< relref "../enterprise/access-control/" >}}) in Grafana Enterprise to learn more about controlling access to alerts using role-based permissions.

- [About alert rules]({{< relref "fundamentals/alert-rules/" >}})
- [Migrating legacy alerts]({{< relref "migrating-alerts/" >}})
- [Disable Grafana Alerting in OSS]({{< relref "migrating-alerts/opt-out/" >}})
- [Create Grafana managed alerting rules]({{< relref "alerting-rules/create-grafana-managed-rule/" >}})
- [Create Grafana Mimir or Loki managed alerting rules]({{< relref "alerting-rules/create-mimir-loki-managed-rule/" >}})
- [View existing alerting rules and manage their current state]({{< relref "alerting-rules/rule-list/" >}})
- [View the state and health of alerting rules]({{< relref "fundamentals/state-and-health/" >}})
- [View alert groupings]({{< relref "alert-groups/" >}})
- [Add or edit an alert contact point]({{< relref "contact-points/" >}})
- [Add or edit notification policies]({{< relref "notifications/" >}})
- [Add or edit silences]({{< relref "silences/" >}})
- [Performance considerations for alerting]({{< relref "performance/" >}})
