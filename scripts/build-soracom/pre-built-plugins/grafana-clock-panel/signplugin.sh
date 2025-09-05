#!/bin/sh
if [[ -z "${GRAFANA_API_KEY}" ]]; then
  echo "GRAFANA_API_KEY environment variable not defined.  Check 1Password and define before running"
fi
npx @grafana/toolkit plugin:sign --rootUrls https://jp.lagoon.soracom.io,https://jp-v2.lagoon.soracom.io,https://jp-v2-staging.lagoon.soracom.io,https://jp-v2-dev.lagoon.soracom.io,https://jp-v3.lagoon.soracom.io,https://jp-v3-dev.lagoon.soracom.io,https://jp-v3-staging.lagoon.soracom.io,https://g.lagoon.soracom.io,https://g-v2.lagoon.soracom.io,https://g-v2-staging.lagoon.soracom.io,https://g-v3.lagoon.soracom.io,https://g-v3-staging.lagoon.soracom.io,http://localhost:3000