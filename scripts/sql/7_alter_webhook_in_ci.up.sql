INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (1, 'repository ssh url', 'repository.ssh_url', 'f', 'f', 't', NULL, NOW()),
       (3, 'repository ssh url', 'repository.ssh_url', 'f', 'f', 't', NULL, NOW());
