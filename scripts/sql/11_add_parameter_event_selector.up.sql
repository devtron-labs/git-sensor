/* changing webhook trigger in git_sensor */
INSERT INTO git_host_webhook_event_selectors(event_id,name,selector,to_show,to_show_in_ci_filter,is_active,created_on,updated_on,to_use_in_ci_env_variable) 
VALUES(2,'pull request id','pullrequest.id',true,false,true,now(),now(),true);