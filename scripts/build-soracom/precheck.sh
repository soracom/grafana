#!/bin/bash

# make sure we are disabling alerts that were migrated from lagoon v2
grep -Fq ALERT_DISABLED pkg/services/ngalert/store/alert_rule.go || { echo 'Source does not contain PR to DISABLE ALERTS using labels!' ; exit 1; }