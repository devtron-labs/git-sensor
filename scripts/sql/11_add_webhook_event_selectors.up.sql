INSERT INTO git_host_webhook_event_selectors(event_id,name,selector,to_show,to_show_in_ci_filter,is_active,created_on,updated_on,to_use_in_ci_env_variable) 
VALUES(1,'pull request number','pull_request.number',true,false,true,now(),now(),false),
(1,'github repo name','pull_request.head.repo.name',true,false,true,now(),now(),false);
