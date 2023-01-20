#!/bin/bash

# make sure we are disabling alerts that were migrated from lagoon v2
grep -Fq ALERT_DISABLED pkg/services/ngalert/store/alert_rule.go || { echo 'Source does not contain PR to DISABLE ALERTS using labels!' ; exit 1; }
# make sure we dont show admin users in the permission lists
grep -Fq filterAdminUsers pkg/services/accesscontrol/resourcepermissions/api.go || { echo 'Source does not contain PR to FILTER USERS in the permission list!' ; exit 1; }