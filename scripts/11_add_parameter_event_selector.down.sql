/* changing webhook trigger in git_sensor  */
DELETE FROM git_host_webhook_event_selectors where  event_id=2 and name='pull request id';